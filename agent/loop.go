package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
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
	"github.com/weibaohui/nanobot-go/cron"
	"github.com/weibaohui/nanobot-go/eino_adapter"
	"github.com/weibaohui/nanobot-go/providers"
	"github.com/weibaohui/nanobot-go/session"
	"go.uber.org/zap"
)

// Loop 代理循环核心
type Loop struct {
	bus                 *bus.MessageBus
	provider            providers.LLMProvider
	workspace           string
	model               string
	maxIterations       int
	execTimeout         int
	restrictToWorkspace bool
	cronService         *cron.Service
	context             *ContextBuilder
	sessions            *session.Manager
	tools               *tools.Registry
	running             bool
	logger              *zap.Logger

	// ADK Agent（保留用于向后兼容）
	adkAgent  *adk.ChatModelAgent
	adkRunner *adk.Runner

	// 中断管理
	interruptManager *InterruptManager

	// Plan-Execute mode support（保留用于向后兼容）
	planAgent *eino_adapter.PlanExecuteAgent
	selector  *eino_adapter.ModeSelector

	// Supervisor 入口 Agent（新增）
	supervisor *SupervisorAgent

	// 是否启用 Supervisor 模式
	enableSupervisor bool
}

// NewLoop 创建代理循环
func NewLoop(messageBus *bus.MessageBus, provider providers.LLMProvider, workspace string, model string, maxIterations int, execTimeout int, restrictToWorkspace bool, cronService *cron.Service, sessionManager *session.Manager, logger *zap.Logger) *Loop {
	if logger == nil {
		logger = zap.NewNop()
	}

	loop := &Loop{
		bus:                 messageBus,
		provider:            provider,
		workspace:           workspace,
		model:               model,
		maxIterations:       maxIterations,
		execTimeout:         execTimeout,
		restrictToWorkspace: restrictToWorkspace,
		cronService:         cronService,
		context:             NewContextBuilder(workspace),
		sessions:            sessionManager,
		tools:               tools.NewRegistry(),
		logger:              logger,
	}

	// 创建中断管理器
	loop.interruptManager = NewInterruptManager(messageBus, logger)

	// 注册默认工具
	loop.registerDefaultTools()

	registeredTools := loop.GetTools()
	toolNames := make([]string, 0, len(registeredTools))
	for _, t := range registeredTools {
		if t != nil {
			info, err := t.Info(context.Background())
			if err == nil && info != nil && info.Name != "" {
				toolNames = append(toolNames, info.Name)
			}
		}
	}
	logger.Info("已注册工具",
		zap.Int("数量", len(registeredTools)),
		zap.Strings("工具列表", toolNames),
	)

	// 创建 ADK Agent
	ctx := context.Background()
	adapter := eino_adapter.NewProviderAdapter(logger, provider, model)

	// 配置适配器：设置技能加载器和已注册工具列表
	adapter.SetSkillLoader(loop.context.GetSkillsLoader().LoadSkill)
	adapter.SetRegisteredTools(toolNames)

	adkTools := loop.tools.GetADKTools()
	var toolsConfig adk.ToolsConfig
	if len(adkTools) > 0 {
		toolsConfig = adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: adkTools,
			},
		}
	}
	adkAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "nanobot",
		Description:   "nanobot AI assistant",
		Model:         adapter,
		ToolsConfig:   toolsConfig,
		MaxIterations: maxIterations,
	})
	if err != nil {
		logger.Error("创建 ADK Agent 失败，将使用回退模式", zap.Error(err))
	} else {
		loop.adkAgent = adkAgent
		loop.adkRunner = adk.NewRunner(ctx, adk.RunnerConfig{
			Agent:           adkAgent,
			EnableStreaming: true,
			CheckPointStore: loop.interruptManager.GetCheckpointStore(),
		})
		logger.Info("ADK Agent 创建成功")
	}

	// 创建 Supervisor Agent（入口型 Agent）
	supervisor, err := NewSupervisorAgent(ctx, &SupervisorConfig{
		Provider:        provider,
		Model:           model,
		Workspace:       workspace,
		Tools:           adkTools,
		Logger:          logger,
		Sessions:        sessionManager,
		Bus:             messageBus,
		Context:         loop.context, // 传入上下文构建器，复用基础系统提示词
		InterruptMgr:    loop.interruptManager,
		CheckpointStore: loop.interruptManager.GetCheckpointStore(),
		MaxIterations:   maxIterations,
		EnableStream:    true,
		RegisteredTools: toolNames, // 传入已注册的工具名称列表
	})
	if err != nil {
		logger.Error("创建 Supervisor Agent 失败，将使用传统模式", zap.Error(err))
	} else {
		loop.supervisor = supervisor
		loop.enableSupervisor = true
		logger.Info("Supervisor Agent 创建成功，已启用入口型 Agent 模式")
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

	// // 回退到传统模式
	// // 检查是否使用计划模式
	// if l.planAgent != nil && l.selector != nil && l.selector.ShouldUsePlanMode(msg.Content) {
	// 	l.logger.Info("使用计划执行模式")
	// 	if err := l.processWithPlan(ctx, msg); err != nil {
	// 		if strings.HasPrefix(err.Error(), "INTERRUPT:") {
	// 			return nil
	// 		}
	// 		return err
	// 	}
	// 	return nil
	// }

	// // 检查是否使用流式处理
	// if l.ShouldUseStream(msg.Channel) {
	// 	if err := l.processWithADKStream(ctx, msg); err != nil {
	// 		if strings.HasPrefix(err.Error(), "INTERRUPT:") {
	// 			return nil
	// 		}
	// 		return err
	// 	}
	// 	return nil
	// }

	// // 使用普通模式
	// if err := l.processWithADK(ctx, msg); err != nil {
	// 	if strings.HasPrefix(err.Error(), "INTERRUPT:") {
	// 		return nil
	// 	}
	// 	return err
	// }
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

// buildMessagesWithSystem 构建包含系统提示词的消息列表
func (l *Loop) buildMessagesWithSystem(history []*schema.Message, userInput, channel, chatID string) []*schema.Message {
	// 构建系统提示词
	systemPrompt := l.context.BuildSystemPrompt(nil)
	// 复用公共方法构建消息列表
	return BuildMessageList(systemPrompt, history, userInput, channel, chatID)
}

// processWithADK 使用 ADK Agent 处理消息
func (l *Loop) processWithADK(ctx context.Context, msg *bus.InboundMessage) error {
	if l.adkAgent == nil || l.adkRunner == nil {
		return fmt.Errorf("ADK Agent 未初始化")
	}

	// 获取会话历史
	sessionKey := msg.SessionKey()
	sess := l.sessions.GetOrCreate(sessionKey)
	history := l.convertHistory(sess.GetHistory(10))

	// 构建消息（包含系统提示词）
	messages := l.buildMessagesWithSystem(history, msg.Content, msg.Channel, msg.ChatID)

	// 生成 checkpoint ID
	checkpointID := fmt.Sprintf("%s_%d", sessionKey, time.Now().UnixNano())

	iter := l.adkRunner.Run(ctx, messages, adk.WithCheckPointID(checkpointID))
	var response string
	var lastEvent *adk.AgentEvent

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return fmt.Errorf("ADK Agent 执行失败: %w", event.Err)
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			msgOutput, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				continue
			}
			response = msgOutput.Content
		}
		lastEvent = event
	}

	// 检查是否被中断
	if lastEvent != nil && lastEvent.Action != nil && lastEvent.Action.Interrupted != nil {
		if err := l.handleInterrupt(ctx, msg, checkpointID, lastEvent, sess, sessionKey, false); err != nil {
			if strings.HasPrefix(err.Error(), "INTERRUPT:") {
				return nil
			}
			return err
		}
		return nil
	}

	// 发布响应
	l.bus.PublishOutbound(bus.NewOutboundMessage(msg.Channel, msg.ChatID, response))

	// 保存会话
	sess.AddMessage("user", msg.Content)
	sess.AddMessage("assistant", response)
	l.sessions.Save(sess)

	l.logger.Info("响应完成",
		zap.String("渠道", msg.Channel),
		zap.Int("内容长度", len(response)),
	)

	return nil
}

