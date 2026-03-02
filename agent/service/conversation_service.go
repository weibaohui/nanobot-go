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
	ErrRecordNotFound    = errors.New("record not found")
	ErrInvalidParameter  = errors.New("invalid parameter")
	ErrDatabaseOperation = errors.New("database operation failed")
)

// ConversationDTO 对话数据传输对象
type ConversationDTO struct {
	ID           uint           `json:"id"`
	TraceID      string         `json:"trace_id"`
	SpanID       string         `json:"span_id,omitempty"`
	ParentSpanID string         `json:"parent_span_id,omitempty"`
	EventType    string         `json:"event_type"`
	Timestamp    time.Time      `json:"timestamp"`
	SessionKey   string         `json:"session_key"`
	Role         string         `json:"role"`
	Content      string         `json:"content"`
	TokenUsage   *TokenUsageDTO `json:"token_usage,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
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
	Total         int64             `json:"total"`
	Page          int               `json:"page"`
	PageSize      int               `json:"page_size"`
}

// ConversationService 对话服务接口
type ConversationService interface {
	GetByTraceID(ctx context.Context, traceID string) ([]ConversationDTO, error)
	ListBySessionKey(ctx context.Context, sessionKey string, page, pageSize int) (*ConversationListResult, error)
	ListByTimeRange(ctx context.Context, startTime, endTime time.Time, page, pageSize int) (*ConversationListResult, error)
	ListRecent(ctx context.Context, page, pageSize int) (*ConversationListResult, error)
	Create(ctx context.Context, dto *ConversationDTO) error
	CreateBatch(ctx context.Context, dtos []ConversationDTO) error
}

type conversationService struct {
	repo repository.ConversationRecordRepository
}

// NewConversationService 创建服务实例
func NewConversationService(repo repository.ConversationRecordRepository) ConversationService {
	return &conversationService{repo: repo}
}

func (s *conversationService) GetByTraceID(ctx context.Context, traceID string) ([]ConversationDTO, error) {
	if traceID == "" {
		return nil, fmt.Errorf("%w: traceID cannot be empty", ErrInvalidParameter)
	}

	records, err := s.repo.FindByTraceID(ctx, traceID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	return s.recordsToDTOs(records), nil
}

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
	if pageSize > 1000 {
		pageSize = 1000
	}

	total, err := s.repo.CountBySessionKey(ctx, sessionKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	offset := (page - 1) * pageSize
	records, err := s.repo.FindBySessionKey(ctx, sessionKey, &repository.QueryOptions{
		OrderBy: "timestamp",
		Order:   "ASC",
		Limit:   pageSize,
		Offset:  offset,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	return &ConversationListResult{
		Conversations: s.recordsToDTOs(records),
		Total:         total,
		Page:          page,
		PageSize:      pageSize,
	}, nil
}

func (s *conversationService) ListByTimeRange(ctx context.Context, startTime, endTime time.Time, page, pageSize int) (*ConversationListResult, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 1000 {
		pageSize = 1000
	}

	total, err := s.repo.CountByTimeRange(ctx, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	offset := (page - 1) * pageSize
	records, err := s.repo.FindByTimeRange(ctx, startTime, endTime, &repository.QueryOptions{
		OrderBy: "timestamp",
		Order:   "ASC",
		Limit:   pageSize,
		Offset:  offset,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	return &ConversationListResult{
		Conversations: s.recordsToDTOs(records),
		Total:         total,
		Page:          page,
		PageSize:      pageSize,
	}, nil
}

func (s *conversationService) ListRecent(ctx context.Context, page, pageSize int) (*ConversationListResult, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 1000 {
		pageSize = 1000
	}

	total, err := s.repo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	offset := (page - 1) * pageSize
	records, err := s.repo.FindByTimeRange(ctx, time.Time{}, time.Now(), &repository.QueryOptions{
		OrderBy: "timestamp",
		Order:   "DESC",
		Limit:   pageSize,
		Offset:  offset,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	result := s.recordsToDTOs(records)
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

func (s *conversationService) Create(ctx context.Context, dto *ConversationDTO) error {
	if dto == nil {
		return fmt.Errorf("%w: dto cannot be nil", ErrInvalidParameter)
	}

	record := s.dtoToRecord(dto)
	if err := s.repo.Create(ctx, &record); err != nil {
		return fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	dto.ID = record.ID
	return nil
}

func (s *conversationService) CreateBatch(ctx context.Context, dtos []ConversationDTO) error {
	if len(dtos) == 0 {
		return nil
	}

	records := make([]models.ConversationRecord, 0, len(dtos))
	for _, dto := range dtos {
		records = append(records, s.dtoToRecord(&dto))
	}

	if err := s.repo.CreateBatch(ctx, records); err != nil {
		return fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	return nil
}

func (s *conversationService) recordsToDTOs(records []models.ConversationRecord) []ConversationDTO {
	dtos := make([]ConversationDTO, 0, len(records))
	for _, record := range records {
		dtos = append(dtos, s.recordToDTO(&record))
	}
	return dtos
}

func (s *conversationService) recordToDTO(record *models.ConversationRecord) ConversationDTO {
	dto := ConversationDTO{
		ID:           record.ID,
		TraceID:      record.TraceID,
		SpanID:       record.SpanID,
		ParentSpanID: record.ParentSpanID,
		EventType:    record.EventType,
		Timestamp:    record.Timestamp,
		SessionKey:   record.SessionKey,
		Role:         record.Role,
		Content:      record.Content,
		CreatedAt:    record.CreatedAt,
	}

	if record.TotalTokens > 0 || record.PromptTokens > 0 || record.CompletionTokens > 0 {
		dto.TokenUsage = &TokenUsageDTO{
			PromptTokens:     record.PromptTokens,
			CompletionTokens: record.CompletionTokens,
			TotalTokens:      record.TotalTokens,
			ReasoningTokens:  record.ReasoningTokens,
			CachedTokens:     record.CachedTokens,
		}
	}

	return dto
}

func (s *conversationService) dtoToRecord(dto *ConversationDTO) models.ConversationRecord {
	record := models.ConversationRecord{
		ID:           dto.ID,
		TraceID:      dto.TraceID,
		SpanID:       dto.SpanID,
		ParentSpanID: dto.ParentSpanID,
		EventType:    dto.EventType,
		Timestamp:    dto.Timestamp,
		SessionKey:   dto.SessionKey,
		Role:         dto.Role,
		Content:      dto.Content,
		CreatedAt:    dto.CreatedAt,
	}

	if dto.TokenUsage != nil {
		record.PromptTokens = dto.TokenUsage.PromptTokens
		record.CompletionTokens = dto.TokenUsage.CompletionTokens
		record.TotalTokens = dto.TokenUsage.TotalTokens
		record.ReasoningTokens = dto.TokenUsage.ReasoningTokens
		record.CachedTokens = dto.TokenUsage.CachedTokens
	}

	return record
}
