package service

import (
	"context"
	"time"

	memorymodels "github.com/weibaohui/nanobot-go/memory/models"
	"gorm.io/gorm"
)

// StreamMemoryService 短期记忆服务接口
type StreamMemoryService interface {
	List(ctx context.Context, sessionKey string, eventType string, offset int, limit int) ([]memorymodels.StreamMemory, int64, error)
	Get(ctx context.Context, id uint64) (*memorymodels.StreamMemory, error)
	Create(ctx context.Context, memory *memorymodels.StreamMemory) error
	Update(ctx context.Context, id uint64, memory *memorymodels.StreamMemory) error
	Delete(ctx context.Context, id uint64) error
	MarkProcessed(ctx context.Context, id uint64) error
	GetUnprocessed(ctx context.Context) ([]memorymodels.StreamMemory, error)
}

// streamMemoryService 短期记忆服务实现
type streamMemoryService struct {
	db *gorm.DB
}

// NewStreamMemoryService 创建短期记忆服务
func NewStreamMemoryService(db *gorm.DB) StreamMemoryService {
	return &streamMemoryService{db: db}
}

// List 获取短期记忆列表
func (s *streamMemoryService) List(ctx context.Context, sessionKey string, eventType string, offset int, limit int) ([]memorymodels.StreamMemory, int64, error) {
	var memories []memorymodels.StreamMemory
	var total int64

	query := s.db.WithContext(ctx).Model(&memorymodels.StreamMemory{})

	if sessionKey != "" {
		query = query.Where("session_key = ?", sessionKey)
	}
	if eventType != "" {
		query = query.Where("event_type = ?", eventType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&memories).Error; err != nil {
		return nil, 0, err
	}

	return memories, total, nil
}

// Get 获取单条短期记忆
func (s *streamMemoryService) Get(ctx context.Context, id uint64) (*memorymodels.StreamMemory, error) {
	var memory memorymodels.StreamMemory
	if err := s.db.WithContext(ctx).First(&memory, id).Error; err != nil {
		return nil, err
	}
	return &memory, nil
}

// Create 创建短期记忆
func (s *streamMemoryService) Create(ctx context.Context, memory *memorymodels.StreamMemory) error {
	return s.db.WithContext(ctx).Create(memory).Error
}

// Update 更新短期记忆
func (s *streamMemoryService) Update(ctx context.Context, id uint64, memory *memorymodels.StreamMemory) error {
	return s.db.WithContext(ctx).Model(&memorymodels.StreamMemory{}).Where("id = ?", id).Updates(memory).Error
}

// Delete 删除短期记忆
func (s *streamMemoryService) Delete(ctx context.Context, id uint64) error {
	return s.db.WithContext(ctx).Delete(&memorymodels.StreamMemory{}, id).Error
}

// MarkProcessed 标记为已处理
func (s *streamMemoryService) MarkProcessed(ctx context.Context, id uint64) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&memorymodels.StreamMemory{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"processed":    true,
			"processed_at": now,
		}).Error
}

// GetUnprocessed 获取未处理的短期记忆
func (s *streamMemoryService) GetUnprocessed(ctx context.Context) ([]memorymodels.StreamMemory, error) {
	var memories []memorymodels.StreamMemory
	if err := s.db.WithContext(ctx).
		Where("processed = ?", false).
		Order("created_at ASC").
		Find(&memories).Error; err != nil {
		return nil, err
	}
	return memories, nil
}
