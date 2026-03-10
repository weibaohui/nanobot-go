package service

import (
	"encoding/json"
	"fmt"

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

// AgentService Agent 服务接口
type AgentService interface {
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

// agentService Agent 服务实现
type agentService struct {
	agentRepo repository.AgentRepository
}

// NewAgentService 创建 Agent 服务
func NewAgentService(agentRepo repository.AgentRepository) AgentService {
	return &agentService{
		agentRepo: agentRepo,
	}
}

// CreateAgent 创建 Agent
func (s *agentService) CreateAgent(userID uint, req CreateAgentRequest) (*models.Agent, error) {
	// 序列化技能列表
	skillsJSON, err := json.Marshal(req.SkillsList)
	if err != nil {
		return nil, fmt.Errorf("序列化技能列表失败: %w", err)
	}

	// 序列化工具列表
	toolsJSON, err := json.Marshal(req.ToolsList)
	if err != nil {
		return nil, fmt.Errorf("序列化工具列表失败: %w", err)
	}

	// 设置默认值
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	temperature := req.Temperature
	if temperature == 0 {
		temperature = 0.7
	}
	maxIterations := req.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 15
	}

	agent := &models.Agent{
		UserID:          userID,
		Name:            req.Name,
		Description:     req.Description,
		IdentityContent: req.IdentityContent,
		SoulContent:     req.SoulContent,
		AgentsContent:   req.AgentsContent,
		UserContent:     req.UserContent,
		ToolsContent:    req.ToolsContent,
		SkillsList:      string(skillsJSON),
		ToolsList:       string(toolsJSON),
		Model:           req.Model,
		MaxTokens:       maxTokens,
		Temperature:     temperature,
		MaxIterations:   maxIterations,
		IsActive:        true,
		IsDefault:       req.IsDefault,
	}

	if err := s.agentRepo.Create(agent); err != nil {
		return nil, err
	}

	return agent, nil
}

// GetAgent 获取 Agent
func (s *agentService) GetAgent(id uint) (*models.Agent, error) {
	return s.agentRepo.GetByID(id)
}

// GetUserAgents 获取用户的所有 Agent
func (s *agentService) GetUserAgents(userID uint) ([]models.Agent, error) {
	return s.agentRepo.GetByUserID(userID)
}

// UpdateAgent 更新 Agent
func (s *agentService) UpdateAgent(id uint, req UpdateAgentRequest) (*models.Agent, error) {
	agent, err := s.agentRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, fmt.Errorf("agent not found")
	}

	// 更新字段
	if req.Name != "" {
		agent.Name = req.Name
	}
	if req.Description != "" {
		agent.Description = req.Description
	}
	if req.IdentityContent != "" {
		agent.IdentityContent = req.IdentityContent
	}
	if req.SoulContent != "" {
		agent.SoulContent = req.SoulContent
	}
	if req.AgentsContent != "" {
		agent.AgentsContent = req.AgentsContent
	}
	if req.UserContent != "" {
		agent.UserContent = req.UserContent
	}
	if req.ToolsContent != "" {
		agent.ToolsContent = req.ToolsContent
	}
	if req.Model != "" {
		agent.Model = req.Model
	}
	if req.MaxTokens > 0 {
		agent.MaxTokens = req.MaxTokens
	}
	if req.Temperature != 0 {
		agent.Temperature = req.Temperature
	}
	if req.MaxIterations > 0 {
		agent.MaxIterations = req.MaxIterations
	}
	if req.IsActive != nil {
		agent.IsActive = *req.IsActive
	}
	if req.IsDefault != nil {
		agent.IsDefault = *req.IsDefault
	}
	if req.SkillsList != nil {
		skillsJSON, err := json.Marshal(req.SkillsList)
		if err != nil {
			return nil, fmt.Errorf("序列化技能列表失败: %w", err)
		}
		agent.SkillsList = string(skillsJSON)
	}
	if req.ToolsList != nil {
		toolsJSON, err := json.Marshal(req.ToolsList)
		if err != nil {
			return nil, fmt.Errorf("序列化工具列表失败: %w", err)
		}
		agent.ToolsList = string(toolsJSON)
	}

	if err := s.agentRepo.Update(agent); err != nil {
		return nil, err
	}

	return agent, nil
}

// DeleteAgent 删除 Agent
func (s *agentService) DeleteAgent(id uint) error {
	return s.agentRepo.Delete(id)
}

// GetAgentConfig 获取 Agent 配置
func (s *agentService) GetAgentConfig(agentID uint) (*AgentConfig, error) {
	agent, err := s.agentRepo.GetByID(agentID)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, fmt.Errorf("agent not found")
	}

	return &AgentConfig{
		IdentityContent: agent.IdentityContent,
		SoulContent:     agent.SoulContent,
		AgentsContent:   agent.AgentsContent,
		UserContent:     agent.UserContent,
		ToolsContent:    agent.ToolsContent,
		Model:           agent.Model,
		MaxTokens:       agent.MaxTokens,
		Temperature:     agent.Temperature,
		MaxIterations:   agent.MaxIterations,
	}, nil
}

