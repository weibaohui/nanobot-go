package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"github.com/weibaohui/nanobot-go/memory/models"
)

// LongTermMemoryRepository 长期记忆仓储接口
type LongTermMemoryRepository interface {
	// Create 创建长期记忆
	Create(ctx context.Context, memory *models.LongTermMemory) error
	// FindByID 根据ID查询
	FindByID(ctx context.Context, id uint64) (*models.LongTermMemory, error)
	// FindByDate 根据日期查询
	FindByDate(ctx context.Context, date string) (*models.LongTermMemory, error)
	// FindByTimeRange 根据时间范围查询
	FindByTimeRange(ctx context.Context, startTime, endTime time.Time, opts *models.QueryOptions) ([]models.LongTermMemory, error)
	// SearchByKeyword 关键词搜索（使用LIKE）
	SearchByKeyword(ctx context.Context, keyword string, opts *models.QueryOptions) ([]models.LongTermMemory, error)
	// Update 更新长期记忆
	Update(ctx context.Context, memory *models.LongTermMemory) error
	// DeleteByDate 根据日期删除
	DeleteByDate(ctx context.Context, date string) error
}

// longTermMemoryRepository 长期记忆仓储实现
type longTermMemoryRepository struct {
	db *gorm.DB
}

// NewLongTermMemoryRepository 创建长期记忆仓储实例
func NewLongTermMemoryRepository(db *gorm.DB) LongTermMemoryRepository {
	return &longTermMemoryRepository{db: db}
}

// Create 创建长期记忆
func (r *longTermMemoryRepository) Create(ctx context.Context, memory *models.LongTermMemory) error {
	memory.CreatedAt = time.Now()
	memory.UpdatedAt = time.Now()
	if err := r.db.WithContext(ctx).Create(memory).Error; err != nil {
		return fmt.Errorf("failed to create long term memory: %w", err)
	}
	return nil
}

// FindByID 根据ID查询长期记忆
func (r *longTermMemoryRepository) FindByID(ctx context.Context, id uint64) (*models.LongTermMemory, error) {
	var memory models.LongTermMemory
	if err := r.db.WithContext(ctx).First(&memory, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find long term memory by id: %w", err)
	}
	return &memory, nil
}

// FindByDate 根据日期查询长期记忆
func (r *longTermMemoryRepository) FindByDate(ctx context.Context, date string) (*models.LongTermMemory, error) {
	var memory models.LongTermMemory
	if err := r.db.WithContext(ctx).Where("date = ?", date).First(&memory).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find long term memory by date: %w", err)
	}
	return &memory, nil
}

// FindByTimeRange 根据时间范围查询长期记忆
func (r *longTermMemoryRepository) FindByTimeRange(ctx context.Context, startTime, endTime time.Time, opts *models.QueryOptions) ([]models.LongTermMemory, error) {
	if opts == nil {
		opts = &models.QueryOptions{}
	}

	// 将时间转换为日期字符串
	startDate := startTime.Format("2006-01-02")
	endDate := endTime.Format("2006-01-02")

	query := r.db.WithContext(ctx).
		Where("date >= ?", startDate).
		Where("date <= ?", endDate)

	// 排序
	orderBy := opts.OrderBy
	if orderBy == "" {
		orderBy = "date"
	}
	order := opts.Order
	if order == "" {
		order = "DESC"
	}
	query = query.Order(orderBy + " " + order)

	// 分页
	if opts.Limit > 0 {
		query = query.Limit(opts.Limit)
	}
	if opts.Offset > 0 {
		query = query.Offset(opts.Offset)
	}

	var memories []models.LongTermMemory
	if err := query.Find(&memories).Error; err != nil {
		return nil, fmt.Errorf("failed to find long term memories by time range: %w", err)
	}
	return memories, nil
}

// SearchByKeyword 关键词搜索长期记忆
func (r *longTermMemoryRepository) SearchByKeyword(ctx context.Context, keyword string, opts *models.QueryOptions) ([]models.LongTermMemory, error) {
	if opts == nil {
		opts = &models.QueryOptions{}
	}

	likePattern := "%" + keyword + "%"
	query := r.db.WithContext(ctx).
		Where("summary LIKE ? OR what_happened LIKE ? OR conclusion LIKE ? OR value LIKE ? OR highlights LIKE ?",
			likePattern, likePattern, likePattern, likePattern, likePattern)

	// 排序
	orderBy := opts.OrderBy
	if orderBy == "" {
		orderBy = "date"
	}
	order := opts.Order
	if order == "" {
		order = "DESC"
	}
	query = query.Order(orderBy + " " + order)

	// 分页
	if opts.Limit > 0 {
		query = query.Limit(opts.Limit)
	}
	if opts.Offset > 0 {
		query = query.Offset(opts.Offset)
	}

	var memories []models.LongTermMemory
	if err := query.Find(&memories).Error; err != nil {
		return nil, fmt.Errorf("failed to search long term memories by keyword: %w", err)
	}
	return memories, nil
}

// Update 更新长期记忆
func (r *longTermMemoryRepository) Update(ctx context.Context, memory *models.LongTermMemory) error {
	memory.UpdatedAt = time.Now()
	if err := r.db.WithContext(ctx).Save(memory).Error; err != nil {
		return fmt.Errorf("failed to update long term memory: %w", err)
	}
	return nil
}

// DeleteByDate 根据日期删除长期记忆
func (r *longTermMemoryRepository) DeleteByDate(ctx context.Context, date string) error {
	if err := r.db.WithContext(ctx).Where("date = ?", date).Delete(&models.LongTermMemory{}).Error; err != nil {
		return fmt.Errorf("failed to delete long term memory by date: %w", err)
	}
	return nil
}
