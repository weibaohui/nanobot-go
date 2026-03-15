package callbacks

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/trace"
	"go.uber.org/zap"
)

// EinoCallbacks Eino 回调处理器
type EinoCallbacks struct {
	enabled      bool
	logger       *zap.Logger
	startTimes   map[string]time.Time
	callSequence int
}

// NewEinoCallbacks 创建新的 Eino 回调处理器
func NewEinoCallbacks(enabled bool, logger *zap.Logger) *EinoCallbacks {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &EinoCallbacks{
		enabled:    enabled,
		logger:     logger,
		startTimes: make(map[string]time.Time),
	}
}

// Handler 获取 Eino 的 Handler 接口实现
func (ec *EinoCallbacks) Handler() callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnStartFn(ec.onStart).
		OnEndFn(ec.onEnd).
		OnErrorFn(ec.onError).
		OnStartWithStreamInputFn(ec.onStartWithStreamInput).
		OnEndWithStreamOutputFn(ec.onEndWithStreamOutput).
		Build()
}

// traceFields 从 context 中提取链路信息
func (ec *EinoCallbacks) traceFields(ctx context.Context) []zap.Field {
	fields := []zap.Field{
		zap.String("trace_id", trace.MustGetTraceID(ctx)),
		zap.String("span_id", trace.MustGetSpanID(ctx)),
	}
	if parentSpanID := trace.GetParentSpanID(ctx); parentSpanID != "" {
		fields = append(fields, zap.String("parent_span_id", parentSpanID))
	}
	return fields
}

func (ec *EinoCallbacks) withTraceFields(ctx context.Context, extra ...zap.Field) []zap.Field {
	return append(ec.traceFields(ctx), extra...)
}

func (ec *EinoCallbacks) marshalJSON(v any) string {
	data, _ := json.Marshal(v)
	return string(data)
}

func (ec *EinoCallbacks) nodeKey(info *callbacks.RunInfo) string {
	return fmt.Sprintf("%s:%s:%s", info.Component, info.Type, info.Name)
}

// onStart 处理组件开始执行的回调
func (ec *EinoCallbacks) onStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	if !ec.enabled {
		return ctx
	}

	ec.callSequence++
	nodeKey := ec.nodeKey(info)
	ec.startTimes[nodeKey] = time.Now()

	ec.logger.Info("[EinoCallback] 节点开始执行",
		ec.withTraceFields(ctx,
			zap.Int("sequence", ec.callSequence),
			zap.String("component", string(info.Component)),
			zap.String("type", info.Type),
			zap.String("name", info.Name),
		)...,
	)

	switch info.Component {
	case "ChatModel", "Model":
		ec.logModelInput(ctx, input)
	case "Tool":
		ec.logToolInput(ctx, input)
	default:
		ec.logGenericInput(ctx, input, info)
	}

	return ctx
}

// onEnd 处理组件执行完成的回调
func (ec *EinoCallbacks) onEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	if !ec.enabled {
		return ctx
	}

	nodeKey := ec.nodeKey(info)
	duration := time.Duration(0)
	if startTime, exists := ec.startTimes[nodeKey]; exists {
		duration = time.Since(startTime)
	}

	ec.logger.Info("[EinoCallback] 节点执行完成",
		ec.withTraceFields(ctx,
			zap.String("component", string(info.Component)),
			zap.String("type", info.Type),
			zap.String("name", info.Name),
			zap.Int64("duration_ms", duration.Milliseconds()),
		)...,
	)

	switch info.Component {
	case "ChatModel", "Model":
		ec.logModelOutput(ctx, output)
	case "Tool":
		ec.logToolOutput(ctx, output)
	default:
		ec.logGenericOutput(ctx, output, info)
	}

	delete(ec.startTimes, nodeKey)
	return ctx
}

// onError 处理组件执行出错的回调
func (ec *EinoCallbacks) onError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	if !ec.enabled {
		return ctx
	}

	nodeKey := ec.nodeKey(info)
	duration := time.Duration(0)
	if startTime, exists := ec.startTimes[nodeKey]; exists {
		duration = time.Since(startTime)
	}

	if isInterruptError(err) {
		ec.logger.Info("[EinoCallback] 节点触发中断",
			ec.withTraceFields(ctx,
				zap.Error(err),
				zap.String("component", string(info.Component)),
				zap.String("type", info.Type),
				zap.String("name", info.Name),
				zap.Int64("duration_ms", duration.Milliseconds()),
			)...,
		)
	} else {
		ec.logger.Error("[EinoCallback] 节点执行出错",
			ec.withTraceFields(ctx,
				zap.Error(err),
				zap.String("component", string(info.Component)),
				zap.String("type", info.Type),
				zap.String("name", info.Name),
				zap.Int64("duration_ms", duration.Milliseconds()),
			)...,
		)
	}

	delete(ec.startTimes, nodeKey)
	return ctx
}

func isInterruptError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.HasPrefix(msg, "INTERRUPT:") ||
		strings.Contains(msg, "interrupt signal:") ||
		strings.Contains(msg, "interrupt happened")
}

func (ec *EinoCallbacks) onStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	if !ec.enabled {
		return ctx
	}

	ec.logger.Info("[EinoCallback] 流式输入开始",
		ec.withTraceFields(ctx,
			zap.String("component", string(info.Component)),
			zap.String("type", info.Type),
			zap.String("name", info.Name),
		)...,
	)

	return ctx
}

func (ec *EinoCallbacks) onEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	if !ec.enabled {
		return ctx
	}
	return ctx
}

// RegisterGlobalCallbacks 注册全局回调处理器
func RegisterGlobalCallbacks(einoCallbacks *EinoCallbacks) {
	if einoCallbacks != nil && einoCallbacks.enabled {
		callbacks.AppendGlobalHandlers(einoCallbacks.Handler())
	}
}
