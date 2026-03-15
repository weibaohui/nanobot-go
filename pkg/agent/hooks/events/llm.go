package events

import (
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// LLMCallStartEvent LLM 调用开始事件 (来自 Eino callbacks)
type LLMCallStartEvent struct {
	*BaseEvent
	Component string                 `json:"component"`  // 组件名称
	Model     string                 `json:"model"`      // 模型名称
	Messages  []*schema.Message      `json:"messages"`   // 消息列表
	ToolNames []string               `json:"tool_names"` // 工具名称列表
	Config    map[string]interface{} `json:"config"`     // 配置
}

// NewLLMCallStartEvent 创建 LLM 调用开始事件
func NewLLMCallStartEvent(traceID, spanID, parentSpanID string, info *callbacks.RunInfo, input *model.CallbackInput) *LLMCallStartEvent {
	toolNames := make([]string, 0, len(input.Tools))
	for _, t := range input.Tools {
		if t != nil {
			toolNames = append(toolNames, t.Name)
		}
	}

	config := make(map[string]interface{})
	if input.Config != nil {
		config["model"] = input.Config.Model
		config["max_tokens"] = input.Config.MaxTokens
		config["temperature"] = input.Config.Temperature
		config["top_p"] = input.Config.TopP
	}

	return &LLMCallStartEvent{
		BaseEvent: NewBaseEvent(traceID, spanID, parentSpanID, EventLLMCallStart),
		Component: string(info.Component),
		Model:     info.Name,
		Messages:  input.Messages,
		ToolNames: toolNames,
		Config:    config,
	}
}

// LLMCallEndEvent LLM 调用结束事件 (来自 Eino callbacks)
type LLMCallEndEvent struct {
	*BaseEvent
	Component       string            `json:"component"`        // 组件名称
	Model           string            `json:"model"`            // 模型名称
	ResponseContent string            `json:"response_content"` // 响应内容
	ToolCalls       []schema.ToolCall `json:"tool_calls"`       // 工具调用列表
	TokenUsage      *model.TokenUsage `json:"token_usage"`      // Token 使用情况
	DurationMs      int64             `json:"duration_ms"`      // 持续时间 (毫秒)
}

// NewLLMCallEndEvent 创建 LLM 调用结束事件
func NewLLMCallEndEvent(traceID, spanID, parentSpanID string, info *callbacks.RunInfo, output *model.CallbackOutput, durationMs int64) *LLMCallEndEvent {
	responseContent := ""
	toolCalls := []schema.ToolCall{}
	if output.Message != nil {
		responseContent = output.Message.Content
		toolCalls = output.Message.ToolCalls
	}

	return &LLMCallEndEvent{
		BaseEvent:       NewBaseEvent(traceID, spanID, parentSpanID, EventLLMCallEnd),
		Component:       string(info.Component),
		Model:           info.Name,
		ResponseContent: responseContent,
		ToolCalls:       toolCalls,
		TokenUsage:      output.TokenUsage,
		DurationMs:      durationMs,
	}
}

// LLMCallErrorEvent LLM 调用错误事件 (来自 Eino callbacks)
type LLMCallErrorEvent struct {
	*BaseEvent
	Component  string `json:"component"`   // 组件名称
	Model      string `json:"model"`       // 模型名称
	Error      string `json:"error"`       // 错误信息
	DurationMs int64  `json:"duration_ms"` // 持续时间 (毫秒)
}

// NewLLMCallErrorEvent 创建 LLM 调用错误事件
func NewLLMCallErrorEvent(traceID, spanID, parentSpanID string, info *callbacks.RunInfo, err error, durationMs int64) *LLMCallErrorEvent {
	return &LLMCallErrorEvent{
		BaseEvent:  NewBaseEvent(traceID, spanID, parentSpanID, EventLLMCallError),
		Component:  string(info.Component),
		Model:      info.Name,
		Error:      err.Error(),
		DurationMs: durationMs,
	}
}
