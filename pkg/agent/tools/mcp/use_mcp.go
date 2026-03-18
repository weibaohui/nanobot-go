package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// UseMCPTool use_mcp 工具 - 用于按需加载 MCP Server
type UseMCPTool struct {
	manager *Manager
}

// NewUseMCPTool 创建 use_mcp 工具
func NewUseMCPTool(manager *Manager) *UseMCPTool {
	return &UseMCPTool{
		manager: manager,
	}
}

// Name 返回工具名称
func (t *UseMCPTool) Name() string {
	return "use_mcp"
}

// Info 返回工具信息
func (t *UseMCPTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "use_mcp",
		Desc: "加载并使用指定 MCP Server 的工具。调用此工具后，该 MCP Server 的所有工具将可在当前对话中使用。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"server_code": {
				Type:     schema.String,
				Desc:     "MCP Server 编码（如 'weather-server', 'file-system'）",
				Required: true,
			},
			"action": {
				Type:     schema.String,
				Desc:     "操作类型：'load'(加载并返回工具列表), 'info'(仅返回 Server 信息，不加载工具)",
				Required: false,
			},
		}),
	}, nil
}

// Run 执行工具
func (t *UseMCPTool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.InvokableRun(ctx, argumentsInJSON, opts...)
}

// InvokableRun 可直接调用的执行入口
func (t *UseMCPTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// 解析参数
	var args struct {
		ServerCode string `json:"server_code"`
		Action     string `json:"action"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	if args.ServerCode == "" {
		return "", fmt.Errorf("server_code 不能为空")
	}

	// 默认操作为 load
	if args.Action == "" {
		args.Action = "load"
	}

	args.Action = strings.ToLower(args.Action)

	switch args.Action {
	case "load":
		return t.handleLoad(args.ServerCode)
	case "info":
		return t.handleInfo(args.ServerCode)
	default:
		return "", fmt.Errorf("不支持的操作: %s", args.Action)
	}
}

// buildMCPServerResponse 构建 MCP Server 响应（供缓存和初始加载共用）
func (t *UseMCPTool) buildMCPServerResponse(server *LoadedServer, alreadyLoaded bool) map[string]interface{} {
	tools := make([]map[string]interface{}, 0, len(server.Tools))
	for _, tool := range server.Tools {
		if mcpTool, ok := tool.(*MCPTool); ok {
			toolInfo := map[string]interface{}{
				"name":         mcpTool.toolName,
				"description":  mcpTool.description,
				"input_schema": mcpTool.inputSchema,
			}
			tools = append(tools, toolInfo)
		}
	}

	message := fmt.Sprintf("MCP Server '%s' 加载成功，包含 %d 个工具。"+
		"现在你可以使用 call_mcp_tool 工具调用这些工具。"+
		"示例：call_mcp_tool(server_code='%s', tool_name='工具名', params={...})",
		server.Name, len(server.Tools), server.Code)
	if alreadyLoaded {
		message = fmt.Sprintf("MCP Server '%s' 已加载，包含 %d 个工具", server.Name, len(server.Tools))
	}

	return map[string]interface{}{
		"success":        true,
		"server_code":    server.Code,
		"server_name":    server.Name,
		"already_loaded": alreadyLoaded,
		"message":        message,
		"tools":          tools,
		"tool_count":     len(server.Tools),
		"usage":          "使用 call_mcp_tool(server_code, tool_name, params) 调用工具",
	}
}

// handleLoad 加载 MCP Server
func (t *UseMCPTool) handleLoad(serverCode string) (string, error) {
	// 检查是否已加载
	if t.manager.IsLoaded(serverCode) {
		server := t.manager.GetLoadedServer(serverCode)
		result := t.buildMCPServerResponse(server, true)

		resultJSON, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", fmt.Errorf("序列化结果失败: %w", err)
		}
		return string(resultJSON), nil
	}

	// 加载 Server
	loaded, err := t.manager.LoadServer(serverCode)
	if err != nil {
		// 注意：返回 JSON 格式的错误信息（而非 Go error），以便 LLM 能够理解错误原因
		result := map[string]interface{}{
			"success":     false,
			"server_code": serverCode,
			"error":       err.Error(),
			"message":     fmt.Sprintf("加载 MCP Server '%s' 失败: %v", serverCode, err),
		}
		resultJSON, marshalErr := json.MarshalIndent(result, "", "  ")
		if marshalErr != nil {
			return "", fmt.Errorf("序列化错误结果失败: %w", marshalErr)
		}
		return string(resultJSON), nil
	}

	// 使用共用函数构建响应
	result := t.buildMCPServerResponse(loaded, false)

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}
	return string(resultJSON), nil
}

// handleInfo 获取 Server 信息（不加载）
func (t *UseMCPTool) handleInfo(serverCode string) (string, error) {
	info, err := t.manager.GetServerInfo(serverCode)
	if err != nil {
		result := map[string]interface{}{
			"success":     false,
			"server_code": serverCode,
			"error":       err.Error(),
		}
		resultJSON, marshalErr := json.MarshalIndent(result, "", "  ")
		if marshalErr != nil {
			return "", fmt.Errorf("序列化错误结果失败: %w", marshalErr)
		}
		return string(resultJSON), nil
	}

	result := map[string]interface{}{
		"success":     true,
		"server_code": info.Code,
		"server_name": info.Name,
		"description": info.Description,
		"status":      info.Status,
		"tool_count":  info.ToolCount,
		"is_loaded":   t.manager.IsLoaded(serverCode),
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}
	return string(resultJSON), nil
}
