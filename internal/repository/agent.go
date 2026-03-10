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
	GetByUserID(userID uint) ([]models.Agent, error)
	GetDefaultByUserID(userID uint) (*models.Agent, error)
	Update(agent *models.Agent) error
	Delete(id uint) error
	GetWithChannels(id uint) (*models.Agent, error)
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

// GetByUserID 获取用户的所有 Agent
func (r *agentRepository) GetByUserID(userID uint) ([]models.Agent, error) {
	var agents []models.Agent
	if err := r.db.Where("user_id = ?", userID).Find(&agents).Error; err != nil {
		return nil, fmt.Errorf("获取用户 Agent 列表失败: %w", err)
	}
	return agents, nil
}

// GetDefaultByUserID 获取用户的默认 Agent
func (r *agentRepository) GetDefaultByUserID(userID uint) (*models.Agent, error) {
	var agent models.Agent
	if err := r.db.Where("user_id = ? AND is_default = ?", userID, true).First(&agent).Error; err != nil {
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
