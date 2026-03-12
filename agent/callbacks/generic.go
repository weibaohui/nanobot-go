package callbacks

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/callbacks"
	"go.uber.org/zap"
)

// logGenericInput 记录通用输入详情
func (ec *EinoCallbacks) logGenericInput(ctx context.Context, input callbacks.CallbackInput, info *callbacks.RunInfo) {
	ec.logger.Debug("[EinoCallback] 通用输入",
		ec.withTraceFields(ctx,
			zap.String("component", string(info.Component)),
			zap.String("input_type", fmt.Sprintf("%T", input)),
			zap.String("input", fmt.Sprintf("%+v", input)),
		)...,
	)
}

// logGenericOutput 记录通用输出详情
func (ec *EinoCallbacks) logGenericOutput(ctx context.Context, output callbacks.CallbackOutput, info *callbacks.RunInfo) {
	ec.logger.Debug("[EinoCallback] 通用输出",
		ec.withTraceFields(ctx,
			zap.String("component", string(info.Component)),
			zap.String("output_type", fmt.Sprintf("%T", output)),
			zap.String("output", fmt.Sprintf("%+v", output)),
		)...,
	)
}
