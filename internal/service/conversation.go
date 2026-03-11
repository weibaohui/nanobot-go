package service

import (
	"context"

	"github.com/weibaohui/nanobot-go/internal/models"
	"gorm.io/gorm"
)

// ConversationRecordService 对话记录服务接口
type ConversationRecordService interface {
	List(ctx context.Context, userID uint, agentID uint, channelID uint, sessionKey string, offset int, limit int) ([]models.ConversationRecord, int64, error)
	Get(ctx context.Context, id uint) (*models.ConversationRecord, error)
	Create(ctx context.Context, record *models.ConversationRecord) error
	Update(ctx context.Context, id uint, record *models.ConversationRecord) error
	Delete(ctx context.Context, id uint) error
	GetBySessionKey(ctx context.Context, sessionKey string, offset int, limit int) ([]models.ConversationRecord, int64, error)
	GetByTraceID(ctx context.Context, traceID string) ([]models.ConversationRecord, error)
}

// conversationRecordService 对话记录服务实现
type conversationRecordService struct {
	db *gorm.DB
}

// NewConversationRecordService 创建对话记录服务
func NewConversationRecordService(db *gorm.DB) ConversationRecordService {
	return &conversationRecordService{db: db}
}

// List 获取对话记录列表
func (s *conversationRecordService) List(ctx context.Context, userID uint, agentID uint, channelID uint, sessionKey string, offset int, limit int) ([]models.ConversationRecord, int64, error) {
	var records []models.ConversationRecord
	var total int64

	query := s.db.WithContext(ctx).Model(&models.ConversationRecord{})

	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if agentID > 0 {
		query = query.Where("agent_id = ?", agentID)
	}
	if channelID > 0 {
		query = query.Where("channel_id = ?", channelID)
	}
	if sessionKey != "" {
		query = query.Where("session_key = ?", sessionKey)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("timestamp DESC").Offset(offset).Limit(limit).Find(&records).Error; err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

// Get 获取单条对话记录
func (s *conversationRecordService) Get(ctx context.Context, id uint) (*models.ConversationRecord, error) {
	var record models.ConversationRecord
	if err := s.db.WithContext(ctx).First(&record, id).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

// Create 创建对话记录
func (s *conversationRecordService) Create(ctx context.Context, record *models.ConversationRecord) error {
	return s.db.WithContext(ctx).Create(record).Error
}

// Update 更新对话记录
func (s *conversationRecordService) Update(ctx context.Context, id uint, record *models.ConversationRecord) error {
	return s.db.WithContext(ctx).Model(&models.ConversationRecord{}).Where("id = ?", id).Updates(record).Error
}

// Delete 删除对话记录
func (s *conversationRecordService) Delete(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.ConversationRecord{}, id).Error
}

// GetBySessionKey 根据 SessionKey 获取对话记录
func (s *conversationRecordService) GetBySessionKey(ctx context.Context, sessionKey string, offset int, limit int) ([]models.ConversationRecord, int64, error) {
	var records []models.ConversationRecord
	var total int64

	query := s.db.WithContext(ctx).Where("session_key = ?", sessionKey)

	if err := query.Model(&models.ConversationRecord{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("timestamp ASC").Offset(offset).Limit(limit).Find(&records).Error; err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

// GetByTraceID 根据 TraceID 获取对话记录
func (s *conversationRecordService) GetByTraceID(ctx context.Context, traceID string) ([]models.ConversationRecord, error) {
	var records []models.ConversationRecord
	if err := s.db.WithContext(ctx).Where("trace_id = ?", traceID).Order("timestamp ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}
