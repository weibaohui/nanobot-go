package agent

import (
	"github.com/weibaohui/nanobot-go/internal/models"
	"github.com/weibaohui/nanobot-go/internal/repository"
)

// CreateAgentRequest 创建 Agent 请求
type CreateAgentRequest struct {
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	IdentityContent string   `json:"identity_content"`
	SoulContent     string   `json:"soul_content"`
	AgentsContent   string   `json:"agents_content"`
	UserContent     string   `json:"user_content"`
	ToolsContent    string   `json:"tools_content"`
	Model           string   `json:"model"`
	MaxTokens       int      `json:"max_tokens"`
	Temperature     float64  `json:"temperature"`
	MaxIterations   int      `json:"max_iterations"`
	SkillsList      []string `json:"skills_list"`
	ToolsList       []string `json:"tools_list"`
	IsDefault       bool     `json:"is_default"`
}

// UpdateAgentRequest 更新 Agent 请求
type UpdateAgentRequest struct {
	Name            string   `json:"name,omitempty"`
	Description     string   `json:"description,omitempty"`
	IdentityContent string   `json:"identity_content,omitempty"`
	SoulContent     string   `json:"soul_content,omitempty"`
	AgentsContent   string   `json:"agents_content,omitempty"`
	UserContent     string   `json:"user_content,omitempty"`
	ToolsContent    string   `json:"tools_content,omitempty"`
	Model           string   `json:"model,omitempty"`
	MaxTokens       int      `json:"max_tokens,omitempty"`
	Temperature     float64  `json:"temperature,omitempty"`
	MaxIterations   int      `json:"max_iterations,omitempty"`
	SkillsList      []string `json:"skills_list,omitempty"`
	ToolsList       []string `json:"tools_list,omitempty"`
	IsActive        *bool    `json:"is_active,omitempty"`
	IsDefault       *bool    `json:"is_default,omitempty"`
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
}

// Service Agent 服务接口
type Service interface {
	// CRUD
	CreateAgent(userID uint, req CreateAgentRequest) (*models.Agent, error)
	GetAgent(id uint) (*models.Agent, error)
	GetUserAgents(userID uint) ([]models.Agent, error)
	UpdateAgent(id uint, req UpdateAgentRequest) (*models.Agent, error)
	DeleteAgent(id uint) error

	// 配置管理
	GetAgentConfig(agentID uint) (*AgentConfig, error)
	UpdateAgentConfig(agentID uint, config *AgentConfig) error

	// 记忆管理
	GetMemory(agentID uint) (string, error)
	UpdateMemory(agentID uint, content string) error
	GetMemorySummary(agentID uint) (string, error)
	UpdateMemorySummary(agentID uint, summary string) error

	// 能力管理
	GetAvailableSkills(agentID uint) ([]string, error)
	SetAvailableSkills(agentID uint, skills []string) error
	GetAvailableTools(agentID uint) ([]string, error)
	SetAvailableTools(agentID uint, tools []string) error

	// 默认 Agent
	GetDefaultAgent(userID uint) (*models.Agent, error)
	SetDefaultAgent(userID uint, agentID uint) error
}

// service Agent 服务实现
type service struct {
	agentRepo repository.AgentRepository
}

// NewService 创建 Agent 服务
func NewService(agentRepo repository.AgentRepository) Service {
	return &service{
		agentRepo: agentRepo,
	}
}
