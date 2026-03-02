package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"github.com/weibaohui/nanobot-go/memory/models"
)

// StreamMemoryRepository 流水记忆仓储接口
type StreamMemoryRepository interface {
	// Create 创建流水记忆
	Create(ctx context.Context, memory *models.StreamMemory) error
	// CreateBatch 批量创建
	CreateBatch(ctx context.Context, memories []models.StreamMemory) error
	// FindByID 根据ID查询
	FindByID(ctx context.Context, id uint64) (*models.StreamMemory, error)
	// FindByTraceID 根据TraceID查询
	FindByTraceID(ctx context.Context, traceID string) ([]models.StreamMemory, error)
	// FindBySessionKey 根据SessionKey查询（支持分页）
	FindBySessionKey(ctx context.Context, sessionKey string, opts *models.QueryOptions) ([]models.StreamMemory, error)
	// FindByTimeRange 根据时间范围查询
	FindByTimeRange(ctx context.Context, startTime, endTime time.Time, opts *models.QueryOptions) ([]models.StreamMemory, error)
	// FindUnprocessed 查询未处理的流水记忆（用于定时升级）
	FindUnprocessed(ctx context.Context, before time.Time, limit int) ([]models.StreamMemory, error)
	// MarkAsProcessed 标记为已处理
	MarkAsProcessed(ctx context.Context, ids []uint64) error
	// CountByTimeRange 统计时间范围内的数量
	CountByTimeRange(ctx context.Context, startTime, endTime time.Time) (int64, error)
	// CountUnprocessed 统计未处理的数量
	CountUnprocessed(ctx context.Context, before time.Time) (int64, error)
}

// streamMemoryRepository 流水记忆仓储实现
type streamMemoryRepository struct {
	db *gorm.DB
}

// NewStreamMemoryRepository 创建流水记忆仓储实例
func NewStreamMemoryRepository(db *gorm.DB) StreamMemoryRepository {
	return &streamMemoryRepository{db: db}
}

// Create 创建流水记忆
func (r *streamMemoryRepository) Create(ctx context.Context, memory *models.StreamMemory) error {
	if err := r.db.WithContext(ctx).Create(memory).Error; err != nil {
		return fmt.Errorf("failed to create stream memory: %w", err)
	}
	return nil
}

// CreateBatch 批量创建流水记忆
func (r *streamMemoryRepository) CreateBatch(ctx context.Context, memories []models.StreamMemory) error {
	if len(memories) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).CreateInBatches(memories, 100).Error; err != nil {
		return fmt.Errorf("failed to create batch stream memories: %w", err)
	}
	return nil
}

// FindByID 根据ID查询流水记忆
func (r *streamMemoryRepository) FindByID(ctx context.Context, id uint64) (*models.StreamMemory, error) {
	var memory models.StreamMemory
	if err := r.db.WithContext(ctx).First(&memory, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find stream memory by id: %w", err)
	}
	return &memory, nil
}

// FindByTraceID 根据TraceID查询流水记忆
func (r *streamMemoryRepository) FindByTraceID(ctx context.Context, traceID string) ([]models.StreamMemory, error) {
	var memories []models.StreamMemory
	if err := r.db.WithContext(ctx).
		Where("trace_id = ?", traceID).
		Order("created_at ASC").
		Find(&memories).Error; err != nil {
		return nil, fmt.Errorf("failed to find stream memories by trace_id: %w", err)
	}
	return memories, nil
}

// FindBySessionKey 根据SessionKey查询流水记忆
func (r *streamMemoryRepository) FindBySessionKey(ctx context.Context, sessionKey string, opts *models.QueryOptions) ([]models.StreamMemory, error) {
	if opts == nil {
		opts = &models.QueryOptions{}
	}

	query := r.db.WithContext(ctx).Where("session_key = ?", sessionKey)

	// 排序
	orderBy := opts.OrderBy
	if orderBy == "" {
		orderBy = "created_at"
	}
	order := opts.Order
	if order == "" {
		order = "ASC"
	}
	query = query.Order(orderBy + " " + order)

	// 分页
	if opts.Limit > 0 {
		query = query.Limit(opts.Limit)
	}
	if opts.Offset > 0 {
		query = query.Offset(opts.Offset)
	}

	var memories []models.StreamMemory
	if err := query.Find(&memories).Error; err != nil {
		return nil, fmt.Errorf("failed to find stream memories by session_key: %w", err)
	}
	return memories, nil
}

// FindByTimeRange 根据时间范围查询流水记忆
func (r *streamMemoryRepository) FindByTimeRange(ctx context.Context, startTime, endTime time.Time, opts *models.QueryOptions) ([]models.StreamMemory, error) {
	if opts == nil {
		opts = &models.QueryOptions{}
	}

	query := r.db.WithContext(ctx).
		Where("created_at >= ?", startTime).
		Where("created_at <= ?", endTime)

	// 排序
	orderBy := opts.OrderBy
	if orderBy == "" {
		orderBy = "created_at"
	}
	order := opts.Order
	if order == "" {
		order = "ASC"
	}
	query = query.Order(orderBy + " " + order)

	// 分页
	if opts.Limit > 0 {
		query = query.Limit(opts.Limit)
	}
	if opts.Offset > 0 {
		query = query.Offset(opts.Offset)
	}

	var memories []models.StreamMemory
	if err := query.Find(&memories).Error; err != nil {
		return nil, fmt.Errorf("failed to find stream memories by time range: %w", err)
	}
	return memories, nil
}

// FindUnprocessed 查询未处理的流水记忆
func (r *streamMemoryRepository) FindUnprocessed(ctx context.Context, before time.Time, limit int) ([]models.StreamMemory, error) {
	if limit <= 0 {
		limit = 100
	}

	var memories []models.StreamMemory
	if err := r.db.WithContext(ctx).
		Where("processed = ?", false).
		Where("created_at < ?", before).
		Order("created_at ASC").
		Limit(limit).
		Find(&memories).Error; err != nil {
		return nil, fmt.Errorf("failed to find unprocessed stream memories: %w", err)
	}
	return memories, nil
}

// MarkAsProcessed 标记流水记忆为已处理
func (r *streamMemoryRepository) MarkAsProcessed(ctx context.Context, ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}

	now := time.Now()
	if err := r.db.WithContext(ctx).
		Model(&models.StreamMemory{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"processed":    true,
			"processed_at": now,
		}).Error; err != nil {
		return fmt.Errorf("failed to mark stream memories as processed: %w", err)
	}
	return nil
}

// CountByTimeRange 统计时间范围内的流水记忆数量
func (r *streamMemoryRepository) CountByTimeRange(ctx context.Context, startTime, endTime time.Time) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.StreamMemory{}).
		Where("created_at >= ?", startTime).
		Where("created_at <= ?", endTime).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count stream memories by time range: %w", err)
	}
	return count, nil
}

// CountUnprocessed 统计未处理的流水记忆数量
func (r *streamMemoryRepository) CountUnprocessed(ctx context.Context, before time.Time) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.StreamMemory{}).
		Where("processed = ?", false).
		Where("created_at < ?", before).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count unprocessed stream memories: %w", err)
	}
	return count, nil
}
