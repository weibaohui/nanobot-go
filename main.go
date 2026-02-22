package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/weibaohui/nanobot-go/agent"
	"github.com/weibaohui/nanobot-go/agent/hooks"
	"github.com/weibaohui/nanobot-go/bus"
	"github.com/weibaohui/nanobot-go/channels"
	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/cron"
	"github.com/weibaohui/nanobot-go/heartbeat"
	"github.com/weibaohui/nanobot-go/session"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	version   = "dev"
	buildDate = "unknown"
)

var (
	debugGlobal    bool
	agentMessage   string
	agentSession   string
	agentMarkdown  bool
	agentLogs      bool
	agentModel     string
	agentWorkspace string
	gatewayPort    int
	gatewayVerbose bool
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

var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "初始化配置",
	Long:  `初始化 nanobot 配置和工作区。`,
	Run:   runOnboard,
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

	rootCmd.AddCommand(gatewayCmd)
	rootCmd.AddCommand(onboardCmd)
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// ========== Gateway 命令实现 ==========

func runGateway(cmd *cobra.Command, args []string) {
	logger := initLogger(debugGlobal || gatewayVerbose)
	defer logger.Sync()

	cfg, workspacePath := loadConfigAndWorkspace(logger)

	logger.Info("nanobot gateway 启动中",
		zap.Int("端口", gatewayPort),
		zap.String("工作区", workspacePath),
		zap.String("版本", version),
		zap.String("构建时间", buildDate),
	)

	messageBus := bus.NewMessageBus(logger)

	dataDir := filepath.Join(workspacePath, ".nanobot")
	sessionManager := session.NewManager(cfg, dataDir)

	// 创建 Hook Manager
	hookManager := agent.NewHookManager()
	hookManager.SetLogger(logger)
	if cfg.Compress.Enabled {
		memory := agent.NewMemoryStore(workspacePath)
		compressLLM, err := hooks.CreateCompressLLM(cfg)
		if err != nil {
			logger.Error("创建压缩 LLM 失败", zap.Error(err))
		} else {
			compressHook := hooks.NewCompressHook(cfg, logger, memory, compressLLM)
			hookManager.Register(compressHook)
			logger.Info("对话压缩 Hook 已启用",
				zap.Int("minMessages", cfg.Compress.MinMessages),
				zap.Int("minTokens", cfg.Compress.MinTokens),
				zap.Int("maxHistory", cfg.Compress.MaxHistory),
			)
		}
	}

	cronStorePath := filepath.Join(dataDir, "cron_jobs.json")
	cronService := cron.NewService(cronStorePath, logger)

	maxIter := cfg.Agents.MaxIterations
	if maxIter <= 0 {
		maxIter = 15
	}
	execTimeout := cfg.Tools.Exec.Timeout
	if execTimeout <= 0 {
		execTimeout = 120
	}

	callbacks := agent.NewEinoCallbacks(true, logger)
	agent.RegisterGlobalCallbacks(callbacks)

	loop := agent.NewLoop(&agent.LoopConfig{
		Config:              cfg,
		MessageBus:          messageBus,
		Workspace:           workspacePath,
		MaxIterations:       maxIter,
		ExecTimeout:         execTimeout,
		RestrictToWorkspace: cfg.Tools.RestrictToWorkspace,
		CronService:         cronService,
		SessionManager:      sessionManager,
		Logger:              logger,
		HookManager:         hookManager,
	})

	ctx := context.Background()

	channelManager := channels.NewManager(messageBus)

	cliChannel := channels.NewCLIChannel(messageBus, "default", logger)
	channelManager.Register(cliChannel)

	// 注册配置中启用的渠道
	registerChannels(channelManager, cfg, messageBus, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动消息分发器，将出站消息分发给各渠道
	messageBus.StartDispatcher(ctx)

	if err := cronService.Start(ctx); err != nil {
		logger.Error("启动定时任务服务失败", zap.Error(err))
	}

	if err := channelManager.StartAll(ctx); err != nil {
		logger.Fatal("启动渠道失败", zap.Error(err))
	}

	// 创建并启动心跳服务
	heartbeatService := heartbeat.NewService(
		logger,
		cfg,
		workspacePath,
		func(ctx context.Context, cfg *config.Config, prompt string, model string, session string) (string, error) {
			agent := loop.GetMasterAgent()
			if agent == nil {
				logger.Error("MasterAgent 未初始化，跳过心跳处理")
				return "", fmt.Errorf("MasterAgent not initialized")
			}
			resp, err := agent.Process(ctx, &bus.InboundMessage{
				Channel: "heartbeat",
				Content: prompt,
			})
			if err != nil {
				logger.Error("处理心跳消息失败", zap.Error(err))
				return "", err
			}

			// 获取心跳目标并发送消息
			target := cfg.Heartbeat.Target
			if target != "" && target != "none" {
				messageBus.PublishOutbound(&bus.OutboundMessage{
					Channel: "heartbeat",
					ChatID:  target,
					Content: resp,
				})
			}
			return resp, nil
		},
	)
	if err := heartbeatService.Start(ctx); err != nil {
		logger.Error("启动心跳服务失败", zap.Error(err))
	}

	go func() {
		if err := loop.Run(ctx); err != nil {
			logger.Error("代理循环错误", zap.Error(err))
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("正在关闭...")
	cancel()
	cronService.Stop()
	heartbeatService.Stop()
	channelManager.StopAll()
	logger.Info("已关闭")
}

// ========== Onboard 命令实现 ==========

func runOnboard(cmd *cobra.Command, args []string) {
	logger := initLogger(debugGlobal)
	defer logger.Sync()

	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".nanobot")
	configPath := filepath.Join(configDir, "config.json")
	workspacePath := filepath.Join(configDir, "workspace")

	os.MkdirAll(configDir, 0755)

	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("配置已存在于 %s\n", configPath)
		fmt.Print("是否覆盖? (y/N): ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "y" && confirm != "Y" {
			fmt.Println("已取消")
			return
		}
	}

	cfg := createDefaultConfig()
	cfg.Agents.Defaults.Workspace = workspacePath

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "序列化配置失败: %s\n", err)
		os.Exit(1)
	}
	os.WriteFile(configPath, data, 0644)
	fmt.Printf("✓ 创建配置: %s\n", configPath)

	os.MkdirAll(workspacePath, 0755)
	fmt.Printf("✓ 创建工作区: %s\n", workspacePath)

	createWorkspaceTemplates(workspacePath)

	fmt.Println()
	fmt.Println("🐈 nanobot 已准备就绪!")
	fmt.Println()
	fmt.Println("下一步:")
	fmt.Println("  1. 在 ~/.nanobot/config.json 中添加 API Key")
	fmt.Println("     获取: https://openrouter.ai/keys")
	fmt.Println("  2. 聊天: nanobot agent -m \"你好!\"")
}

