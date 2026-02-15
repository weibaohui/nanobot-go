package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
)

// EinoCallbacks Eino 回调处理器
// 用于观察和记录 Workflow 执行过程中的各种事件，包括：
// - LLM 调用的 prompt、tools、tokens 等
// - 工具调用的参数和响应
// - 各节点的执行时间和状态
type EinoCallbacks struct {
	enabled      bool                 // 是否启用回调
	logger       *zap.Logger          // 日志记录器
	startTimes   map[string]time.Time // 记录各节点开始时间
	callSequence int                  // 调用序列号
}

// NewEinoCallbacks 创建新的 Eino 回调处理器
// enabled: 是否启用回调
// logger: zap 日志记录器
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
// 返回配置好的 callbacks.Handler，可用于 Chain 或 Graph 的回调配置
func (ec *EinoCallbacks) Handler() callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnStartFn(ec.onStart).
		OnEndFn(ec.onEnd).
		OnErrorFn(ec.onError).
		OnStartWithStreamInputFn(ec.onStartWithStreamInput).
		OnEndWithStreamOutputFn(ec.onEndWithStreamOutput).
		Build()
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
		zap.Int("sequence", ec.callSequence),
		zap.String("component", string(info.Component)),
		zap.String("type", info.Type),
		zap.String("name", info.Name),
	)

	// 根据组件类型记录详细信息
	switch info.Component {
	case "ChatModel", "Model":
		ec.logModelInput(input, info)
	case "Tool":
		ec.logToolInput(input, info)
	default:
		ec.logGenericInput(input, info)
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
		zap.String("component", string(info.Component)),
		zap.String("type", info.Type),
		zap.String("name", info.Name),
		zap.Int64("duration_ms", duration.Milliseconds()),
	)

	// 根据组件类型记录详细信息
	switch info.Component {
	case "ChatModel", "Model":
		ec.logModelOutput(output, info)
	case "Tool":
		ec.logToolOutput(output, info)
	default:
		ec.logGenericOutput(output, info)
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

	ec.logger.Error("[EinoCallback] 节点执行出错",
		zap.Error(err),
		zap.String("component", string(info.Component)),
		zap.String("type", info.Type),
		zap.String("name", info.Name),
		zap.Int64("duration_ms", duration.Milliseconds()),
	)

	delete(ec.startTimes, nodeKey)
	return ctx
}

// onStartWithStreamInput 处理流式输入开始的回调
func (ec *EinoCallbacks) onStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	if !ec.enabled {
		return ctx
	}

	ec.logger.Info("[EinoCallback] 流式输入开始",
		zap.String("component", string(info.Component)),
		zap.String("type", info.Type),
		zap.String("name", info.Name),
	)

	return ctx
}

// onEndWithStreamOutput 处理流式输出结束的回调
func (ec *EinoCallbacks) onEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	if !ec.enabled {
		return ctx
	}
	return ctx
}

// ========== Model (LLM) 回调详情 ==========

// logModelInput 记录模型输入详情
func (ec *EinoCallbacks) logModelInput(input callbacks.CallbackInput, info *callbacks.RunInfo) {
	modelInput := model.ConvCallbackInput(input)
	if modelInput == nil {
		ec.logger.Debug("[EinoCallback] Model 输入转换失败",
			zap.String("input_type", fmt.Sprintf("%T", input)),
		)
		return
	}

	// 收集 Tools 信息
	toolCount := len(modelInput.Tools)
	toolNames := make([]string, 0, toolCount)
	for _, t := range modelInput.Tools {
		if t == nil {
			continue
		}
		toolNames = append(toolNames, t.Name)
	}

	marshalJSON := func(v any) string {
		data, _ := json.Marshal(v)
		return string(data)
	}

	// 在第一条日志中显示关键信息：Messages 和 Tools 摘要
	ec.logger.Info("[EinoCallback] Model 输入",
		zap.Int("message_count", len(modelInput.Messages)),
		zap.Int("tool_count", toolCount),
		zap.Strings("tool_names", toolNames),
	)

	// 记录 Prompt/Messages 详情
	for _, msg := range modelInput.Messages {
		if msg == nil {
			continue
		}

		// 特别处理 System Message（包含指令），以 Info 级别记录
		if msg.Role == schema.System {
			ec.logger.Info("[EinoCallback] System Message (包含注入的指令)",
				zap.String("content", msg.Content),
			)
			continue  // System Message 已在 Info 级别记录，跳过 Debug 记录
		}

		// 其他消息在调试级别下记录完整内容
		ec.logger.Debug("[EinoCallback]   Message Content",
			zap.String("role", string(msg.Role)),
			zap.String("content", msg.Content),
		)
	}

	// 记录 Tools 详情
	if len(toolNames) > 0 {
		ec.logger.Info("[EinoCallback] Model 输入 Tools",
			zap.Strings("tool_names", toolNames),
		)
	}

	// 记录 ToolChoice
	if modelInput.ToolChoice != nil {
		ec.logger.Info("[EinoCallback] Model 输入 ToolChoice",
			zap.String("tool_choice", marshalJSON(modelInput.ToolChoice)),
		)
	}

	// 记录 Config
	if modelInput.Config != nil {
		ec.logger.Info("[EinoCallback] Model 输入 Config",
			zap.String("model", modelInput.Config.Model),
			zap.Int("max_tokens", modelInput.Config.MaxTokens),
			zap.Float32("temperature", modelInput.Config.Temperature),
			zap.Float32("top_p", modelInput.Config.TopP),
		)
	}

	// 记录 Extra
	if len(modelInput.Extra) > 0 {
		ec.logger.Debug("[EinoCallback] Model 输入 Extra",
			zap.String("extra", marshalJSON(modelInput.Extra)),
		)
	}
}

