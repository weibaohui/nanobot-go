package observers

import (
	"context"
	"fmt"
	"strings"

	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/observer"
	"go.uber.org/zap"
)

// isInterruptError 检查错误是否为中断信号
// 中断是正常流程，不应该记录为错误
func isInterruptError(errorMsg string) bool {
	return strings.HasPrefix(strings.ToLower(errorMsg), "interrupt signal") ||
		strings.HasPrefix(strings.ToLower(errorMsg), "interrupt:")
}

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
	base := event.ToBaseEvent()

	// 构建链路信息日志前缀
	traceFields := []zap.Field{
		zap.String("trace_id", base.TraceID),
		zap.String("span_id", base.SpanID),
	}
	if base.ParentSpanID != "" {
		traceFields = append(traceFields, zap.String("parent_span_id", base.ParentSpanID))
	}

	switch event.GetEventType() {
	case events.EventMessageReceived:
		fields := append(traceFields, zap.Any("event", event))
		lo.logger.Info("[Hook] 收到消息", fields...)
	case events.EventMessageSent:
		fields := append(traceFields, zap.Any("event", event))
		lo.logger.Info("[Hook] 发送消息", fields...)
	case events.EventPromptSubmitted:
		fields := append(traceFields, zap.Any("event", event))
		lo.logger.Info("[Hook] 提交 Prompt", fields...)
	case events.EventSystemPromptBuilt:
		fields := append(traceFields, zap.Any("event", event))
		lo.logger.Info("[Hook] 生成系统 Prompt", fields...)
	case events.EventToolCall:
		fields := append(traceFields, zap.Any("event", event))
		lo.logger.Info("[Hook] 工具调用", fields...)
	case events.EventToolIntercepted:
		fields := append(traceFields, zap.Any("event", event))
		lo.logger.Info("[Hook] 工具调用被拦截", fields...)
	case events.EventToolUsed:
		fields := append(traceFields, zap.Any("event", event))
		lo.logger.Info("[Hook] 使用工具", fields...)
	case events.EventToolCompleted:
		fields := append(traceFields, zap.Any("event", event))
		lo.logger.Info("[Hook] 工具执行完成", fields...)
	case events.EventToolError:
		// 检查是否为中断信号（正常流程）
		if toolErr, ok := event.(*events.ToolErrorEvent); ok && isInterruptError(toolErr.Error) {
			fields := append(traceFields,
				zap.String("tool_name", toolErr.ToolName),
				zap.String("reason", "用户中断/等待输入"),
			)
			lo.logger.Info("[Hook] 工具执行中断", fields...)
		} else {
			fields := append(traceFields, zap.Any("event", event))
			lo.logger.Error("[Hook] 工具执行错误", fields...)
		}
	case events.EventSkillCall:
		fields := append(traceFields, zap.Any("event", event))
		lo.logger.Info("[Hook] 技能调用", fields...)
	case events.EventSkillLookup:
		fields := append(traceFields, zap.Any("event", event))
		lo.logger.Debug("[Hook] 查找技能", fields...)
	case events.EventSkillUsed:
		fields := append(traceFields, zap.Any("event", event))
		lo.logger.Info("[Hook] 使用技能", fields...)
	case events.EventLLMCallStart:
		fields := append(traceFields, zap.Any("event", event))
		lo.logger.Info("[Hook] LLM 调用开始", fields...)
	case events.EventLLMCallEnd:
		tokenUsage := "N/A"
		// 尝试获取 token usage 信息
		if llmEvent, ok := event.(*events.LLMCallEndEvent); ok && llmEvent.TokenUsage != nil {
			tokenUsage = fmt.Sprintf("%d/%d/%d",
				llmEvent.TokenUsage.PromptTokens,
				llmEvent.TokenUsage.CompletionTokens,
				llmEvent.TokenUsage.TotalTokens,
			)
		}
		fields := append(traceFields, zap.Any("event", event), zap.String("token_usage", tokenUsage))
		lo.logger.Info("[Hook] LLM 调用结束", fields...)
	case events.EventLLMCallError:
		// 检查是否为中断信号（正常流程）
		if llmErr, ok := event.(*events.LLMCallErrorEvent); ok && isInterruptError(llmErr.Error) {
			fields := append(traceFields,
				zap.String("model", llmErr.Model),
				zap.String("reason", "用户中断/等待输入"),
			)
			lo.logger.Info("[Hook] LLM 调用中断", fields...)
		} else {
			fields := append(traceFields, zap.Any("event", event))
			lo.logger.Error("[Hook] LLM 调用错误", fields...)
		}
	case events.EventComponentStart:
		fields := append(traceFields, zap.Any("event", event))
		lo.logger.Debug("[Hook] 组件开始执行", fields...)
	case events.EventComponentEnd:
		fields := append(traceFields, zap.Any("event", event))
		lo.logger.Debug("[Hook] 组件执行完成", fields...)
	case events.EventComponentError:
		// 检查是否为中断信号（正常流程）
		if compErr, ok := event.(*events.ComponentErrorEvent); ok && isInterruptError(compErr.Error) {
			fields := append(traceFields,
				zap.String("component", compErr.Component),
				zap.String("reason", "用户中断/等待输入"),
			)
			lo.logger.Info("[Hook] 组件执行中断", fields...)
		} else {
			fields := append(traceFields, zap.Any("event", event))
			lo.logger.Error("[Hook] 组件执行错误", fields...)
		}
	default:
		fields := append(traceFields,
			zap.String("event_type", string(event.GetEventType())),
			zap.Any("event", event),
		)
		lo.logger.Debug("未知事件类型", fields...)
	}
	return nil
}
