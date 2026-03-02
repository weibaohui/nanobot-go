package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/weibaohui/nanobot-go/agent/models"
	"github.com/weibaohui/nanobot-go/agent/repository"
)

var (
	// ErrRecordNotFound 记录不存在
	ErrRecordNotFound = errors.New("record not found")

	// ErrInvalidParameter 无效参数
	ErrInvalidParameter = errors.New("invalid parameter")

	// ErrDatabaseOperation 数据库操作错误
	ErrDatabaseOperation = errors.New("database operation failed")
)

// ConversationDTO 对话数据传输对象
// 用于外部模块调用的统一数据结构
type ConversationDTO struct {
	ID               uint              `json:"id"`
	TraceID          string            `json:"trace_id"`
	SpanID           string            `json:"span_id,omitempty"`
	ParentSpanID     string            `json:"parent_span_id,omitempty"`
	EventType        string            `json:"event_type"`
	Timestamp        time.Time         `json:"timestamp"`
	SessionKey       string            `json:"session_key"`
	Role             string            `json:"role"`
	Content          string            `json:"content"`
	TokenUsage       *TokenUsageDTO    `json:"token_usage,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
}

// TokenUsageDTO Token 使用信息
type TokenUsageDTO struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	ReasoningTokens  int `json:"reasoning_tokens,omitempty"`
	CachedTokens     int `json:"cached_tokens,omitempty"`
}

// ConversationListResult 对话列表查询结果
type ConversationListResult struct {
	Conversations []ConversationDTO `json:"conversations"`
	Total         int64            `json:"total"`
	Page          int              `json:"page"`
	PageSize      int              `json:"page_size"`
}

// ConversationService 对话服务接口
// 提供统一的对话记录访问接口，屏蔽底层数据库细节
type ConversationService interface {
	// GetByTraceID 根据 TraceID 获取完整对话
	GetByTraceID(ctx context.Context, traceID string) ([]ConversationDTO, error)

	// ListBySessionKey 根据会话 ID 获取对话列表（支持分页）
	ListBySessionKey(ctx context.Context, sessionKey string, page, pageSize int) (*ConversationListResult, error)

	// ListByTimeRange 根据时间范围获取对话列表（支持分页）
	ListByTimeRange(ctx context.Context, startTime, endTime time.Time, page, pageSize int) (*ConversationListResult, error)

	// ListRecent 获取最近的对话列表（支持分页）
	ListRecent(ctx context.Context, page, pageSize int) (*ConversationListResult, error)

	// Create 创建新对话记录
	Create(ctx context.Context, dto *ConversationDTO) error

	// CreateBatch 批量创建对话记录
	CreateBatch(ctx context.Context, dtos []ConversationDTO) error
}

// conversationService ConversationService 的实现
type conversationService struct {
	repo repository.EventRepository
}

// NewConversationService 创建 ConversationService 实例
func NewConversationService(repo repository.EventRepository) ConversationService {
	return &conversationService{repo: repo}
}

// GetByTraceID 根据 TraceID 获取完整对话
func (s *conversationService) GetByTraceID(ctx context.Context, traceID string) ([]ConversationDTO, error) {
	if traceID == "" {
		return nil, fmt.Errorf("%w: traceID cannot be empty", ErrInvalidParameter)
	}

	events, err := s.repo.FindByTraceID(ctx, traceID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	return s.eventsToDTOs(events), nil
}

// ListBySessionKey 根据会话 ID 获取对话列表（支持分页）
func (s *conversationService) ListBySessionKey(ctx context.Context, sessionKey string, page, pageSize int) (*ConversationListResult, error) {
	if sessionKey == "" {
		return nil, fmt.Errorf("%w: sessionKey cannot be empty", ErrInvalidParameter)
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	// 限制最大分页大小
	if pageSize > 1000 {
		pageSize = 1000
	}

	// 获取总数
	total, err := s.repo.CountBySessionKey(ctx, sessionKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 查询数据
	events, err := s.repo.FindBySessionKey(ctx, sessionKey, &repository.QueryOptions{
		OrderBy: "timestamp",
		Order:   "ASC",
		Limit:   pageSize,
		Offset:  offset,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	return &ConversationListResult{
		Conversations: s.eventsToDTOs(events),
		Total:         total,
		Page:          page,
		PageSize:      pageSize,
	}, nil
}

// ListByTimeRange 根据时间范围获取对话列表（支持分页）
func (s *conversationService) ListByTimeRange(ctx context.Context, startTime, endTime time.Time, page, pageSize int) (*ConversationListResult, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	// 限制最大分页大小
	if pageSize > 1000 {
		pageSize = 1000
	}

	// 获取总数
	total, err := s.repo.CountByTimeRange(ctx, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 查询数据
	events, err := s.repo.FindByTimeRange(ctx, startTime, endTime, &repository.QueryOptions{
		OrderBy: "timestamp",
		Order:   "ASC",
		Limit:   pageSize,
		Offset:  offset,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	return &ConversationListResult{
		Conversations: s.eventsToDTOs(events),
		Total:         total,
		Page:          page,
		PageSize:      pageSize,
	}, nil
}

// ListRecent 获取最近的对话列表（支持分页）
func (s *conversationService) ListRecent(ctx context.Context, page, pageSize int) (*ConversationListResult, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	// 限制最大分页大小
	if pageSize > 1000 {
		pageSize = 1000
	}

	// 获取总数
	total, err := s.repo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 查询数据（使用空时间范围获取所有数据）
	events, err := s.repo.FindByTimeRange(ctx, time.Time{}, time.Now(), &repository.QueryOptions{
		OrderBy: "timestamp",
		Order:   "DESC", // 按时间倒序，获取最近的
		Limit:   pageSize,
		Offset:  offset,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	// 按时间正序返回（保持对话顺序）
	result := s.eventsToDTOs(events)
	// 反转切片
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return &ConversationListResult{
		Conversations: result,
		Total:         total,
		Page:          page,
		PageSize:      pageSize,
	}, nil
}

// Create 创建新对话记录
func (s *conversationService) Create(ctx context.Context, dto *ConversationDTO) error {
	if dto == nil {
		return fmt.Errorf("%w: dto cannot be nil", ErrInvalidParameter)
	}

	event := s.dtoToEvent(dto)
	if err := s.repo.Create(ctx, &event); err != nil {
		return fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	// 回填 ID
	dto.ID = event.ID
	return nil
}

// CreateBatch 批量创建对话记录
func (s *conversationService) CreateBatch(ctx context.Context, dtos []ConversationDTO) error {
	if len(dtos) == 0 {
		return nil
	}

	events := make([]models.Event, 0, len(dtos))
	for _, dto := range dtos {
		events = append(events, s.dtoToEvent(&dto))
	}

	if err := s.repo.CreateBatch(ctx, events); err != nil {
		return fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	return nil
}

// eventsToDTOs 将 Event 列表转换为 DTO 列表
func (s *conversationService) eventsToDTOs(events []models.Event) []ConversationDTO {
	dtos := make([]ConversationDTO, 0, len(events))
	for _, event := range events {
		dtos = append(dtos, s.eventToDTO(&event))
	}
	return dtos
}

// eventToDTO 将 Event 转换为 DTO
func (s *conversationService) eventToDTO(event *models.Event) ConversationDTO {
	dto := ConversationDTO{
		ID:           event.ID,
		TraceID:      event.TraceID,
		SpanID:       event.SpanID,
		ParentSpanID: event.ParentSpanID,
		EventType:    event.EventType,
		Timestamp:    event.Timestamp,
		SessionKey:   event.SessionKey,
		Role:         event.Role,
		Content:      event.Content,
		CreatedAt:    event.CreatedAt,
	}

	// 只有当有 Token 使用信息时才添加
	if event.TotalTokens > 0 || event.PromptTokens > 0 || event.CompletionTokens > 0 {
		dto.TokenUsage = &TokenUsageDTO{
			PromptTokens:     event.PromptTokens,
			CompletionTokens: event.CompletionTokens,
			TotalTokens:      event.TotalTokens,
			ReasoningTokens:  event.ReasoningTokens,
			CachedTokens:     event.CachedTokens,
		}
	}

	return dto
}

// dtoToEvent 将 DTO 转换为 Event
func (s *conversationService) dtoToEvent(dto *ConversationDTO) models.Event {
	event := models.Event{
		ID:               dto.ID,
		TraceID:          dto.TraceID,
		SpanID:           dto.SpanID,
		ParentSpanID:     dto.ParentSpanID,
		EventType:        dto.EventType,
		Timestamp:        dto.Timestamp,
		SessionKey:       dto.SessionKey,
		Role:             dto.Role,
		Content:          dto.Content,
		CreatedAt:        dto.CreatedAt,
	}

	if dto.TokenUsage != nil {
		event.PromptTokens = dto.TokenUsage.PromptTokens
		event.CompletionTokens = dto.TokenUsage.CompletionTokens
		event.TotalTokens = dto.TokenUsage.TotalTokens
		event.ReasoningTokens = dto.TokenUsage.ReasoningTokens
		event.CachedTokens = dto.TokenUsage.CachedTokens
	}

	return event
}