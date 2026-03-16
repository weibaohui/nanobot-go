package app

import (
	"context"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/observers"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/trace"
	"github.com/weibaohui/nanobot-go/pkg/bus"
	"github.com/weibaohui/nanobot-go/config"
	"go.uber.org/zap"
)

// HookComponents Hook系统组件
type HookComponents struct {
	Manager *hooks.HookManager
	// Hook回调函数
	Callback func(eventType events.EventType, data map[string]interface{})
}

// InitHookSystem 初始化Hook系统
func InitHookSystem(cfg *config.Config, messageBus *bus.MessageBus, logger *zap.Logger) *HookComponents {
	// 创建统一的 Hook 系统
	hookSystem := hooks.NewHookManager(logger, true)

	// 注册 LoggingObserver
	loggingObserver := observers.NewLoggingObserver(logger, nil)
	hookSystem.Register(loggingObserver)
	logger.Info("日志观察器已注册到 Hook 系统",
		zap.Strings("events", []string{
			"message_received", "message_sent",
			"prompt_submitted", "system_prompt_built",
			"tool_call", "tool_intercepted", "tool_used", "tool_completed", "tool_error",
			"skill_call", "skill_lookup", "skill_used",
			"llm_call_start", "llm_call_end", "llm_call_error",
			"component_start", "component_end", "component_error",
		}),
	)

	// 注册 ThinkingProcessObserver
	// 每个 Agent 可以独立控制是否启用思考过程输出
	thinkingProcessObserver := observers.NewThinkingProcessObserver(&cfg.ThinkingProcess, messageBus, logger, nil)
	hookSystem.Register(thinkingProcessObserver)
	logger.Info("思考过程观察器已注册",
		zap.Bool("global_enabled", cfg.ThinkingProcess.Enabled),
		zap.Strings("events", cfg.ThinkingProcess.Events),
	)

	// 注册 SQLiteObserver
	if sqliteObserver, err := observers.NewSQLiteObserverFromConfig(cfg, logger, nil); err != nil {
		logger.Error("创建 SQLite 观察器失败", zap.Error(err))
	} else if sqliteObserver != nil {
		hookSystem.Register(sqliteObserver)
		logger.Info("SQLite 观察器已注册到 Hook 系统", zap.String("db_path", sqliteObserver.GetDBPath()))
	}

	// 设置 Hook 回调
	callback := createHookCallback(hookSystem, logger)

	return &HookComponents{
		Manager:  hookSystem,
		Callback: callback,
	}
}

// createHookCallback 创建Hook回调函数
func createHookCallback(hookSystem *hooks.HookManager, logger *zap.Logger) func(eventType events.EventType, data map[string]interface{}) {
	return func(eventType events.EventType, data map[string]interface{}) {
		if !hookSystem.Enabled() {
			return
		}

		ctx := context.Background()
		var traceID string
		if tid, ok := data["trace_id"].(string); ok && tid != "" {
			traceID = tid
			ctx = hooks.WithTraceID(ctx, traceID)
		} else {
			traceID = hooks.GetTraceID(ctx)
		}
		if spanID, ok := data["span_id"].(string); ok && spanID != "" {
			ctx = trace.WithSpanID(ctx, spanID)
		}
		if parentSpanID, ok := data["parent_span_id"].(string); ok && parentSpanID != "" {
			ctx = trace.WithParentSpanID(ctx, parentSpanID)
		}
		// 从 data 中提取 enable_thinking_process 并设置到 context
		if enableThinking, ok := data["enable_thinking_process"].(bool); ok {
			ctx = trace.WithEnableThinkingProcess(ctx, enableThinking)
		}
		// 从 data 中提取 Code 字段并设置到 context
		if userCode, ok := data["user_code"].(string); ok && userCode != "" {
			ctx = trace.WithUserCode(ctx, userCode)
		}
		if agentCode, ok := data["agent_code"].(string); ok && agentCode != "" {
			ctx = trace.WithAgentCode(ctx, agentCode)
		}
		if channelCode, ok := data["channel_code"].(string); ok && channelCode != "" {
			ctx = trace.WithChannelCode(ctx, channelCode)
		}

		var sessionKey, channel string
		if sk, ok := data["session_key"].(string); ok {
			sessionKey = sk
		}
		if ch, ok := data["channel"].(string); ok {
			channel = ch
		}

		switch eventType {
		case events.EventLLMCallEnd:
			event := &events.LLMCallEndEvent{
				BaseEvent: &events.BaseEvent{
					TraceID:   traceID,
					EventType: eventType,
					Timestamp: time.Now(),
				},
			}
			tokenUsageRaw := data["token_usage"]
			if schemaUsage, ok := tokenUsageRaw.(*schema.TokenUsage); ok && schemaUsage != nil {
				event.TokenUsage = &model.TokenUsage{
					PromptTokens:            schemaUsage.PromptTokens,
					PromptTokenDetails:      model.PromptTokenDetails(schemaUsage.PromptTokenDetails),
					CompletionTokens:        schemaUsage.CompletionTokens,
					TotalTokens:             schemaUsage.TotalTokens,
					CompletionTokensDetails: model.CompletionTokensDetails(schemaUsage.CompletionTokensDetails),
				}
			}
			if spanID, ok := data["span_id"].(string); ok {
				event.SpanID = spanID
			}
			if parentSpanID, ok := data["parent_span_id"].(string); ok {
				event.ParentSpanID = parentSpanID
			}
			hookSystem.Dispatch(ctx, event, channel, sessionKey)

		case events.EventLLMCallStart:
			event := &events.LLMCallStartEvent{
				BaseEvent: &events.BaseEvent{
					TraceID:   traceID,
					EventType: eventType,
					Timestamp: time.Now(),
				},
			}
			if spanID, ok := data["span_id"].(string); ok {
				event.SpanID = spanID
			}
			if parentSpanID, ok := data["parent_span_id"].(string); ok {
				event.ParentSpanID = parentSpanID
			}
			hookSystem.Dispatch(ctx, event, channel, sessionKey)

		default:
			baseEvent := &events.BaseEvent{
				TraceID:   traceID,
				EventType: eventType,
				Timestamp: time.Now(),
			}
			hookSystem.Dispatch(ctx, baseEvent, channel, sessionKey)
		}
	}
}
