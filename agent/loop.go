package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/agent/tools"
	"github.com/weibaohui/nanobot-go/agent/tools/askuser"
	toolcron "github.com/weibaohui/nanobot-go/agent/tools/cron"
	"github.com/weibaohui/nanobot-go/agent/tools/editfile"
	"github.com/weibaohui/nanobot-go/agent/tools/exec"
	"github.com/weibaohui/nanobot-go/agent/tools/listdir"
	"github.com/weibaohui/nanobot-go/agent/tools/message"
	"github.com/weibaohui/nanobot-go/agent/tools/readfile"
	"github.com/weibaohui/nanobot-go/agent/tools/skill"
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

	interruptManager *InterruptManager
	supervisor       *SupervisorAgent
	masterAgent      *MasterAgent
	enableSupervisor bool
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
	}

	loop.interruptManager = NewInterruptManager(cfg.MessageBus, logger)

	loop.registerDefaultTools()

	ctx := context.Background()
	toolNames := loop.tools.GetToolNames(ctx)
	logger.Info("已注册工具",
		zap.Int("数量", len(toolNames)),
		zap.Strings("工具列表", toolNames),
	)

	adapter, err := NewChatModelAdapter(logger, loop.cfg)
	if err != nil {
		logger.Error("创建 Provider 适配器失败", zap.Error(err))
		return loop
	}

	adapter.SetSkillLoader(loop.context.GetSkillsLoader().LoadSkill)
	adapter.SetRegisteredTools(toolNames)

	adkTools := loop.tools.GetADKToolsByNames(toolNames)

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
	})
	if err != nil {
		logger.Error("创建 Master Agent 失败，将使用传统模式", zap.Error(err))
	} else {
		loop.masterAgent = masterAgent
	}

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

