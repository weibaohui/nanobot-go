package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	hooks "github.com/weibaohui/nanobot-go/pkg/agent/hooks"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/trace"
	"github.com/weibaohui/nanobot-go/pkg/agent/interrupt"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/askuser"
	"github.com/weibaohui/nanobot-go/pkg/bus"
	"github.com/weibaohui/nanobot-go/pkg/session"
	"go.uber.org/zap"
)

var interruptErrorPrefix = "INTERRUPT:"

// buildChatModelAdapter 创建并配置 ChatModelAdapter
// 将 LLM 初始化逻辑集中在此，避免遗漏必要配置
func buildChatModelAdapter(logger *zap.Logger, configLoader LLMConfigLoader, sessions *session.Manager, skillsLoader func(string) string, registeredTools []string, hookCallback func(eventType events.EventType, data map[string]interface{})) (*ChatModelAdapter, error) {
	llm, err := NewChatModelAdapter(logger, configLoader, sessions)
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

	// 设置 Hook Callback
	if hookCallback != nil {
		llm.SetHookCallback(hookCallback)
	}

	return llm, nil
}

// interruptible 可嵌入的中断处理能力
// 为 Agent 提供中断处理、恢复执行等通用能力
type interruptible struct {
	configLoader     LLMConfigLoader
	workspace        string
	tools            []tool.BaseTool
	logger           *zap.Logger
	sessions         *session.Manager
	bus              *bus.MessageBus
	context          *ContextBuilder
	adkRunner        *adk.Runner
	interruptManager *interrupt.Manager
	checkpointStore  compose.CheckPointStore
	registeredTools  []string
	maxIterations    int
	agentType        string // "master" 或 "supervisor"
	adkAgent         adk.Agent
	hookManager      *hooks.HookManager
}

// interruptibleConfig 中断处理能力的配置
type interruptibleConfig struct {
	ConfigLoader    LLMConfigLoader
	Workspace       string
	Tools           []tool.BaseTool
	Logger          *zap.Logger
	Sessions        *session.Manager
	Bus             *bus.MessageBus
	Context         *ContextBuilder
	InterruptMgr    *interrupt.Manager
	CheckpointStore compose.CheckPointStore
	MaxIterations   int
	RegisteredTools []string
	AgentType       string
	ADKAgent        adk.Agent
	ADKRunner       *adk.Runner
	HookManager     *hooks.HookManager
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
		configLoader:     cfg.ConfigLoader,
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
	llm, err := NewChatModelAdapter(i.logger, i.configLoader, i.sessions)
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

	// 设置 HookCallback - 将事件转发到 HookManager
	if hookCallback := CreateHookCallback(i.hookManager, i.logger); hookCallback != nil {
		llm.SetHookCallback(hookCallback)
	}

	return llm, nil
}

// Process 处理用户消息的统一入口
// 包含中断检查和恢复逻辑
func (i *interruptible) Process(ctx context.Context, msg *bus.InboundMessage, buildMessagesFunc func(history []*schema.Message, userInput, channel, chatID string) []*schema.Message) (string, error) {
	sessionKey := msg.SessionKey()
	sess := i.sessions.GetOrCreate(sessionKey)

	ctx = context.WithValue(ctx, SessionKeyContextKey, sessionKey)
	// 同时存储为 "session_key" 供 SessionObserver 使用
	ctx = context.WithValue(ctx, "session_key", sessionKey)

	// 创建 Agent 处理 span
	ctx, agentSpanID := trace.StartSpan(ctx)
	i.logger.Debug("Agent 处理开始",
		zap.String("agent_type", i.agentType),
		zap.String("span_id", agentSpanID),
	)

	// 检查是否有待处理的中断需要响应
	if pendingInterrupt := i.interruptManager.GetPendingInterrupt(sessionKey); pendingInterrupt != nil {
		return i.processInterrupted(ctx, sess, msg, pendingInterrupt, buildMessagesFunc)
	}

	// Normal processing flow
	// 从 AgentConfig 获取历史消息数量配置（默认10，最大50）
	historyWindow := 10
	if i.context != nil {
		historyWindow = i.context.GetHistoryMessages()
	}
	history := i.convertHistory(i.sessions.GetHistory(ctx, sessionKey, historyWindow))
	i.logger.Debug("加载会话历史",
		zap.Int("history_window", historyWindow),
		zap.Int("loaded_messages", len(history)))
	messages := buildMessagesFunc(history, msg.Content, msg.Channel, msg.ChatID)
	checkpointID := fmt.Sprintf("%s_%d", sessionKey, time.Now().UnixNano())

	// 触发 PromptSubmitted 事件，让 SessionObserver 保存用户消息
	if i.hookManager != nil {
		i.hookManager.OnPromptSubmitted(ctx, msg.Content, messages, sessionKey)
	}

	response, err := i.processNormal(ctx, messages, checkpointID, msg)
	if err != nil {
		if IsInterruptError(err) {
			return "", err
		}
		// 非中断错误：返回错误信息作为响应和错误，让上层处理
		errorMsg := fmt.Sprintf("处理失败: %v", err)
		return errorMsg, fmt.Errorf("%s", errorMsg)
	}

	return response, nil
}

