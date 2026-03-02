package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/weibaohui/nanobot-go/internal/models"
)

// ConversationRecordRepository 对话记录仓储接口
type ConversationRecordRepository interface {
	FindByID(ctx context.Context, id uint) (*models.ConversationRecord, error)
	FindByTraceID(ctx context.Context, traceID string) ([]models.ConversationRecord, error)
	FindBySessionKey(ctx context.Context, sessionKey string, opts *models.QueryOptions) ([]models.ConversationRecord, error)
	FindByTimeRange(ctx context.Context, startTime, endTime time.Time, opts *models.QueryOptions) ([]models.ConversationRecord, error)
	FindByTraceIDRoleAndContent(ctx context.Context, traceID, role, content string) ([]models.ConversationRecord, error)
	CountBySessionKey(ctx context.Context, sessionKey string) (int64, error)
	CountByTimeRange(ctx context.Context, startTime, endTime time.Time) (int64, error)
	Count(ctx context.Context) (int64, error)
	Create(ctx context.Context, record *models.ConversationRecord) error
	CreateBatch(ctx context.Context, records []models.ConversationRecord) error
	DeleteByID(ctx context.Context, id uint) error
}

type conversationRecordRepository struct {
	db *gorm.DB
}

// NewConversationRecordRepository 创建仓储实例
func NewConversationRecordRepository(db *gorm.DB) ConversationRecordRepository {
	return &conversationRecordRepository{db: db}
}

func (r *conversationRecordRepository) FindByID(ctx context.Context, id uint) (*models.ConversationRecord, error) {
	var record models.ConversationRecord
	if err := r.db.WithContext(ctx).First(&record, id).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *conversationRecordRepository) FindByTraceID(ctx context.Context, traceID string) ([]models.ConversationRecord, error) {
	var records []models.ConversationRecord
	if err := r.db.WithContext(ctx).
		Where("trace_id = ?", traceID).
		Order("timestamp ASC").
		Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (r *conversationRecordRepository) FindBySessionKey(ctx context.Context, sessionKey string, opts *models.QueryOptions) ([]models.ConversationRecord, error) {
	query := r.db.WithContext(ctx).Where("session_key = ?", sessionKey)

	if opts != nil {
		if len(opts.Roles) > 0 {
			query = query.Where("role IN ?", opts.Roles)
		}
		orderBy := "timestamp"
		if opts.OrderBy != "" {
			orderBy = opts.OrderBy
		}
		order := "ASC"
		if opts.Order != "" {
			order = opts.Order
		}
		query = query.Order(orderBy + " " + order)
		if opts.Offset > 0 {
			query = query.Offset(opts.Offset)
		}
		if opts.Limit > 0 {
			query = query.Limit(opts.Limit)
		}
	} else {
		query = query.Order("timestamp ASC")
	}

	var records []models.ConversationRecord
	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (r *conversationRecordRepository) FindByTimeRange(ctx context.Context, startTime, endTime time.Time, opts *models.QueryOptions) ([]models.ConversationRecord, error) {
	query := r.db.WithContext(ctx).
		Where("timestamp >= ?", startTime).
		Where("timestamp <= ?", endTime)

	if opts != nil {
		if len(opts.Roles) > 0 {
			query = query.Where("role IN ?", opts.Roles)
		}
		orderBy := "timestamp"
		if opts.OrderBy != "" {
			orderBy = opts.OrderBy
		}
		order := "ASC"
		if opts.Order != "" {
			order = opts.Order
		}
		query = query.Order(orderBy + " " + order)
		if opts.Offset > 0 {
			query = query.Offset(opts.Offset)
		}
		if opts.Limit > 0 {
			query = query.Limit(opts.Limit)
		}
	} else {
		query = query.Order("timestamp ASC")
	}

	var records []models.ConversationRecord
	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (r *conversationRecordRepository) FindByTraceIDRoleAndContent(ctx context.Context, traceID, role, content string) ([]models.ConversationRecord, error) {
	var records []models.ConversationRecord
	if err := r.db.WithContext(ctx).
		Where("trace_id = ? AND role = ? AND content = ?", traceID, role, content).
		Order("id ASC").
		Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (r *conversationRecordRepository) CountBySessionKey(ctx context.Context, sessionKey string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.ConversationRecord{}).
		Where("session_key = ?", sessionKey).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *conversationRecordRepository) CountByTimeRange(ctx context.Context, startTime, endTime time.Time) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.ConversationRecord{}).
		Where("timestamp >= ?", startTime).
		Where("timestamp <= ?", endTime).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *conversationRecordRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.ConversationRecord{}).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *conversationRecordRepository) Create(ctx context.Context, record *models.ConversationRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

func (r *conversationRecordRepository) CreateBatch(ctx context.Context, records []models.ConversationRecord) error {
	if len(records) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(records, 100).Error
}

func (r *conversationRecordRepository) DeleteByID(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.ConversationRecord{}, id).Error
}
