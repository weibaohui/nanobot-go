package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/weibaohui/nanobot-go/internal/api"
	"github.com/weibaohui/nanobot-go/internal/app"
	"github.com/weibaohui/nanobot-go/internal/database"
	"go.uber.org/zap"
)

var (
	version   = "dev"
	buildDate = "unknown"
)

var (
	debugGlobal    bool
	gatewayPort    int
	gatewayVerbose bool
	apiPort        int
	apiEnabled     bool
)

var rootCmd = &cobra.Command{
	Use:   "nanobot",
	Short: "🐈 nanobot - 个人 AI 助手",
	Long:  `🐈 nanobot - 一个轻量级的个人 AI 助手，支持多种渠道和工具。`,
}

var gatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "启动网关服务",
	Long:  `启动 nanobot 网关服务，监听所有启用的渠道。`,
	Run:   runGateway,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("🐈 nanobot-go %s (built %s)\n", version, buildDate)
	},
}

var memoryUpgradeCmd = &cobra.Command{
	Use:   "memory-upgrade [date]",
	Short: "手动触发记忆升级",
	Long: `手动触发记忆升级任务，将流水记忆提炼为长期记忆。
如果不指定日期，默认处理昨天的记录。
日期格式: YYYY-MM-DD
示例:
  nanobot memory-upgrade           # 处理昨天
  nanobot memory-upgrade 2026-03-07 # 处理指定日期`,
	Run: runMemoryUpgrade,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&debugGlobal, "debug", "d", false, "调试模式")

	gatewayCmd.Flags().IntVarP(&gatewayPort, "port", "p", 18790, "网关端口")
	gatewayCmd.Flags().BoolVarP(&gatewayVerbose, "verbose", "v", false, "详细输出")
	gatewayCmd.Flags().IntVar(&apiPort, "api-port", 8081, "API 服务端口（0 表示禁用）")
	gatewayCmd.Flags().BoolVar(&apiEnabled, "api", true, "启用管理 API")

	rootCmd.AddCommand(gatewayCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(memoryUpgradeCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runGateway(cmd *cobra.Command, args []string) {
	logger := app.InitLogger(debugGlobal || gatewayVerbose)
	defer logger.Sync()

	cfg := app.CreateDefaultConfig()

	logger.Info("nanobot gateway 启动中",
		zap.Int("端口", gatewayPort),
		zap.String("版本", version),
		zap.String("构建时间", buildDate),
	)

	// 创建网关
	gateway := app.NewGateway(&app.GatewayOptions{
		Logger:     logger,
		Config:     cfg,
		APIPort:    apiPort,
		APIEnabled: apiEnabled,
	})

	// 初始化各模块
	gateway.InitDatabase()
	gateway.InitSessionManager()

	// 验证 JWT 配置（仅在启用 API 时）
	if apiEnabled {
		if err := api.ValidateJWTConfig(); err != nil {
			logger.Warn("JWT 配置警告", zap.Error(err))
			logger.Warn("请设置 JWT_SECRET 环境变量以确保安全性")
		}
	}

	gateway.InitAPI()
	gateway.InitMemory()
	gateway.InitHookSystem()
	gateway.InitAgentLoop()
	gateway.InitChannels()

	// 启动 API 服务器（在 InitAgentLoop 之后，以便注入 TaskManager）
	gateway.StartAPIServer()

	// 启动网关
	if err := gateway.Start(); err != nil {
		logger.Fatal("启动网关失败", zap.Error(err))
	}
}

func runMemoryUpgrade(cmd *cobra.Command, args []string) {
	logger := app.InitLogger(debugGlobal)
	defer logger.Sync()

	// 解析日期参数
	targetDate := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if len(args) > 0 {
		if _, err := time.Parse("2006-01-02", args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "错误: 无效的日期格式 '%s'，请使用 YYYY-MM-DD 格式\n", args[0])
			os.Exit(1)
		}
		targetDate = args[0]
	}

	cfg := app.CreateDefaultConfig()

	if !cfg.Memory.Enabled {
		fmt.Println("记忆模块未启用，请在配置中设置 memory.enabled = true")
		os.Exit(1)
	}

	// 初始化数据库
	dbConfig := database.NewConfigFromConfig(cfg)
	if dbConfig == nil {
		fmt.Println("错误: 数据库配置无效")
		os.Exit(1)
	}

	dbClient, err := database.NewClient(dbConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 初始化数据库失败: %v\n", err)
		os.Exit(1)
	}
	defer dbClient.Close()

	if err := dbClient.InitSchema(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: 初始化数据库 schema 失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("开始执行记忆升级任务，目标日期: %s\n", targetDate)

	// 执行升级
	if err := app.RunMemoryUpgrade(cfg, dbClient.DB(), logger, targetDate); err != nil {
		fmt.Fprintf(os.Stderr, "错误: 记忆升级失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ 记忆升级任务完成: %s\n", targetDate)
}
