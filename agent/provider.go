package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/agent/hooks/trace"
	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/session"
	"go.uber.org/zap"
)

// ContextKey session key 的 context key
type ContextKey string

const SessionKeyContextKey ContextKey = "session_key"

// SkillLoader 技能加载函数类型
type SkillLoader func(name string) string

// HookCallback Hook 回调函数类型
// 避免循环导入，使用函数回调而不是直接依赖 Hook 系统
type HookCallback func(eventType events.EventType, data map[string]interface{})

// ChatModelAdapter 包装 eino 的 ChatModel，添加工具调用拦截功能
// 主要功能：
//   - 实现 model.ToolCallingChatModel 接口
//   - 拦截未注册的工具调用，转换为技能调用
type ChatModelAdapter struct {
	logger        *zap.Logger
	chatModel     model.ToolCallingChatModel
	registeredMap map[string]bool  // 已注册的工具名称
	skillLoader   SkillLoader      // 技能加载器
	sessions      *session.Manager // 会话管理器，用于记录 token 用量
	hookCallback  HookCallback     // Hook 回调函数
}

// Sentinel errors 定义包级别的错误常量
var (
	ErrNilConfig       = fmt.Errorf("配置不能为空")
	ErrCreateChatModel = fmt.Errorf("创建 ChatModel 失败")
	ErrNilAPIKey       = fmt.Errorf("API Key 不能为空")
)

func createChatModelConfig(logger *zap.Logger, cfg *config.Config) (apiKey, apiBase, modelName string, err error) {
	if cfg == nil {
		return "", "", "", ErrNilConfig
	}

	providerCfg := cfg.GetProvider(cfg.Agents.Defaults.Model)
	if providerCfg == nil || providerCfg.APIKey == "" {
		logger.Warn("未找到有效的 API Key，请设置环境变量")
		return "", "", "gpt-4o-mini", ErrNilAPIKey
	}

	apiBase = providerCfg.APIBase
	if apiBase == "" {
		apiBase = "https://api.openai.com/v1"
	}

	return providerCfg.APIKey, apiBase, cfg.Agents.Defaults.Model, nil
}

// NewChatModelAdapter 创建 ChatModel 适配器
func NewChatModelAdapter(logger *zap.Logger, cfg *config.Config, sessions *session.Manager) (*ChatModelAdapter, error) {
	apiKey, apiBase, modelName, err := createChatModelConfig(logger, cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNilConfig, err)
	}

	chatModel, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		APIKey:  apiKey,
		Model:   modelName,
		BaseURL: apiBase,
	})
	if err != nil {
		if logger != nil {
			logger.Error("创建 OpenAI ChatModel 失败", zap.Error(err))
		}
		return nil, fmt.Errorf("%w: %w", ErrCreateChatModel, err)
	}

	return &ChatModelAdapter{
		logger:        logger,
		chatModel:     chatModel,
		registeredMap: make(map[string]bool),
		sessions:      sessions,
	}, nil
}

// SetSkillLoader 设置技能加载器
func (a *ChatModelAdapter) SetSkillLoader(loader SkillLoader) {
	a.skillLoader = loader
}

// SetHookCallback 设置 Hook 回调函数
func (a *ChatModelAdapter) SetHookCallback(callback HookCallback) {
	a.hookCallback = callback
}

// SetRegisteredTools 设置已注册的工具名称列表
func (a *ChatModelAdapter) SetRegisteredTools(names []string) {
	a.registeredMap = make(map[string]bool)
	for _, name := range names {
		a.registeredMap[name] = true
	}
}

// isRegisteredTool 检查工具是否已注册
func (a *ChatModelAdapter) isRegisteredTool(name string) bool {
	if a.registeredMap == nil {
		return false
	}
	return a.registeredMap[name]
}

// isKnownSkill 检查是否是已知技能
func (a *ChatModelAdapter) isKnownSkill(name string) bool {
	if a.skillLoader == nil {
		return false
	}
	content := a.skillLoader(name)
	return content != ""
}

