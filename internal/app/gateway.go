package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/internal/api"
	"github.com/weibaohui/nanobot-go/internal/models"
	tasksvc "github.com/weibaohui/nanobot-go/internal/service/task"
	"github.com/weibaohui/nanobot-go/pkg/agent"
	"github.com/weibaohui/nanobot-go/pkg/agent/provider"
	"github.com/weibaohui/nanobot-go/pkg/bus"
	"github.com/weibaohui/nanobot-go/pkg/channels"
	"github.com/weibaohui/nanobot-go/pkg/channels/websocket"
	"github.com/weibaohui/nanobot-go/pkg/session"
	"go.uber.org/zap"
)

// Gateway 网关服务
type Gateway struct {
	Logger         *zap.Logger
	Config         *config.Config
	MessageBus     *bus.MessageBus
	DB             *DatabaseComponents
	SessionManager *session.Manager
	Providers      *api.Providers
	APIServer      *api.Server
	Hook           *HookComponents
	Loop           *agent.Loop
	ChannelManager *channels.Manager
	apiPort        int
	apiEnabled     bool
}

// GatewayOptions 网关选项
type GatewayOptions struct {
	Logger     *zap.Logger
	Config     *config.Config
	APIPort    int
	APIEnabled bool
}

// NewGateway 创建网关服务
func NewGateway(opts *GatewayOptions) *Gateway {
	return &Gateway{
		Logger:     opts.Logger,
		Config:     opts.Config,
		MessageBus: bus.NewMessageBus(opts.Logger),
		apiPort:    opts.APIPort,
		apiEnabled: opts.APIEnabled,
	}
}

// InitDatabase 初始化数据库
func (g *Gateway) InitDatabase() {
	g.DB = InitDatabase(g.Config, g.Logger)
}

// InitSessionManager 初始化会话管理器
func (g *Gateway) InitSessionManager() {
	var convRepo session.ConversationRecordRepository
	if g.DB != nil {
		convRepo = g.DB.ConvRepo
	}
	g.SessionManager = session.NewManager(g.Config, g.Logger, convRepo)
}

// InitAPI 初始化 API 服务
func (g *Gateway) InitAPI() {
	if g.DB == nil || g.DB.DB == nil {
		return
	}

	g.Providers = api.NewProviders(g.DB.DB.DB(), g.Config, g.Logger)

	// 注入 SessionManager 用于取消会话功能
	g.Providers.SessionManager = g.SessionManager

	if err := g.Providers.InitDefaultData(); err != nil {
		g.Logger.Error("初始化默认数据失败", zap.Error(err))
	} else {
		g.Logger.Info("Agent 管理系统已初始化")
	}
}

// StartAPIServer 启动 API 服务器（需要在 InitAgentLoop 之后调用）
func (g *Gateway) StartAPIServer() {
	if !g.apiEnabled || g.Providers == nil {
		return
	}

	// 注入 TaskManager（如果存在）
	if g.Loop != nil {
		g.Providers.TaskManager = g.Loop.GetTaskManager()
		// 初始化 TaskService
		g.Providers.TaskService = tasksvc.NewService(g.Providers.TaskManager)
	}

	apiAddr := ":" + strconv.Itoa(g.apiPort)
	g.APIServer = api.NewServer(apiAddr, g.Providers, g.Logger)

	// 注册 WebSocket 处理器
	wsHandler := NewWebSocketHandler(g, g.Logger)
	g.APIServer.SetWebSocketHandler(wsHandler)
	g.Logger.Info("WebSocket 处理器已注册")

	// 注册 Task WebSocket 处理器
	taskWSHandler := api.NewTaskWebSocketHandler(g.Logger)
	g.APIServer.SetTaskWebSocketHandler(taskWSHandler)
	g.Logger.Info("Task WebSocket 处理器已注册")

	// 订阅任务事件并通过 WebSocket 广播
	g.MessageBus.SubscribeTaskEvent(func(eventType string, payload map[string]any) {
		data, _ := json.Marshal(payload)
		taskWSHandler.Broadcast(data)
	})

	if err := g.APIServer.Start(); err != nil {
		g.Logger.Error("启动 API 服务器失败", zap.Error(err))
	}
}

// InitHookSystem 初始化 Hook 系统
func (g *Gateway) InitHookSystem() {
	g.Hook = InitHookSystem(g.Config, g.MessageBus, g.Logger)
}

