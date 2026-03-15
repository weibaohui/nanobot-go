package service

import (
	"context"

	memorymodels "github.com/weibaohui/nanobot-go/internal/memory/models"
	"gorm.io/gorm"
)

// LongTermMemoryService 长期记忆服务接口
type LongTermMemoryService interface {
	List(ctx context.Context, offset int, limit int) ([]memorymodels.LongTermMemory, int64, error)
	Get(ctx context.Context, id uint64) (*memorymodels.LongTermMemory, error)
	GetByDate(ctx context.Context, date string) (*memorymodels.LongTermMemory, error)
	Create(ctx context.Context, memory *memorymodels.LongTermMemory) error
	Update(ctx context.Context, id uint64, memory *memorymodels.LongTermMemory) error
	Delete(ctx context.Context, id uint64) error
	Search(ctx context.Context, query string, offset int, limit int) ([]memorymodels.LongTermMemory, int64, error)
	GetRecent(ctx context.Context, days int, offset int, limit int) ([]memorymodels.LongTermMemory, int64, error)
}

// longTermMemoryService 长期记忆服务实现
type longTermMemoryService struct {
	db *gorm.DB
}

// NewLongTermMemoryService 创建长期记忆服务
func NewLongTermMemoryService(db *gorm.DB) LongTermMemoryService {
	return &longTermMemoryService{db: db}
}

// List 获取长期记忆列表
func (s *longTermMemoryService) List(ctx context.Context, offset int, limit int) ([]memorymodels.LongTermMemory, int64, error) {
	var memories []memorymodels.LongTermMemory
	var total int64

	if err := s.db.WithContext(ctx).Model(&memorymodels.LongTermMemory{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.WithContext(ctx).Order("date DESC").Offset(offset).Limit(limit).Find(&memories).Error; err != nil {
		return nil, 0, err
	}

	return memories, total, nil
}

// Get 获取单条长期记忆
func (s *longTermMemoryService) Get(ctx context.Context, id uint64) (*memorymodels.LongTermMemory, error) {
	var memory memorymodels.LongTermMemory
	if err := s.db.WithContext(ctx).First(&memory, id).Error; err != nil {
		return nil, err
	}
	return &memory, nil
}

// GetByDate 根据日期获取长期记忆
func (s *longTermMemoryService) GetByDate(ctx context.Context, date string) (*memorymodels.LongTermMemory, error) {
	var memory memorymodels.LongTermMemory
	if err := s.db.WithContext(ctx).Where("date = ?", date).First(&memory).Error; err != nil {
		return nil, err
	}
	return &memory, nil
}

// Create 创建长期记忆
func (s *longTermMemoryService) Create(ctx context.Context, memory *memorymodels.LongTermMemory) error {
	return s.db.WithContext(ctx).Create(memory).Error
}

// Update 更新长期记忆
func (s *longTermMemoryService) Update(ctx context.Context, id uint64, memory *memorymodels.LongTermMemory) error {
	return s.db.WithContext(ctx).Model(&memorymodels.LongTermMemory{}).Where("id = ?", id).Updates(memory).Error
}

// Delete 删除长期记忆
func (s *longTermMemoryService) Delete(ctx context.Context, id uint64) error {
	return s.db.WithContext(ctx).Delete(&memorymodels.LongTermMemory{}, id).Error
}

// Search 搜索长期记忆
func (s *longTermMemoryService) Search(ctx context.Context, query string, offset int, limit int) ([]memorymodels.LongTermMemory, int64, error) {
	var memories []memorymodels.LongTermMemory
	var total int64

	searchQuery := "%" + query + "%"

	countQuery := s.db.WithContext(ctx).Model(&memorymodels.LongTermMemory{}).
		Where("summary LIKE ? OR what_happened LIKE ? OR conclusion LIKE ? OR highlights LIKE ?",
			searchQuery, searchQuery, searchQuery, searchQuery)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.WithContext(ctx).
		Where("summary LIKE ? OR what_happened LIKE ? OR conclusion LIKE ? OR highlights LIKE ?",
			searchQuery, searchQuery, searchQuery, searchQuery).
		Order("date DESC").
		Offset(offset).
		Limit(limit).
		Find(&memories).Error; err != nil {
		return nil, 0, err
	}

	return memories, total, nil
}

// GetRecent 获取最近几天的长期记忆
func (s *longTermMemoryService) GetRecent(ctx context.Context, days int, offset int, limit int) ([]memorymodels.LongTermMemory, int64, error) {
	var memories []memorymodels.LongTermMemory
	var total int64

	if days <= 0 {
		days = 30 // 默认30天
	}

	query := s.db.WithContext(ctx).Model(&memorymodels.LongTermMemory{}).
		Where("date >= DATE('now', '-' || ? || ' day')", days)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("date DESC").Offset(offset).Limit(limit).Find(&memories).Error; err != nil {
		return nil, 0, err
	}

	return memories, total, nil
}