// processWithADKStream 使用流式处理
func (l *Loop) processWithADKStream(ctx context.Context, msg *bus.InboundMessage) error {
	if l.adkAgent == nil || l.adkRunner == nil {
		return fmt.Errorf("ADK Agent 未初始化")
	}

	// 获取会话历史
	sessionKey := msg.SessionKey()
	sess := l.sessions.GetOrCreate(sessionKey)
	history := l.convertHistory(sess.GetHistory(10))

	// 构建消息（包含系统提示词）
	messages := l.buildMessagesWithSystem(history, msg.Content, msg.Channel, msg.ChatID)

	// 生成 checkpoint ID（用于中断恢复）
	checkpointID := fmt.Sprintf("%s_%d", sessionKey, time.Now().UnixNano())

	iter := l.adkRunner.Run(ctx, messages, adk.WithCheckPointID(checkpointID))
	var response string
	var fullContent string
	var lastEvent *adk.AgentEvent

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return fmt.Errorf("流式执行失败: %w", event.Err)
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			msgOutput, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				continue
			}
			delta := ""
			if fullContent != "" && strings.HasPrefix(msgOutput.Content, fullContent) {
				delta = msgOutput.Content[len(fullContent):]
			} else {
				delta = msgOutput.Content
			}
			if delta != "" {
				l.bus.PublishStream(bus.NewStreamChunk(msg.Channel, msg.ChatID, delta, msgOutput.Content, false))
			}
			fullContent = msgOutput.Content
			response = msgOutput.Content
		}
		lastEvent = event
	}

	// 检查是否被中断
	if lastEvent != nil && lastEvent.Action != nil && lastEvent.Action.Interrupted != nil {
		return l.handleInterrupt(ctx, msg, checkpointID, lastEvent, sess, sessionKey, false)
	}

	// 发送完成标记
	l.bus.PublishStream(bus.NewStreamChunk(msg.Channel, msg.ChatID, "", response, true))

	// 保存会话
	sess.AddMessage("user", msg.Content)
	sess.AddMessage("assistant", response)
	l.sessions.Save(sess)

	l.logger.Info("流式响应完成",
		zap.String("渠道", msg.Channel),
		zap.Int("内容长度", len(response)),
	)

	return nil
}