// logModelOutput 记录模型输出详情
func (ec *EinoCallbacks) logModelOutput(output callbacks.CallbackOutput, info *callbacks.RunInfo) {
	modelOutput := model.ConvCallbackOutput(output)
	if modelOutput == nil {
		ec.logger.Debug("[EinoCallback] Model 输出转换失败",
			zap.String("output_type", fmt.Sprintf("%T", output)),
		)
		return
	}

	// 记录生成的 Message
	if modelOutput.Message != nil {
		// 在调试级别下记录完整内容
		ec.logger.Debug("[EinoCallback] Model 输出 Content",
			zap.String("role", string(modelOutput.Message.Role)),
			zap.String("content", modelOutput.Message.Content),
		)

		// 记录工具调用
		for i, tc := range modelOutput.Message.ToolCalls {
			ec.logger.Info("[EinoCallback] 调用工具 ToolCall",
				zap.Int("index", i),
				zap.String("type", tc.Type),
				zap.String("function_name", tc.Function.Name),
				zap.String("function_arguments", tc.Function.Arguments),
			)
		}
	}

	// 记录 Token 使用情况 (重点!)
	if modelOutput.TokenUsage != nil {
		ec.logger.Info("[EinoCallback] Model Token 使用情况",
			zap.Int("prompt_tokens", modelOutput.TokenUsage.PromptTokens),
			zap.Int("completion_tokens", modelOutput.TokenUsage.CompletionTokens),
			zap.Int("total_tokens", modelOutput.TokenUsage.TotalTokens),
			zap.Int("reasoning_tokens", modelOutput.TokenUsage.CompletionTokensDetails.ReasoningTokens),
			zap.Int("cached_tokens", modelOutput.TokenUsage.PromptTokenDetails.CachedTokens),
		)
	} else {
		ec.logger.Info("[EinoCallback] Model Token 使用情况",
			zap.String("token_usage", "未返回"),
		)
	}

	// 记录 Extra
	if len(modelOutput.Extra) > 0 {
		extraJSON, _ := json.Marshal(modelOutput.Extra)
		ec.logger.Debug("[EinoCallback] Model 输出 Extra",
			zap.String("extra", string(extraJSON)),
		)
	}
}

// ========== Tool 回调详情 ==========

// logToolInput 记录工具输入详情
func (ec *EinoCallbacks) logToolInput(input callbacks.CallbackInput, info *callbacks.RunInfo) {
	toolInput := tool.ConvCallbackInput(input)
	if toolInput == nil {
		ec.logger.Debug("[EinoCallback] Tool 输入转换失败",
			zap.String("input_type", fmt.Sprintf("%T", input)),
		)
		return
	}

	ec.logger.Info("[EinoCallback] Tool 输入参数",
		zap.String("arguments", toolInput.ArgumentsInJSON),
	)

	if len(toolInput.Extra) > 0 {
		extraJSON, _ := json.Marshal(toolInput.Extra)
		ec.logger.Debug("[EinoCallback] Tool 输入 Extra",
			zap.String("extra", string(extraJSON)),
		)
	}
}

// logToolOutput 记录工具输出详情
func (ec *EinoCallbacks) logToolOutput(output callbacks.CallbackOutput, info *callbacks.RunInfo) {
	toolOutput := tool.ConvCallbackOutput(output)
	if toolOutput == nil {
		ec.logger.Debug("[EinoCallback] Tool 输出转换失败",
			zap.String("output_type", fmt.Sprintf("%T", output)),
		)
		return
	}

	ec.logger.Info("[EinoCallback] Tool 输出响应",
		zap.Int("response_length", len(toolOutput.Response)),
	)

	// 在调试级别下记录完整响应
	ec.logger.Debug("[EinoCallback] Tool 输出响应详情",
		zap.String("response", toolOutput.Response),
	)

	if len(toolOutput.Extra) > 0 {
		extraJSON, _ := json.Marshal(toolOutput.Extra)
		ec.logger.Debug("[EinoCallback] Tool 输出 Extra",
			zap.String("extra", string(extraJSON)),
		)
	}
}

// ========== 通用回调详情 ==========

// logGenericInput 记录通用输入详情
func (ec *EinoCallbacks) logGenericInput(input callbacks.CallbackInput, info *callbacks.RunInfo) {
	ec.logger.Debug("[EinoCallback] 通用输入",
		zap.String("component", string(info.Component)),
		zap.String("input_type", fmt.Sprintf("%T", input)),
		zap.String("input", fmt.Sprintf("%+v", input)),
	)
}

// logGenericOutput 记录通用输出详情
func (ec *EinoCallbacks) logGenericOutput(output callbacks.CallbackOutput, info *callbacks.RunInfo) {
	ec.logger.Debug("[EinoCallback] 通用输出",
		zap.String("component", string(info.Component)),
		zap.String("output_type", fmt.Sprintf("%T", output)),
		zap.String("output", fmt.Sprintf("%+v", output)),
	)
}

// nodeKey 生成节点唯一键
func (ec *EinoCallbacks) nodeKey(info *callbacks.RunInfo) string {
	return fmt.Sprintf("%s:%s:%s", info.Component, info.Type, info.Name)
}

// ========== 全局回调注册 ==========

// RegisterGlobalCallbacks 注册全局回调处理器
// 全局回调会在所有 Chain/Graph 执行前被调用
// 注意：此函数不是线程安全的，应在程序初始化时调用
func RegisterGlobalCallbacks(einoCallbacks *EinoCallbacks) {
	if einoCallbacks != nil && einoCallbacks.enabled {
		callbacks.AppendGlobalHandlers(einoCallbacks.Handler())
	}
}
