package agent

import (
	"encoding/json"
	"fmt"

	"github.com/weibaohui/nanobot-go/internal/models"
)

// GetAvailableSkills 获取可用技能列表
func (s *service) GetAvailableSkills(agentID uint) ([]string, error) {
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
func (s *service) SetAvailableSkills(agentID uint, skills []string) error {
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
func (s *service) GetAvailableTools(agentID uint) ([]string, error) {
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
func (s *service) SetAvailableTools(agentID uint, tools []string) error {
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
func (s *service) GetDefaultAgent(userCode string) (*models.Agent, error) {
	return s.agentRepo.GetDefaultByUserCode(userCode)
}

// SetDefaultAgent 设置默认 Agent
func (s *service) SetDefaultAgent(userCode string, agentID uint) error {
	// 获取该用户的所有 Agent
	agents, err := s.agentRepo.GetByUserCode(userCode)
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