func createWorkspaceTemplates(workspace string) {
	templates := map[string]string{
		"AGENTS.md": `# 代理指令

你是一个有帮助的 AI 助手。保持简洁、准确和友好。

## 指南

- 在采取行动前解释你在做什么
- 当请求不明确时请求澄清
- 使用工具帮助完成任务
- 在内存文件中记住重要信息
`,
		"SOUL.md": `# 灵魂

我是 nanobot，一个轻量级的 AI 助手。

## 个性

- 有帮助且友好
- 简洁明了
- 好奇且渴望学习

## 价值观

- 准确性优于速度
- 用户隐私和安全
- 行动透明
`,
		"USER.md": `# 用户

用户信息放在这里。

## 偏好

- 沟通风格: (随意/正式)
- 时区: (你的时区)
- 语言: (你的首选语言)
`,
	}

	for filename, content := range templates {
		filePath := filepath.Join(workspace, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			os.WriteFile(filePath, []byte(content), 0644)
			fmt.Printf("  创建 %s\n", filename)
		}
	}

	memoryDir := filepath.Join(workspace, "memory")
	os.MkdirAll(memoryDir, 0755)

	memoryFile := filepath.Join(memoryDir, "MEMORY.md")
	if _, err := os.Stat(memoryFile); os.IsNotExist(err) {
		memoryContent := `# 长期内存

此文件存储跨会话持久化的重要信息。

## 用户信息

(关于用户的重要事实)

## 偏好

(随时间学习的用户偏好)

## 重要笔记

(需要记住的事情)
`
		os.WriteFile(memoryFile, []byte(memoryContent), 0644)
		fmt.Println("  创建 memory/MEMORY.md")
	}

	skillsDir := filepath.Join(workspace, "skills")
	os.MkdirAll(skillsDir, 0755)
}

// ========== 辅助函数 ==========

func initLogger(debug bool) *zap.Logger {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	level := zapcore.InfoLevel
	if debug {
		level = zapcore.DebugLevel
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.AddSync(os.Stderr),
		level,
	)

	return zap.New(core, zap.AddCaller())
}

func loadConfigAndWorkspace(logger *zap.Logger) (*config.Config, string) {
	workspace := agentWorkspace
	if workspace == "" {
		workspace = "."
	}

	workspacePath, err := filepath.Abs(workspace)
	if err != nil {
		logger.Fatal("解析工作区路径失败", zap.Error(err))
	}

	cfg, err := loadConfig("", workspacePath)
	if err != nil {
		logger.Fatal("加载配置失败", zap.Error(err))
	}

	// 如果配置文件中指定了 workspace 且命令行未指定，使用配置中的路径
	if agentWorkspace == "" && cfg.Agents.Defaults.Workspace != "" {
		workspacePath = config.GetWorkspacePath(cfg.Agents.Defaults.Workspace)
	}

	return cfg, workspacePath
}