// Generate produces a complete model response
func (a *ChatModelAdapter) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	// 创建 LLM 调用 span
	ctx, llmSpanID := trace.StartSpan(ctx)
	a.logger.Debug("LLM 调用开始",
		zap.String("span_id", llmSpanID),
		zap.Int("message_count", len(input)),
	)

	// 触发 LLM 调用开始事件
	a.triggerLLMCallStart(ctx, input)

	// 调试：记录输入消息
	if a.logger != nil && len(input) > 0 {
		for i, msg := range input {
			a.logger.Debug("[LLM] 发送消息",
				zap.Int("index", i),
				zap.String("role", string(msg.Role)),
				zap.String("content_preview", func() string {
					preview := msg.Content
					if len(preview) > 200 {
						preview = preview[:200] + "..."
					}
					return preview
				}()),
			)
		}
	}

	// 调用底层 ChatModel
	response, err := a.chatModel.Generate(ctx, input, opts...)
	if err != nil {
		if a.logger != nil {
			a.logger.Error("调用 LLM 失败", zap.Error(err))
		}
		// 触发 LLM 调用错误事件
		a.triggerLLMCallError(ctx, err)
		return nil, err
	}

	a.logger.Debug("LLM 调用完成",
		zap.String("span_id", llmSpanID),
		zap.Int("tool_calls", len(response.ToolCalls)),
	)

	// 触发 LLM 调用结束事件（包含 Token 使用）
	a.triggerLLMCallEnd(ctx, response)

	// 拦截并转换工具调用
	a.interceptToolCalls(response)

	return response, nil
}

// interceptToolCall 拦截工具调用，如果工具不存在则转换为技能调用
func (a *ChatModelAdapter) interceptToolCall(toolName string, argumentsJSON string) (string, string, error) {

	// 如果工具是 use_skill，解析参数触发 skill_call Hook
	if toolName == "use_skill" {
		var args map[string]any
		if err := json.Unmarshal([]byte(argumentsJSON), &args); err == nil {
			if skillName, ok := args["skill_name"].(string); ok {
				skillContent := ""
				if a.skillLoader != nil {
					skillContent = a.skillLoader(skillName)
				}
				// 触发技能调用 Hook
				a.triggerHook(events.EventSkillCall, map[string]any{
					"skill_name":   skillName,
					"skill_length": len(skillContent),
				})
			}
		}
		return toolName, argumentsJSON, nil
	}

	// 如果工具已注册，不拦截
	if a.isRegisteredTool(toolName) {
		a.logger.Info("工具已注册，不拦截", zap.String("名称", toolName))
		return toolName, argumentsJSON, nil
	}

	// 如果是已知技能，将工具调用转换为技能调用
	if a.isKnownSkill(toolName) {
		a.logger.Info("isKnownSkill 工具转换为技能调用", zap.String("名称", toolName))
		// 解析原始参数
		var originalArgs map[string]any
		if err := json.Unmarshal([]byte(argumentsJSON), &originalArgs); err != nil {
			originalArgs = make(map[string]any)
		}

		// 将原始参数包装成技能参数
		skillParams := map[string]any{
			"skill_name": toolName,
			"action":     originalArgs["action"],
		}

		// 移除 action 后的其他参数放入 params
		filteredParams := make(map[string]any)
		for k, v := range originalArgs {
			if k != "action" {
				filteredParams[k] = v
			}
		}
		if len(filteredParams) > 0 {
			skillParams["params"] = filteredParams
		}

		// 序列化新参数
		newArgsJSON, err := json.Marshal(skillParams)
		if err != nil {
			return toolName, argumentsJSON, err
		}

		// 返回 use_skill 作为工具名
		return "use_skill", string(newArgsJSON), nil
	}

	a.logger.Info("既不是工具也不是技能，保持原样", zap.String("名称", toolName))
	// 既不是工具也不是技能，保持原样（会在执行时报错）
	return toolName, argumentsJSON, nil
}

// interceptToolCalls 拦截并转换工具调用
func (a *ChatModelAdapter) interceptToolCalls(msg *schema.Message) {
	if len(msg.ToolCalls) == 0 {
		return
	}

	for i, tc := range msg.ToolCalls {
		// 触发工具调用 Hook
		a.triggerHook(events.EventToolCall, map[string]any{
			"tool_name": tc.Function.Name,
			"arguments": tc.Function.Arguments,
		})

		newName, newArgs, err := a.interceptToolCall(tc.Function.Name, tc.Function.Arguments)

		if err != nil {
			continue
		}
		if newName != tc.Function.Name {
			// 触发工具调用被拦截 Hook
			a.triggerHook(events.EventToolIntercepted, map[string]any{
				"original_name": tc.Function.Name,
				"new_name":      newName,
				"new_args":      newArgs,
			})
			// 工具名被修改了，更新工具调用
			msg.ToolCalls[i].Function.Name = newName
			msg.ToolCalls[i].Function.Arguments = newArgs
		}
	}
}

