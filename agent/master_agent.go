package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/agent/tools/askuser"
	"github.com/weibaohui/nanobot-go/bus"
	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/session"
	"go.uber.org/zap"
)

const interruptErrorPrefix = "INTERRUPT:"

// MasterAgent 监督者 Agent
// 作为统一入口，根据用户输入自动路由到合适的子 Agent
type MasterAgent struct {
	cfg       *config.Config
	workspace string
	tools     []tool.BaseTool
	logger    *zap.Logger
	sessions  *session.Manager
	context   *ContextBuilder

	adkRunner *adk.Runner

	interruptManager *InterruptManager
	checkpointStore  compose.CheckPointStore
	registeredTools  []string
}

// MasterAgentConfig Master 配置
type MasterAgentConfig struct {
	Cfg             *config.Config
	Workspace       string
	Tools           []tool.BaseTool
	Logger          *zap.Logger
	Sessions        *session.Manager
	Bus             *bus.MessageBus
	Context         *ContextBuilder // 上下文构建器
	InterruptMgr    *InterruptManager
	CheckpointStore compose.CheckPointStore
	MaxIterations   int
	// 已注册的工具名称列表
	RegisteredTools []string
}

// NewMasterAgent 创建 Master Agent
func NewMasterAgent(ctx context.Context, cfg *MasterAgentConfig) (*MasterAgent, error) {
	if cfg == nil {
		return nil, ErrConfigNil
	}

	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	sa := &MasterAgent{
		cfg:              cfg.Cfg,
		workspace:        cfg.Workspace,
		tools:            cfg.Tools,
		logger:           logger,
		sessions:         cfg.Sessions,
		context:          cfg.Context,
		interruptManager: cfg.InterruptMgr,
		checkpointStore:  cfg.CheckpointStore,
		registeredTools:  cfg.RegisteredTools,
	}

	if err := sa.initMaster(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMasterInit, err)
	}

	logger.Info("Master Agent 创建成功",
		zap.String("model", cfg.Workspace),
	)

	return sa, nil
}

// initMaster 创建 ADK Master
func (sa *MasterAgent) initMaster(ctx context.Context) error {
	adapter, err := NewChatModelAdapter(sa.logger, sa.cfg, sa.sessions)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrChatModelAdapter, err)
	}
	if sa.context != nil {
		adapter.SetSkillLoader(sa.context.GetSkillsLoader().LoadSkill)
	}
	if len(sa.registeredTools) > 0 {
		adapter.SetRegisteredTools(sa.registeredTools)
	}

	var toolsConfig adk.ToolsConfig
	if len(sa.tools) > 0 {
		toolsConfig = adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: sa.tools,
			},
		}
	}

	masterAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "Master",
		Description: "主智能体",
		Instruction: sa.context.BuildSystemPrompt(),
		Model:       adapter,
		ToolsConfig: toolsConfig,
		Exit:        &adk.ExitTool{},
	})
	if err != nil {
		return fmt.Errorf("%w: %w", ErrAgentCreate, err)
	}

	sa.adkRunner = adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           masterAgent,
		CheckPointStore: sa.checkpointStore,
	})

	return nil
}

// Process 处理用户消息
func (sa *MasterAgent) Process(ctx context.Context, msg *bus.InboundMessage) (string, error) {
	sessionKey := msg.SessionKey()
	sess := sa.sessions.GetOrCreate(sessionKey)

	// 将 session key 放入 context，用于记录 token 用量
	ctx = context.WithValue(ctx, SessionKeyContextKey, sessionKey)

	// 检查是否有待处理的中断需要响应，如果有则处理恢复
	if pendingInterrupt := sa.interruptManager.GetPendingInterrupt(sessionKey); pendingInterrupt != nil {
		return sa.processInterrupted(ctx, sess, msg, pendingInterrupt)
	}

	// 正常处理流程
	history := sa.convertHistory(sess.GetHistory(10))
	messages := sa.buildMessages(history, msg.Content, msg.Channel, msg.ChatID)
	checkpointID := fmt.Sprintf("%s_%d", sessionKey, time.Now().UnixNano())

	response, err := sa.processNormal(ctx, messages, checkpointID, msg)
	if err != nil && isInterruptError(err) {
		return "", err
	}

	// 保存用户消息（assistant 消息在 Generate 中已自动保存）
	sa.saveSession(sess, msg.Content)

	return response, nil
}

// saveSession 保存用户消息到会话
func (sa *MasterAgent) saveSession(sess *session.Session, userMessage string) {
	sess.AddMessage("user", userMessage)
	sa.sessions.Save(sess)
}

