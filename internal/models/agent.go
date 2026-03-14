package models

import (
	"encoding/json"
	"time"
)

// Agent Agent 模型
// 存储 Agent 配置信息，包括所有 Markdown 文档内容和能力配置
type Agent struct {
	ID        uint   `gorm:"primarykey" json:"id"`
	AgentCode string `gorm:"type:varchar(16);uniqueIndex" json:"agent_code"`
	UserCode  string `gorm:"type:varchar(16);index" json:"user_code"`
	Name        string `gorm:"type:text;not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`

	// Markdown 配置内容（每个文档对应一个列）
	IdentityContent string `gorm:"type:text" json:"identity_content"` // IDENTITY.md - Agent 身份信息
	SoulContent     string `gorm:"type:text" json:"soul_content"`     // SOUL.md - Agent 灵魂/个性
	AgentsContent   string `gorm:"type:text" json:"agents_content"`   // AGENTS.md - Agent 指令配置
	UserContent     string `gorm:"type:text" json:"user_content"`     // USER.md - 用户信息
	ToolsContent    string `gorm:"type:text" json:"tools_content"`    // TOOLS.md - 工具本地备注

	// 长期记忆
	MemoryContent string `gorm:"type:text" json:"memory_content"` // MEMORY.md - 长期记忆内容
	MemorySummary string `gorm:"type:text" json:"memory_summary"` // 记忆摘要

	// 能力配置
	SkillsList string `gorm:"type:text" json:"skills_list"` // 可用技能列表，JSON 数组
	ToolsList  string `gorm:"type:text" json:"tools_list"`  // 可用工具列表，JSON 数组

	// 模型配置
	Model         string  `gorm:"type:text" json:"model"`
	MaxTokens     int     `gorm:"default:4096" json:"max_tokens"`
	Temperature   float64 `gorm:"default:0.7" json:"temperature"`
	MaxIterations int     `gorm:"default:15" json:"max_iterations"`

	IsActive             bool `gorm:"default:true" json:"is_active"`
	IsDefault            bool `gorm:"default:false" json:"is_default"`             // 是否默认 Agent
	EnableThinkingProcess bool `gorm:"default:false" json:"enable_thinking_process"` // 是否启用思考过程输出

	CreatedAt time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName 指定表名
func (Agent) TableName() string {
	return "agents"
}

// GetAvailableSkills 获取可用技能列表
func (a *Agent) GetAvailableSkills() []string {
	if a.SkillsList == "" || a.SkillsList == "null" {
		return nil
	}
	var skills []string
	if err := json.Unmarshal([]byte(a.SkillsList), &skills); err != nil {
		return nil
	}
	return skills
}

// GetAvailableTools 获取可用工具列表
func (a *Agent) GetAvailableTools() []string {
	if a.ToolsList == "" || a.ToolsList == "null" {
		return nil
	}
	var tools []string
	if err := json.Unmarshal([]byte(a.ToolsList), &tools); err != nil {
		return nil
	}
	return tools
}
