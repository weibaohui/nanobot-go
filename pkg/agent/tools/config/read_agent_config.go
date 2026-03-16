package config

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/internal/models"
	agentsvc "github.com/weibaohui/nanobot-go/internal/service/agent"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/common"
)

// configTypeGetters 配置类型到字段 getter 的包级映射表
var configTypeGetters = map[string]func(*models.Agent) string{
	"identity": func(a *models.Agent) string { return a.IdentityContent },
	"soul":     func(a *models.Agent) string { return a.SoulContent },
	"agents":   func(a *models.Agent) string { return a.AgentsContent },
	"tools":    func(a *models.Agent) string { return a.ToolsContent },
	"user":     func(a *models.Agent) string { return a.UserContent },
}

// ReadAgentConfigTool 读取 Agent 配置工具
type ReadAgentConfigTool struct {
	baseTool
}

// NewReadAgentConfigTool 创建读取配置工具实例
func NewReadAgentConfigTool(agentService agentsvc.Service) *ReadAgentConfigTool {
	return &ReadAgentConfigTool{
		baseTool: baseTool{agentService: agentService},
	}
}

// Name 返回工具名称
func (t *ReadAgentConfigTool) Name() string {
	return "read_agent_config"
}

// Info 返回工具信息
func (t *ReadAgentConfigTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.Name(),
		Desc: "读取 Agent 的配置项（identity/soul/agents/tools/user）",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"config_type": {
				Type:     schema.DataType("string"),
				Desc:     "配置类型，可选: identity, soul, agents, tools, user",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun 可直接调用的执行入口
func (t *ReadAgentConfigTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// 1. 解析参数
	var args struct {
		ConfigType string `json:"config_type"`
	}
	if err := common.DecodeArgs(argumentsInJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// 2. 验证 config_type
	getter, ok := configTypeGetters[args.ConfigType]
	if !ok {
		return "", fmt.Errorf("invalid config_type: %s, must be one of: identity, soul, agents, tools, user", args.ConfigType)
	}

	// 3. 验证权限并获取 Agent
	_, agent, err := t.validatePermission(ctx)
	if err != nil {
		return "", err
	}

	// 4. 获取配置内容
	content := getter(agent)

	// 5. 构造返回结果
	return jsonResponse(map[string]interface{}{
		"success":     true,
		"config_type": args.ConfigType,
		"content":     content,
		"updated_at":  agent.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"size_bytes":  len(content),
	})
}

// Run 执行工具逻辑（兼容接口）
func (t *ReadAgentConfigTool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.InvokableRun(ctx, argumentsInJSON, opts...)
}