// Run 运行代理循环
func (l *Loop) Run(ctx context.Context) error {
	l.running = true
	l.logger.Info("代理循环已启动")

	for l.running {
		select {
		case <-ctx.Done():
			l.running = false
			return ctx.Err()
		default:
		}

		// 等待消息
		msg, err := l.bus.ConsumeInbound(ctx)
		if err != nil {
			if err == context.DeadlineExceeded {
				continue
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

	// 检查是否有待处理的中断需要响应
	sessionKey := msg.SessionKey()
	if pendingInterrupt := l.interruptManager.GetPendingInterrupt(sessionKey); pendingInterrupt != nil {
		l.logger.Info("检测到待处理的中断，尝试恢复执行",
			zap.String("checkpoint_id", pendingInterrupt.CheckpointID),
			zap.String("session_key", sessionKey),
		)

		// 提交用户响应
		response := &UserResponse{
			CheckpointID: pendingInterrupt.CheckpointID,
			Answer:       msg.Content,
		}
		if err := l.interruptManager.SubmitUserResponse(response); err != nil {
			l.logger.Error("提交用户响应失败", zap.Error(err))
			return err
		}

		// 恢复执行
		result, err := l.ResumeExecution(ctx, pendingInterrupt.CheckpointID, pendingInterrupt.InterruptID, msg.Content, msg.Channel, msg.ChatID, sessionKey, pendingInterrupt.IsAskUser, pendingInterrupt.IsPlan, pendingInterrupt.IsSupervisor)
		if err != nil {
			// 检查是否是新的中断
			if strings.HasPrefix(err.Error(), "INTERRUPT:") {
				// 新的中断已由 handleInterrupt 处理
				return nil
			}
			return fmt.Errorf("恢复执行失败: %w", err)
		}

		// 清理已完成的中断
		l.interruptManager.ClearInterrupt(pendingInterrupt.CheckpointID)

		// 发布恢复后的响应
		l.bus.PublishOutbound(bus.NewOutboundMessage(msg.Channel, msg.ChatID, result))

		// 保存会话
		sess := l.sessions.GetOrCreate(sessionKey)
		sess.AddMessage("user", msg.Content)
		sess.AddMessage("assistant", result)
		l.sessions.Save(sess)

		return nil
	}

	// 更新工具上下文
	l.updateToolContext(msg.Channel, msg.ChatID)

	l.logger.Info("使用 Master Agent 处理消息")
	response, err := l.masterAgent.Process(ctx, msg)

	if err != nil {
		// 检查是否是中断
		if isInterruptError(err) {
			return nil
		}
		return fmt.Errorf("Master Agent 处理失败: %w", err)
	}

	// 发布响应
	l.bus.PublishOutbound(bus.NewOutboundMessage(msg.Channel, msg.ChatID, response))
	return nil

	// 优先使用 Supervisor Agent 处理消息
	if l.enableSupervisor && l.supervisor != nil {
		l.logger.Info("使用 Supervisor Agent 处理消息")
		response, err := l.supervisor.Process(ctx, msg)
		if err != nil {
			// 检查是否是中断
			if strings.HasPrefix(err.Error(), "INTERRUPT:") {
				return nil
			}
			return fmt.Errorf("Supervisor 处理失败: %w", err)
		}

		// 发布响应
		l.bus.PublishOutbound(bus.NewOutboundMessage(msg.Channel, msg.ChatID, response))
		return nil
	}

	return nil
}

// updateToolContext 更新工具上下文
func (l *Loop) updateToolContext(channel, chatID string) {
	if mt, ok := l.tools.Get("message").(tools.ContextSetter); ok {
		mt.SetContext(channel, chatID)
	}
	if st, ok := l.tools.Get("spawn").(tools.ContextSetter); ok {
		st.SetContext(channel, chatID)
	}
	if ct, ok := l.tools.Get("cron").(tools.ContextSetter); ok {
		ct.SetContext(channel, chatID)
	}
	if at, ok := l.tools.Get("ask_user").(tools.ContextSetter); ok {
		at.SetContext(channel, chatID)
	}
}

// ResumeExecution 恢复被中断的执行
func (l *Loop) ResumeExecution(ctx context.Context, checkpointID, interruptID string, userAnswer string, channel, chatID string, sessionKey string, isAskUser bool, isPlan bool, isSupervisor bool) (string, error) {
	// 准备恢复参数
	var resumePayload any
	if isAskUser {
		resumePayload = &askuser.AskUserInfo{
			UserAnswer: userAnswer,
		}
	} else {
		resumePayload = map[string]any{
			"user_answer": userAnswer,
		}
	}
	resumeParams := &adk.ResumeParams{
		Targets: map[string]any{
			interruptID: resumePayload,
		},
	}

	// 构建消息对象（用于 Supervisor 模式的中断恢复）
	msg := &bus.InboundMessage{
		Channel: channel,
		ChatID:  chatID,
	}

	// 根据中断类型选择恢复方式
	if isSupervisor {
		// Supervisor 模式的中断恢复
		if l.supervisor == nil {
			return "", fmt.Errorf("Supervisor Agent 未初始化")
		}
		return l.supervisor.Resume(ctx, checkpointID, resumeParams, msg)
	}

	// 非 Supervisor 模式暂未支持
	return "", fmt.Errorf("非 Supervisor 模式暂未实现，当前仅支持 Supervisor 模式的中断恢复")
}

// GetInterruptManager 获取中断管理器
func (l *Loop) GetInterruptManager() *InterruptManager {
	return l.interruptManager
}

// GetSupervisor 获取 Supervisor Agent
func (l *Loop) GetSupervisor() *SupervisorAgent {
	return l.supervisor
}

// IsSupervisorEnabled 检查是否启用 Supervisor 模式
func (l *Loop) IsSupervisorEnabled() bool {
	return l.enableSupervisor && l.supervisor != nil
}

// convertHistory 转换会话历史为 eino Message 格式
func (l *Loop) convertHistory(history []map[string]any) []*schema.Message {
	result := make([]*schema.Message, 0, len(history))
	for _, h := range history {
		role := schema.User
		if r, ok := h["role"].(string); ok && r == "assistant" {
			role = schema.Assistant
		}
		content := ""
		if c, ok := h["content"].(string); ok {
			content = c
		}
		result = append(result, &schema.Message{
			Role:    role,
			Content: content,
		})
	}
	return result
}

// ShouldUseStream 判断是否应该使用流式处理
func (l *Loop) ShouldUseStream(channel string) bool {
	return channel == "websocket"
}

// GetTools returns all registered tools as a slice
func (l *Loop) GetTools() []tool.BaseTool {
	return l.tools.GetADKTools()
}
