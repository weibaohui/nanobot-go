package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// CallMCPTool 通用 MCP 工具调用器
// 用于调用已加载的 MCP Server 中的具体工具
type CallMCPTool struct {
	manager *Manager
}

// NewCallMCPTool 创建通用 MCP 工具调用器
func NewCallMCPTool(manager *Manager) *CallMCPTool {
	return &CallMCPTool{
		manager: manager,
	}
}

// Name 返回工具名称
func (t *CallMCPTool) Name() string {
	return "call_mcp_tool"
}

// Info 返回工具信息
func (t *CallMCPTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "call_mcp_tool",
		Desc: "调用 MCP Server 中的工具。如果 Server 未加载，会自动加载后再执行。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"server_code": {
				Type:     schema.String,
				Desc:     "MCP Server 编码（如 'weather-server'）",
				Required: true,
			},
			"tool_name": {
				Type:     schema.String,
				Desc:     "工具名称（如 'get_current_weather'）",
				Required: true,
			},
			"params": {
				Type:     schema.Object,
				Desc:     "工具参数（JSON 对象，根据具体工具的要求）",
				Required: false,
			},
		}),
	}, nil
}

// Run 执行工具
func (t *CallMCPTool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.InvokableRun(ctx, argumentsInJSON, opts...)
}

// InvokableRun 可直接调用的执行入口
func (t *CallMCPTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// 解析参数
	var args struct {
		ServerCode string                 `json:"server_code"`
		ToolName   string                 `json:"tool_name"`
		Params     map[string]interface{} `json:"params"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	if args.ServerCode == "" {
		return "", fmt.Errorf("server_code 不能为空")
	}
	if args.ToolName == "" {
		return "", fmt.Errorf("tool_name 不能为空")
	}

	// 规范化 nil params 为空对象，避免序列化为 "null"
	if args.Params == nil {
		args.Params = map[string]interface{}{}
	}

	// 执行 MCP 工具
	result, err := t.manager.ExecuteTool(ctx, args.ServerCode, args.ToolName, args.Params)
	if err != nil {
		return "", err
	}

	return result, nil
}
