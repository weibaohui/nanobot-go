package tools

import (
	"context"
)

// Tool 工具接口
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	Execute(ctx context.Context, params map[string]any) (string, error)
	ToSchema() map[string]any
}

// ContextSetter 可设置上下文的工具接口
type ContextSetter interface {
	SetContext(channel, chatID string)
}
