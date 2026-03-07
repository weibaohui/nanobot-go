package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	// "github.com/cloudwego/eino/callbacks" // 已移除，事件通过 provider.go 直接触发
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/spf13/cobra"
	"github.com/weibaohui/nanobot-go/agent"
	"github.com/weibaohui/nanobot-go/agent/hooks"
	hookevents "github.com/weibaohui/nanobot-go/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/agent/hooks/observers"
	"github.com/weibaohui/nanobot-go/conversation/database"
	"github.com/weibaohui/nanobot-go/conversation/repository"
	"github.com/weibaohui/nanobot-go/bus"
	"github.com/weibaohui/nanobot-go/channels"
	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/cron"
	"github.com/weibaohui/nanobot-go/heartbeat"
	"github.com/weibaohui/nanobot-go/session"
	"github.com/weibaohui/nanobot-go/token_usage"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/weibaohui/nanobot-go/internal/models"
)

// convRepoAdapter 将 repository.ConversationRecordRepository 适配为 session.ConversationRecordRepository
type convRepoAdapter struct {
	repo repository.ConversationRecordRepository
}

func newConvRepoAdapter(repo repository.ConversationRecordRepository) session.ConversationRecordRepository {
	return &convRepoAdapter{repo: repo}
}

func (a *convRepoAdapter) FindBySessionKey(ctx context.Context, sessionKey string, opts *models.QueryOptions) ([]models.ConversationRecord, error) {
	return a.repo.FindBySessionKey(ctx, sessionKey, opts)
}

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

	// 初始化数据库和对话记录仓库
	var convRepo session.ConversationRecordRepository
	if dbConfig := database.NewConfigFromConfig(cfg); dbConfig != nil {
		dbClient, err := database.NewClient(dbConfig)
		if err != nil {
			logger.Error("初始化数据库失败", zap.Error(err))
		} else {
			if err := dbClient.InitSchema(); err != nil {
				logger.Error("初始化数据库 schema 失败", zap.Error(err))
				dbClient.Close()
			} else {
				convRepo = newConvRepoAdapter(repository.NewConversationRecordRepository(dbClient.DB()))
				logger.Info("数据库和对话记录仓库已初始化")
			}
		}
	}

	sessionManager := session.NewManager(cfg, logger, dataDir, convRepo)
	tokenUsageManager := token_usage.NewTokenUsageManager(dataDir)

	// 创建统一的 Hook 系统
	hookSystem := hooks.NewHookManager(logger, true)

	// 注册 SessionObserver - 负责保存消息到会话
	sessionObserver := observers.NewSessionObserver(sessionManager, logger, nil)
	hookSystem.Register(sessionObserver)
	logger.Info("会话观察器已注册到 Hook 系统")

	// 注册 TokenUsageObserver - 负责记录 Token 使用量
	tokenUsageObserver := observers.NewTokenUsageObserver(tokenUsageManager, logger, nil)
	hookSystem.Register(tokenUsageObserver)
	logger.Info("Token 使用量观察器已注册到 Hook 系统")

	// 注册 LoggingObserver
	loggingObserver := observers.NewLoggingObserver(logger, nil)
	hookSystem.Register(loggingObserver)
	logger.Info("日志观察器已注册到 Hook 系统")

	// 如果启用了压缩，注册 CompressObserver
	if cfg.Compress.Enabled {
		memory := agent.NewMemoryStore(workspacePath)
		compressLLM, err := observers.CreateCompressLLM(cfg)
		if err != nil {
			logger.Error("创建压缩 LLM 失败", zap.Error(err))
		} else {
			compressObserver := observers.NewCompressObserver(cfg, logger, memory, compressLLM, sessionManager, nil)
			hookSystem.Register(compressObserver)
			logger.Info("对话压缩观察器已启用",
				zap.Int("minMessages", cfg.Compress.MinMessages),
				zap.Int("minTokens", cfg.Compress.MinTokens),
				zap.Int("maxHistory", cfg.Compress.MaxHistory),
			)
		}
	}

	// 如果启用了思考过程推送，注册 ThinkingProcessObserver
	if cfg.ThinkingProcess.Enabled {
		thinkingProcessObserver := observers.NewThinkingProcessObserver(&cfg.ThinkingProcess, messageBus, logger, nil)
		hookSystem.Register(thinkingProcessObserver)
		logger.Info("思考过程观察器已启用",
			zap.Bool("enabled", cfg.ThinkingProcess.Enabled),
			zap.Strings("events", cfg.ThinkingProcess.Events),
		)
	}

	// 注册 SQLiteObserver - 负责将所有事件存储到 SQLite 数据库
	if sqliteObserver, err := observers.NewSQLiteObserverFromConfig(cfg, logger, nil); err != nil {
		logger.Error("创建 SQLite 观察器失败", zap.Error(err))
	} else if sqliteObserver != nil {
		hookSystem.Register(sqliteObserver)
		logger.Info("SQLite 观察器已注册到 Hook 系统", zap.String("db_path", sqliteObserver.GetDBPath()))
	}

	// 注意：Eino Callback 已移除，事件通过 provider.go 直接触发
	// 如需恢复，取消下面这行的注释：
	// callbacks.AppendGlobalHandlers(hookSystem.EinoHandler())

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

	// 设置 Hook 回调，将 Loop 中的事件转发到 Hook 系统
	setHookCallback := func(eventType hookevents.EventType, data map[string]interface{}) {
		if !hookSystem.Enabled() {
			return
		}

		ctx := context.Background()
		traceID := hooks.GetTraceID(ctx)

		// 从 data 中提取 session_key 和 channel
		var sessionKey, channel string
		if sk, ok := data["session_key"].(string); ok {
			sessionKey = sk
		}
		if ch, ok := data["channel"].(string); ok {
			channel = ch
		}

		// 根据事件类型创建具体的事件对象
		switch eventType {
		case hookevents.EventLLMCallEnd:
			// 创建 LLMCallEndEvent，包含 Token 使用信息
			event := &hookevents.LLMCallEndEvent{
				BaseEvent: &hookevents.BaseEvent{
					TraceID:   traceID,
					EventType: eventType,
					Timestamp: time.Now(),
				},
			}
			// 从 data 中提取 TokenUsage（schema.TokenUsage 需要转换为 model.TokenUsage）
			if schemaUsage, ok := data["token_usage"].(*schema.TokenUsage); ok && schemaUsage != nil {
				event.TokenUsage = &model.TokenUsage{
					PromptTokens:            schemaUsage.PromptTokens,
					PromptTokenDetails:      model.PromptTokenDetails(schemaUsage.PromptTokenDetails),
					CompletionTokens:        schemaUsage.CompletionTokens,
					TotalTokens:             schemaUsage.TotalTokens,
					CompletionTokensDetails: model.CompletionTokensDetails(schemaUsage.CompletionTokensDetails),
				}
			}
			// 从 data 中提取其他字段
			if spanID, ok := data["span_id"].(string); ok {
				event.SpanID = spanID
			}
			if parentSpanID, ok := data["parent_span_id"].(string); ok {
				event.ParentSpanID = parentSpanID
			}
			hookSystem.Dispatch(ctx, event, channel, sessionKey)

		case hookevents.EventLLMCallStart:
			// 创建 LLMCallStartEvent
			event := &hookevents.LLMCallStartEvent{
				BaseEvent: &hookevents.BaseEvent{
					TraceID:   traceID,
					EventType: eventType,
					Timestamp: time.Now(),
				},
			}
			if spanID, ok := data["span_id"].(string); ok {
				event.SpanID = spanID
			}
			if parentSpanID, ok := data["parent_span_id"].(string); ok {
				event.ParentSpanID = parentSpanID
			}
			hookSystem.Dispatch(ctx, event, channel, sessionKey)

		default:
			// 其他事件类型，创建 BaseEvent
			baseEvent := &hookevents.BaseEvent{
				TraceID:   traceID,
				EventType: eventType,
				Timestamp: time.Now(),
			}
			hookSystem.Dispatch(ctx, baseEvent, channel, sessionKey)
		}
	}

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
		HookManager:         hookSystem,
		HookCallback:        setHookCallback,
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
			// 心跳使用固定 session key "heartbeat:"，所有心跳共享一个会话
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

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := loop.Run(ctx); err != nil {
			logger.Error("代理循环错误", zap.Error(err))
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("正在关闭...")
	cancel()

	// 等待 goroutine 完成（带超时）
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("代理循环已正常停止")
	case <-time.After(5 * time.Second):
		logger.Warn("代理循环停止超时")
	}

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

	// 飞书渠道
	if cfg.Channels.Feishu.Enabled {
		feishuConfig := &channels.FeishuConfig{
			AppID:             cfg.Channels.Feishu.AppID,
			AppSecret:         cfg.Channels.Feishu.AppSecret,
			EncryptKey:        cfg.Channels.Feishu.EncryptKey,
			VerificationToken: cfg.Channels.Feishu.VerificationToken,
			AllowFrom:         cfg.Channels.Feishu.AllowFrom,
		}
		feishu := channels.NewFeishuChannel(feishuConfig, messageBus, logger)
		mgr.Register(feishu)
		logger.Info("已注册飞书渠道",
			zap.String("app_id", feishuConfig.AppID),
		)
	}

}
