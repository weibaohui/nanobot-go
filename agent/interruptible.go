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

// buildChatModelAdapter 创建并配置 ChatModelAdapter
// 将 LLM 初始化逻辑集中在此，避免遗漏必要配置
func buildChatModelAdapter(logger *zap.Logger, cfg *config.Config, sessions *session.Manager, skillsLoader func(string) string, registeredTools []string) (*ChatModelAdapter, error) {
	llm, err := NewChatModelAdapter(logger, cfg, sessions)
	if err != nil {
		return nil, err
	}

	// 设置 SkillLoader
	if skillsLoader != nil {
		llm.SetSkillLoader(skillsLoader)
	}

	// 设置 RegisteredTools
	if len(registeredTools) > 0 {
		llm.SetRegisteredTools(registeredTools)
	}

	return llm, nil
}

// interruptible 可嵌入的中断处理能力
// 为 Agent 提供中断处理、恢复执行等通用能力
type interruptible struct {
	cfg              *config.Config
	workspace        string
	tools            []tool.BaseTool
	logger           *zap.Logger
	sessions         *session.Manager
	bus              *bus.MessageBus
	context          *ContextBuilder
	adkRunner        *adk.Runner
	interruptManager *InterruptManager
	checkpointStore  compose.CheckPointStore
	registeredTools  []string
	maxIterations    int
	agentType        string // "master" 或 "supervisor"
	adkAgent         adk.Agent
	hookManager      *HookManager
}

// interruptibleConfig 中断处理能力的配置
type interruptibleConfig struct {
	Cfg             *config.Config
	Workspace       string
	Tools           []tool.BaseTool
	Logger          *zap.Logger
	Sessions        *session.Manager
	Bus             *bus.MessageBus
	Context         *ContextBuilder
	InterruptMgr    *InterruptManager
	CheckpointStore compose.CheckPointStore
	MaxIterations   int
	RegisteredTools []string
	AgentType       string
	ADKAgent        adk.Agent
	ADKRunner       *adk.Runner
	HookManager     *HookManager
}

// newInterruptible 创建中断处理能力
func newInterruptible(ctx context.Context, cfg *interruptibleConfig) (*interruptible, error) {
	if cfg == nil {
		return nil, ErrConfigNil
	}

	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	maxIter := cfg.MaxIterations
	if maxIter <= 0 {
		maxIter = 10
	}

	i := &interruptible{
		cfg:              cfg.Cfg,
		workspace:        cfg.Workspace,
		tools:            cfg.Tools,
		logger:           logger,
		sessions:         cfg.Sessions,
		bus:              cfg.Bus,
		context:          cfg.Context,
		adkRunner:        cfg.ADKRunner,
		interruptManager: cfg.InterruptMgr,
		checkpointStore:  cfg.CheckpointStore,
		registeredTools:  cfg.RegisteredTools,
		maxIterations:    maxIter,
		agentType:        cfg.AgentType,
		adkAgent:         cfg.ADKAgent,
		hookManager:      cfg.HookManager,
	}

	logger.Info(fmt.Sprintf("%s Agent 能力初始化成功", cfg.AgentType),
		zap.String("workspace", cfg.Workspace),
		zap.Int("max_iterations", maxIter),
	)

	return i, nil
}

// BuildChatModelAdapter 创建并配置 ChatModelAdapter
// 将 LLM 初始化逻辑集中在此，避免遗漏必要配置
func (i *interruptible) BuildChatModelAdapter() (*ChatModelAdapter, error) {
	llm, err := NewChatModelAdapter(i.logger, i.cfg, i.sessions)
	if err != nil {
		return nil, err
	}

	// 设置 SkillLoader
	if i.context != nil {
		llm.SetSkillLoader(i.context.GetSkillsLoader().LoadSkill)
	}

	// 设置 RegisteredTools
	if len(i.registeredTools) > 0 {
		llm.SetRegisteredTools(i.registeredTools)
	}

	return llm, nil
}

// Process 处理用户消息的统一入口
// 包含中断检查和恢复逻辑
func (i *interruptible) Process(ctx context.Context, msg *bus.InboundMessage, buildMessagesFunc func(history []*schema.Message, userInput, channel, chatID string) []*schema.Message) (string, error) {
	sessionKey := msg.SessionKey()
	sess := i.sessions.GetOrCreate(sessionKey)

	ctx = context.WithValue(ctx, SessionKeyContextKey, sessionKey)

	// 检查是否有待处理的中断需要响应
	if pendingInterrupt := i.interruptManager.GetPendingInterrupt(sessionKey); pendingInterrupt != nil {
		return i.processInterrupted(ctx, sess, msg, pendingInterrupt, buildMessagesFunc)
	}

	// 正常处理流程
	history := i.convertHistory(sess.GetHistory(10))
	messages := buildMessagesFunc(history, msg.Content, msg.Channel, msg.ChatID)
	checkpointID := fmt.Sprintf("%s_%d", sessionKey, time.Now().UnixNano())

	response, err := i.processNormal(ctx, messages, checkpointID, msg)
	if err != nil {
		if isInterruptError(err) {
			return "", err
		}
		// 非中断错误：返回错误信息作为响应，让上层处理
		return fmt.Sprintf("处理失败: %v", err), err
	}

	i.saveSession(sess, msg.Content)

	// 执行消息后处理 Hook
	if i.hookManager != nil {
		if err := i.hookManager.ExecuteAfterMessageProcess(ctx, msg, sess, response); err != nil {
			i.logger.Error("Hook 执行失败", zap.Error(err))
		}
	}

	return response, nil
}