// processInterrupted 处理中断恢复流程
func (i *interruptible) processInterrupted(ctx context.Context, sess *session.Session, msg *bus.InboundMessage, pendingInterrupt *interrupt.InterruptInfo, buildMessagesFunc func(history []*schema.Message, userInput, channel, chatID string) []*schema.Message) (string, error) {
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
	response := &interrupt.UserResponse{
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
		if IsInterruptError(err) {
			return "", err
		}
		// 返回错误信息作为响应和错误，让上层处理
		errorMsg := fmt.Sprintf("恢复执行失败: %v", err)
		return errorMsg, fmt.Errorf("%s", errorMsg)
	}

	// 清理已完成的中断
	i.interruptManager.ClearInterrupt(pendingInterrupt.CheckpointID)

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

// processNormal 普通模式处理
func (i *interruptible) processNormal(ctx context.Context, messages []*schema.Message, checkpointID string, msg *bus.InboundMessage) (string, error) {
	// 创建 ADK 执行 span
	ctx, adkSpanID := trace.StartSpan(ctx)
	i.logger.Debug("ADK Runner 执行开始",
		zap.String("agent_type", i.agentType),
		zap.String("span_id", adkSpanID),
		zap.String("checkpoint_id", checkpointID),
	)

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

	// 创建恢复执行 span
	ctx, resumeSpanID := trace.StartSpan(ctx)
	i.logger.Debug("恢复执行开始",
		zap.String("agent_type", i.agentType),
		zap.String("span_id", resumeSpanID),
		zap.String("checkpoint_id", checkpointID),
	)

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
	i.interruptManager.HandleInterrupt(&interrupt.InterruptInfo{
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
		// 获取角色
		roleStr, _ := h["role"].(string)

		// 跳过工具相关消息（tool 和 tool_result）
		// 工具调用上下文由 Eino 框架内部维护，不需要在 Session 中保存
		if roleStr == "tool" || roleStr == "tool_result" {
			i.logger.Debug("跳过工具消息",
				zap.String("role", roleStr),
				zap.String("content_preview", fmt.Sprintf("%.50v", h["content"])),
			)
			continue
		}

		role := schema.User
		if roleStr == "assistant" {
			role = schema.Assistant
		}

		msg := &schema.Message{
			Role: role,
		}

		// 处理 content：可能是字符串或多部分内容
		content := h["content"]
		if contentStr, ok := content.(string); ok {
			// 纯文本内容
			msg.Content = contentStr
		} else if contentSlice, ok := content.([]map[string]any); ok {
			// 多部分内容（例如包含图片）
			msg.UserInputMultiContent = make([]schema.MessageInputPart, 0, len(contentSlice))
			for _, part := range contentSlice {
				partType, _ := part["type"].(string)
				switch partType {
				case "text":
					if text, ok := part["text"].(string); ok {
						msg.UserInputMultiContent = append(msg.UserInputMultiContent, schema.MessageInputPart{
							Type: schema.ChatMessagePartTypeText,
							Text: text,
						})
					}
				case "image_url":
					if imageObj, ok := part["image_url"].(map[string]any); ok {
						if url, ok := imageObj["url"].(string); ok {
							msg.UserInputMultiContent = append(msg.UserInputMultiContent, schema.MessageInputPart{
								Type: schema.ChatMessagePartTypeImageURL,
								Image: &schema.MessageInputImage{
									MessagePartCommon: schema.MessagePartCommon{
										URL: &url,
									},
									Detail: schema.ImageURLDetailAuto,
								},
							})
						}
					}
				}
			}
		} else {
			// 其他类型，尝试转为字符串作为后备
			if content != nil {
				msg.Content = fmt.Sprintf("%v", content)
			}
		}

		result = append(result, msg)
	}
	return result
}
