package mcp

import (
	"github.com/weibaohui/nanobot-go/internal/models"
	"github.com/weibaohui/nanobot-go/internal/repository"
)

// CreateMCPServerRequest 创建 MCP 服务器请求
type CreateMCPServerRequest struct {
	Code          string            `json:"code" binding:"required"`
	Name          string            `json:"name" binding:"required"`
	Description   string            `json:"description"`
	TransportType string            `json:"transport_type" binding:"required,oneof=stdio http sse"`
	Command       string            `json:"command"`
	Args          []string          `json:"args"`
	URL           string            `json:"url"`
	EnvVars       map[string]string `json:"env_vars"`
}

// UpdateMCPServerRequest 更新 MCP 服务器请求
type UpdateMCPServerRequest struct {
	Name          string            `json:"name,omitempty"`
	Description   string            `json:"description,omitempty"`
	TransportType string            `json:"transport_type,omitempty" binding:"omitempty,oneof=stdio http sse"`
	Command       string            `json:"command,omitempty"`
	Args          []string          `json:"args,omitempty"`
	URL           string            `json:"url,omitempty"`
	EnvVars       map[string]string `json:"env_vars,omitempty"`
}

// MCPServerStatusUpdate 更新 MCP 服务器状态
type MCPServerStatusUpdate struct {
	Status       string `json:"status" binding:"required,oneof=active inactive error"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// MCPServerResponse MCP 服务器响应
type MCPServerResponse struct {
	ID          uint              `json:"id"`
	Code        string            `json:"code"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	TransportType string          `json:"transport_type"`
	Command     string            `json:"command,omitempty"`
	Args        []string          `json:"args,omitempty"`
	URL         string            `json:"url,omitempty"`
	EnvVars     map[string]string `json:"env_vars,omitempty"`
	Status      string            `json:"status"`
	Capabilities []models.MCPTool `json:"capabilities,omitempty"`
	LastConnectedAt *string       `json:"last_connected_at,omitempty"`
	ErrorMessage    string        `json:"error_message,omitempty"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
}

// CreateAgentMCPBindingRequest 创建 Agent MCP 绑定请求
type CreateAgentMCPBindingRequest struct {
	MCPServerID  uint     `json:"mcp_server_id" binding:"required"`
	EnabledTools []string `json:"enabled_tools"`
	IsActive     *bool    `json:"is_active,omitempty"`
	AutoLoad     *bool    `json:"auto_load,omitempty"` // 是否在对话开始时自动加载
}

// UpdateAgentMCPBindingRequest 更新 Agent MCP 绑定请求
type UpdateAgentMCPBindingRequest struct {
	EnabledTools []string `json:"enabled_tools,omitempty"`
	IsActive     *bool    `json:"is_active,omitempty"`
	AutoLoad     *bool    `json:"auto_load,omitempty"` // 是否在对话开始时自动加载
}

// AgentMCPBindingResponse Agent MCP 绑定响应
type AgentMCPBindingResponse struct {
	ID           uint               `json:"id"`
	AgentID      uint               `json:"agent_id"`
	MCPServerID  uint               `json:"mcp_server_id"`
	MCPServer    *MCPServerResponse `json:"mcp_server,omitempty"`
	EnabledTools []string           `json:"enabled_tools"`
	IsActive     bool               `json:"is_active"`
	AutoLoad     bool               `json:"auto_load"` // 是否在对话开始时自动加载
	CreatedAt    string             `json:"created_at"`
	UpdatedAt    string             `json:"updated_at"`
}

// Service MCP 服务接口
type Service interface {
	// MCP 服务器管理
	CreateServer(req CreateMCPServerRequest) (*models.MCPServer, error)
	GetServer(id uint) (*models.MCPServer, error)
	GetServerByCode(code string) (*models.MCPServer, error)
	ListServers() ([]models.MCPServer, error)
	ListServersByStatus(status string) ([]models.MCPServer, error)
	UpdateServer(id uint, req UpdateMCPServerRequest) (*models.MCPServer, error)
	DeleteServer(id uint) error
	UpdateServerStatus(id uint, status string, errorMsg string) error

	// MCP 服务器测试和刷新
	TestServer(id uint) error
	RefreshCapabilities(id uint) error

	// MCP 工具管理
	ListTools(serverID uint) ([]models.MCPToolModel, error)
	GetTool(toolID uint) (*models.MCPToolModel, error)

	// MCP 工具调用日志
	ListToolLogs(serverID uint, limit int) ([]models.MCPToolLog, error)
	LogToolExecution(sessionKey string, serverID uint, toolName string, params interface{}, result string, errMsg string, executeTimeMs int64) error

	// Agent MCP 绑定管理
	CreateAgentBinding(agentID uint, req CreateAgentMCPBindingRequest) (*models.AgentMCPBinding, error)
	GetAgentBindings(agentID uint) ([]models.AgentMCPBinding, error)
	GetAgentBindingByID(bindingID uint) (*models.AgentMCPBinding, error)
	UpdateAgentBinding(bindingID uint, req UpdateAgentMCPBindingRequest) (*models.AgentMCPBinding, error)
	DeleteAgentBinding(bindingID uint) error
	DeleteAgentBindingByServer(agentID, mcpServerID uint) error

	// 获取 Agent 可用的 MCP 工具
	GetAgentMCPTools(agentCode string) ([]AgentMCPToolInfo, error)

	// 获取 Agent 绑定的 MCP Servers（包含 auto_load 信息）
	GetAgentMCPServersWithBinding(agentCode string) ([]AgentMCPServerInfo, error)

	// 执行 MCP 工具
	ExecuteTool(serverID uint, toolName string, params map[string]interface{}) (string, error)
}

// AgentMCPServerInfo Agent 绑定的 MCP Server 信息（包含绑定配置）
// 注意：AutoLoad、IsActive、EnabledTools 等字段可直接从 Binding 字段获取
// 示例：info.Binding.AutoLoad, info.Binding.IsActive, info.Binding.GetEnabledTools()
type AgentMCPServerInfo struct {
	MCPServer *models.MCPServer       `json:"mcp_server"`
	Binding   *models.AgentMCPBinding `json:"binding"`
}

// AgentMCPToolInfo Agent 可用的 MCP 工具信息
type AgentMCPToolInfo struct {
	MCPServerCode string         `json:"mcp_server_code"`
	MCPServerName string         `json:"mcp_server_name"`
	Tool          models.MCPTool `json:"tool"`
}

// service MCP 服务实现
type service struct {
	mcpServerRepo       repository.MCPServerRepository
	agentMCPBindingRepo repository.AgentMCPBindingRepository
	agentRepo           repository.AgentRepository
	mcpToolRepo         repository.MCPToolRepository
	mcpToolLogRepo      repository.MCPToolLogRepository
}

// NewService 创建 MCP 服务
func NewService(
	mcpServerRepo repository.MCPServerRepository,
	agentMCPBindingRepo repository.AgentMCPBindingRepository,
	agentRepo repository.AgentRepository,
	mcpToolRepo repository.MCPToolRepository,
	mcpToolLogRepo repository.MCPToolLogRepository,
) Service {
	return &service{
		mcpServerRepo:       mcpServerRepo,
		agentMCPBindingRepo: agentMCPBindingRepo,
		agentRepo:           agentRepo,
		mcpToolRepo:         mcpToolRepo,
		mcpToolLogRepo:      mcpToolLogRepo,
	}
}