// UpdateAgentConfig 更新 Agent 配置
func (s *agentService) UpdateAgentConfig(agentID uint, config *AgentConfig) error {
	agent, err := s.agentRepo.GetByID(agentID)
	if err != nil {
		return err
	}
	if agent == nil {
		return fmt.Errorf("agent not found")
	}

	agent.IdentityContent = config.IdentityContent
	agent.SoulContent = config.SoulContent
	agent.AgentsContent = config.AgentsContent
	agent.UserContent = config.UserContent
	agent.ToolsContent = config.ToolsContent
	agent.Model = config.Model
	agent.MaxTokens = config.MaxTokens
	agent.Temperature = config.Temperature
	agent.MaxIterations = config.MaxIterations

	return s.agentRepo.Update(agent)
}

// GetMemory 获取 Agent 长期记忆
func (s *agentService) GetMemory(agentID uint) (string, error) {
	agent, err := s.agentRepo.GetByID(agentID)
	if err != nil {
		return "", err
	}
	if agent == nil {
		return "", fmt.Errorf("agent not found")
	}
	return agent.MemoryContent, nil
}

// UpdateMemory 更新 Agent 长期记忆
func (s *agentService) UpdateMemory(agentID uint, content string) error {
	agent, err := s.agentRepo.GetByID(agentID)
	if err != nil {
		return err
	}
	if agent == nil {
		return fmt.Errorf("agent not found")
	}

	agent.MemoryContent = content
	return s.agentRepo.Update(agent)
}

// GetMemorySummary 获取记忆摘要
func (s *agentService) GetMemorySummary(agentID uint) (string, error) {
	agent, err := s.agentRepo.GetByID(agentID)
	if err != nil {
		return "", err
	}
	if agent == nil {
		return "", fmt.Errorf("agent not found")
	}
	return agent.MemorySummary, nil
}

// UpdateMemorySummary 更新记忆摘要
func (s *agentService) UpdateMemorySummary(agentID uint, summary string) error {
	agent, err := s.agentRepo.GetByID(agentID)
	if err != nil {
		return err
	}
	if agent == nil {
		return fmt.Errorf("agent not found")
	}

	agent.MemorySummary = summary
	return s.agentRepo.Update(agent)
}

// GetAvailableSkills 获取可用技能列表
func (s *agentService) GetAvailableSkills(agentID uint) ([]string, error) {
	agent, err := s.agentRepo.GetByID(agentID)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, fmt.Errorf("agent not found")
	}

	if agent.SkillsList == "" || agent.SkillsList == "null" {
		return []string{}, nil
	}

	var skills []string
	if err := json.Unmarshal([]byte(agent.SkillsList), &skills); err != nil {
		return nil, fmt.Errorf("解析技能列表失败: %w", err)
	}

	return skills, nil
}

// SetAvailableSkills 设置可用技能列表
func (s *agentService) SetAvailableSkills(agentID uint, skills []string) error {
	agent, err := s.agentRepo.GetByID(agentID)
	if err != nil {
		return err
	}
	if agent == nil {
		return fmt.Errorf("agent not found")
	}

	skillsJSON, err := json.Marshal(skills)
	if err != nil {
		return fmt.Errorf("序列化技能列表失败: %w", err)
	}

	agent.SkillsList = string(skillsJSON)
	return s.agentRepo.Update(agent)
}

// GetAvailableTools 获取可用工具列表
func (s *agentService) GetAvailableTools(agentID uint) ([]string, error) {
	agent, err := s.agentRepo.GetByID(agentID)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, fmt.Errorf("agent not found")
	}

	if agent.ToolsList == "" || agent.ToolsList == "null" {
		return []string{}, nil
	}

	var tools []string
	if err := json.Unmarshal([]byte(agent.ToolsList), &tools); err != nil {
		return nil, fmt.Errorf("解析工具列表失败: %w", err)
	}

	return tools, nil
}

// SetAvailableTools 设置可用工具列表
func (s *agentService) SetAvailableTools(agentID uint, tools []string) error {
	agent, err := s.agentRepo.GetByID(agentID)
	if err != nil {
		return err
	}
	if agent == nil {
		return fmt.Errorf("agent not found")
	}

	toolsJSON, err := json.Marshal(tools)
	if err != nil {
		return fmt.Errorf("序列化工具列表失败: %w", err)
	}

	agent.ToolsList = string(toolsJSON)
	return s.agentRepo.Update(agent)
}

// GetDefaultAgent 获取用户的默认 Agent
func (s *agentService) GetDefaultAgent(userID uint) (*models.Agent, error) {
	return s.agentRepo.GetDefaultByUserID(userID)
}

// SetDefaultAgent 设置默认 Agent
func (s *agentService) SetDefaultAgent(userID uint, agentID uint) error {
	// 获取该用户的所有 Agent
	agents, err := s.agentRepo.GetByUserID(userID)
	if err != nil {
		return err
	}

	// 找到目标 Agent 并验证归属
	var targetAgent *models.Agent
	for _, agent := range agents {
		if agent.ID == agentID {
			targetAgent = &agent
			break
		}
	}
	if targetAgent == nil {
		return fmt.Errorf("agent not found or not belong to user")
	}

	// 清除其他 Agent 的默认标记
	for _, agent := range agents {
		if agent.IsDefault && agent.ID != agentID {
			agent.IsDefault = false
			if err := s.agentRepo.Update(&agent); err != nil {
				return err
			}
		}
	}

	// 设置目标 Agent 为默认
	targetAgent.IsDefault = true
	return s.agentRepo.Update(targetAgent)
}
