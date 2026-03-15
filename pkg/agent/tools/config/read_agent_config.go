package config

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/internal/models"
	agentsvc "github.com/weibaohui/nanobot-go/internal/service/agent"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/common"
)

// ReadAgentConfigTool 读取 Agent 配置工具
type ReadAgentConfigTool struct {
	agentService agentsvc.Service
}

// NewReadAgentConfigTool 创建读取配置工具实例
func NewReadAgentConfigTool(agentService agentsvc.Service) *ReadAgentConfigTool {
	return &ReadAgentConfigTool{
		agentService: agentService,
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

// configTypeGetters 配置类型到字段 getter 的包级映射表
var configTypeGetters = map[string]func(*models.Agent) string{
	"identity": func(a *models.Agent) string { return a.IdentityContent },
	"soul":     func(a *models.Agent) string { return a.SoulContent },
	"agents":   func(a *models.Agent) string { return a.AgentsContent },
	"tools":    func(a *models.Agent) string { return a.ToolsContent },
	"user":     func(a *models.Agent) string { return a.UserContent },
}

// configTypeToGetter 配置类型到字段 getter 的映射
func configTypeToGetter(configType string) func(*models.Agent) string {
	return configTypeGetters[configType]
}

// InvokableRun 可直接调用的执行入口
func (t *ReadAgentConfigTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// 1. 强制提取并验证上下文
	cfgCtx, err := GetAgentConfigContext(ctx)
	if err != nil {
		return "", fmt.Errorf("security check failed: %w", err)
	}

	// 2. 解析参数
	var args struct {
		ConfigType string `json:"config_type"`
	}
	if err := common.DecodeArgs(argumentsInJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// 3. 验证 config_type
	getter := configTypeToGetter(args.ConfigType)
	if getter == nil {
		return "", fmt.Errorf("invalid config_type: %s, must be one of: identity, soul, agents, tools, user", args.ConfigType)
	}

	// 4. 权限检查：Agent 存在且属于当前用户
	agent, err := t.agentService.GetAgentByCode(cfgCtx.AgentCode)
	if err != nil {
		return "", fmt.Errorf("failed to get agent: %w", err)
	}
	if agent == nil {
		return "", fmt.Errorf("agent not found: %s", cfgCtx.AgentCode)
	}
	if agent.UserCode != cfgCtx.UserCode {
		return "", fmt.Errorf("access denied: agent %s does not belong to user %s", cfgCtx.AgentCode, cfgCtx.UserCode)
	}

	// 5. 获取配置内容
	content := getter(agent)

	// 6. 构造返回结果
	result := map[string]interface{}{
		"success":     true,
		"config_type": args.ConfigType,
		"content":     content,
		"updated_at":  agent.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"size_bytes":  len(content),
	}

	out, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal result failed: %w", err)
	}
	return string(out), nil
}

// Run 执行工具逻辑（兼容接口）
func (t *ReadAgentConfigTool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.InvokableRun(ctx, argumentsInJSON, opts...)
}
