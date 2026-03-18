package agent

import (
	"context"

	"github.com/weibaohui/nanobot-go/pkg/agent/hooks"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/pkg/agent/interrupt"
	"github.com/weibaohui/nanobot-go/pkg/agent/task"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/mcp"
	"github.com/weibaohui/nanobot-go/pkg/bus"
	"github.com/weibaohui/nanobot-go/pkg/cron"
	"github.com/weibaohui/nanobot-go/internal/service"
	agentsvc "github.com/weibaohui/nanobot-go/internal/service/agent"
	mcpsvc "github.com/weibaohui/nanobot-go/internal/service/mcp"
	"github.com/weibaohui/nanobot-go/pkg/session"
	"go.uber.org/zap"
)

// Loop 代理循环核心
type Loop struct {
	bus                 *bus.MessageBus
	configLoader        LLMConfigLoader
	workspace           string
	maxIterations       int
	execTimeout         int
	restrictToWorkspace bool
	cronService         *cron.Service
	context             *ContextBuilder
	sessions            *session.Manager
	tools               *tools.Registry
	running             bool
	logger              *zap.Logger
	hookManager         *hooks.HookManager
	hookCallback        func(eventType events.EventType, data map[string]interface{}) // Hook 回调
	channelService      service.ChannelService // 渠道服务，用于获取渠道绑定的 Agent
	agentService        agentsvc.Service       // Agent 服务，用于获取 Agent 配置
	sessionService      service.SessionService // Session 服务，用于创建和管理数据库 Session
	mcpService          mcpsvc.Service         // MCP 服务
	mcpManager          *mcp.Manager           // MCP 会话管理器

	interruptManager *interrupt.Manager
	masterAgent      *MasterAgent
	taskManager      *task.Manager
}

// LoopConfig Loop 配置
type LoopConfig struct {
	ConfigLoader        LLMConfigLoader
	MessageBus          *bus.MessageBus
	Workspace           string
	MaxIterations       int
	ExecTimeout         int
	RestrictToWorkspace bool
	CronService         *cron.Service
	SessionManager      *session.Manager
	Logger              *zap.Logger
	HookManager         *hooks.HookManager                                            // Hook 系统管理器
	HookCallback        func(eventType events.EventType, data map[string]interface{}) // Hook 回调
	ChannelService      service.ChannelService                                       // 渠道服务
	AgentService        agentsvc.Service                                             // Agent 服务
	SessionService      service.SessionService                                       // Session 服务
	MCPService          mcpsvc.Service                                               // MCP 服务
}

// NewLoop 创建代理循环
func NewLoop(cfg *LoopConfig) *Loop {
	if cfg == nil {
		return nil
	}

	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	loop := &Loop{
		bus:                 cfg.MessageBus,
		configLoader:        cfg.ConfigLoader,
		workspace:           cfg.Workspace,
		maxIterations:       cfg.MaxIterations,
		execTimeout:         cfg.ExecTimeout,
		restrictToWorkspace: cfg.RestrictToWorkspace,
		cronService:         cfg.CronService,
		context:             NewContextBuilder(cfg.Workspace),
		sessions:            cfg.SessionManager,
		tools:               tools.NewRegistry(),
		logger:              logger,
		hookManager:         cfg.HookManager,
		hookCallback:        cfg.HookCallback,
		channelService:      cfg.ChannelService,
		agentService:        cfg.AgentService,
		sessionService:      cfg.SessionService,
		mcpService:          cfg.MCPService,
	}

	// 初始化 MCP 管理器（如果配置了 MCP 服务）
	if cfg.MCPService != nil {
		loop.mcpManager = mcp.NewManager(cfg.MCPService, logger)
		logger.Info("MCP 管理器已初始化")
	}

	// 设置工具的 HookManager，使工具执行时能触发 Hook 事件
	if cfg.HookManager != nil {
		loop.tools.SetHookManager(cfg.HookManager, logger)
	}

	loop.interruptManager = interrupt.NewManager(cfg.MessageBus, logger)

	loop.registerDefaultTools()

	loop.taskManager = loop.createBackgroundAgentTaskManager()
	if loop.taskManager != nil {
		adapter := task.NewAdapter(loop.taskManager)
		loop.registerTaskTools(adapter)
	}

	ctx := context.Background()
	toolNames := loop.tools.GetToolNames(ctx)
	logger.Info("已注册工具",
		zap.Int("数量", len(toolNames)),
		zap.Strings("工具列表", toolNames),
	)

	if loop.taskManager != nil {
		loop.taskManager.SetRegisteredTools(toolNames)
	}

	adapter, err := NewChatModelAdapter(logger, loop.configLoader, loop.sessions)
	if err != nil {
		logger.Error("创建 Provider 适配器失败", zap.Error(err))
		return loop
	}

	adapter.SetSkillLoader(loop.context.GetSkillsLoader().LoadSkill)
	adapter.SetRegisteredTools(toolNames)
	adapter.SetHookCallback(loop.hookCallback)

	adkTools := loop.tools.GetToolsByNames(toolNames)

	masterAgent, err := NewMasterAgent(ctx, &MasterAgentConfig{
		ConfigLoader:    loop.configLoader,
		Workspace:       loop.workspace,
		Tools:           adkTools,
		Logger:          logger,
		Sessions:        cfg.SessionManager,
		Bus:             cfg.MessageBus,
		Context:         loop.context,
		InterruptMgr:    loop.interruptManager,
		CheckpointStore: loop.interruptManager.GetCheckpointStore(),
		MaxIterations:   cfg.MaxIterations,
		RegisteredTools: toolNames,
		HookManager:     loop.hookManager,
	})
	if err != nil {
		logger.Error("创建 Master Agent 失败，将使用传统模式", zap.Error(err))
		loop.masterAgent = nil
	} else {
		loop.masterAgent = masterAgent
	}
	logger.Info("Loop 初始化完成", zap.Bool("has_master_agent", loop.masterAgent != nil))

	return loop
}

// Stop 停止代理循环
func (l *Loop) Stop() {
	l.running = false
	l.logger.Info("代理循环正在停止")
}

// GetMasterAgent 获取 Master Agent
func (l *Loop) GetMasterAgent() *MasterAgent {
	if l.masterAgent == nil {
		l.logger.Warn("GetMasterAgent() 被调用但 MasterAgent 未初始化")
	}
	return l.masterAgent
}

// GetTaskManager 获取任务管理器
func (l *Loop) GetTaskManager() *task.Manager {
	return l.taskManager
}
