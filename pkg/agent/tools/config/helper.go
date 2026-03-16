package config

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/weibaohui/nanobot-go/internal/models"
	agentsvc "github.com/weibaohui/nanobot-go/internal/service/agent"
)

// baseTool 工具基础结构，包含公共方法
type baseTool struct {
	agentService agentsvc.Service
}

// validatePermission 验证上下文和权限
// 返回配置上下文和 Agent 对象，如果验证失败则返回错误
func (b *baseTool) validatePermission(ctx context.Context) (*AgentConfigContext, *models.Agent, error) {
	// 1. 提取并验证上下文
	cfgCtx, err := GetAgentConfigContext(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("security check failed: %w", err)
	}

	// 2. 权限检查：Agent 存在且属于当前用户
	agent, err := b.agentService.GetAgentByCode(cfgCtx.AgentCode)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get agent: %w", err)
	}
	if agent == nil {
		return nil, nil, fmt.Errorf("agent not found: %s", cfgCtx.AgentCode)
	}
	if agent.UserCode != cfgCtx.UserCode {
		return nil, nil, fmt.Errorf("access denied: agent %s does not belong to user %s", cfgCtx.AgentCode, cfgCtx.UserCode)
	}

	return cfgCtx, agent, nil
}

// jsonResponse 序列化响应为 JSON 字符串
func jsonResponse(data interface{}) (string, error) {
	out, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal result failed: %w", err)
	}
	return string(out), nil
}
