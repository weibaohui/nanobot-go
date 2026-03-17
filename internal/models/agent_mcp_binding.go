package models

import (
	"encoding/json"
	"time"
)

// AgentMCPBinding Agent 与 MCP 服务器的绑定关系
type AgentMCPBinding struct {
	ID          uint `gorm:"primarykey" json:"id"`
	AgentID     uint `gorm:"not null;index" json:"agent_id"`
	MCPServerID uint `gorm:"not null;index" json:"mcp_server_id"`

	// 绑定配置
	EnabledTools string `gorm:"type:text" json:"enabled_tools"`    // 启用的工具列表 JSON (null 表示全部启用)
	IsActive     bool   `gorm:"default:true" json:"is_active"`       // 是否启用该 MCP 服务器
	AutoLoad     bool   `gorm:"default:false" json:"auto_load"`      // 是否在对话开始时自动加载该 MCP 服务器的工具

	// 关联模型
	Agent     Agent     `gorm:"foreignKey:AgentID" json:"agent,omitempty"`
	MCPServer MCPServer `gorm:"foreignKey:MCPServerID" json:"mcp_server,omitempty"`

	CreatedAt time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName 指定表名
func (AgentMCPBinding) TableName() string {
	return "agent_mcp_bindings"
}

// GetEnabledTools 获取启用的工具列表
// 返回 nil 表示全部启用
func (b *AgentMCPBinding) GetEnabledTools() []string {
	if b.EnabledTools == "" || b.EnabledTools == "null" {
		return nil
	}
	var tools []string
	if err := json.Unmarshal([]byte(b.EnabledTools), &tools); err != nil {
		return nil
	}
	return tools
}

// SetEnabledTools 设置启用的工具列表
// 传入 nil 或空数组表示全部启用
func (b *AgentMCPBinding) SetEnabledTools(tools []string) error {
	if len(tools) == 0 {
		b.EnabledTools = ""
		return nil
	}
	data, err := json.Marshal(tools)
	if err != nil {
		return err
	}
	b.EnabledTools = string(data)
	return nil
}

// IsToolEnabled 检查指定工具是否启用
func (b *AgentMCPBinding) IsToolEnabled(toolName string) bool {
	if !b.IsActive {
		return false
	}
	enabledTools := b.GetEnabledTools()
	if enabledTools == nil {
		return true // 全部启用
	}
	for _, name := range enabledTools {
		if name == toolName {
			return true
		}
	}
	return false
}