// handleInterrupt 处理中断
func (l *Loop) handleInterrupt(ctx context.Context, msg *bus.InboundMessage, checkpointID string, event *adk.AgentEvent, sess *session.Session, sessionKey string, isPlan bool) error {
	if event.Action == nil || event.Action.Interrupted == nil {
		return nil
	}

	// 获取中断上下文
	interruptCtx := event.Action.Interrupted.InterruptContexts[0]
	interruptID := interruptCtx.ID

	// 解析中断信息
	var question string
	var options []string
	isAskUser := false

	if info, ok := interruptCtx.Info.(*askuser.AskUserInfo); ok {
		question = info.Question
		options = append(options, info.Options...)
		isAskUser = true
	} else if info, ok := interruptCtx.Info.(map[string]any); ok {
		if q, ok := info["question"].(string); ok {
			question = q
		}
		if opts, ok := info["options"].([]any); ok {
			for _, opt := range opts {
				if s, ok := opt.(string); ok {
					options = append(options, s)
				}
			}
		}
	} else {
		// 尝试直接作为 Stringer
		question = fmt.Sprintf("%v", interruptCtx.Info)
	}
	if sessionKey == "" && msg != nil {
		sessionKey = msg.SessionKey()
	}

	// 发送中断请求到用户
	l.interruptManager.HandleInterrupt(&InterruptInfo{
		CheckpointID: checkpointID,
		InterruptID:  interruptID,
		Channel:      msg.Channel,
		ChatID:       msg.ChatID,
		Question:     question,
		Options:      options,
		SessionKey:   sessionKey,
		IsAskUser:    isAskUser,
		IsPlan:       isPlan,
	})

	l.logger.Info("等待用户输入以恢复执行",
		zap.String("checkpoint_id", checkpointID),
		zap.String("interrupt_id", interruptID),
		zap.String("question", question),
	)

	// 注意：这里不等待用户响应，而是返回
	// 用户响应会通过 SubmitUserResponse 提交，然后通过 ResumeExecution 恢复
	return fmt.Errorf("INTERRUPT:%s:%s", checkpointID, interruptID)
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

	// 非 Supervisor 模式的恢复
	if l.adkRunner == nil {
		return "", fmt.Errorf("ADK Agent 未初始化")
	}

	var (
		iter *adk.AsyncIterator[*adk.AgentEvent]
		err  error
	)
	if isPlan {
		if l.planAgent == nil {
			return "", fmt.Errorf("Plan Agent 未初始化")
		}
		iter, err = l.planAgent.ResumeWithParams(ctx, checkpointID, resumeParams)
	} else {
		iter, err = l.adkRunner.ResumeWithParams(ctx, checkpointID, resumeParams)
	}
	if err != nil {
		return "", fmt.Errorf("恢复执行失败: %w", err)
	}

	var response string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", fmt.Errorf("恢复后执行失败: %w", event.Err)
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			msgOutput, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				continue
			}
			response = msgOutput.Content
		}

		// 检查是否再次被中断
		if event.Action != nil && event.Action.Interrupted != nil {
			// 递归处理新的中断
			newCheckpointID := fmt.Sprintf("%s_resume_%d", checkpointID, time.Now().UnixNano())
			return "", l.handleInterrupt(ctx, msg, newCheckpointID, event, nil, sessionKey, isPlan)
		}
	}

	return response, nil
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