// InitAgentLoop 初始化 Agent 循环
func (g *Gateway) InitAgentLoop() {
	if g.Providers == nil {
		g.Logger.Warn("Providers 未初始化，跳过 Agent Loop 初始化")
		return
	}

	maxIter := g.Config.Agents.MaxIterations
	if maxIter <= 0 {
		maxIter = 15
	}

	// 使用公共函数从数据库创建 ConfigLoader
	configLoader := provider.CreateConfigLoaderFromDB(g.DB.DB.DB(), g.Logger)

	g.Loop = agent.NewLoop(&agent.LoopConfig{
		ConfigLoader:   configLoader,
		MessageBus:     g.MessageBus,
		MaxIterations:  maxIter,
		SessionManager: g.SessionManager,
		Logger:         g.Logger,
		HookManager:    g.Hook.Manager,
		HookCallback:   g.Hook.Callback,
		ChannelService: g.Providers.ChannelService,
		AgentService:   g.Providers.AgentService,
		SessionService: g.Providers.SessionService,
		MCPService:     g.Providers.MCPService,
	})
}

// InitChannels 初始化渠道
func (g *Gateway) InitChannels() {
	g.ChannelManager = channels.NewManager(g.MessageBus)

	if g.DB != nil && g.DB.DB != nil {
		g.registerChannelsFromDB()
	}
}

// registerChannelsFromDB 从数据库注册渠道
func (g *Gateway) registerChannelsFromDB() {
	db := g.DB.DB.DB()
	var channelList []models.Channel
	if err := db.Where("is_active = ?", true).Find(&channelList).Error; err != nil {
		g.Logger.Error("从数据库读取渠道配置失败", zap.Error(err))
		return
	}

	for _, ch := range channelList {
		switch ch.Type {
		case models.ChannelTypeFeishu:
			var cfg models.FeishuChannelConfig
			if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil {
				g.Logger.Error("解析飞书渠道配置失败", zap.Error(err), zap.Uint("channel_id", ch.ID))
				continue
			}
			feishuConfig := &channels.FeishuConfig{
				AppID:             cfg.AppID,
				AppSecret:         cfg.AppSecret,
				EncryptKey:        cfg.EncryptKey,
				VerificationToken: cfg.VerificationToken,
				ChannelID:         ch.ID, // 设置数据库中的渠道ID
			}
			feishu := channels.NewFeishuChannel(feishuConfig, g.MessageBus, g.Logger)
			g.ChannelManager.Register(feishu)
			g.Logger.Info("已注册飞书渠道", zap.String("app_id", cfg.AppID))

		case models.ChannelTypeWebSocket:
			var cfg models.WebSocketChannelConfig
			if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil {
				g.Logger.Error("解析 WebSocket 渠道配置失败", zap.Error(err), zap.Uint("channel_id", ch.ID))
				continue
			}
			wsConfig := &websocket.Config{
				Addr:        cfg.Addr,
				Path:        cfg.Path,
				ChannelCode: ch.ChannelCode,
				ChannelID:   ch.ID,
				AgentCode:   ch.AgentCode,
			}
			wsChannel := websocket.NewChannel(wsConfig, g.MessageBus, g.Logger)
			g.ChannelManager.Register(wsChannel)
			g.Logger.Info("已注册 WebSocket 渠道", zap.String("channel_code", ch.ChannelCode))

		default:
			g.Logger.Warn("未知渠道类型", zap.String("type", string(ch.Type)), zap.Uint("channel_id", ch.ID))
		}
	}
}

// Start 启动网关
func (g *Gateway) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动消息分发器
	g.MessageBus.StartDispatcher(ctx)

	// 启动所有渠道
	if err := g.ChannelManager.StartAll(ctx); err != nil {
		return fmt.Errorf("启动渠道失败: %w", err)
	}

	// 启动 Agent 循环
	var wg sync.WaitGroup
	if g.Loop != nil {
		wg.Go(func() {
			if err := g.Loop.Run(ctx); err != nil {
				g.Logger.Error("代理循环错误", zap.Error(err))
			}
		})
	} else {
		g.Logger.Warn("Agent Loop 未初始化，跳过消息处理")
	}

	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	g.Logger.Info("正在关闭...")
	cancel()

	// 等待关闭
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		g.Logger.Info("代理循环已正常停止")
	case <-time.After(5 * time.Second):
		g.Logger.Warn("代理循环停止超时")
	}

	g.Shutdown()
	return nil
}

// Shutdown 关闭网关
func (g *Gateway) Shutdown() {
	if g.ChannelManager != nil {
		g.ChannelManager.StopAll()
	}

	if g.APIServer != nil {
		if err := g.APIServer.Stop(); err != nil {
			g.Logger.Error("停止 API 服务器失败", zap.Error(err))
		}
	}

	if g.DB != nil {
		g.DB.Close()
	}

	g.Logger.Info("已关闭")
}
