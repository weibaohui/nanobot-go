package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// Registry 工具注册表
type Registry struct {
	tools map[string]tool.BaseTool
	mu    sync.RWMutex
}

// NewRegistry 创建工具注册表
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]tool.BaseTool),
	}
}

// Register 注册工具
func (r *Registry) Register(baseTool tool.BaseTool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	name := r.resolveToolName(context.Background(), baseTool)
	if name == "" {
		return
	}
	r.tools[name] = baseTool
}

// Get 获取工具
func (r *Registry) Get(name string) tool.BaseTool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tools[name]
}

// GetDefinitions 获取所有工具定义
func (r *Registry) GetDefinitions(ctx context.Context) []map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var defs []map[string]any
	for _, baseTool := range r.tools {
		info, err := baseTool.Info(ctx)
		if err != nil || info == nil {
			continue
		}
		definition, err := buildToolDefinition(info)
		if err != nil {
			continue
		}
		defs = append(defs, definition)
	}
	return defs
}

// GetADKTools 获取 ADK 工具列表
func (r *Registry) GetADKTools() []tool.BaseTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]tool.BaseTool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}

// Execute 执行工具
func (r *Registry) Execute(ctx context.Context, name string, params map[string]any) (string, error) {
	r.mu.RLock()
	baseTool, ok := r.tools[name]
	r.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("工具 '%s' 不存在", name)
	}
	invokable, ok := baseTool.(tool.InvokableTool)
	if !ok {
		return "", fmt.Errorf("工具 '%s' 不支持直接调用", name)
	}
	argsJSON, err := json.Marshal(params)
	if err != nil {
		return "", err
	}
	return invokable.InvokableRun(ctx, string(argsJSON))
}

// resolveToolName 解析工具名称
func (r *Registry) resolveToolName(ctx context.Context, baseTool tool.BaseTool) string {
	if named, ok := baseTool.(NamedTool); ok {
		return named.Name()
	}
	info, err := baseTool.Info(ctx)
	if err != nil || info == nil {
		return ""
	}
	return info.Name
}

// buildToolDefinition 构建工具定义
func buildToolDefinition(info *schema.ToolInfo) (map[string]any, error) {
	var params any
	if info.ParamsOneOf != nil {
		jsonSchema, err := info.ParamsOneOf.ToJSONSchema()
		if err != nil {
			return nil, err
		}
		if jsonSchema != nil {
			data, err := json.Marshal(jsonSchema)
			if err != nil {
				return nil, err
			}
			var schemaMap map[string]any
			if err := json.Unmarshal(data, &schemaMap); err != nil {
				return nil, err
			}
			params = schemaMap
		}
	}
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        info.Name,
			"description": info.Desc,
			"parameters":  params,
		},
	}, nil
}
