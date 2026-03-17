package models

import (
	"encoding/json"
	"time"
)

// MCPServer MCP 服务器模型
// 存储外部 MCP Server 的配置和连接信息
type MCPServer struct {
	ID          uint   `gorm:"primarykey" json:"id"`
	Code        string `gorm:"type:varchar(64);uniqueIndex" json:"code"`     // MCP 服务器编码
	Name        string `gorm:"type:varchar(128);not null" json:"name"`       // 显示名称
	Description string `gorm:"type:text" json:"description"`                  // 描述
	TransportType string `gorm:"type:varchar(32);not null" json:"transport_type"` // 传输类型: stdio, http, sse

	// stdio 类型配置
	Command string `gorm:"type:text" json:"command"` // 启动命令
	Args    string `gorm:"type:text" json:"args"`    // 启动参数 JSON 数组

	// http/sse 类型配置
	URL string `gorm:"type:varchar(512)" json:"url"` // 服务 URL

	// 环境变量
	EnvVars string `gorm:"type:text" json:"env_vars"` // 环境变量 JSON

	// 状态和能力
	Status       string `gorm:"type:varchar(32);default:'inactive'" json:"status"` // 状态: active, inactive, error
	Capabilities string `gorm:"type:text" json:"capabilities"`                     // 能力列表 JSON (工具列表)

	// 时间戳
	LastConnectedAt *time.Time `gorm:"type:datetime" json:"last_connected_at"` // 最后连接时间
	ErrorMessage    string     `gorm:"type:text" json:"error_message"`         // 错误信息
	CreatedAt       time.Time  `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName 指定表名
func (MCPServer) TableName() string {
	return "mcp_servers"
}

// GetArgs 获取启动参数列表
func (m *MCPServer) GetArgs() []string {
	if m.Args == "" || m.Args == "null" {
		return nil
	}
	var args []string
	if err := json.Unmarshal([]byte(m.Args), &args); err != nil {
		return nil
	}
	return args
}

// SetArgs 设置启动参数列表
func (m *MCPServer) SetArgs(args []string) error {
	data, err := json.Marshal(args)
	if err != nil {
		return err
	}
	m.Args = string(data)
	return nil
}

// GetEnvVars 获取环境变量
func (m *MCPServer) GetEnvVars() map[string]string {
	if m.EnvVars == "" || m.EnvVars == "null" {
		return nil
	}
	var envVars map[string]string
	if err := json.Unmarshal([]byte(m.EnvVars), &envVars); err != nil {
		return nil
	}
	return envVars
}

// SetEnvVars 设置环境变量
func (m *MCPServer) SetEnvVars(envVars map[string]string) error {
	data, err := json.Marshal(envVars)
	if err != nil {
		return err
	}
	m.EnvVars = string(data)
	return nil
}

// GetCapabilities 获取能力列表 (工具列表)
func (m *MCPServer) GetCapabilities() []MCPTool {
	if m.Capabilities == "" || m.Capabilities == "null" {
		return nil
	}
	var capabilities []MCPTool
	if err := json.Unmarshal([]byte(m.Capabilities), &capabilities); err != nil {
		return nil
	}
	return capabilities
}

// SetCapabilities 设置能力列表
func (m *MCPServer) SetCapabilities(capabilities []MCPTool) error {
	data, err := json.Marshal(capabilities)
	if err != nil {
		return err
	}
	m.Capabilities = string(data)
	return nil
}

// MCPTool MCP 工具定义
type MCPTool struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema,omitempty"`
}

// IsActive 检查 MCP 服务器是否活跃
func (m *MCPServer) IsActive() bool {
	return m.Status == "active"
}

// IsStdio 检查是否为 stdio 传输类型
func (m *MCPServer) IsStdio() bool {
	return m.TransportType == "stdio"
}

// IsHTTP 检查是否为 http 传输类型
func (m *MCPServer) IsHTTP() bool {
	return m.TransportType == "http"
}

// IsSSE 检查是否为 sse 传输类型
func (m *MCPServer) IsSSE() bool {
	return m.TransportType == "sse"
}

// MarshalJSON 自定义 JSON 序列化
// 将 Capabilities 字符串字段解析为数组输出
func (m *MCPServer) MarshalJSON() ([]byte, error) {
	type Alias MCPServer
	return json.Marshal(&struct {
		Capabilities []MCPTool `json:"capabilities,omitempty"`
		*Alias
	}{
		Capabilities: m.GetCapabilities(),
		Alias:        (*Alias)(m),
	})
}
