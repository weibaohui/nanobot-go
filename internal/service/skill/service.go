// Package skill provides skills management service
package skill

import (
	"encoding/json"

	"github.com/weibaohui/nanobot-go/internal/models"
	"github.com/weibaohui/nanobot-go/internal/repository"
	"github.com/weibaohui/nanobot-go/pkg/agent"
)

// SkillInfo 技能信息
type SkillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
}

// SkillDetail 技能详情
type SkillDetail struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Source      string         `json:"source"`
	Content     string         `json:"content"`
	BoundAgents []models.Agent `json:"bound_agents"`
}

// Service 技能服务接口
type Service interface {
	// ListSkills 列出所有可用技能
	ListSkills() ([]SkillInfo, error)
	// GetSkill 获取单个技能详情
	GetSkill(name string) (*SkillDetail, error)
}

// service 技能服务实现
type service struct {
	skillsLoader *agent.SkillsLoader
	agentRepo    repository.AgentRepository
}

// NewService 创建技能服务
func NewService(workspace string, agentRepo repository.AgentRepository) Service {
	return &service{
		skillsLoader: agent.NewSkillsLoader(workspace),
		agentRepo:    agentRepo,
	}
}

// ListSkills 列出所有可用技能
func (s *service) ListSkills() ([]SkillInfo, error) {
	skills := s.skillsLoader.ListSkills(false)
	var result []SkillInfo
	for _, sk := range skills {
		meta := s.skillsLoader.GetSkillMetadata(sk.Name)
		desc := sk.Name
		if meta != nil {
			if d, ok := meta["description"]; ok && d != "" {
				desc = d
			}
		}
		result = append(result, SkillInfo{
			Name:        sk.Name,
			Description: desc,
			Source:      sk.Source,
		})
	}
	return result, nil
}

// GetSkill 获取单个技能详情
func (s *service) GetSkill(name string) (*SkillDetail, error) {
	// 获取技能内容
	content := s.skillsLoader.LoadSkill(name)
	if content == "" {
		return nil, nil
	}

	// 获取技能元数据
	meta := s.skillsLoader.GetSkillMetadata(name)
	desc := name
	if meta != nil {
		if d, ok := meta["description"]; ok && d != "" {
			desc = d
		}
	}

	// 获取来源
	skills := s.skillsLoader.ListSkills(false)
	source := "unknown"
	for _, sk := range skills {
		if sk.Name == name {
			source = sk.Source
			break
		}
	}

	// 获取绑定的 Agent
	agents, err := s.agentRepo.ListAll()
	if err != nil {
		return nil, err
	}

	var boundAgents []models.Agent
	for _, agt := range agents {
		if agt.SkillsList != "" {
			// 使用 JSON 解析检查技能绑定
			if containsSkill(agt.SkillsList, name) {
				boundAgents = append(boundAgents, agt)
			}
		}
	}

	return &SkillDetail{
		Name:        name,
		Description: desc,
		Source:      source,
		Content:     content,
		BoundAgents: boundAgents,
	}, nil
}

// containsSkill 检查技能列表是否包含指定技能
func containsSkill(skillsList, skillName string) bool {
	if skillsList == "" {
		return false
	}

	var skills []string
	if err := json.Unmarshal([]byte(skillsList), &skills); err != nil {
		// 如果解析失败，尝试简单字符串匹配
		return skillsList == skillName || skillsList == "[\""+skillName+"\"]"
	}

	for _, s := range skills {
		if s == skillName {
			return true
		}
	}
	return false
}
