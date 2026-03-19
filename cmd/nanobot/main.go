package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/weibaohui/nanobot-go/internal/api"
	"github.com/weibaohui/nanobot-go/internal/app"
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


func init() {
	rootCmd.PersistentFlags().BoolVarP(&debugGlobal, "debug", "d", false, "调试模式")

	gatewayCmd.Flags().IntVarP(&gatewayPort, "port", "p", 18790, "网关端口")
	gatewayCmd.Flags().BoolVarP(&gatewayVerbose, "verbose", "v", false, "详细输出")
	gatewayCmd.Flags().IntVar(&apiPort, "api-port", 8081, "API 服务端口（0 表示禁用）")
	gatewayCmd.Flags().BoolVar(&apiEnabled, "api", true, "启用管理 API")

	rootCmd.AddCommand(gatewayCmd)
	rootCmd.AddCommand(versionCmd)
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

