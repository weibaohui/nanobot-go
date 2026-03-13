package conversation

import (
	"context"
	"errors"
	"time"

	"github.com/weibaohui/nanobot-go/internal/models"
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

	// 归属信息（用于多租户、多 Agent 架构）
	UserID      *uint  `json:"user_id,omitempty"`      // 用户 ID
	AgentID     *uint  `json:"agent_id,omitempty"`     // Agent ID
	ChannelID   *uint  `json:"channel_id,omitempty"`   // Channel ID
	ChannelType string `json:"channel_type,omitempty"` // 渠道类型
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

// Service 对话服务接口
type Service interface {
	GetByTraceID(ctx context.Context, traceID string) ([]ConversationDTO, error)
	ListBySessionKey(ctx context.Context, sessionKey string, page, pageSize int) (*ConversationListResult, error)
	ListByTimeRange(ctx context.Context, startTime, endTime time.Time, page, pageSize int) (*ConversationListResult, error)
	ListRecent(ctx context.Context, page, pageSize int) (*ConversationListResult, error)
	Create(ctx context.Context, dto *ConversationDTO) error
	CreateBatch(ctx context.Context, dtos []ConversationDTO) error
}

// Repository 对话记录仓储接口
type Repository interface {
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
