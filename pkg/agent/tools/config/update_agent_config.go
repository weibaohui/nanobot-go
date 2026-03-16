package config

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	agentsvc "github.com/weibaohui/nanobot-go/internal/service/agent"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/common"
)

// configTypeSetters 配置类型到 setter 函数的映射
var configTypeSetters = map[string]func(*agentsvc.AgentConfig, string){
	"identity": func(c *agentsvc.AgentConfig, v string) { c.IdentityContent = v },
	"soul":     func(c *agentsvc.AgentConfig, v string) { c.SoulContent = v },
	"agents":   func(c *agentsvc.AgentConfig, v string) { c.AgentsContent = v },
	"tools":    func(c *agentsvc.AgentConfig, v string) { c.ToolsContent = v },
	"user":     func(c *agentsvc.AgentConfig, v string) { c.UserContent = v },
}

// UpdateAgentConfigTool 更新 Agent 配置工具
type UpdateAgentConfigTool struct {
	baseTool
}

// NewUpdateAgentConfigTool 创建更新配置工具实例
func NewUpdateAgentConfigTool(agentService agentsvc.Service) *UpdateAgentConfigTool {
	return &UpdateAgentConfigTool{
		baseTool: baseTool{agentService: agentService},
	}
}

// Name 返回工具名称
func (t *UpdateAgentConfigTool) Name() string {
	return "update_agent_config"
}

// Info 返回工具信息
func (t *UpdateAgentConfigTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.Name(),
		Desc: "更新 Agent 的配置项（identity/soul/agents/tools/user），直接替换整个内容",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"config_type": {
				Type:     schema.DataType("string"),
				Desc:     "配置类型，可选: identity, soul, agents, tools, user",
				Required: true,
			},
			"content": {
				Type:     schema.DataType("string"),
				Desc:     "完整的配置内容（直接替换原内容）",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun 可直接调用的执行入口
func (t *UpdateAgentConfigTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// 1. 解析参数
	var args struct {
		ConfigType string `json:"config_type"`
		Content    string `json:"content"`
	}
	if err := common.DecodeArgs(argumentsInJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// 2. 验证 config_type
	setter, ok := configTypeSetters[args.ConfigType]
	if !ok {
		return "", fmt.Errorf("invalid config_type: %s, must be one of: identity, soul, agents, tools, user", args.ConfigType)
	}

	// 3. 验证内容大小（1MB限制）
	const maxContentSize = 1024 * 1024 // 1MB
	if len(args.Content) > maxContentSize {
		return "", fmt.Errorf("content too large: %d bytes, max allowed is %d bytes", len(args.Content), maxContentSize)
	}

	// 4. 验证权限并获取上下文
	cfgCtx, _, err := t.validatePermission(ctx)
	if err != nil {
		return "", err
	}

	// 5. 获取当前配置，修改指定字段
	config, err := t.agentService.GetAgentConfigByCode(cfgCtx.AgentCode)
	if err != nil {
		return "", fmt.Errorf("failed to get agent config: %w", err)
	}

	// 6. 根据 config_type 更新对应字段
	setter(config, args.Content)

	// 7. 保存到数据库
	if err := t.agentService.UpdateAgentConfigByCode(cfgCtx.AgentCode, config); err != nil {
		return "", fmt.Errorf("failed to update agent config: %w", err)
	}

	// 8. 构造返回结果
	return jsonResponse(map[string]interface{}{
		"success":       true,
		"message":       "配置已更新",
		"config_type":   args.ConfigType,
		"bytes_written": len(args.Content),
		"updated_at":    time.Now().Format(time.RFC3339),
	})
}

// Run 执行工具逻辑（兼容接口）
func (t *UpdateAgentConfigTool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.InvokableRun(ctx, argumentsInJSON, opts...)
}
