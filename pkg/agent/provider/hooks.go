package provider

import (
	"context"

	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/trace"
)

// triggerHook 触发 Hook 事件
func (a *ChatModelAdapter) triggerHook(eventType events.EventType, data map[string]any) {
	if a.hookCallback == nil {
		return
	}
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
	enableThinking := trace.GetEnableThinkingProcess(ctx)
	userCode := trace.GetUserCode(ctx)
	agentCode := trace.GetAgentCode(ctx)
	channelCode := trace.GetChannelCode(ctx)

	var toolNames []string
	for _, msg := range input {
		for _, tc := range msg.ToolCalls {
			toolNames = append(toolNames, tc.Function.Name)
		}
	}

	data := map[string]interface{}{
		"event_type":              events.EventLLMCallStart,
		"trace_id":                traceID,
		"span_id":                 spanID,
		"parent_span_id":          parentSpanID,
		"session_key":             sessionKey,
		"channel":                 channel,
		"input_count":             len(input),
		"tool_names":              toolNames,
		"messages":                input,
		"enable_thinking_process": enableThinking,
		"user_code":               userCode,
		"agent_code":              agentCode,
		"channel_code":            channelCode,
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
	enableThinking := trace.GetEnableThinkingProcess(ctx)
	userCode := trace.GetUserCode(ctx)
	agentCode := trace.GetAgentCode(ctx)
	channelCode := trace.GetChannelCode(ctx)

	var tokenUsage *schema.TokenUsage
	if response.ResponseMeta != nil && response.ResponseMeta.Usage != nil {
		tokenUsage = response.ResponseMeta.Usage
	}

	toolCalls := response.ToolCalls

	data := map[string]interface{}{
		"event_type":              events.EventLLMCallEnd,
		"trace_id":                traceID,
		"span_id":                 spanID,
		"parent_span_id":          parentSpanID,
		"session_key":             sessionKey,
		"channel":                 channel,
		"response":                response.Content,
		"tool_calls":              toolCalls,
		"token_usage":             tokenUsage,
		"enable_thinking_process": enableThinking,
		"user_code":               userCode,
		"agent_code":              agentCode,
		"channel_code":            channelCode,
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
	enableThinking := trace.GetEnableThinkingProcess(ctx)
	userCode := trace.GetUserCode(ctx)
	agentCode := trace.GetAgentCode(ctx)
	channelCode := trace.GetChannelCode(ctx)

	data := map[string]interface{}{
		"event_type":              events.EventLLMCallError,
		"trace_id":                traceID,
		"span_id":                 spanID,
		"parent_span_id":          parentSpanID,
		"session_key":             sessionKey,
		"channel":                 channel,
		"error":                   err.Error(),
		"enable_thinking_process": enableThinking,
		"user_code":               userCode,
		"agent_code":              agentCode,
		"channel_code":            channelCode,
	}
	a.hookCallback(events.EventLLMCallError, data)
}