// processInterrupted 处理中断恢复流程
func (sa *MasterAgent) processInterrupted(ctx context.Context, sess *session.Session, msg *bus.InboundMessage, pendingInterrupt *InterruptInfo) (string, error) {
	sessionKey := msg.SessionKey()

	sa.logger.Info("检测到待处理的中断，尝试恢复执行",
		zap.String("checkpoint_id", pendingInterrupt.CheckpointID),
		zap.String("session_key", sessionKey),
	)

	// 提交用户响应
	response := &UserResponse{
		CheckpointID: pendingInterrupt.CheckpointID,
		Answer:       msg.Content,
	}
	if err := sa.interruptManager.SubmitUserResponse(response); err != nil {
		return "", fmt.Errorf("提交用户响应失败: %w", err)
	}

	// 准备恢复参数
	resumePayload := sa.buildResumePayload(pendingInterrupt.IsAskUser, msg.Content)
	resumeParams := &adk.ResumeParams{
		Targets: map[string]any{
			pendingInterrupt.InterruptID: resumePayload,
		},
	}

	// 构建消息对象用于恢复执行
	resumeMsg := &bus.InboundMessage{
		Channel:  msg.Channel,
		ChatID:   msg.ChatID,
		SenderID: sessionKey,
	}

	// 恢复执行
	result, err := sa.Resume(ctx, pendingInterrupt.CheckpointID, resumeParams, resumeMsg)
	if err != nil {
		if isInterruptError(err) {
			return "", err
		}
		return "", fmt.Errorf("恢复执行失败: %w", err)
	}

	// 清理已完成的中断
	sa.interruptManager.ClearInterrupt(pendingInterrupt.CheckpointID)
	// 保存会话
	sa.saveSession(sess, msg.Content)

	return result, nil
}

// buildResumePayload 构建恢复参数的有效载荷
func (sa *MasterAgent) buildResumePayload(isAskUser bool, userAnswer string) any {
	if isAskUser {
		return &askuser.AskUserInfo{
			UserAnswer: userAnswer,
		}
	}
	return map[string]any{
		"user_answer": userAnswer,
	}
}

// processNormal 普通模式处理
func (sa *MasterAgent) processNormal(ctx context.Context, messages []*schema.Message, checkpointID string, msg *bus.InboundMessage) (string, error) {
	iter := sa.adkRunner.Run(ctx, messages, adk.WithCheckPointID(checkpointID))

	var response string
	var lastEvent *adk.AgentEvent

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return "", fmt.Errorf("Master 执行失败: %w", event.Err)
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

	// 检查中断
	if lastEvent != nil && lastEvent.Action != nil && lastEvent.Action.Interrupted != nil {
		return "", sa.handleInterrupt(msg, checkpointID, lastEvent)
	}

	return response, nil
}

// handleInterrupt 处理中断
func (sa *MasterAgent) handleInterrupt(msg *bus.InboundMessage, checkpointID string, event *adk.AgentEvent) error {
	if event.Action == nil || event.Action.Interrupted == nil {
		return nil
	}

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
		if question != "" {
			isAskUser = true
		}
	} else {
		question = fmt.Sprintf("%v", interruptCtx.Info)
	}

	// 发送中断请求
	sa.interruptManager.HandleInterrupt(&InterruptInfo{
		CheckpointID: checkpointID,
		InterruptID:  interruptID,
		Channel:      msg.Channel,
		ChatID:       msg.ChatID,
		Question:     question,
		Options:      options,
		SessionKey:   msg.SessionKey(),
		IsAskUser:    isAskUser,
		IsMaster:     true, // 标记来自 Master 模式的中断
	})

	sa.logger.Info("等待用户输入以恢复执行",
		zap.String("checkpoint_id", checkpointID),
		zap.String("question", question),
	)

	return fmt.Errorf("%s%s:%s", interruptErrorPrefix, checkpointID, interruptID)
}

// buildMessages 构建消息列表
func (sa *MasterAgent) buildMessages(history []*schema.Message, userInput, channel, chatID string) []*schema.Message {
	// 复用公共方法构建消息列表
	return BuildMessageList("", history, userInput, channel, chatID)
}

// convertHistory 转换会话历史
func (sa *MasterAgent) convertHistory(history []map[string]any) []*schema.Message {
	result := make([]*schema.Message, 0, len(history))
	for _, h := range history {
		role := schema.User
		if r, ok := h["role"].(string); ok && r == "assistant" {
			role = schema.Assistant
		}

		content, _ := h["content"].(string)
		result = append(result, &schema.Message{
			Role:    role,
			Content: content,
		})
	}
	return result
}

// GetADKRunner 获取 ADK Runner
func (sa *MasterAgent) GetADKRunner() *adk.Runner {
	return sa.adkRunner
}

// Resume 恢复被中断的执行
// 用于处理 Master 模式下的中断恢复
func (sa *MasterAgent) Resume(ctx context.Context, checkpointID string, resumeParams *adk.ResumeParams, msg *bus.InboundMessage) (string, error) {
	if sa.adkRunner == nil {
		return "", fmt.Errorf("ADK Runner 未初始化")
	}

	// 将 session key 放入 context，用于记录 token 用量
	sessionKey := msg.SenderID
	ctx = context.WithValue(ctx, SessionKeyContextKey, sessionKey)

	// 使用 Master 的 Runner 恢复执行
	iter, err := sa.adkRunner.ResumeWithParams(ctx, checkpointID, resumeParams)
	if err != nil {
		return "", fmt.Errorf("Master 恢复执行失败: %w", err)
	}

	var response string
	var lastEvent *adk.AgentEvent

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return "", fmt.Errorf("Master 恢复后执行失败: %w", event.Err)
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

	// 检查是否再次被中断
	if lastEvent != nil && lastEvent.Action != nil && lastEvent.Action.Interrupted != nil {
		// 生成新的 checkpoint ID
		newCheckpointID := fmt.Sprintf("%s_resume_%d", checkpointID, time.Now().UnixNano())
		return "", sa.handleInterrupt(msg, newCheckpointID, lastEvent)
	}

	return response, nil
}
