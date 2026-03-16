package config

import (
	"context"
	"fmt"
)

// AgentConfigContext 配置工具上下文
type AgentConfigContext struct {
	UserCode    string // 用户编码（租户隔离）
	AgentCode   string // Agent 编码（操作目标）
	ChannelCode string // 渠道编码（审计追踪）
}

type contextKey struct{}

var agentConfigCtxKey = &contextKey{}

// WithAgentConfigContext 将配置上下文注入 ctx
func WithAgentConfigContext(ctx context.Context, acc *AgentConfigContext) context.Context {
	return context.WithValue(ctx, agentConfigCtxKey, acc)
}

// GetAgentConfigContext 从 ctx 提取配置上下文
// 如果不存在或字段不完整，返回错误（强制要求）
func GetAgentConfigContext(ctx context.Context) (*AgentConfigContext, error) {
	v := ctx.Value(agentConfigCtxKey)
	if v == nil {
		return nil, fmt.Errorf("agent config context required: must provide UserCode, AgentCode, ChannelCode via context")
	}
	acc, ok := v.(*AgentConfigContext)
	if !ok || acc == nil {
		return nil, fmt.Errorf("invalid or nil agent config context")
	}
	if acc.UserCode == "" || acc.AgentCode == "" || acc.ChannelCode == "" {
		return nil, fmt.Errorf("agent config context incomplete: UserCode, AgentCode, ChannelCode are all required")
	}
	return acc, nil
}
