package agent

import (
	"github.com/weibaohui/nanobot-go/internal/models"
	"github.com/weibaohui/nanobot-go/internal/repository"
)

// CreateAgentRequest 创建 Agent 请求
type CreateAgentRequest struct {
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	IdentityContent       string   `json:"identity_content"`
	SoulContent           string   `json:"soul_content"`
	AgentsContent         string   `json:"agents_content"`
	UserContent           string   `json:"user_content"`
	ToolsContent          string   `json:"tools_content"`
	Model                 string   `json:"model"`
	MaxTokens             int      `json:"max_tokens"`
	Temperature           float64  `json:"temperature"`
	MaxIterations         int      `json:"max_iterations"`
	HistoryMessages       int      `json:"history_messages"` // 携带的历史对话消息数量（默认10，范围0-50）
	SkillsList            []string `json:"skills_list"`
	ToolsList             []string `json:"tools_list"`
	IsDefault             bool     `json:"is_default"`
	EnableThinkingProcess bool     `json:"enable_thinking_process"`
}

// UpdateAgentRequest 更新 Agent 请求
type UpdateAgentRequest struct {
	Name                  string   `json:"name,omitempty"`
	Description           string   `json:"description,omitempty"`
	IdentityContent       string   `json:"identity_content,omitempty"`
	SoulContent           string   `json:"soul_content,omitempty"`
	AgentsContent         string   `json:"agents_content,omitempty"`
	UserContent           string   `json:"user_content,omitempty"`
	ToolsContent          string   `json:"tools_content,omitempty"`
	Model                 string   `json:"model,omitempty"`
	MaxTokens             int      `json:"max_tokens,omitempty"`
	Temperature           float64  `json:"temperature,omitempty"`
	MaxIterations         int      `json:"max_iterations,omitempty"`
	HistoryMessages       int      `json:"history_messages,omitempty"` // 携带的历史对话消息数量（默认10，范围0-50）
	SkillsList            []string `json:"skills_list,omitempty"`
	ToolsList             []string `json:"tools_list,omitempty"`
	IsActive              *bool    `json:"is_active,omitempty"`
	IsDefault             *bool    `json:"is_default,omitempty"`
	EnableThinkingProcess *bool    `json:"enable_thinking_process,omitempty"`
}

// AgentConfig Agent 配置响应
type AgentConfig struct {
	IdentityContent string  `json:"identity_content"`
	SoulContent     string  `json:"soul_content"`
	AgentsContent   string  `json:"agents_content"`
	UserContent     string  `json:"user_content"`
	ToolsContent    string  `json:"tools_content"`
	Model           string  `json:"model"`
	MaxTokens       int     `json:"max_tokens"`
	Temperature     float64 `json:"temperature"`
	MaxIterations   int     `json:"max_iterations"`
	HistoryMessages int     `json:"history_messages"` // 携带的历史对话消息数量（默认10，范围0-50）
}

// Service Agent 服务接口
type Service interface {
	// CRUD
	CreateAgent(userCode string, req CreateAgentRequest) (*models.Agent, error)
	GetAgent(id uint) (*models.Agent, error)
	GetAgentByCode(code string) (*models.Agent, error)
	GetUserAgents(userCode string) ([]models.Agent, error)
	UpdateAgent(id uint, req UpdateAgentRequest) (*models.Agent, error)
	DeleteAgent(id uint) error

	// 配置管理
	GetAgentConfig(agentID uint) (*AgentConfig, error)
	GetAgentConfigByCode(agentCode string) (*AgentConfig, error)
	UpdateAgentConfig(agentID uint, config *AgentConfig) error
	UpdateAgentConfigByCode(agentCode string, config *AgentConfig) error

	// 能力管理
	GetAvailableSkills(agentID uint) ([]string, error)
	SetAvailableSkills(agentID uint, skills []string) error
	GetAvailableTools(agentID uint) ([]string, error)
	SetAvailableTools(agentID uint, tools []string) error

	// 默认 Agent
	GetDefaultAgent(userCode string) (*models.Agent, error)
	SetDefaultAgent(userCode string, agentID uint) error
}

// service Agent 服务实现
type service struct {
	agentRepo   repository.AgentRepository
	codeService CodeService
}

// CodeService Code 生成服务接口（从父包导入）
type CodeService interface {
	GenerateAgentCode() (string, error)
}

// NewService 创建 Agent 服务
func NewService(agentRepo repository.AgentRepository, codeService CodeService) Service {
	return &service{
		agentRepo:   agentRepo,
		codeService: codeService,
	}
}
