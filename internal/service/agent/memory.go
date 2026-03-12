package agent

import (
	"fmt"
)

// GetMemory 获取 Agent 长期记忆
func (s *service) GetMemory(agentID uint) (string, error) {
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
func (s *service) UpdateMemory(agentID uint, content string) error {
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
func (s *service) GetMemorySummary(agentID uint) (string, error) {
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
func (s *service) UpdateMemorySummary(agentID uint, summary string) error {
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
