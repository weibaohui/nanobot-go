package tools

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks"
	"go.uber.org/zap"
)

// HookableTool 带 Hook 的工具包装器
// 在工具执行前后触发 Hook 事件
type HookableTool struct {
	inner       tool.InvokableTool
	hookManager *hooks.HookManager
	logger      *zap.Logger
}

// NewHookableTool 创建带 Hook 的工具包装器
func NewHookableTool(inner tool.InvokableTool, hookManager *hooks.HookManager, logger *zap.Logger) *HookableTool {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &HookableTool{
		inner:       inner,
		hookManager: hookManager,
		logger:      logger,
	}
}

// Info 返回工具信息
func (h *HookableTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return h.inner.Info(ctx)
}

// InvokableRun 执行工具并触发 Hook 事件
func (h *HookableTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// 获取工具名称
	toolInfo, err := h.inner.Info(ctx)
	if err != nil {
		toolInfo = &schema.ToolInfo{Name: "unknown"}
	}
	toolName := toolInfo.Name

	// 触发 ToolUsed 事件
	if h.hookManager != nil {
		h.hookManager.OnToolUsed(ctx, toolName, argumentsInJSON)
	}

	// 执行工具
	result, err := h.inner.InvokableRun(ctx, argumentsInJSON, opts...)

	// 触发 ToolCompleted 或 ToolError 事件
	if h.hookManager != nil {
		if err != nil {
			h.hookManager.OnToolError(ctx, toolName, err.Error())
		} else {
			h.hookManager.OnToolCompleted(ctx, toolName, result, true)
		}
	}

	return result, err
}
