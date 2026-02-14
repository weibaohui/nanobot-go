package agent

// AgentType 定义 Agent 类型
type AgentType string

const (
	AgentTypeReAct AgentType = "react"
	AgentTypePlan  AgentType = "plan"
	AgentTypeChat  AgentType = "chat"
)