// triggerHook 触发 Hook 事件
func (a *ChatModelAdapter) triggerHook(eventType events.EventType, data map[string]any) {
	if a.hookCallback == nil {
		return
	}
	// 转换 data 从 map[string]any 到 map[string]interface{}
	dataInterface := make(map[string]interface{})
	for k, v := range data {
		dataInterface[k] = v
	}
	a.hookCallback(eventType, dataInterface)
}

// triggerLLMCallStart 触发 LLM 调用开始事件
func (a *ChatModelAdapter) triggerLLMCallStart(ctx context.Context, input []*schema.Message) {
	if a.hookCallback == nil {
		return
	}

	traceID := trace.GetTraceID(ctx)
	spanID := trace.GetSpanID(ctx)
	parentSpanID := trace.GetParentSpanID(ctx)
	sessionKey := trace.GetSessionKey(ctx)
	channel := trace.GetChannel(ctx)

	// 提取工具名称列表
	var toolNames []string
	for _, msg := range input {
		for _, tc := range msg.ToolCalls {
			toolNames = append(toolNames, tc.Function.Name)
		}
	}

	// 直接构造事件数据
	data := map[string]interface{}{
		"event_type":     events.EventLLMCallStart,
		"trace_id":       traceID,
		"span_id":        spanID,
		"parent_span_id": parentSpanID,
		"session_key":    sessionKey,
		"channel":        channel,
		"input_count":    len(input),
		"tool_names":     toolNames,
		"messages":       input,
	}
	a.hookCallback(events.EventLLMCallStart, data)
}

// triggerLLMCallEnd 触发 LLM 调用结束事件
func (a *ChatModelAdapter) triggerLLMCallEnd(ctx context.Context, response *schema.Message) {
	if a.hookCallback == nil {
		return
	}

	traceID := trace.GetTraceID(ctx)
	spanID := trace.GetSpanID(ctx)
	parentSpanID := trace.GetParentSpanID(ctx)
	sessionKey := trace.GetSessionKey(ctx)
	channel := trace.GetChannel(ctx)

	// 提取 Token 使用信息
	var tokenUsage *schema.TokenUsage
	if response.ResponseMeta != nil && response.ResponseMeta.Usage != nil {
		tokenUsage = response.ResponseMeta.Usage
	}

	// 提取工具调用信息
	toolCalls := response.ToolCalls

	// 直接构造事件数据
	data := map[string]interface{}{
		"event_type":     events.EventLLMCallEnd,
		"trace_id":       traceID,
		"span_id":        spanID,
		"parent_span_id": parentSpanID,
		"session_key":    sessionKey,
		"channel":        channel,
		"response":       response.Content,
		"tool_calls":     toolCalls,
		"token_usage":    tokenUsage,
	}
	a.hookCallback(events.EventLLMCallEnd, data)
}

// triggerLLMCallError 触发 LLM 调用错误事件
func (a *ChatModelAdapter) triggerLLMCallError(ctx context.Context, err error) {
	if a.hookCallback == nil {
		return
	}

	traceID := trace.GetTraceID(ctx)
	spanID := trace.GetSpanID(ctx)
	parentSpanID := trace.GetParentSpanID(ctx)
	sessionKey := trace.GetSessionKey(ctx)
	channel := trace.GetChannel(ctx)

	// 直接构造事件数据
	data := map[string]interface{}{
		"event_type":     events.EventLLMCallError,
		"trace_id":       traceID,
		"span_id":        spanID,
		"parent_span_id": parentSpanID,
		"session_key":    sessionKey,
		"channel":        channel,
		"error":          err.Error(),
	}
	a.hookCallback(events.EventLLMCallError, data)
}

// Stream produces a response as a stream
// 注意：为避免流式响应解析问题，这里使用 Generate 并模拟流式输出
func (a *ChatModelAdapter) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	// 直接调用 Generate 获取完整响应
	msg, err := a.Generate(ctx, input, opts...)
	if err != nil {
		return nil, err
	}

	// 创建 StreamReader 并发送完整消息
	sr, sw := schema.Pipe[*schema.Message](1)
	go func() {
		defer sw.Close()
		sw.Send(msg, nil)
	}()

	return sr, nil
}

// WithTools returns a new adapter instance with the specified tools bound
func (a *ChatModelAdapter) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	// 绑定工具到底层 ChatModel
	boundModel, err := a.chatModel.WithTools(tools)
	if err != nil {
		return nil, err
	}

	return &ChatModelAdapter{
		logger:        a.logger,
		chatModel:     boundModel,
		registeredMap: a.registeredMap,
		skillLoader:   a.skillLoader,
		sessions:      a.sessions,
		hookCallback:  a.hookCallback, // 复制 hookCallback
	}, nil
}