func loadConfig(configPath, workspace string) (*config.Config, error) {
	path := configPath
	if path == "" {
		// 获取用户主目录
		homeDir, _ := os.UserHomeDir()

		defaultPaths := []string{
			filepath.Join(workspace, "nanobot.json"),
			filepath.Join(workspace, "config.json"),
			filepath.Join(workspace, "config", "nanobot.json"),
			filepath.Join(workspace, ".nanobot", "config.json"),
		}

		// 添加用户主目录下的配置路径
		if homeDir != "" {
			defaultPaths = append(defaultPaths, filepath.Join(homeDir, ".nanobot", "config.json"))
		}

		for _, p := range defaultPaths {
			if _, err := os.Stat(p); err == nil {
				path = p
				break
			}
		}
	}

	if path != "" {
		return config.LoadConfig(path)
	}

	return createDefaultConfig(), nil
}

func createDefaultConfig() *config.Config {
	return &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Model:       getEnvOrDefault("NANOBOT_MODEL", "gpt-4o-mini"),
				MaxTokens:   4096,
				Temperature: 0.7,
			},
			MaxIterations: 15,
		},
		Providers: config.ProvidersConfig{
			OpenAI: config.ProviderConfig{
				APIKey:  os.Getenv("OPENAI_API_KEY"),
				APIBase: os.Getenv("OPENAI_API_BASE"),
			},
			Anthropic: config.ProviderConfig{
				APIKey: os.Getenv("ANTHROPIC_API_KEY"),
			},
			DeepSeek: config.ProviderConfig{
				APIKey: os.Getenv("DEEPSEEK_API_KEY"),
			},
			OpenRouter: config.ProviderConfig{
				APIKey: os.Getenv("OPENROUTER_API_KEY"),
			},
			SiliconFlow: config.ProviderConfig{
				APIKey:  os.Getenv("SILICONFLOW_API_KEY"),
				APIBase: "https://api.siliconflow.cn/v1",
			},
		},
		Tools: config.ToolsConfig{
			Web: config.WebToolsConfig{
				Search: config.WebSearchConfig{
					MaxResults: 5,
				},
			},
			Exec: config.ExecToolConfig{
				Timeout: 120,
			},
			RestrictToWorkspace: true,
		},
		Gateway: config.GatewayConfig{
			Host: getEnvOrDefault("NANOBOT_HOST", "0.0.0.0"),
			Port: 8080,
		},
	}
}
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// registerChannels 根据配置注册启用的渠道
func registerChannels(mgr *channels.Manager, cfg *config.Config, messageBus *bus.MessageBus, logger *zap.Logger) {
	// WebSocket 渠道（默认启用）
	if cfg.Channels.WebSocket.Enabled {
		wsConfig := &channels.WebSocketConfig{
			Addr:      cfg.Channels.WebSocket.Addr,
			Path:      cfg.Channels.WebSocket.Path,
			AllowFrom: cfg.Channels.WebSocket.AllowFrom,
		}
		ws := channels.NewWebSocketChannel(wsConfig, messageBus, logger)
		mgr.Register(ws)
		if wsConfig.Addr != "" {
			logger.Info("已注册 WebSocket 渠道", zap.String("addr", wsConfig.Addr), zap.String("path", wsConfig.Path))
		} else {
			logger.Info("已注册 WebSocket 渠道", zap.String("addr", ":8088"), zap.String("path", "/ws"))
		}
	}

	// 钉钉渠道
	if cfg.Channels.DingTalk.Enabled {
		dingtalkConfig := &channels.DingTalkConfig{
			ClientID:     cfg.Channels.DingTalk.ClientID,
			ClientSecret: cfg.Channels.DingTalk.ClientSecret,
			AllowFrom:    cfg.Channels.DingTalk.AllowFrom,
		}
		dingtalk := channels.NewDingTalkChannel(dingtalkConfig, messageBus, logger)
		mgr.Register(dingtalk)
		logger.Info("已注册钉钉渠道")
	}

	// Matrix 渠道
	if cfg.Channels.Matrix.Enabled {
		matrixConfig := &channels.MatrixConfig{
			Homeserver: cfg.Channels.Matrix.Homeserver,
			UserID:     cfg.Channels.Matrix.UserID,
			Token:      cfg.Channels.Matrix.Token,
			AllowFrom:  cfg.Channels.Matrix.AllowFrom,
			DataDir:    cfg.Channels.Matrix.DataDir,
		}
		matrix := channels.NewMatrixChannel(matrixConfig, messageBus, logger)
		mgr.Register(matrix)
		logger.Info("已注册 Matrix 渠道",
			zap.String("homeserver", matrixConfig.Homeserver),
			zap.String("user_id", matrixConfig.UserID),
		)
	}

}
