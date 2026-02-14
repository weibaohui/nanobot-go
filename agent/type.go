package agent

// AgentType 定义 Agent 类型
type AgentType string

const (
	AgentTypeReAct AgentType = "react"
	AgentTypePlan  AgentType = "plan"
	AgentTypeChat  AgentType = "chat"
)

// IntentType 意图类型
type IntentType string

const (
	IntentFileOperation    IntentType = "file_operation"
	IntentWebSearch        IntentType = "web_search"
	IntentCodeExecution    IntentType = "code_execution"
	IntentProjectPlanning  IntentType = "project_planning"
	IntentTaskDelegation   IntentType = "task_delegation"
	IntentSimpleQuestion   IntentType = "simple_question"
	IntentCasualChat       IntentType = "casual_chat"
	IntentInformationQuery IntentType = "information_query"
	IntentUnknown          IntentType = "unknown"
)
