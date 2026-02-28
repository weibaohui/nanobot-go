package eino

import (
	"context"
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/weibaohui/nanobot-go/agent/hooks/dispatcher"
	"github.com/weibaohui/nanobot-go/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/agent/hooks/trace"
	"go.uber.org/zap"
)

// EinoCallbackBridge Eino Callback 桥接器
// 将 Eino 的回调转换为 Hook 系统事件并分发
type EinoCallbackBridge struct {
	dispatcher *dispatcher.Dispatcher
	logger     *zap.Logger
	startTimes map[string]time.Time
}

// NewEinoCallbackBridge 创建 Eino Callback 桥接器
func NewEinoCallbackBridge(dispatcher *dispatcher.Dispatcher, logger *zap.Logger) *EinoCallbackBridge {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &EinoCallbackBridge{
		dispatcher:  dispatcher,
		logger:     logger,
		startTimes: make(map[string]time.Time),
	}
}

// Handler 获取 Eino 的 Handler 接口实现
func (cb *EinoCallbackBridge) Handler() callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnStartFn(cb.onStart).
		OnEndFn(cb.onEnd).
		OnErrorFn(cb.onError).
		Build()
}

// nodeKey 生成节点唯一键
func (cb *EinoCallbackBridge) nodeKey(info *callbacks.RunInfo) string {
	return string(info.Component) + ":" + info.Type + ":" + info.Name
}

// onStart 处理组件开始执行的回调
func (cb *EinoCallbackBridge) onStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	nodeKey := cb.nodeKey(info)
	cb.startTimes[nodeKey] = time.Now()

	traceID := trace.GetTraceID(ctx)
	spanID := trace.GetSpanID(ctx)
	parentSpanID := trace.MustGetSpanID(ctx) // 使用 MustGetSpanID 获取父 SpanID（如果存在）

	// 分发组件开始事件
	componentEvent := events.NewComponentStartEvent(traceID, spanID, parentSpanID, info)
	cb.dispatcher.Dispatch(ctx, componentEvent, "", "")

	// 根据组件类型分发具体事件
	// 只处理 ChatModel，避免与 Model 重复处理
	switch info.Component {
	case "ChatModel":
		cb.handleModelStart(ctx, traceID, spanID, parentSpanID, info, input)
	case "Tool":
		cb.handleToolStart(ctx, traceID, spanID, parentSpanID, info, input)
	default:
		// 其他组件类型，只记录通用事件
	}

	return ctx
}

// onEnd 处理组件执行完成的回调
func (cb *EinoCallbackBridge) onEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	nodeKey := cb.nodeKey(info)
	startTime := cb.startTimes[nodeKey]
	delete(cb.startTimes, nodeKey)

	durationMs := time.Since(startTime).Milliseconds()
	traceID := trace.GetTraceID(ctx)
	spanID := trace.GetSpanID(ctx)
	parentSpanID := trace.MustGetSpanID(ctx)

	// 分发组件结束事件
	componentEvent := events.NewComponentEndEvent(traceID, spanID, parentSpanID, info, durationMs)
	cb.dispatcher.Dispatch(ctx, componentEvent, "", "")

	// 根据组件类型分发具体事件
	// 只处理 ChatModel，避免与 Model 重复处理
	switch info.Component {
	case "ChatModel":
		cb.handleModelEnd(ctx, traceID, spanID, parentSpanID, info, output, durationMs)
	case "Tool":
		cb.handleToolEnd(ctx, traceID, spanID, parentSpanID, info, output, durationMs)
	default:
		// 其他组件类型，只记录通用事件
	}

	return ctx
}

// onError 处理组件执行出错的回调
func (cb *EinoCallbackBridge) onError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	nodeKey := cb.nodeKey(info)
	startTime, exists := cb.startTimes[nodeKey]
	delete(cb.startTimes, nodeKey)

	durationMs := int64(0)
	if exists {
		durationMs = time.Since(startTime).Milliseconds()
	}

	traceID := trace.GetTraceID(ctx)
	spanID := trace.GetSpanID(ctx)
	parentSpanID := trace.MustGetSpanID(ctx)

	// 分发组件错误事件
	componentEvent := events.NewComponentErrorEvent(traceID, spanID, parentSpanID, info, err, durationMs)
	cb.dispatcher.Dispatch(ctx, componentEvent, "", "")

	// 根据组件类型分发具体错误事件
	// 只处理 ChatModel，避免与 Model 重复处理
	switch info.Component {
	case "ChatModel":
		llmErrorEvent := events.NewLLMCallErrorEvent(traceID, spanID, parentSpanID, info, err, durationMs)
		cb.dispatcher.Dispatch(ctx, llmErrorEvent, "", "")
	case "Tool":
		cb.handleToolError(ctx, traceID, spanID, parentSpanID, info, err)
	default:
		// 其他组件类型，只记录通用事件
	}

	return ctx
}

// handleModelStart 处理模型调用开始
func (cb *EinoCallbackBridge) handleModelStart(ctx context.Context, traceID, spanID, parentSpanID string, info *callbacks.RunInfo, input callbacks.CallbackInput) {
	modelInput := model.ConvCallbackInput(input)
	if modelInput == nil {
		return
	}

	event := events.NewLLMCallStartEvent(traceID, spanID, parentSpanID, info, modelInput)
	cb.dispatcher.Dispatch(ctx, event, "", "")
}

// handleModelEnd 处理模型调用结束
func (cb *EinoCallbackBridge) handleModelEnd(ctx context.Context, traceID, spanID, parentSpanID string, info *callbacks.RunInfo, output callbacks.CallbackOutput, durationMs int64) {
	modelOutput := model.ConvCallbackOutput(output)
	if modelOutput == nil {
		return
	}

	event := events.NewLLMCallEndEvent(traceID, spanID, parentSpanID, info, modelOutput, durationMs)
	cb.dispatcher.Dispatch(ctx, event, "", "")
}

// handleToolStart 处理工具调用开始
func (cb *EinoCallbackBridge) handleToolStart(ctx context.Context, traceID, spanID, parentSpanID string, info *callbacks.RunInfo, input callbacks.CallbackInput) {
	toolInput := tool.ConvCallbackInput(input)
	if toolInput == nil {
		return
	}

	// 从 info.Name 获取工具名称
	event := events.NewToolUsedEvent(traceID, spanID, parentSpanID, info.Name, toolInput.ArgumentsInJSON)
	cb.dispatcher.Dispatch(ctx, event, "", "")
}

// handleToolEnd 处理工具调用结束
func (cb *EinoCallbackBridge) handleToolEnd(ctx context.Context, traceID, spanID, parentSpanID string, info *callbacks.RunInfo, output callbacks.CallbackOutput, durationMs int64) {
	toolOutput := tool.ConvCallbackOutput(output)
	if toolOutput == nil {
		return
	}

	event := events.NewToolCompletedEvent(traceID, spanID, parentSpanID, info.Name, toolOutput.Response, true)
	cb.dispatcher.Dispatch(ctx, event, "", "")
}

// handleToolError 处理工具调用错误
func (cb *EinoCallbackBridge) handleToolError(ctx context.Context, traceID, spanID, parentSpanID string, info *callbacks.RunInfo, err error) {
	event := events.NewToolErrorEvent(traceID, spanID, parentSpanID, info.Name, err.Error())
	cb.dispatcher.Dispatch(ctx, event, "", "")
}