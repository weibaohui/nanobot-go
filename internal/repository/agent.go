package repository

import (
	"fmt"

	"github.com/weibaohui/nanobot-go/internal/models"
	"gorm.io/gorm"
)

// AgentRepository Agent 仓库接口
type AgentRepository interface {
	Create(agent *models.Agent) error
	GetByID(id uint) (*models.Agent, error)
	GetByAgentCode(code string) (*models.Agent, error)
	GetByAgentCodes(codes []string) ([]*models.Agent, error) // 批量查询
	GetByUserCode(userCode string) ([]models.Agent, error)
	GetDefaultByUserCode(userCode string) (*models.Agent, error)
	Update(agent *models.Agent) error
	Delete(id uint) error
	GetWithChannels(id uint) (*models.Agent, error)
	// CheckAgentCodeExists 检查 AgentCode 是否已存在
	CheckAgentCodeExists(code string) (bool, error)
	// ListAll 获取所有 Agent
	ListAll() ([]models.Agent, error)
}

// agentRepository Agent 仓库实现
type agentRepository struct {
	db *gorm.DB
}

// NewAgentRepository 创建 Agent 仓库
func NewAgentRepository(db *gorm.DB) AgentRepository {
	return &agentRepository{db: db}
}

// Create 创建 Agent
func (r *agentRepository) Create(agent *models.Agent) error {
	if err := r.db.Create(agent).Error; err != nil {
		return fmt.Errorf("创建 Agent 失败: %w", err)
	}
	return nil
}

// GetByID 根据 ID 获取 Agent
func (r *agentRepository) GetByID(id uint) (*models.Agent, error) {
	var agent models.Agent
	if err := r.db.First(&agent, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("获取 Agent 失败: %w", err)
	}
	return &agent, nil
}

// GetByUserCode 获取用户的所有 Agent
func (r *agentRepository) GetByUserCode(userCode string) ([]models.Agent, error) {
	var agents []models.Agent
	if err := r.db.Where("user_code = ?", userCode).Find(&agents).Error; err != nil {
		return nil, fmt.Errorf("获取用户 Agent 列表失败: %w", err)
	}
	return agents, nil
}

// GetDefaultByUserCode 获取用户的默认 Agent
func (r *agentRepository) GetDefaultByUserCode(userCode string) (*models.Agent, error) {
	var agent models.Agent
	if err := r.db.Where("user_code = ? AND is_default = ?", userCode, true).First(&agent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("获取默认 Agent 失败: %w", err)
	}
	return &agent, nil
}

// Update 更新 Agent
func (r *agentRepository) Update(agent *models.Agent) error {
	if err := r.db.Save(agent).Error; err != nil {
		return fmt.Errorf("更新 Agent 失败: %w", err)
	}
	return nil
}

// Delete 删除 Agent
func (r *agentRepository) Delete(id uint) error {
	if err := r.db.Delete(&models.Agent{}, id).Error; err != nil {
		return fmt.Errorf("删除 Agent 失败: %w", err)
	}
	return nil
}

// GetWithChannels 获取 Agent 及其绑定的 Channels
func (r *agentRepository) GetWithChannels(id uint) (*models.Agent, error) {
	var agent models.Agent
	if err := r.db.Preload("Channels").First(&agent, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("获取 Agent 失败: %w", err)
	}
	return &agent, nil
}

// GetByAgentCode 根据 AgentCode 获取 Agent
func (r *agentRepository) GetByAgentCode(code string) (*models.Agent, error) {
	var agent models.Agent
	if err := r.db.Where("agent_code = ?", code).First(&agent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("获取 Agent 失败: %w", err)
	}
	return &agent, nil
}

// GetByAgentCodes 根据多个 AgentCode 批量获取 Agent
func (r *agentRepository) GetByAgentCodes(codes []string) ([]*models.Agent, error) {
	if len(codes) == 0 {
		return []*models.Agent{}, nil
	}
	var agents []*models.Agent
	if err := r.db.Where("agent_code IN ?", codes).Find(&agents).Error; err != nil {
		return nil, fmt.Errorf("批量获取 Agent 失败: %w", err)
	}
	return agents, nil
}

// CheckAgentCodeExists 检查 AgentCode 是否已存在
func (r *agentRepository) CheckAgentCodeExists(code string) (bool, error) {
	var count int64
	if err := r.db.Model(&models.Agent{}).Where("agent_code = ?", code).Count(&count).Error; err != nil {
		return false, fmt.Errorf("检查 AgentCode 失败: %w", err)
	}
	return count > 0, nil
}

// ListAll 获取所有 Agent
func (r *agentRepository) ListAll() ([]models.Agent, error) {
	var agents []models.Agent
	if err := r.db.Find(&agents).Error; err != nil {
		return nil, fmt.Errorf("获取 Agent 列表失败: %w", err)
	}
	return agents, nil
}
