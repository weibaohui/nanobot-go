package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/weibaohui/nanobot-go/agent/models"
)

// EventRepository 事件仓储接口
// 定义所有事件数据访问操作，屏蔽底层数据库细节
type EventRepository interface {
	// FindByID 根据 ID 查找事件
	FindByID(ctx context.Context, id uint) (*models.Event, error)

	// FindByTraceID 根据 TraceID 查找事件（可能有多条，因为一个 trace 可能包含多个事件）
	FindByTraceID(ctx context.Context, traceID string) ([]models.Event, error)

	// FindBySessionKey 根据 SessionKey 查找事件列表
	FindBySessionKey(ctx context.Context, sessionKey string, opts *QueryOptions) ([]models.Event, error)

	// FindByTimeRange 根据时间范围查找事件列表
	FindByTimeRange(ctx context.Context, startTime, endTime time.Time, opts *QueryOptions) ([]models.Event, error)

	// FindByTraceIDRoleAndContent 根据 TraceID、Role 和 Content 查找事件（用于去重）
	FindByTraceIDRoleAndContent(ctx context.Context, traceID, role, content string) ([]models.Event, error)

	// CountBySessionKey 统计 SessionKey 下的事件数量
	CountBySessionKey(ctx context.Context, sessionKey string) (int64, error)

	// CountByTimeRange 统计指定时间范围内的事件数量
	CountByTimeRange(ctx context.Context, startTime, endTime time.Time) (int64, error)

	// Count 统计事件总数
	Count(ctx context.Context) (int64, error)

	// Create 创建事件
	Create(ctx context.Context, event *models.Event) error

	// CreateBatch 批量创建事件
	CreateBatch(ctx context.Context, events []models.Event) error

	// DeleteByID 根据 ID 删除事件
	DeleteByID(ctx context.Context, id uint) error
}

// QueryOptions 查询选项
type QueryOptions struct {
	OrderBy string   // 排序字段，默认 "timestamp"
	Order   string   // 排序方向，ASC 或 DESC，默认 "DESC"
	Limit   int      // 限制数量
	Offset  int      // 偏移量
	Roles   []string // 筛选角色
}

// eventRepository EventRepository 的 GORM 实现
type eventRepository struct {
	db *gorm.DB
}

// NewEventRepository 创建 EventRepository 实例
func NewEventRepository(db *gorm.DB) EventRepository {
	return &eventRepository{db: db}
}

// FindByID 根据 ID 查找事件
func (r *eventRepository) FindByID(ctx context.Context, id uint) (*models.Event, error) {
	var event models.Event
	if err := r.db.WithContext(ctx).First(&event, id).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

// FindByTraceID 根据 TraceID 查找事件
func (r *eventRepository) FindByTraceID(ctx context.Context, traceID string) ([]models.Event, error) {
	var events []models.Event
	if err := r.db.WithContext(ctx).
		Where("trace_id = ?", traceID).
		Order("timestamp ASC").
		Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

// FindBySessionKey 根据 SessionKey 查找事件列表
func (r *eventRepository) FindBySessionKey(ctx context.Context, sessionKey string, opts *QueryOptions) ([]models.Event, error) {
	query := r.db.WithContext(ctx).Where("session_key = ?", sessionKey)

	if opts != nil {
		// 应用角色筛选
		if len(opts.Roles) > 0 {
			query = query.Where("role IN ?", opts.Roles)
		}

		// 应用排序
		orderBy := "timestamp"
		if opts.OrderBy != "" {
			orderBy = opts.OrderBy
		}
		order := "ASC"
		if opts.Order != "" {
			order = opts.Order
		}
		query = query.Order(orderBy + " " + order)

		// 应用分页
		if opts.Offset > 0 {
			query = query.Offset(opts.Offset)
		}
		if opts.Limit > 0 {
			query = query.Limit(opts.Limit)
		}
	} else {
		// 默认按时间正序排列
		query = query.Order("timestamp ASC")
	}

	var events []models.Event
	if err := query.Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

// FindByTimeRange 根据时间范围查找事件列表
func (r *eventRepository) FindByTimeRange(ctx context.Context, startTime, endTime time.Time, opts *QueryOptions) ([]models.Event, error) {
	query := r.db.WithContext(ctx).
		Where("timestamp >= ?", startTime).
		Where("timestamp <= ?", endTime)

	if opts != nil {
		// 应用角色筛选
		if len(opts.Roles) > 0 {
			query = query.Where("role IN ?", opts.Roles)
		}

		// 应用排序
		orderBy := "timestamp"
		if opts.OrderBy != "" {
			orderBy = opts.OrderBy
		}
		order := "ASC"
		if opts.Order != "" {
			order = opts.Order
		}
		query = query.Order(orderBy + " " + order)

		// 应用分页
		if opts.Offset > 0 {
			query = query.Offset(opts.Offset)
		}
		if opts.Limit > 0 {
			query = query.Limit(opts.Limit)
		}
	} else {
		// 默认按时间正序排列
		query = query.Order("timestamp ASC")
	}

	var events []models.Event
	if err := query.Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

// CountBySessionKey 统计 SessionKey 下的事件数量
func (r *eventRepository) CountBySessionKey(ctx context.Context, sessionKey string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.Event{}).
		Where("session_key = ?", sessionKey).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByTimeRange 统计指定时间范围内的事件数量
func (r *eventRepository) CountByTimeRange(ctx context.Context, startTime, endTime time.Time) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.Event{}).
		Where("timestamp >= ?", startTime).
		Where("timestamp <= ?", endTime).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// Count 统计事件总数
func (r *eventRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.Event{}).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// Create 创建事件
func (r *eventRepository) Create(ctx context.Context, event *models.Event) error {
	return r.db.WithContext(ctx).Create(event).Error
}

// CreateBatch 批量创建事件
func (r *eventRepository) CreateBatch(ctx context.Context, events []models.Event) error {
	if len(events) == 0 {
		return nil
	}
	// 使用批量插入，每批最多 100 条
	return r.db.WithContext(ctx).CreateInBatches(events, 100).Error
}

// FindByTraceIDRoleAndContent 根据 TraceID、Role 和 Content 查找事件（用于去重）
func (r *eventRepository) FindByTraceIDRoleAndContent(ctx context.Context, traceID, role, content string) ([]models.Event, error) {
	var events []models.Event
	if err := r.db.WithContext(ctx).
		Where("trace_id = ? AND role = ? AND content = ?", traceID, role, content).
		Order("id ASC").
		Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

// DeleteByID 根据 ID 删除事件
func (r *eventRepository) DeleteByID(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Event{}, id).Error
}