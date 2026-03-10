package repository

import (
	"fmt"

	"github.com/weibaohui/nanobot-go/internal/models"
	"gorm.io/gorm"
)

// ChannelRepository Channel 仓库接口
type ChannelRepository interface {
	Create(channel *models.Channel) error
	GetByID(id uint) (*models.Channel, error)
	GetByUserID(userID uint) ([]models.Channel, error)
	GetByAgentID(agentID uint) ([]models.Channel, error)
	GetActiveByUserID(userID uint) ([]models.Channel, error)
	Update(channel *models.Channel) error
	Delete(id uint) error
	BindAgent(channelID, agentID uint) error
	UnbindAgent(channelID uint) error
}

// channelRepository Channel 仓库实现
type channelRepository struct {
	db *gorm.DB
}

// NewChannelRepository 创建 Channel 仓库
func NewChannelRepository(db *gorm.DB) ChannelRepository {
	return &channelRepository{db: db}
}

// Create 创建 Channel
func (r *channelRepository) Create(channel *models.Channel) error {
	if err := r.db.Create(channel).Error; err != nil {
		return fmt.Errorf("创建 Channel 失败: %w", err)
	}
	return nil
}

// GetByID 根据 ID 获取 Channel
func (r *channelRepository) GetByID(id uint) (*models.Channel, error) {
	var channel models.Channel
	if err := r.db.First(&channel, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("获取 Channel 失败: %w", err)
	}
	return &channel, nil
}

// GetByUserID 获取用户的所有 Channel
func (r *channelRepository) GetByUserID(userID uint) ([]models.Channel, error) {
	var channels []models.Channel
	if err := r.db.Where("user_id = ?", userID).Find(&channels).Error; err != nil {
		return nil, fmt.Errorf("获取用户 Channel 列表失败: %w", err)
	}
	return channels, nil
}

// GetByAgentID 获取绑定到指定 Agent 的所有 Channel
func (r *channelRepository) GetByAgentID(agentID uint) ([]models.Channel, error) {
	var channels []models.Channel
	if err := r.db.Where("agent_id = ?", agentID).Find(&channels).Error; err != nil {
		return nil, fmt.Errorf("获取 Agent 的 Channel 列表失败: %w", err)
	}
	return channels, nil
}

// GetActiveByUserID 获取用户的所有活跃 Channel
func (r *channelRepository) GetActiveByUserID(userID uint) ([]models.Channel, error) {
	var channels []models.Channel
	if err := r.db.Where("user_id = ? AND is_active = ?", userID, true).Find(&channels).Error; err != nil {
		return nil, fmt.Errorf("获取用户活跃 Channel 列表失败: %w", err)
	}
	return channels, nil
}

// Update 更新 Channel
func (r *channelRepository) Update(channel *models.Channel) error {
	if err := r.db.Save(channel).Error; err != nil {
		return fmt.Errorf("更新 Channel 失败: %w", err)
	}
	return nil
}

// Delete 删除 Channel
func (r *channelRepository) Delete(id uint) error {
	if err := r.db.Delete(&models.Channel{}, id).Error; err != nil {
		return fmt.Errorf("删除 Channel 失败: %w", err)
	}
	return nil
}

// BindAgent 绑定 Agent 到 Channel
func (r *channelRepository) BindAgent(channelID, agentID uint) error {
	if err := r.db.Model(&models.Channel{}).Where("id = ?", channelID).Update("agent_id", agentID).Error; err != nil {
		return fmt.Errorf("绑定 Agent 失败: %w", err)
	}
	return nil
}

// UnbindAgent 解除 Channel 的 Agent 绑定
func (r *channelRepository) UnbindAgent(channelID uint) error {
	if err := r.db.Model(&models.Channel{}).Where("id = ?", channelID).Update("agent_id", nil).Error; err != nil {
		return fmt.Errorf("解绑 Agent 失败: %w", err)
	}
	return nil
}