// processInterrupted 处理中断恢复流程
func (i *interruptible) processInterrupted(ctx context.Context, sess *session.Session, msg *bus.InboundMessage, pendingInterrupt *InterruptInfo, buildMessagesFunc func(history []*schema.Message, userInput, channel, chatID string) []*schema.Message) (string, error) {
	sessionKey := msg.SessionKey()

	// 使用原始 checkpoint ID 进行恢复
	resumeCheckpointID := pendingInterrupt.OriginalCheckpointID
	if resumeCheckpointID == "" {
		resumeCheckpointID = pendingInterrupt.CheckpointID
	}

	i.logger.Info("检测到待处理的中断，尝试恢复执行",
		zap.String("checkpoint_id", resumeCheckpointID),
		zap.String("original_checkpoint_id", pendingInterrupt.OriginalCheckpointID),
		zap.String("session_key", sessionKey),
		zap.String("agent_type", i.agentType),
	)

	// 提交用户响应
	response := &UserResponse{
		CheckpointID: resumeCheckpointID,
		Answer:       msg.Content,
	}
	if err := i.interruptManager.SubmitUserResponse(response); err != nil {
		return "", fmt.Errorf("提交用户响应失败: %w", err)
	}

	// 准备恢复参数
	resumePayload := i.buildResumePayload(pendingInterrupt.IsAskUser, msg.Content)
	resumeParams := &adk.ResumeParams{
		Targets: map[string]any{
			pendingInterrupt.InterruptID: resumePayload,
		},
	}

	resumeMsg := &bus.InboundMessage{
		Channel:  msg.Channel,
		ChatID:   msg.ChatID,
		SenderID: sessionKey,
	}

	// 恢复执行
	result, err := i.Resume(ctx, resumeCheckpointID, resumeParams, resumeMsg)
	if err != nil {
		if isInterruptError(err) {
			return "", err
		}
		// 返回错误信息作为响应，让上层处理
		return fmt.Sprintf("恢复执行失败: %v", err), err
	}

	// 清理已完成的中断
	i.interruptManager.ClearInterrupt(pendingInterrupt.CheckpointID)
	i.saveSession(sess, msg.Content)

	return result, nil
}

// buildResumePayload 构建恢复参数的有效载荷
func (i *interruptible) buildResumePayload(isAskUser bool, userAnswer string) any {
	if isAskUser {
		return &askuser.AskUserInfo{
			UserAnswer: userAnswer,
		}
	}
	return map[string]any{
		"user_answer": userAnswer,
	}
}

// saveSession 保存用户消息到会话
func (i *interruptible) saveSession(sess *session.Session, userMessage string) {
	sess.AddMessage("user", userMessage)
	i.sessions.Save(sess)
}

// processNormal 普通模式处理
func (i *interruptible) processNormal(ctx context.Context, messages []*schema.Message, checkpointID string, msg *bus.InboundMessage) (string, error) {
	iter := i.adkRunner.Run(ctx, messages, adk.WithCheckPointID(checkpointID))

	var response string
	var lastEvent *adk.AgentEvent

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return "", fmt.Errorf("%s 执行失败: %w", i.agentType, event.Err)
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
		return "", i.handleInterrupt(msg, checkpointID, checkpointID, lastEvent)
	}

	return response, nil
}

// Resume 恢复被中断的执行
func (i *interruptible) Resume(ctx context.Context, checkpointID string, resumeParams *adk.ResumeParams, msg *bus.InboundMessage) (string, error) {
	if i.adkRunner == nil {
		return "", fmt.Errorf("ADK Runner 未初始化")
	}

	sessionKey := msg.SenderID
	ctx = context.WithValue(ctx, SessionKeyContextKey, sessionKey)

	iter, err := i.adkRunner.ResumeWithParams(ctx, checkpointID, resumeParams)
	if err != nil {
		return "", fmt.Errorf("%s 恢复执行失败: %w", i.agentType, err)
	}

	var response string
	var lastEvent *adk.AgentEvent

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return "", fmt.Errorf("%s 恢复后执行失败: %w", i.agentType, event.Err)
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
		newCheckpointID := fmt.Sprintf("%s_resume_%d", checkpointID, time.Now().UnixNano())
		return "", i.handleInterrupt(msg, newCheckpointID, checkpointID, lastEvent)
	}

	return response, nil
}

// handleInterrupt 处理中断
func (i *interruptible) handleInterrupt(msg *bus.InboundMessage, checkpointID string, originalCheckpointID string, event *adk.AgentEvent) error {
	if event.Action == nil || event.Action.Interrupted == nil {
		return nil
	}

	interruptCtx := event.Action.Interrupted.InterruptContexts[0]
	interruptID := interruptCtx.ID

	// 如果没有提供原始 checkpointID，使用当前的
	if originalCheckpointID == "" {
		originalCheckpointID = checkpointID
	}

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
	i.interruptManager.HandleInterrupt(&InterruptInfo{
		CheckpointID:         checkpointID,
		OriginalCheckpointID: originalCheckpointID,
		InterruptID:          interruptID,
		Channel:              msg.Channel,
		ChatID:               msg.ChatID,
		Question:             question,
		Options:              options,
		SessionKey:           msg.SessionKey(),
		IsAskUser:            isAskUser,
		IsMaster:             i.agentType == "master",
		IsSupervisor:         i.agentType == "supervisor",
	})

	i.logger.Info("等待用户输入以恢复执行",
		zap.String("checkpoint_id", checkpointID),
		zap.String("question", question),
		zap.String("agent_type", i.agentType),
	)

	return fmt.Errorf("%s%s:%s", interruptErrorPrefix, checkpointID, interruptID)
}

// convertHistory 转换会话历史
func (i *interruptible) convertHistory(history []map[string]any) []*schema.Message {
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
