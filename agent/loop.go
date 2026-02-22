package agent

import (
	"context"
	"fmt"

	"github.com/weibaohui/nanobot-go/agent/tools"
	"github.com/weibaohui/nanobot-go/agent/tools/askuser"
	toolcron "github.com/weibaohui/nanobot-go/agent/tools/cron"
	"github.com/weibaohui/nanobot-go/agent/tools/editfile"
	"github.com/weibaohui/nanobot-go/agent/tools/exec"
	"github.com/weibaohui/nanobot-go/agent/tools/listdir"
	"github.com/weibaohui/nanobot-go/agent/tools/message"
	"github.com/weibaohui/nanobot-go/agent/tools/readfile"
	"github.com/weibaohui/nanobot-go/agent/tools/skill"
	tasktool "github.com/weibaohui/nanobot-go/agent/tools/task"
	"github.com/weibaohui/nanobot-go/agent/tools/webfetch"
	"github.com/weibaohui/nanobot-go/agent/tools/websearch"
	"github.com/weibaohui/nanobot-go/agent/tools/writefile"
	"github.com/weibaohui/nanobot-go/bus"
	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/cron"
	"github.com/weibaohui/nanobot-go/session"
	"go.uber.org/zap"
)

// Loop 代理循环核心
type Loop struct {
	bus                 *bus.MessageBus
	cfg                 *config.Config
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
	hookManager         *HookManager

	interruptManager *InterruptManager
	supervisor       *SupervisorAgent
	masterAgent      *MasterAgent
	enableSupervisor bool
	taskManager      *AgentTaskManager
}

