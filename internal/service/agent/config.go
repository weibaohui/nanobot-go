package agent

import (
	"fmt"
)

// GetAgentConfig 获取 Agent 配置
func (s *service) GetAgentConfig(agentID uint) (*AgentConfig, error) {
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
		HistoryMessages: agent.HistoryMessages,
	}, nil
}

// GetAgentConfigByCode 根据 Code 获取 Agent 配置
func (s *service) GetAgentConfigByCode(agentCode string) (*AgentConfig, error) {
	agent, err := s.agentRepo.GetByAgentCode(agentCode)
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
		HistoryMessages: agent.HistoryMessages,
	}, nil
}

// UpdateAgentConfig 更新 Agent 配置
func (s *service) UpdateAgentConfig(agentID uint, config *AgentConfig) error {
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
	agent.HistoryMessages = config.HistoryMessages

	return s.agentRepo.Update(agent)
}

// UpdateAgentConfigByCode 根据 Code 更新 Agent 配置
func (s *service) UpdateAgentConfigByCode(agentCode string, config *AgentConfig) error {
	agent, err := s.agentRepo.GetByAgentCode(agentCode)
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
	agent.HistoryMessages = config.HistoryMessages

	return s.agentRepo.Update(agent)
}
