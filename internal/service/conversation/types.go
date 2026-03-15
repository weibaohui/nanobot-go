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

	// 归属信息（用于多租户、多 Agent 架构，使用 Code 进行关联）
	UserCode    string `json:"user_code,omitempty"`    // 用户 Code
	AgentCode   string `json:"agent_code,omitempty"`   // Agent Code
	ChannelCode string `json:"channel_code,omitempty"` // Channel Code
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
	ListByUserAndDate(ctx context.Context, userCode string, date string) ([]ConversationDTO, error)
	Create(ctx context.Context, dto *ConversationDTO) error
	CreateBatch(ctx context.Context, dtos []ConversationDTO) error
	GetStats(ctx context.Context, req *StatsRequest) (*StatsResponse, error)
}

// StatsRequest 统计请求参数
type StatsRequest struct {
	StartTime    time.Time
	EndTime      time.Time
	AgentCodes   []string
	ChannelCodes []string
	Roles        []string
}

// StatsResponse 统计响应
type StatsResponse struct {
	TokenStats          TokenStats          `json:"token_stats"`
	AgentDistribution   []AgentDistribution `json:"agent_distribution"`
	ChannelDistribution []ChannelDistribution `json:"channel_distribution"`
	RoleDistribution    []RoleDistribution  `json:"role_distribution"`
	SessionStats        SessionStats        `json:"session_stats"`
}

// TokenStats Token 统计
type TokenStats struct {
	TotalPromptTokens     int64       `json:"total_prompt_tokens"`
	TotalCompletionTokens int64       `json:"total_completion_tokens"`
	TotalTokens           int64       `json:"total_tokens"`
	DailyTrends           []DailyStat `json:"daily_trends"`
}

// DailyStat 每日统计
type DailyStat struct {
	Date           string `json:"date"`
	PromptTokens   int64  `json:"prompt_tokens"`
	CompleteTokens int64  `json:"complete_tokens"`
	TotalTokens    int64  `json:"total_tokens"`
}

// AgentDistribution Agent 分布
type AgentDistribution struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Count  int64  `json:"count"`
	Tokens int64  `json:"tokens"`
}

// ChannelDistribution Channel 分布
type ChannelDistribution struct {
	Type  string `json:"type"`
	Count int64  `json:"count"`
}

// RoleDistribution 角色分布
type RoleDistribution struct {
	Role  string `json:"role"`
	Count int64  `json:"count"`
}

// SessionStats 会话统计
type SessionStats struct {
	TotalSessions   int64   `json:"total_sessions"`
	AvgMessages     float64 `json:"avg_messages_per_session"`
	AvgResponseTime float64 `json:"avg_response_time_ms"`
}

// Repository 对话记录仓储接口
type Repository interface {
	FindByID(ctx context.Context, id uint) (*models.ConversationRecord, error)
	FindByTraceID(ctx context.Context, traceID string) ([]models.ConversationRecord, error)
	FindBySessionKey(ctx context.Context, sessionKey string, opts *models.QueryOptions) ([]models.ConversationRecord, error)
	FindByTimeRange(ctx context.Context, startTime, endTime time.Time, opts *models.QueryOptions) ([]models.ConversationRecord, error)
	FindByUserCodeAndDate(ctx context.Context, userCode string, startTime, endTime time.Time) ([]models.ConversationRecord, error)
	FindByTraceIDRoleAndContent(ctx context.Context, traceID, role, content string) ([]models.ConversationRecord, error)
	CountBySessionKey(ctx context.Context, sessionKey string) (int64, error)
	CountByTimeRange(ctx context.Context, startTime, endTime time.Time) (int64, error)
	Count(ctx context.Context) (int64, error)
	Create(ctx context.Context, record *models.ConversationRecord) error
	CreateBatch(ctx context.Context, records []models.ConversationRecord) error
	DeleteByID(ctx context.Context, id uint) error

	// 统计方法
	GetTokenStats(ctx context.Context, startTime, endTime time.Time, agentCodes, channelCodes, roles []string) (*TokenStats, error)
	GetAgentDistribution(ctx context.Context, startTime, endTime time.Time, agentCodes, channelCodes, roles []string) ([]AgentDistribution, error)
	GetChannelDistribution(ctx context.Context, startTime, endTime time.Time, agentCodes, channelCodes, roles []string) ([]ChannelDistribution, error)
	GetRoleDistribution(ctx context.Context, startTime, endTime time.Time, agentCodes, channelCodes, roles []string) ([]RoleDistribution, error)
	GetSessionStats(ctx context.Context, startTime, endTime time.Time, agentCodes, channelCodes, roles []string) (*SessionStats, error)
}
