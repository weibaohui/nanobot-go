package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	mcpsvc "github.com/weibaohui/nanobot-go/internal/service/mcp"
	"go.uber.org/zap"
)

// MCPTool MCP 工具实现
type MCPTool struct {
	serverID    uint
	serverCode  string
	toolName    string
	description string
	inputSchema map[string]interface{}
	mcpService  mcpsvc.Service
	logger      *zap.Logger
}

// NewMCPTool 创建 MCP 工具
func NewMCPTool(
	serverID uint,
	serverCode string,
	toolName string,
	description string,
	inputSchema map[string]interface{},
	mcpService mcpsvc.Service,
	logger *zap.Logger,
) *MCPTool {
	return &MCPTool{
		serverID:    serverID,
		serverCode:  serverCode,
		toolName:    toolName,
		description: description,
		inputSchema: inputSchema,
		mcpService:  mcpService,
		logger:      logger,
	}
}

// Name 返回工具名称
// 格式: mcp::{server_code}::{tool_name}
// 使用 :: 作为分隔符，避免与 server_code 或 tool_name 中的下划线冲突
func (t *MCPTool) Name() string {
	return fmt.Sprintf("mcp::%s::%s", t.serverCode, t.toolName)
}

// Info 返回工具信息
func (t *MCPTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	params := t.buildParams()

	return &schema.ToolInfo{
		Name:        t.Name(),
		Desc:        fmt.Sprintf("[%s] %s", t.serverCode, t.description),
		ParamsOneOf: schema.NewParamsOneOfByParams(params),
	}, nil
}

// Run 执行工具逻辑
func (t *MCPTool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.InvokableRun(ctx, argumentsInJSON, opts...)
}

// InvokableRun 可直接调用的执行入口
func (t *MCPTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// 解析参数
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	t.logger.Debug("执行 MCP 工具",
		zap.String("server_code", t.serverCode),
		zap.String("tool_name", t.toolName),
		zap.Any("params", params),
	)

	// 调用 MCP 服务执行工具
	result, err := t.mcpService.ExecuteTool(t.serverID, t.toolName, params)
	if err != nil {
		t.logger.Error("MCP 工具执行失败",
			zap.String("server_code", t.serverCode),
			zap.String("tool_name", t.toolName),
			zap.Error(err),
		)
		return "", fmt.Errorf("MCP 工具 '%s' 执行失败: %w", t.toolName, err)
	}

	t.logger.Info("MCP 工具执行成功",
		zap.String("server_code", t.serverCode),
		zap.String("tool_name", t.toolName),
	)

	return result, nil
}

// buildParams 构建参数定义
func (t *MCPTool) buildParams() map[string]*schema.ParameterInfo {
	params := make(map[string]*schema.ParameterInfo)

	if t.inputSchema == nil {
		return params
	}

	// 解析 inputSchema
	properties, ok := t.inputSchema["properties"].(map[string]interface{})
	if !ok {
		return params
	}

	required := make(map[string]bool)
	if reqArr, ok := t.inputSchema["required"].([]interface{}); ok {
		for _, r := range reqArr {
			if s, ok := r.(string); ok {
				required[s] = true
			}
		}
	}

	for name, prop := range properties {
		propMap, ok := prop.(map[string]interface{})
		if !ok {
			continue
		}

		paramType := "string"
		if t, ok := propMap["type"].(string); ok {
			paramType = t
		}

		description := ""
		if d, ok := propMap["description"].(string); ok {
			description = d
		}

		params[name] = &schema.ParameterInfo{
			Type:     schema.DataType(paramType),
			Desc:     description,
			Required: required[name],
		}
	}

	return params
}
