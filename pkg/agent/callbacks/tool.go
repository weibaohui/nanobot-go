package callbacks

import (
	"context"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/tool"
	"go.uber.org/zap"
)

// logToolInput 记录工具输入详情
func (ec *EinoCallbacks) logToolInput(ctx context.Context, input callbacks.CallbackInput) {
	toolInput := tool.ConvCallbackInput(input)
	if toolInput == nil {
		ec.logger.Debug("[EinoCallback] Tool 输入转换失败",
			ec.withTraceFields(ctx, zap.String("input_type", "unknown"))...,
		)
		return
	}

	ec.logger.Info("[EinoCallback] Tool 输入参数",
		ec.withTraceFields(ctx, zap.String("arguments", toolInput.ArgumentsInJSON))...,
	)

	if len(toolInput.Extra) > 0 {
		ec.logger.Debug("[EinoCallback] Tool 输入 Extra",
			ec.withTraceFields(ctx, zap.String("extra", ec.marshalJSON(toolInput.Extra)))...,
		)
	}
}

// logToolOutput 记录工具输出详情
func (ec *EinoCallbacks) logToolOutput(ctx context.Context, output callbacks.CallbackOutput) {
	toolOutput := tool.ConvCallbackOutput(output)
	if toolOutput == nil {
		ec.logger.Debug("[EinoCallback] Tool 输出转换失败",
			ec.withTraceFields(ctx, zap.String("output_type", "unknown"))...,
		)
		return
	}

	ec.logger.Info("[EinoCallback] Tool 输出响应",
		ec.withTraceFields(ctx, zap.Int("response_length", len(toolOutput.Response)))...,
	)

	ec.logger.Debug("[EinoCallback] Tool 输出响应详情",
		ec.withTraceFields(ctx, zap.String("response", toolOutput.Response))...,
	)

	if len(toolOutput.Extra) > 0 {
		ec.logger.Debug("[EinoCallback] Tool 输出 Extra",
			ec.withTraceFields(ctx, zap.String("extra", ec.marshalJSON(toolOutput.Extra)))...,
		)
	}
}
