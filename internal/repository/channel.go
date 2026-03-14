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
	GetByChannelCode(code string) (*models.Channel, error)
	GetByChannelCodes(codes []string) ([]*models.Channel, error) // 批量查询
	GetByUserCode(userCode string) ([]models.Channel, error)
	GetByAgentCode(agentCode string) ([]models.Channel, error)
	GetActiveByUserCode(userCode string) ([]models.Channel, error)
	Update(channel *models.Channel) error
	Delete(id uint) error
	BindAgent(channelCode, agentCode string) error
	UnbindAgent(channelCode string) error
	// CheckChannelCodeExists 检查 ChannelCode 是否已存在
	CheckChannelCodeExists(code string) (bool, error)
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

// GetByUserCode 获取用户的所有 Channel
func (r *channelRepository) GetByUserCode(userCode string) ([]models.Channel, error) {
	var channels []models.Channel
	if err := r.db.Where("user_code = ?", userCode).Find(&channels).Error; err != nil {
		return nil, fmt.Errorf("获取用户 Channel 列表失败: %w", err)
	}
	return channels, nil
}

// GetByAgentCode 获取绑定到指定 Agent 的所有 Channel
func (r *channelRepository) GetByAgentCode(agentCode string) ([]models.Channel, error) {
	var channels []models.Channel
	if err := r.db.Where("agent_code = ?", agentCode).Find(&channels).Error; err != nil {
		return nil, fmt.Errorf("获取 Agent 的 Channel 列表失败: %w", err)
	}
	return channels, nil
}

// GetActiveByUserCode 获取用户的所有活跃 Channel
func (r *channelRepository) GetActiveByUserCode(userCode string) ([]models.Channel, error) {
	var channels []models.Channel
	if err := r.db.Where("user_code = ? AND is_active = ?", userCode, true).Find(&channels).Error; err != nil {
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
func (r *channelRepository) BindAgent(channelCode, agentCode string) error {
	if err := r.db.Model(&models.Channel{}).Where("channel_code = ?", channelCode).Update("agent_code", agentCode).Error; err != nil {
		return fmt.Errorf("绑定 Agent 失败: %w", err)
	}
	return nil
}

// UnbindAgent 解除 Channel 的 Agent 绑定
func (r *channelRepository) UnbindAgent(channelCode string) error {
	if err := r.db.Model(&models.Channel{}).Where("channel_code = ?", channelCode).Update("agent_code", "").Error; err != nil {
		return fmt.Errorf("解绑 Agent 失败: %w", err)
	}
	return nil
}

// GetByChannelCode 根据 ChannelCode 获取 Channel
func (r *channelRepository) GetByChannelCode(code string) (*models.Channel, error) {
	var channel models.Channel
	if err := r.db.Where("channel_code = ?", code).First(&channel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("获取 Channel 失败: %w", err)
	}
	return &channel, nil
}

// GetByChannelCodes 根据多个 ChannelCode 批量获取 Channel
func (r *channelRepository) GetByChannelCodes(codes []string) ([]*models.Channel, error) {
	if len(codes) == 0 {
		return []*models.Channel{}, nil
	}
	var channels []*models.Channel
	if err := r.db.Where("channel_code IN ?", codes).Find(&channels).Error; err != nil {
		return nil, fmt.Errorf("批量获取 Channel 失败: %w", err)
	}
	return channels, nil
}

// CheckChannelCodeExists 检查 ChannelCode 是否已存在
func (r *channelRepository) CheckChannelCodeExists(code string) (bool, error) {
	var count int64
	if err := r.db.Model(&models.Channel{}).Where("channel_code = ?", code).Count(&count).Error; err != nil {
		return false, fmt.Errorf("检查 ChannelCode 失败: %w", err)
	}
	return count > 0, nil
}
