package logging

import (
	"context"
	"fmt"

	"github.com/weibaohui/nanobot-go/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/agent/hooks/observer"
	"go.uber.org/zap"
)

// LoggingObserver 日志观察器
// 将所有 Hook 事件记录到日志
type LoggingObserver struct {
	*observer.BaseObserver
	logger *zap.Logger
}

// NewLoggingObserver 创建日志观察器
func NewLoggingObserver(logger *zap.Logger, filter *observer.ObserverFilter) *LoggingObserver {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &LoggingObserver{
		BaseObserver: observer.NewBaseObserver("logging", filter),
		logger:       logger,
	}
}

// OnEvent 处理事件
func (lo *LoggingObserver) OnEvent(ctx context.Context, event events.Event) error {
	switch event.GetEventType() {
	case events.EventMessageReceived:
		lo.logMessageReceived(event)
	case events.EventMessageSent:
		lo.logMessageSent(event)
	case events.EventPromptSubmitted:
		lo.logPromptSubmitted(event)
	case events.EventSystemPromptBuilt:
		lo.logSystemPromptBuilt(event)
	case events.EventToolUsed:
		lo.logToolUsed(event)
	case events.EventToolCompleted:
		lo.logToolCompleted(event)
	case events.EventToolError:
		lo.logToolError(event)
	case events.EventSkillLookup:
		lo.logSkillLookup(event)
	case events.EventSkillUsed:
		lo.logSkillUsed(event)
	case events.EventLLMCallStart:
		lo.logLLMCallStart(event)
	case events.EventLLMCallEnd:
		lo.logLLMCallEnd(event)
	case events.EventLLMCallError:
		lo.logLLMCallError(event)
	case events.EventComponentStart:
		lo.logComponentStart(event)
	case events.EventComponentEnd:
		lo.logComponentEnd(event)
	case events.EventComponentError:
		lo.logComponentError(event)
	default:
		lo.logger.Debug("未知事件类型",
			zap.String("event_type", string(event.GetEventType())),
			zap.String("trace_id", event.GetTraceID()),
		)
	}
	return nil
}

func (lo *LoggingObserver) logMessageReceived(event events.Event) {
	base := event.ToBaseEvent()
	lo.logger.Info("[Hook] 收到消息",
		zap.String("trace_id", base.TraceID),
		zap.Any("event", event),
	)
}

func (lo *LoggingObserver) logMessageSent(event events.Event) {
	base := event.ToBaseEvent()
	lo.logger.Info("[Hook] 发送消息",
		zap.String("trace_id", base.TraceID),
		zap.Any("event", event),
	)
}

func (lo *LoggingObserver) logPromptSubmitted(event events.Event) {
	base := event.ToBaseEvent()
	lo.logger.Info("[Hook] 提交 Prompt",
		zap.String("trace_id", base.TraceID),
		zap.Any("event", event),
	)
}

func (lo *LoggingObserver) logSystemPromptBuilt(event events.Event) {
	base := event.ToBaseEvent()
	lo.logger.Info("[Hook] 生成系统 Prompt",
		zap.String("trace_id", base.TraceID),
		zap.Any("event", event),
	)
}

func (lo *LoggingObserver) logToolUsed(event events.Event) {
	base := event.ToBaseEvent()
	lo.logger.Info("[Hook] 使用工具",
		zap.String("trace_id", base.TraceID),
		zap.Any("event", event),
	)
}

func (lo *LoggingObserver) logToolCompleted(event events.Event) {
	base := event.ToBaseEvent()
	lo.logger.Info("[Hook] 工具执行完成",
		zap.String("trace_id", base.TraceID),
		zap.Any("event", event),
	)
}

func (lo *LoggingObserver) logToolError(event events.Event) {
	base := event.ToBaseEvent()
	lo.logger.Error("[Hook] 工具执行错误",
		zap.String("trace_id", base.TraceID),
		zap.Any("event", event),
	)
}

func (lo *LoggingObserver) logSkillLookup(event events.Event) {
	base := event.ToBaseEvent()
	lo.logger.Debug("[Hook] 查找技能",
		zap.String("trace_id", base.TraceID),
		zap.Any("event", event),
	)
}

func (lo *LoggingObserver) logSkillUsed(event events.Event) {
	base := event.ToBaseEvent()
	lo.logger.Info("[Hook] 使用技能",
		zap.String("trace_id", base.TraceID),
		zap.Any("event", event),
	)
}

func (lo *LoggingObserver) logLLMCallStart(event events.Event) {
	base := event.ToBaseEvent()
	lo.logger.Info("[Hook] LLM 调用开始",
		zap.String("trace_id", base.TraceID),
		zap.Any("event", event),
	)
}

func (lo *LoggingObserver) logLLMCallEnd(event events.Event) {
	base := event.ToBaseEvent()
	tokenUsage := "N/A"
	// 尝试获取 token usage 信息
	if llmEvent, ok := event.(*events.LLMCallEndEvent); ok && llmEvent.TokenUsage != nil {
		tokenUsage = fmt.Sprintf("%d/%d/%d",
			llmEvent.TokenUsage.PromptTokens,
			llmEvent.TokenUsage.CompletionTokens,
			llmEvent.TokenUsage.TotalTokens,
		)
	}
	lo.logger.Info("[Hook] LLM 调用结束",
		zap.String("trace_id", base.TraceID),
		zap.Any("event", event),
		zap.String("token_usage", tokenUsage),
	)
}

func (lo *LoggingObserver) logLLMCallError(event events.Event) {
	base := event.ToBaseEvent()
	lo.logger.Error("[Hook] LLM 调用错误",
		zap.String("trace_id", base.TraceID),
		zap.Any("event", event),
	)
}

func (lo *LoggingObserver) logComponentStart(event events.Event) {
	base := event.ToBaseEvent()
	lo.logger.Debug("[Hook] 组件开始执行",
		zap.String("trace_id", base.TraceID),
		zap.Any("event", event),
	)
}

func (lo *LoggingObserver) logComponentEnd(event events.Event) {
	base := event.ToBaseEvent()
	lo.logger.Debug("[Hook] 组件执行完成",
		zap.String("trace_id", base.TraceID),
		zap.Any("event", event),
	)
}

func (lo *LoggingObserver) logComponentError(event events.Event) {
	base := event.ToBaseEvent()
	lo.logger.Error("[Hook] 组件执行错误",
		zap.String("trace_id", base.TraceID),
		zap.Any("event", event),
	)
}

// JSONLogger JSON 日志观察器
// 将事件以 JSON 格式输出
type JSONLogger struct {
	*observer.BaseObserver
	logger *zap.Logger
}

// NewJSONLogger 创建 JSON 日志观察器
func NewJSONLogger(logger *zap.Logger, filter *observer.ObserverFilter) *JSONLogger {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &JSONLogger{
		BaseObserver: observer.NewBaseObserver("json_logger", filter),
		logger:       logger,
	}
}

// OnEvent 处理事件，以 JSON 格式输出
func (jl *JSONLogger) OnEvent(ctx context.Context, event events.Event) error {
	// 使用 zap 的 Any 字段来记录完整事件
	jl.logger.Info("[Hook-JSON]",
		zap.String("event_type", string(event.GetEventType())),
		zap.String("trace_id", event.GetTraceID()),
		zap.Any("event", event),
	)
	return nil
}