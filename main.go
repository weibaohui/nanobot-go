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
	"github.com/weibaohui/nanobot-go/bus"
	"github.com/weibaohui/nanobot-go/channels"
	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/cron"
	"github.com/weibaohui/nanobot-go/providers"
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

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "与代理交互",
	Long:  `直接与 nanobot 代理交互，支持单条消息或交互模式。`,
	Run:   runAgent,
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

	agentCmd.Flags().StringVarP(&agentMessage, "message", "m", "", "发送给代理的消息")
	agentCmd.Flags().StringVarP(&agentSession, "session", "s", "cli:default", "会话 ID")
	agentCmd.Flags().BoolVar(&agentMarkdown, "markdown", true, "渲染 Markdown 输出")
	agentCmd.Flags().BoolVar(&agentLogs, "logs", false, "显示运行时日志")
	agentCmd.Flags().StringVarP(&agentModel, "model", "M", "", "模型名称")
	agentCmd.Flags().StringVarP(&agentWorkspace, "workspace", "w", "", "工作区路径")

	gatewayCmd.Flags().IntVarP(&gatewayPort, "port", "p", 18790, "网关端口")
	gatewayCmd.Flags().BoolVarP(&gatewayVerbose, "verbose", "v", false, "详细输出")

	rootCmd.AddCommand(agentCmd)
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

// ========== Agent 命令实现 ==========

func runAgent(cmd *cobra.Command, args []string) {
	logger := initLogger(debugGlobal || agentLogs)
	defer logger.Sync()

	cfg, workspacePath := loadConfigAndWorkspace(logger)

	if agentModel != "" {
		cfg.Agents.Defaults.Model = agentModel
	}

	logger.Info("nanobot agent 启动",
		zap.String("工作区", workspacePath),
		zap.String("模型", cfg.Agents.Defaults.Model),
	)

	messageBus := bus.NewMessageBus(logger)
	provider := createProvider(cfg, logger)

	dataDir := filepath.Join(workspacePath, ".nanobot")
	sessionManager := session.NewManager(dataDir)

	maxIter := cfg.Agents.MaxIterations
	if maxIter <= 0 {
		maxIter = 15
	}
	execTimeout := cfg.Tools.ExecTimeout
	if execTimeout <= 0 {
		execTimeout = 120
	}

	loop := agent.NewLoop(
		messageBus,
		provider,
		workspacePath,
		cfg.Agents.Defaults.Model,
		maxIter,
		cfg.Tools.BraveAPIKey,
		execTimeout,
		cfg.Tools.RestrictToWorkspace,
		nil,
		sessionManager,
		logger,
	)

	ctx := context.Background()

	if agentMessage != "" {
		response, err := loop.ProcessDirect(ctx, agentMessage, agentSession, "cli", "default")
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: %s\n", err)
			os.Exit(1)
		}
		fmt.Println(response)
	} else {
		runInteractiveMode(ctx, loop, logger)
	}
}

func runInteractiveMode(ctx context.Context, loop *agent.Loop, logger *zap.Logger) {
	fmt.Println("🐈 nanobot 交互模式 (输入 'exit' 或按 Ctrl+C 退出)")
	fmt.Println()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n再见!")
		os.Exit(0)
	}()

	for {
		fmt.Print("You: ")
		var input string
		if _, err := fmt.Scanln(&input); err != nil {
			if err.Error() == "unexpected newline" {
				continue
			}
			break
		}

		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" || input == "/exit" || input == "/quit" {
			fmt.Println("\n再见!")
			break
		}

		response, err := loop.ProcessDirect(ctx, input, agentSession, "cli", "default")
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: %s\n", err)
			continue
		}

		fmt.Println()
		fmt.Println("🐈 nanobot")
		fmt.Println(response)
		fmt.Println()
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
	)

	messageBus := bus.NewMessageBus(logger)
	provider := createProvider(cfg, logger)

	dataDir := filepath.Join(workspacePath, ".nanobot")
	sessionManager := session.NewManager(dataDir)

	cronStorePath := filepath.Join(dataDir, "cron_jobs.json")
	cronService := cron.NewService(cronStorePath, logger)

	maxIter := cfg.Agents.MaxIterations
	if maxIter <= 0 {
		maxIter = 15
	}
	execTimeout := cfg.Tools.ExecTimeout
	if execTimeout <= 0 {
		execTimeout = 120
	}

	loop := agent.NewLoop(
		messageBus,
		provider,
		workspacePath,
		cfg.Agents.Defaults.Model,
		maxIter,
		cfg.Tools.BraveAPIKey,
		execTimeout,
		cfg.Tools.RestrictToWorkspace,
		cronService,
		sessionManager,
		logger,
	)

	channelManager := channels.NewManager(messageBus)

	cliChannel := channels.NewCLIChannel(messageBus, "default", logger)
	channelManager.Register(cliChannel)

	cronService.SetOnJobCallback(func(job *cron.Job) (string, error) {
		ctx := context.Background()
		return loop.ProcessDirect(ctx, job.Payload.Message, job.Payload.Channel+":"+job.Payload.To, job.Payload.Channel, job.Payload.To)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cronService.Start(ctx); err != nil {
		logger.Error("启动定时任务服务失败", zap.Error(err))
	}

	if err := channelManager.StartAll(ctx); err != nil {
		logger.Fatal("启动渠道失败", zap.Error(err))
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

	return cfg, workspacePath
}

func loadConfig(configPath, workspace string) (*config.Config, error) {
	path := configPath
	if path == "" {
		defaultPaths := []string{
			filepath.Join(workspace, "nanobot.json"),
			filepath.Join(workspace, "config", "nanobot.json"),
			filepath.Join(workspace, ".nanobot", "config.json"),
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
			BraveAPIKey:         os.Getenv("BRAVE_API_KEY"),
			ExecTimeout:         120,
			RestrictToWorkspace: true,
		},
		Gateway: config.GatewayConfig{
			Host: getEnvOrDefault("NANOBOT_HOST", "127.0.0.1"),
			Port: 8080,
		},
	}
}

func createProvider(cfg *config.Config, logger *zap.Logger) providers.LLMProvider {
	providerCfg := cfg.GetProvider(cfg.Agents.Defaults.Model)
	if providerCfg == nil || providerCfg.APIKey == "" {
		logger.Warn("未找到有效的 API Key，请设置环境变量")
		return providers.NewLiteLLMProvider("", "", "gpt-4o-mini", nil)
	}

	apiBase := providerCfg.APIBase
	if apiBase == "" {
		apiBase = "https://api.openai.com/v1"
	}

	return providers.NewLiteLLMProvider(
		providerCfg.APIKey,
		apiBase,
		cfg.Agents.Defaults.Model,
		providerCfg.ExtraHeaders,
	)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