// processWithPlan 使用计划执行模式
func (l *Loop) processWithPlan(ctx context.Context, msg *bus.InboundMessage) error {

	// 获取会话历史
	sessionKey := msg.SessionKey()
	sess := l.sessions.GetOrCreate(sessionKey)
	history := l.convertHistory(sess.GetHistory(10))

	// 构建消息（包含系统提示词）
	messages := l.buildMessagesWithSystem(history, msg.Content, msg.Channel, msg.ChatID)

	checkpointID := fmt.Sprintf("%s_%d", sessionKey, time.Now().UnixNano())
	iter := l.planAgent.StreamWithHistory(ctx, msg.Content, messages[:len(messages)-1], checkpointID)
	var response string
	var lastEvent *adk.AgentEvent
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return fmt.Errorf("计划执行失败: %w", event.Err)
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			msgOutput, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				continue
			}
			response = msgOutput.Content
		}
		lastEvent = event
		if event.Action != nil && event.Action.Interrupted != nil {
			break
		}
	}

	if lastEvent != nil && lastEvent.Action != nil && lastEvent.Action.Interrupted != nil {
		if err := l.handleInterrupt(ctx, msg, checkpointID, lastEvent, sess, sessionKey, true); err != nil {
			if strings.HasPrefix(err.Error(), "INTERRUPT:") {
				return nil
			}
			return err
		}
		return nil
	}

	// 发布响应
	l.bus.PublishOutbound(bus.NewOutboundMessage(msg.Channel, msg.ChatID, response))

	sess.AddMessage("user", msg.Content)
	sess.AddMessage("assistant", response)
	l.sessions.Save(sess)

	return nil
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

// ProcessDirect 直接处理消息（用于 CLI 或 cron）
func (l *Loop) ProcessDirect(ctx context.Context, content, sessionKey, channel, chatID string) (string, error) {
	// 更新工具上下文
	l.updateToolContext(channel, chatID)

	if l.adkAgent == nil || l.adkRunner == nil {
		return "", fmt.Errorf("ADK Agent 未初始化")
	}

	// 获取会话历史
	sess := l.sessions.GetOrCreate(sessionKey)
	history := l.convertHistory(sess.GetHistory(10))

	// 构建消息（包含系统提示词）
	messages := l.buildMessagesWithSystem(history, content, channel, chatID)

	iter := l.adkRunner.Run(ctx, messages)
	var response string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", event.Err
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			msgOutput, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				continue
			}
			response = msgOutput.Content
		}
	}

	// 保存会话
	sess.AddMessage("user", content)
	sess.AddMessage("assistant", response)
	l.sessions.Save(sess)

	return response, nil
}

// GetTools returns all registered tools as a slice
func (l *Loop) GetTools() []tool.BaseTool {
	return l.tools.GetADKTools()
}

// SetPlanAgent sets the plan-execute agent for complex task handling
func (l *Loop) SetPlanAgent(planAgent *eino_adapter.PlanExecuteAgent) {
	l.planAgent = planAgent
}

// SetModeSelector sets the mode selector for automatic mode switching
func (l *Loop) SetModeSelector(selector *eino_adapter.ModeSelector) {
	l.selector = selector
}

// GetPlanAgent returns the plan-execute agent
func (l *Loop) GetPlanAgent() *eino_adapter.PlanExecuteAgent {
	return l.planAgent
}

// GetModeSelector returns the mode selector
func (l *Loop) GetModeSelector() *eino_adapter.ModeSelector {
	return l.selector
}

// GetADKAgent returns the ADK agent
func (l *Loop) GetADKAgent() *adk.ChatModelAgent {
	return l.adkAgent
}
