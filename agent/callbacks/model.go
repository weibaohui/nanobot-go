package callbacks

import (
	"context"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
)

// logModelInput 记录模型输入详情
func (ec *EinoCallbacks) logModelInput(ctx context.Context, input callbacks.CallbackInput) {
	modelInput := model.ConvCallbackInput(input)
	if modelInput == nil {
		ec.logger.Debug("[EinoCallback] Model 输入转换失败",
			ec.withTraceFields(ctx, zap.String("input_type", "unknown"))...,
		)
		return
	}

	toolCount := len(modelInput.Tools)
	toolNames := make([]string, 0, toolCount)
	for _, t := range modelInput.Tools {
		if t == nil {
			continue
		}
		toolNames = append(toolNames, t.Name)
	}

	ec.logger.Info("[EinoCallback] Model 输入",
		ec.withTraceFields(ctx,
			zap.Int("message_count", len(modelInput.Messages)),
			zap.Int("tool_count", toolCount),
			zap.Strings("tool_names", toolNames),
		)...,
	)

	for _, msg := range modelInput.Messages {
		if msg == nil {
			continue
		}

		if msg.Role == schema.System {
			ec.logger.Info("[EinoCallback] System Message",
				ec.withTraceFields(ctx, zap.String("content", msg.Content))...,
			)
			continue
		}

		ec.logger.Debug("[EinoCallback]   Message Content",
			ec.withTraceFields(ctx,
				zap.String("role", string(msg.Role)),
				zap.String("content", msg.Content),
			)...,
		)
	}

	if len(toolNames) > 0 {
		ec.logger.Info("[EinoCallback] Model 输入 Tools",
			ec.withTraceFields(ctx, zap.Strings("tool_names", toolNames))...,
		)
	}

	if modelInput.ToolChoice != nil {
		ec.logger.Info("[EinoCallback] Model 输入 ToolChoice",
			ec.withTraceFields(ctx, zap.String("tool_choice", ec.marshalJSON(modelInput.ToolChoice)))...,
		)
	}

	if modelInput.Config != nil {
		ec.logger.Info("[EinoCallback] Model 输入 Config",
			ec.withTraceFields(ctx,
				zap.String("model", modelInput.Config.Model),
				zap.Int("max_tokens", modelInput.Config.MaxTokens),
				zap.Float32("temperature", modelInput.Config.Temperature),
				zap.Float32("top_p", modelInput.Config.TopP),
			)...,
		)
	}

	if len(modelInput.Extra) > 0 {
		ec.logger.Debug("[EinoCallback] Model 输入 Extra",
			ec.withTraceFields(ctx, zap.String("extra", ec.marshalJSON(modelInput.Extra)))...,
		)
	}
}

// logModelOutput 记录模型输出详情
func (ec *EinoCallbacks) logModelOutput(ctx context.Context, output callbacks.CallbackOutput) {
	modelOutput := model.ConvCallbackOutput(output)
	if modelOutput == nil {
		ec.logger.Debug("[EinoCallback] Model 输出转换失败",
			ec.withTraceFields(ctx, zap.String("output_type", "unknown"))...,
		)
		return
	}

	if modelOutput.Message != nil {
		ec.logger.Debug("[EinoCallback] Model 输出 Content",
			ec.withTraceFields(ctx,
				zap.String("role", string(modelOutput.Message.Role)),
				zap.String("content", modelOutput.Message.Content),
			)...,
		)

		for i, tc := range modelOutput.Message.ToolCalls {
			ec.logger.Info("[EinoCallback] 调用工具 ToolCall",
				ec.withTraceFields(ctx,
					zap.Int("index", i),
					zap.String("type", tc.Type),
					zap.String("function_name", tc.Function.Name),
					zap.String("function_arguments", tc.Function.Arguments),
				)...,
			)
		}
	}

	if modelOutput.TokenUsage != nil {
		ec.logger.Info("[EinoCallback] Model Token 使用情况",
			ec.withTraceFields(ctx,
				zap.Int("prompt_tokens", modelOutput.TokenUsage.PromptTokens),
				zap.Int("completion_tokens", modelOutput.TokenUsage.CompletionTokens),
				zap.Int("total_tokens", modelOutput.TokenUsage.TotalTokens),
				zap.Int("reasoning_tokens", modelOutput.TokenUsage.CompletionTokensDetails.ReasoningTokens),
				zap.Int("cached_tokens", modelOutput.TokenUsage.PromptTokenDetails.CachedTokens),
			)...,
		)
	} else {
		ec.logger.Info("[EinoCallback] Model Token 使用情况",
			ec.withTraceFields(ctx, zap.String("token_usage", "未返回"))...,
		)
	}

	if len(modelOutput.Extra) > 0 {
		ec.logger.Debug("[EinoCallback] Model 输出 Extra",
			ec.withTraceFields(ctx, zap.String("extra", ec.marshalJSON(modelOutput.Extra)))...,
		)
	}
}
