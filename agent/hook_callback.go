package agent

import (
	"context"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	hooks "github.com/weibaohui/nanobot-go/agent/hooks"
	"github.com/weibaohui/nanobot-go/agent/hooks/events"
	"go.uber.org/zap"
)

// CreateHookCallback 创建一个将事件转发到 HookManager 的回调函数
// 用于在 interruptible.BuildChatModelAdapter 和 task_manager.executeTask 中避免代码重复
func CreateHookCallback(hookManager *hooks.HookManager, logger *zap.Logger) HookCallback {
	if hookManager == nil {
		return nil
	}

	return func(eventType events.EventType, data map[string]interface{}) {
		// 从 data 中提取 session_key 和 channel
		var sessionKey, channel string
		if sk, ok := data["session_key"].(string); ok {
			sessionKey = sk
		}
		if ch, ok := data["channel"].(string); ok {
			channel = ch
		}

		// 创建事件并分发
		ctx := context.Background()
		baseEvent := &events.BaseEvent{
			TraceID:   hooks.GetTraceID(ctx),
			EventType: eventType,
			Timestamp: time.Now(),
		}

		// 根据事件类型创建具体事件
		switch eventType {
		case events.EventLLMCallStart:
			event := &events.LLMCallStartEvent{
				BaseEvent: baseEvent,
			}
			if spanID, ok := data["span_id"].(string); ok {
				event.SpanID = spanID
			}
			if parentSpanID, ok := data["parent_span_id"].(string); ok {
				event.ParentSpanID = parentSpanID
			}
			if toolNames, ok := data["tool_names"].([]string); ok {
				event.ToolNames = toolNames
			}
			if messages, ok := data["messages"].([]*schema.Message); ok {
				event.Messages = messages
			}
			hookManager.Dispatch(ctx, event, channel, sessionKey)

		case events.EventLLMCallEnd:
			event := &events.LLMCallEndEvent{
				BaseEvent: baseEvent,
			}
			// 从 data 中提取 TokenUsage
			if tokenUsage, ok := data["token_usage"].(*schema.TokenUsage); ok && tokenUsage != nil {
				event.TokenUsage = &model.TokenUsage{
					PromptTokens:            tokenUsage.PromptTokens,
					PromptTokenDetails:      model.PromptTokenDetails(tokenUsage.PromptTokenDetails),
					CompletionTokens:        tokenUsage.CompletionTokens,
					TotalTokens:             tokenUsage.TotalTokens,
					CompletionTokensDetails: model.CompletionTokensDetails(tokenUsage.CompletionTokensDetails),
				}
			}
			if spanID, ok := data["span_id"].(string); ok {
				event.SpanID = spanID
			}
			if parentSpanID, ok := data["parent_span_id"].(string); ok {
				event.ParentSpanID = parentSpanID
			}
			if response, ok := data["response"].(string); ok {
				event.ResponseContent = response
			}
			if toolCalls, ok := data["tool_calls"].([]schema.ToolCall); ok {
				event.ToolCalls = toolCalls
			}
			hookManager.Dispatch(ctx, event, channel, sessionKey)

		case events.EventLLMCallError:
			event := &events.LLMCallErrorEvent{
				BaseEvent: baseEvent,
			}
			if spanID, ok := data["span_id"].(string); ok {
				event.SpanID = spanID
			}
			if parentSpanID, ok := data["parent_span_id"].(string); ok {
				event.ParentSpanID = parentSpanID
			}
			if errMsg, ok := data["error"].(string); ok {
				event.Error = errMsg
			}
			hookManager.Dispatch(ctx, event, channel, sessionKey)

		default:
			// 其他事件类型，直接分发 BaseEvent
			hookManager.Dispatch(ctx, baseEvent, channel, sessionKey)
		}
	}
}
