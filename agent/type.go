package agent

// AgentType 定义 Agent 类型
type AgentType string

const (
	AgentTypeReAct AgentType = "react_agent"
	AgentTypePlan  AgentType = "plan_agent"
	AgentTypeChat  AgentType = "chat_agent"
)
