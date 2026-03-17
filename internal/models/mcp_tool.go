package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// MCPToolModel MCP工具数据库模型
// 存储从 MCP Server 获取的工具信息
type MCPToolModel struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	MCPServerID uint           `gorm:"not null;index" json:"mcp_server_id"`
	Name        string         `gorm:"size:255;not null" json:"name"`
	Description string         `json:"description"`
	InputSchema datatypes.JSON `json:"input_schema"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联关系
	MCPServer *MCPServer `gorm:"foreignKey:MCPServerID" json:"-"`
}

// TableName 指定表名
func (MCPToolModel) TableName() string {
	return "mcp_tools"
}
