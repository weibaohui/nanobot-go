package agent

import (
	"encoding/json"
	"fmt"

	"github.com/weibaohui/nanobot-go/internal/models"
	"github.com/weibaohui/nanobot-go/internal/utils/codeutil"
)

// CreateAgent 创建 Agent
func (s *service) CreateAgent(userCode string, req CreateAgentRequest) (*models.Agent, error) {
	// 生成唯一 AgentCode
	agentCode, err := codeutil.GenerateUniqueCodeWithRetry(
		s.codeService.GenerateAgentCode,
		s.agentRepo.CheckAgentCodeExists,
		3,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate agent code: %w", err)
	}

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
	historyMessages := req.HistoryMessages
	if historyMessages <= 0 {
		historyMessages = 10
	}

	// 如果个性化配置为空，使用 OpenClaw 默认配置
	identityContent := req.IdentityContent
	if identityContent == "" {
		identityContent = DefaultIdentityContent
	}
	soulContent := req.SoulContent
	if soulContent == "" {
		soulContent = DefaultSoulContent
	}
	agentsContent := req.AgentsContent
	if agentsContent == "" {
		agentsContent = DefaultAgentsContent
	}
	userContent := req.UserContent
	if userContent == "" {
		userContent = DefaultUserContent
	}
	toolsContent := req.ToolsContent
	if toolsContent == "" {
		toolsContent = DefaultToolsContent
	}

	agent := &models.Agent{
		UserCode:              userCode,
		AgentCode:             agentCode,
		Name:                  req.Name,
		Description:           req.Description,
		IdentityContent:       identityContent,
		SoulContent:           soulContent,
		AgentsContent:         agentsContent,
		UserContent:           userContent,
		ToolsContent:          toolsContent,
		SkillsList:            string(skillsJSON),
		ToolsList:             string(toolsJSON),
		Model:                 req.Model,
		MaxTokens:             maxTokens,
		Temperature:           temperature,
		MaxIterations:         maxIterations,
		HistoryMessages:       historyMessages,
		IsActive:              true,
		IsDefault:             req.IsDefault,
		EnableThinkingProcess: req.EnableThinkingProcess,
	}

	if err := s.agentRepo.Create(agent); err != nil {
		return nil, err
	}

	return agent, nil
}

// GetAgent 获取 Agent
func (s *service) GetAgent(id uint) (*models.Agent, error) {
	return s.agentRepo.GetByID(id)
}

// GetAgentByCode 根据 Code 获取 Agent
func (s *service) GetAgentByCode(code string) (*models.Agent, error) {
	return s.agentRepo.GetByAgentCode(code)
}

// GetUserAgents 获取用户的所有 Agent
func (s *service) GetUserAgents(userCode string) ([]models.Agent, error) {
	return s.agentRepo.GetByUserCode(userCode)
}

// UpdateAgent 更新 Agent
func (s *service) UpdateAgent(id uint, req UpdateAgentRequest) (*models.Agent, error) {
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
	if req.HistoryMessages > 0 {
		agent.HistoryMessages = req.HistoryMessages
	}
	if req.IsActive != nil {
		agent.IsActive = *req.IsActive
	}
	if req.IsDefault != nil {
		agent.IsDefault = *req.IsDefault
	}
	if req.EnableThinkingProcess != nil {
		agent.EnableThinkingProcess = *req.EnableThinkingProcess
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
func (s *service) DeleteAgent(id uint) error {
	return s.agentRepo.Delete(id)
}
