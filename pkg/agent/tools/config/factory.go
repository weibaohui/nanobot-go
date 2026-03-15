package config

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	agentsvc "github.com/weibaohui/nanobot-go/internal/service/agent"
)

// Tools 配置工具集合
type Tools struct {
	ReadAgentConfigTool   *ReadAgentConfigTool
	UpdateAgentConfigTool *UpdateAgentConfigTool
	ManageAgentMemoryTool *ManageAgentMemoryTool
}

// NewTools 创建配置工具集合
func NewTools(agentService agentsvc.Service) *Tools {
	return &Tools{
		ReadAgentConfigTool:   NewReadAgentConfigTool(agentService),
		UpdateAgentConfigTool: NewUpdateAgentConfigTool(agentService),
		ManageAgentMemoryTool: NewManageAgentMemoryTool(agentService),
	}
}

// All 返回所有工具实例，用于注册
func (t *Tools) All() []interface {
	Name() string
	Info(ctx context.Context) (*schema.ToolInfo, error)
	InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error)
} {
	return []interface {
		Name() string
		Info(ctx context.Context) (*schema.ToolInfo, error)
		InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error)
	}{
		t.ReadAgentConfigTool,
		t.UpdateAgentConfigTool,
		t.ManageAgentMemoryTool,
	}
}

// Read 返回读取工具
func (t *Tools) Read() *ReadAgentConfigTool {
	return t.ReadAgentConfigTool
}

// Update 返回更新工具
func (t *Tools) Update() *UpdateAgentConfigTool {
	return t.UpdateAgentConfigTool
}

// Memory 返回记忆管理工具
func (t *Tools) Memory() *ManageAgentMemoryTool {
	return t.ManageAgentMemoryTool
}
