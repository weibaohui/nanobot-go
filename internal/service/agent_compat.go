package service

// 为了向后兼容，从 agent 子包导出类型
// 新代码应该直接使用 internal/service/agent 包

import (
	"github.com/weibaohui/nanobot-go/internal/service/agent"
)

// 类型别名（向后兼容）
type (
	CreateAgentRequest = agent.CreateAgentRequest
	UpdateAgentRequest = agent.UpdateAgentRequest
	AgentConfig        = agent.AgentConfig
	AgentService       = agent.Service
)

// 函数（向后兼容）
var NewAgentService = agent.NewService
