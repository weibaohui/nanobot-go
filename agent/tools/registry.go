package tools

import (
	"context"
	"fmt"
	"sync"

	"github.com/cloudwego/eino/components/tool"
)

// Registry 工具注册表
type Registry struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

// NewRegistry 创建工具注册表
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register 注册工具
func (r *Registry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
}

// Get 获取工具
func (r *Registry) Get(name string) Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tools[name]
}

// GetDefinitions 获取所有工具定义
func (r *Registry) GetDefinitions() []map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var defs []map[string]any
	for _, tool := range r.tools {
		defs = append(defs, tool.ToSchema())
	}
	return defs
}

func (r *Registry) GetADKTools() []tool.BaseTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]tool.BaseTool, 0, len(r.tools))
	for _, t := range r.tools {
		if adkTool, ok := t.(tool.BaseTool); ok {
			result = append(result, adkTool)
		}
	}
	return result
}

// Execute 执行工具
func (r *Registry) Execute(ctx context.Context, name string, params map[string]any) (string, error) {
	r.mu.RLock()
	tool, ok := r.tools[name]
	r.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("工具 '%s' 不存在", name)
	}
	return tool.Execute(ctx, params)
}