// LoopConfig Loop 配置
type LoopConfig struct {
	Config              *config.Config
	MessageBus          *bus.MessageBus
	Workspace           string
	MaxIterations       int
	ExecTimeout         int
	RestrictToWorkspace bool
	CronService         *cron.Service
	SessionManager      *session.Manager
	Logger              *zap.Logger
	HookManager         *HookManager
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
		cfg:                 cfg.Config,
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
	}

	loop.interruptManager = NewInterruptManager(cfg.MessageBus, logger)

	loop.registerDefaultTools()

	loop.taskManager = loop.createBackgroundAgentTaskManager()
	if loop.taskManager != nil {
		adapter := NewTaskManagerAdapter(loop.taskManager)
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

	adapter, err := NewChatModelAdapter(logger, loop.cfg, loop.sessions)
	if err != nil {
		logger.Error("创建 Provider 适配器失败", zap.Error(err))
		return loop
	}

	adapter.SetSkillLoader(loop.context.GetSkillsLoader().LoadSkill)
	adapter.SetRegisteredTools(toolNames)

	adkTools := loop.tools.GetToolsByNames(toolNames)

	supervisor, err := NewSupervisorAgent(ctx, &SupervisorConfig{
		Cfg:             loop.cfg,
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
		logger.Error("创建 Supervisor Agent 失败，将使用传统模式", zap.Error(err))
	} else {
		loop.supervisor = supervisor
		loop.enableSupervisor = true
		logger.Info("Supervisor Agent 创建成功，已启用入口型 Agent 模式")
	}

	masterAgent, err := NewMasterAgent(ctx, &MasterAgentConfig{
		Cfg:             loop.cfg,
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

// registerDefaultTools 注册默认工具
func (l *Loop) registerDefaultTools() {
	allowedDir := ""
	if l.restrictToWorkspace {
		allowedDir = l.workspace
	}

	// 文件工具
	l.tools.Register(&readfile.Tool{AllowedDir: allowedDir})
	l.tools.Register(&writefile.Tool{AllowedDir: allowedDir})
	l.tools.Register(&editfile.Tool{AllowedDir: allowedDir})
	l.tools.Register(&listdir.Tool{AllowedDir: allowedDir})

	// Shell 工具
	l.tools.Register(&exec.Tool{Timeout: l.execTimeout, WorkingDir: l.workspace, RestrictToWorkspace: l.restrictToWorkspace})

	// Web 工具
	l.tools.Register(&websearch.Tool{MaxResults: 5})
	l.tools.Register(&webfetch.Tool{MaxChars: 50000})

	// 消息工具
	l.tools.Register(&message.Tool{SendCallback: func(msg any) error {
		if outMsg, ok := msg.(*bus.OutboundMessage); ok {
			l.bus.PublishOutbound(outMsg)
		}
		return nil
	}})

	// Cron 工具
	if l.cronService != nil {
		l.tools.Register(&toolcron.Tool{CronService: l.cronService})
	}

	// Ask User 工具（用于向用户提问并中断等待响应）
	l.tools.Register(askuser.NewTool(func(channel, chatID, question string, options []string) (string, error) {
		// 这个回调会在 InterruptManager 中处理
		// 实际的中断处理在 tool 的 InvokableRun 中通过 StatefulInterrupt 完成
		return "", nil
	}))

	// 注册通用技能工具（用于拦截后的技能调用）
	l.tools.Register(skill.NewGenericSkillTool(l.context.GetSkillsLoader().LoadSkill))

}

// registerTaskTools 注册后台任务工具
func (l *Loop) registerTaskTools(manager tasktool.Manager) {
	if manager == nil {
		return
	}
	l.tools.Register(&tasktool.StartTool{Manager: manager, Logger: l.logger})
	l.tools.Register(&tasktool.GetTool{Manager: manager, Logger: l.logger})
	l.tools.Register(&tasktool.StopTool{Manager: manager, Logger: l.logger})
	l.tools.Register(&tasktool.ListTool{Manager: manager, Logger: l.logger})
}

// createBackgroundAgentTaskManager 创建任务管理器
func (l *Loop) createBackgroundAgentTaskManager() *AgentTaskManager {
	taskManager, err := NewBackgroundAgentTaskManager(&AgentTaskManagerConfig{
		Cfg:             l.cfg,
		Workspace:       l.workspace,
		Tools:           l.tools.GetTools(),
		Logger:          l.logger,
		Context:         l.context,
		CheckpointStore: l.interruptManager.GetCheckpointStore(),
		MaxIterations:   l.maxIterations,
		Sessions:        l.sessions,
		OnTaskComplete: func(channel, chatID, taskID string, status TaskStatus, result string) {
			// 任务完成时发送通知消息
			statusText := map[TaskStatus]string{
				TaskFinished: "完成",
				TaskFailed:   "失败",
				TaskStopped:  "已停止",
			}[status]
			msg := fmt.Sprintf("后台任务 %s\n状态: %s\n任务ID: %s", statusText, statusText, taskID)
			if result != "" && status == TaskFinished {
				msg = fmt.Sprintf("后台任务完成\n任务ID: %s\n\n%s", taskID, result)
			}
			l.bus.PublishOutbound(bus.NewOutboundMessage(channel, chatID, msg))
		},
	})
	if err != nil {
		l.logger.Error("创建任务管理器失败", zap.Error(err))
		return nil
	}
	return taskManager
}

// Run 运行代理循环
func (l *Loop) Run(ctx context.Context) error {
	l.running = true
	l.logger.Info("消息监听循环处理功能已启动")

	for l.running {
		// 等待消息
		msg, err := l.bus.ConsumeInbound(ctx)
		if err != nil {
			if err == context.DeadlineExceeded {
				continue
			}
			if err == context.Canceled {
				return nil
			}
			return err
		}

		// 处理消息
		if err := l.processMessage(ctx, msg); err != nil {
			l.logger.Error("处理消息失败", zap.Error(err))
			l.bus.PublishOutbound(bus.NewOutboundMessage(msg.Channel, msg.ChatID, fmt.Sprintf("抱歉，我遇到了错误: %s", err)))
		}
	}

	return nil
}

// Stop 停止代理循环
func (l *Loop) Stop() {
	l.running = false
	l.logger.Info("代理循环正在停止")
}

// processMessage 处理单条消息
func (l *Loop) processMessage(ctx context.Context, msg *bus.InboundMessage) error {
	preview := msg.Content
	if len(preview) > 80 {
		preview = preview[:80] + "..."
	}
	l.logger.Info("处理消息",
		zap.String("渠道", msg.Channel),
		zap.String("发送者", msg.SenderID),
		zap.String("内容", preview),
	)

	// 更新工具上下文
	l.updateToolContext(msg.Channel, msg.ChatID)

	// 使用 Master Agent 处理消息（包括中断恢复和正常处理）
	l.logger.Info("使用 Master Agent 处理消息")
	response, err := l.masterAgent.Process(ctx, msg)

	if err != nil {
		// 检查是否是中断
		if isInterruptError(err) {
			return nil
		}
		// 非中断错误：如果 response 包含错误信息（由 interruptible 构造），直接发送
		// 否则构造默认错误消息
		if response != "" {
			l.logger.Error("Master Agent 处理失败", zap.Error(err), zap.String("response", response))
			l.bus.PublishOutbound(bus.NewOutboundMessage(msg.Channel, msg.ChatID, response))
		} else {
			l.logger.Error("Master Agent 处理失败", zap.Error(err))
			l.bus.PublishOutbound(bus.NewOutboundMessage(msg.Channel, msg.ChatID, fmt.Sprintf("抱歉，处理消息时遇到错误: %v", err)))
		}
		return nil
	}

	// 发布响应
	l.bus.PublishOutbound(bus.NewOutboundMessage(msg.Channel, msg.ChatID, response))
	return nil

}

// updateToolContext 更新工具上下文
func (l *Loop) updateToolContext(channel, chatID string) {
	if mt, ok := l.tools.Get("message").(tools.ContextSetter); ok {
		mt.SetContext(channel, chatID)
	}

	if ct, ok := l.tools.Get("cron").(tools.ContextSetter); ok {
		ct.SetContext(channel, chatID)
	}
	if at, ok := l.tools.Get("ask_user").(tools.ContextSetter); ok {
		at.SetContext(channel, chatID)
	}
	if st, ok := l.tools.Get("start_task").(tools.ContextSetter); ok {
		st.SetContext(channel, chatID)
	}
}

// GetSupervisor 获取 Supervisor Agent
func (l *Loop) GetMasterAgent() *MasterAgent {
	if l.masterAgent == nil {
		l.logger.Warn("GetMasterAgent() 被调用但 MasterAgent 未初始化")
	}
	return l.masterAgent
}

// GetSupervisor 获取 Supervisor Agent
func (l *Loop) GetSupervisor() *SupervisorAgent {
	return l.supervisor
}
