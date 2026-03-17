package models

import (
	"time"

	"gorm.io/datatypes"
)

// MCPToolLog MCP工具调用日志
// 记录每次工具调用的详细信息
type MCPToolLog struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	SessionKey   string         `gorm:"size:64;not null;index" json:"session_key"`
	MCPServerID  uint           `gorm:"not null;index" json:"mcp_server_id"`
	ToolName     string         `gorm:"size:255;not null;index" json:"tool_name"`
	Parameters   datatypes.JSON `json:"parameters"`
	Result       string         `gorm:"type:text" json:"result"`
	ErrorMessage string         `gorm:"type:text" json:"error_message"`
	ExecuteTime  uint           `gorm:"default:0" json:"execute_time"` // 执行耗时(ms)
	CreatedAt    time.Time      `gorm:"index" json:"created_at"`
}

// TableName 指定表名
func (MCPToolLog) TableName() string {
	return "mcp_tool_logs"
}
