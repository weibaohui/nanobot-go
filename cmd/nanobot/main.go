package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/spf13/cobra"
	"github.com/weibaohui/nanobot-go/agent"
	"github.com/weibaohui/nanobot-go/agent/hooks"
	hookevents "github.com/weibaohui/nanobot-go/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/agent/hooks/observers"
	"github.com/weibaohui/nanobot-go/agent/hooks/trace"
	"github.com/weibaohui/nanobot-go/bus"
	"github.com/weibaohui/nanobot-go/channels"
	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/conversation/repository"
	"github.com/weibaohui/nanobot-go/cron"
	"github.com/weibaohui/nanobot-go/internal/api"
	"github.com/weibaohui/nanobot-go/internal/database"
	"github.com/weibaohui/nanobot-go/internal/models"
	"github.com/weibaohui/nanobot-go/internal/service"
	memoryhandler "github.com/weibaohui/nanobot-go/memory/handler"
	memoryjob "github.com/weibaohui/nanobot-go/memory/job"
	memoryrepo "github.com/weibaohui/nanobot-go/memory/repository"
	memoryservice "github.com/weibaohui/nanobot-go/memory/service"
	"github.com/weibaohui/nanobot-go/session"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"
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
	agentWorkspace string
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

// ========== Gateway 命令实现 ==========

