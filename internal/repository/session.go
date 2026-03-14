package repository

import (
	"fmt"
	"time"

	"github.com/weibaohui/nanobot-go/internal/models"
	"gorm.io/gorm"
)

// SessionRepository 会话仓库接口
type SessionRepository interface {
	Create(session *models.Session) error
	GetByID(id uint) (*models.Session, error)
	GetBySessionKey(key string) (*models.Session, error)
	GetByChannelCode(channelCode string) ([]models.Session, error)
	GetByUserCode(userCode string) ([]models.Session, error)
	GetActiveByUserCode(userCode string) ([]models.Session, error)
	UpdateLastActive(sessionKey string) error
	Update(session *models.Session) error
	Delete(sessionKey string) error
	DeleteByChannelCode(channelCode string) error
}

// sessionRepository 会话仓库实现
type sessionRepository struct {
	db *gorm.DB
}

// NewSessionRepository 创建会话仓库
func NewSessionRepository(db *gorm.DB) SessionRepository {
	return &sessionRepository{db: db}
}

// Create 创建会话
func (r *sessionRepository) Create(session *models.Session) error {
	if err := r.db.Create(session).Error; err != nil {
		return fmt.Errorf("创建会话失败: %w", err)
	}
	return nil
}

// GetByID 根据 ID 获取会话
func (r *sessionRepository) GetByID(id uint) (*models.Session, error) {
	var session models.Session
	if err := r.db.First(&session, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("获取会话失败: %w", err)
	}
	return &session, nil
}

// GetBySessionKey 根据 SessionKey 获取会话
func (r *sessionRepository) GetBySessionKey(key string) (*models.Session, error) {
	var session models.Session
	if err := r.db.Where("session_key = ?", key).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("获取会话失败: %w", err)
	}
	return &session, nil
}

// GetByChannelCode 获取 Channel 的所有会话
func (r *sessionRepository) GetByChannelCode(channelCode string) ([]models.Session, error) {
	var sessions []models.Session
	if err := r.db.Where("channel_code = ?", channelCode).Find(&sessions).Error; err != nil {
		return nil, fmt.Errorf("获取 Channel 会话列表失败: %w", err)
	}
	return sessions, nil
}

// GetByUserCode 获取用户的所有会话
func (r *sessionRepository) GetByUserCode(userCode string) ([]models.Session, error) {
	var sessions []models.Session
	if err := r.db.Where("user_code = ?", userCode).Find(&sessions).Error; err != nil {
		return nil, fmt.Errorf("获取用户会话列表失败: %w", err)
	}
	return sessions, nil
}

// GetActiveByUserCode 获取用户的所有活跃会话
func (r *sessionRepository) GetActiveByUserCode(userCode string) ([]models.Session, error) {
	var sessions []models.Session
	// 注意：Session 模型没有 deleted_at 字段，所以不使用软删除条件
	if err := r.db.Where("user_code = ?", userCode).Find(&sessions).Error; err != nil {
		return nil, fmt.Errorf("获取用户活跃会话列表失败: %w", err)
	}
	return sessions, nil
}

// UpdateLastActive 更新会话最后活跃时间
func (r *sessionRepository) UpdateLastActive(sessionKey string) error {
	now := time.Now()
	if err := r.db.Model(&models.Session{}).Where("session_key = ?", sessionKey).Update("last_active_at", now).Error; err != nil {
		return fmt.Errorf("更新会话活跃时间失败: %w", err)
	}
	return nil
}

// Update 更新会话
func (r *sessionRepository) Update(session *models.Session) error {
	if err := r.db.Save(session).Error; err != nil {
		return fmt.Errorf("更新会话失败: %w", err)
	}
	return nil
}

// Delete 删除会话
func (r *sessionRepository) Delete(sessionKey string) error {
	if err := r.db.Where("session_key = ?", sessionKey).Delete(&models.Session{}).Error; err != nil {
		return fmt.Errorf("删除会话失败: %w", err)
	}
	return nil
}

// DeleteByChannelCode 删除 Channel 的所有会话
func (r *sessionRepository) DeleteByChannelCode(channelCode string) error {
	if err := r.db.Where("channel_code = ?", channelCode).Delete(&models.Session{}).Error; err != nil {
		return fmt.Errorf("删除 Channel 会话失败: %w", err)
	}
	return nil
}