func createDefaultConfig() *config.Config {
	return &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Model:       "",
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
		Database: config.DatabaseConfig{
			Enabled: true,
			// DataDir 留空，使用 database.DefaultConfig() 中的固定路径 (程序目录/data)
			DBName: "nanobot.db",
		},
	}
}
func runGateway(cmd *cobra.Command, args []string) {
	logger := initLogger(debugGlobal || gatewayVerbose)
	defer logger.Sync()

	cfg := createDefaultConfig()

	logger.Info("nanobot gateway 启动中",
		zap.Int("端口", gatewayPort),
		zap.String("工作区", cfg.GetWorkspacePath()),
		zap.String("版本", version),
		zap.String("构建时间", buildDate),
	)

	messageBus := bus.NewMessageBus(logger)

	dataDir := filepath.Join(cfg.GetWorkspacePath(), ".nanobot")

	// 初始化数据库和对话记录仓库
	var convRepo session.ConversationRecordRepository
	var dbClient *database.Client
	if dbConfig := database.NewConfigFromConfig(cfg); dbConfig != nil {
		var err error
		dbClient, err = database.NewClient(dbConfig)
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

	// 设置 Provider 加载器：优先从数据库读取 LLM 配置
	if dbClient != nil {
		setDatabaseProviderLoader(cfg, dbClient.DB(), logger)
	}

	// 初始化 Agent 管理系统
	var providers *api.Providers
	var apiServer *api.Server
	if dbClient != nil {
		providers = api.NewProviders(dbClient.DB())

		// 初始化默认数据
		if err := providers.InitDefaultData(); err != nil {
			logger.Error("初始化默认数据失败", zap.Error(err))
		} else {
			logger.Info("Agent 管理系统已初始化")
		}

		// 启动 API 服务器（如果启用）
		if apiEnabled {
			apiAddr := ":" + strconv.Itoa(apiPort)
			apiServer = api.NewServer(apiAddr, providers, logger)
			if err := apiServer.Start(); err != nil {
				logger.Error("启动 API 服务器失败", zap.Error(err))
			}
		}
	}

	// 初始化记忆模块（如果启用）
	var memoryService memoryservice.MemoryService
	var memoryEventHandler *memoryhandler.MemoryEventHandler
	var memoryUpgradeJob *memoryjob.MemoryUpgradeJob

	if cfg.Memory.Enabled && dbClient != nil {
		// 创建记忆仓库
		streamRepo := memoryrepo.NewStreamMemoryRepository(dbClient.DB())
		longTermRepo := memoryrepo.NewLongTermMemoryRepository(dbClient.DB())

		// 创建 LLM 客户端（使用系统整体配置）
		llmClient := memoryservice.NewSystemLLMClient(cfg, logger)

		// 创建总结器
		summarizer := memoryservice.NewMemorySummarizer(
			llmClient,
			cfg.Memory.Summarization.ConversationPrompt,
			cfg.Memory.Summarization.LongTermPrompt,
		)

		// 创建记忆服务
		memoryService = memoryservice.NewMemoryService(
			streamRepo,
			longTermRepo,
			summarizer,
			cfg.Memory.Enabled,
		)

		// 创建事件处理器
		memoryEventHandler = memoryhandler.NewMemoryEventHandler(
			memoryService,
			nil, // conversationSvc 暂时为 nil，避免循环依赖
			summarizer,
			logger,
			cfg.Memory.Enabled,
		)

		// 创建定时任务
		memoryUpgradeJob = memoryjob.NewMemoryUpgradeJob(
			memoryService,
			logger,
			cfg.Memory.Scheduled.Enabled,
			cfg.Memory.Scheduled.TimeWindow,
			cfg.Memory.Scheduled.Timezone,
		)

		logger.Info("记忆模块已初始化",
			zap.Bool("enabled", cfg.Memory.Enabled),
			zap.String("summarization_model", cfg.Memory.Summarization.Model),
		)
	}
	// 避免 unused 错误
	_ = memoryEventHandler

	// 创建统一的 Hook 系统
	hookSystem := hooks.NewHookManager(logger, true)

	// 注册 LoggingObserver
	loggingObserver := observers.NewLoggingObserver(logger, nil)
	hookSystem.Register(loggingObserver)
	logger.Info("日志观察器已注册到 Hook 系统",
		zap.Strings("events", []string{
			"message_received", "message_sent",
			"prompt_submitted", "system_prompt_built",
			"tool_call", "tool_intercepted", "tool_used", "tool_completed", "tool_error",
			"skill_call", "skill_lookup", "skill_used",
			"llm_call_start", "llm_call_end", "llm_call_error",
			"component_start", "component_end", "component_error",
		}),
	)

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

		// 从 data 中提取 trace 信息并设置到 context
		ctx := context.Background()
		var traceID string
		if tid, ok := data["trace_id"].(string); ok && tid != "" {
			traceID = tid
			ctx = hooks.WithTraceID(ctx, traceID)
		} else {
			traceID = hooks.GetTraceID(ctx)
		}
		if spanID, ok := data["span_id"].(string); ok && spanID != "" {
			ctx = trace.WithSpanID(ctx, spanID)
		}
		if parentSpanID, ok := data["parent_span_id"].(string); ok && parentSpanID != "" {
			ctx = trace.WithParentSpanID(ctx, parentSpanID)
		}

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
			tokenUsageRaw := data["token_usage"]
			logger.Debug("setHookCallback: 收到 EventLLMCallEnd",
				zap.String("session_key", sessionKey),
				zap.String("channel", channel),
				zap.Any("token_usage_raw", tokenUsageRaw),
				zap.Bool("token_usage_is_nil", tokenUsageRaw == nil),
			)
			if schemaUsage, ok := tokenUsageRaw.(*schema.TokenUsage); ok && schemaUsage != nil {
				event.TokenUsage = &model.TokenUsage{
					PromptTokens:            schemaUsage.PromptTokens,
					PromptTokenDetails:      model.PromptTokenDetails(schemaUsage.PromptTokenDetails),
					CompletionTokens:        schemaUsage.CompletionTokens,
					TotalTokens:             schemaUsage.TotalTokens,
					CompletionTokensDetails: model.CompletionTokensDetails(schemaUsage.CompletionTokensDetails),
				}
				logger.Debug("setHookCallback: TokenUsage 转换成功",
					zap.Int("prompt_tokens", event.TokenUsage.PromptTokens),
					zap.Int("completion_tokens", event.TokenUsage.CompletionTokens),
					zap.Int("total_tokens", event.TokenUsage.TotalTokens),
				)
			} else {
				logger.Debug("setHookCallback: TokenUsage 类型断言失败或为 nil",
					zap.Bool("is_token_usage", ok),
					zap.Bool("is_nil", tokenUsageRaw == nil),
				)
			}
			// 从 data 中提取其他字段
			if spanID, ok := data["span_id"].(string); ok {
				event.SpanID = spanID
			}
			if parentSpanID, ok := data["parent_span_id"].(string); ok {
				event.ParentSpanID = parentSpanID
			}
			logger.Debug("setHookCallback: 准备 Dispatch 事件",
				zap.String("session_key", sessionKey),
				zap.Bool("has_token_usage", event.TokenUsage != nil),
			)
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

	// 创建 LLM 配置加载器
	configLoader := func(ctx context.Context) (*agent.LLMConfig, error) {
		providerSvc := service.NewProviderService(dbClient.DB())
		svcConfig, err := providerSvc.GetLLMConfig(ctx, 0)
		if err != nil {
			return nil, fmt.Errorf("获取 LLM 配置失败: %w", err)
		}
		// 转换类型
		return &agent.LLMConfig{
			APIKey:       svcConfig.APIKey,
			APIBase:      svcConfig.APIBase,
			DefaultModel: svcConfig.DefaultModel,
			ExtraHeaders: svcConfig.ExtraHeaders,
		}, nil
	}

	loop := agent.NewLoop(&agent.LoopConfig{
		ConfigLoader:        configLoader,
		MessageBus:          messageBus,
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

	// 从数据库注册启用的渠道
	if dbClient != nil {
		registerChannelsFromDB(channelManager, dbClient.DB(), messageBus, logger)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动消息分发器，将出站消息分发给各渠道
	messageBus.StartDispatcher(ctx)

	if err := cronService.Start(ctx); err != nil {
		logger.Error("启动定时任务服务失败", zap.Error(err))
	}

	// 启动记忆升级定时任务（如果启用）
	if cfg.Memory.Enabled && memoryUpgradeJob != nil {
		// 启动定时器，每小时检查一次是否需要在时间窗口内执行
		go func() {
			ticker := time.NewTicker(1 * time.Hour)
			defer ticker.Stop()

			// 立即执行一次检查
			memoryUpgradeJob.Run()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					memoryUpgradeJob.Run()
				}
			}
		}()
		logger.Info("记忆升级定时任务已启动",
			zap.String("time_window", cfg.Memory.Scheduled.TimeWindow),
		)
	}

	if err := channelManager.StartAll(ctx); err != nil {
		logger.Fatal("启动渠道失败", zap.Error(err))
	}

	// 心跳服务已禁用：多租户场景下每个人应使用自己的定时任务
	// heartbeatService := (*heartbeat.Service)(nil)

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
	// heartbeat 服务已禁用
	channelManager.StopAll()

	// 停止 API 服务器
	if apiServer != nil {
		if err := apiServer.Stop(); err != nil {
			logger.Error("停止 API 服务器失败", zap.Error(err))
		}
	}

	logger.Info("已关闭")
}

// ========== Memory Upgrade 命令实现 ==========

func runMemoryUpgrade(cmd *cobra.Command, args []string) {
	logger := initLogger(debugGlobal)
	defer logger.Sync()

	// 解析日期参数
	targetDate := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if len(args) > 0 {
		// 验证日期格式
		if _, err := time.Parse("2006-01-02", args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "错误: 无效的日期格式 '%s'，请使用 YYYY-MM-DD 格式\n", args[0])
			os.Exit(1)
		}
		targetDate = args[0]
	}

	cfg := createDefaultConfig()

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

	// 创建记忆模块组件
	streamRepo := memoryrepo.NewStreamMemoryRepository(dbClient.DB())
	longTermRepo := memoryrepo.NewLongTermMemoryRepository(dbClient.DB())

	// 创建 LLM 客户端
	llmClient := memoryservice.NewSystemLLMClient(cfg, logger)

	// 创建总结器
	summarizer := memoryservice.NewMemorySummarizer(
		llmClient,
		cfg.Memory.Summarization.ConversationPrompt,
		cfg.Memory.Summarization.LongTermPrompt,
	)

	// 创建记忆服务
	memoryService := memoryservice.NewMemoryService(
		streamRepo,
		longTermRepo,
		summarizer,
		cfg.Memory.Enabled,
	)

	// 创建升级任务
	upgradeJob := memoryjob.NewMemoryUpgradeJob(
		memoryService,
		logger,
		cfg.Memory.Enabled,
		cfg.Memory.Scheduled.TimeWindow,
		cfg.Memory.Scheduled.Timezone,
	)

	fmt.Printf("开始执行记忆升级任务，目标日期: %s\n", targetDate)

	// 执行升级
	if err := upgradeJob.RunForDate(targetDate); err != nil {
		fmt.Fprintf(os.Stderr, "错误: 记忆升级失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ 记忆升级任务完成: %s\n", targetDate)
}

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

// registerChannelsFromDB 从数据库读取渠道配置并注册
func registerChannelsFromDB(mgr *channels.Manager, db *gorm.DB, messageBus *bus.MessageBus, logger *zap.Logger) {
	var channelList []models.Channel
	if err := db.Where("is_active = ?", true).Find(&channelList).Error; err != nil {
		logger.Error("从数据库读取渠道配置失败", zap.Error(err))
		return
	}

	for _, ch := range channelList {
		switch ch.Type {
		case models.ChannelTypeFeishu:
			var cfg models.FeishuChannelConfig
			if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil {
				logger.Error("解析飞书渠道配置失败", zap.Error(err), zap.Uint("channel_id", ch.ID))
				continue
			}
			feishuConfig := &channels.FeishuConfig{
				AppID:             cfg.AppID,
				AppSecret:         cfg.AppSecret,
				EncryptKey:        cfg.EncryptKey,
				VerificationToken: cfg.VerificationToken,
			}
			feishu := channels.NewFeishuChannel(feishuConfig, messageBus, logger)
			mgr.Register(feishu)
			logger.Info("已注册飞书渠道", zap.String("app_id", cfg.AppID))

		case models.ChannelTypeDingTalk:
			var cfg models.DingTalkChannelConfig
			if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil {
				logger.Error("解析钉钉渠道配置失败", zap.Error(err), zap.Uint("channel_id", ch.ID))
				continue
			}
			dingtalkConfig := &channels.DingTalkConfig{
				ClientID:     cfg.ClientID,
				ClientSecret: cfg.ClientSecret,
			}
			dingtalk := channels.NewDingTalkChannel(dingtalkConfig, messageBus, logger)
			mgr.Register(dingtalk)
			logger.Info("已注册钉钉渠道")

		case models.ChannelTypeMatrix:
			var cfg models.MatrixChannelConfig
			if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil {
				logger.Error("解析 Matrix 渠道配置失败", zap.Error(err), zap.Uint("channel_id", ch.ID))
				continue
			}
			matrixConfig := &channels.MatrixConfig{
				Homeserver: cfg.Homeserver,
				UserID:     cfg.UserID,
				Token:      cfg.Token,
			}
			matrix := channels.NewMatrixChannel(matrixConfig, messageBus, logger)
			mgr.Register(matrix)
			logger.Info("已注册 Matrix 渠道", zap.String("homeserver", cfg.Homeserver), zap.String("user_id", cfg.UserID))

		case models.ChannelTypeWebSocket:
			var cfg models.WebSocketChannelConfig
			if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil {
				logger.Error("解析 WebSocket 渠道配置失败", zap.Error(err), zap.Uint("channel_id", ch.ID))
				continue
			}
			wsConfig := &channels.WebSocketConfig{
				Addr: cfg.Addr,
				Path: cfg.Path,
			}
			ws := channels.NewWebSocketChannel(wsConfig, messageBus, logger)
			mgr.Register(ws)
			logger.Info("已注册 WebSocket 渠道", zap.String("addr", cfg.Addr), zap.String("path", cfg.Path))

		default:
			logger.Warn("未知渠道类型", zap.String("type", string(ch.Type)), zap.Uint("channel_id", ch.ID))
		}
	}
}

// setDatabaseProviderLoader 设置从数据库加载 Provider 配置的函数
func setDatabaseProviderLoader(cfg *config.Config, db *gorm.DB, logger *zap.Logger) {
	cfg.SetProviderLoader(func(model string) *config.ProviderConfig {
		// 查询数据库获取默认的 Provider
		var provider models.LLMProvider
		if err := db.Where("is_default = ? AND is_active = ?", true, true).First(&provider).Error; err != nil {
			logger.Warn("从数据库获取默认 Provider 失败，将使用配置文件", zap.Error(err))
			return nil
		}

		// 获取 extra headers
		extraHeaders := provider.GetExtraHeaders()

		logger.Info("从数据库加载 Provider 配置",
			zap.String("provider", provider.ProviderKey),
			zap.String("model", provider.DefaultModel),
			zap.String("api_base", provider.APIBase),
		)

		return &config.ProviderConfig{
			APIKey:       provider.APIKey,
			APIBase:      provider.APIBase,
			ExtraHeaders: extraHeaders,
		}
	})
}
