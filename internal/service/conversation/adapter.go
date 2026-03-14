package conversation

import (
	"context"
	"time"

	"github.com/weibaohui/nanobot-go/internal/models"
)

// RecordServiceAdapter 适配器，将 conversation.Service 适配为传统的 ConversationRecordService 接口
type RecordServiceAdapter struct {
	repo Repository
}

// NewRecordServiceAdapter 创建适配器实例
func NewRecordServiceAdapter(repo Repository) *RecordServiceAdapter {
	return &RecordServiceAdapter{repo: repo}
}

// List 获取对话记录列表
func (a *RecordServiceAdapter) List(ctx context.Context, userID uint, agentID uint, channelID uint, sessionKey string, offset int, limit int) ([]models.ConversationRecord, int64, error) {
	var records []models.ConversationRecord
	var total int64

	query := make(map[string]interface{})
	if userID > 0 {
		query["user_id"] = userID
	}
	if agentID > 0 {
		query["agent_id"] = agentID
	}
	if channelID > 0 {
		query["channel_id"] = channelID
	}
	if sessionKey != "" {
		query["session_key"] = sessionKey
	}

	// 使用 Repository 直接查询
	if sessionKey != "" {
		var err error
		records, err = a.repo.FindBySessionKey(ctx, sessionKey, &models.QueryOptions{
			OrderBy: "timestamp",
			Order:   "DESC",
			Limit:   limit,
			Offset:  offset,
		})
		if err != nil {
			return nil, 0, err
		}
		total, err = a.repo.CountBySessionKey(ctx, sessionKey)
		if err != nil {
			return nil, 0, err
		}
	} else {
		// 默认查询最近记录
		var err error
		records, err = a.repo.FindByTimeRange(ctx, time.Time{}, time.Now(), &models.QueryOptions{
			OrderBy: "timestamp",
			Order:   "DESC",
			Limit:   limit,
			Offset:  offset,
		})
		if err != nil {
			return nil, 0, err
		}
		total, err = a.repo.Count(ctx)
		if err != nil {
			return nil, 0, err
		}
	}

	return records, total, nil
}

// Get 获取单条对话记录
func (a *RecordServiceAdapter) Get(ctx context.Context, id uint) (*models.ConversationRecord, error) {
	return a.repo.FindByID(ctx, id)
}

// Create 创建对话记录
func (a *RecordServiceAdapter) Create(ctx context.Context, record *models.ConversationRecord) error {
	return a.repo.Create(ctx, record)
}

// Update 更新对话记录
func (a *RecordServiceAdapter) Update(ctx context.Context, id uint, record *models.ConversationRecord) error {
	// 先获取原有记录
	existing, err := a.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	// 更新字段
	existing.UserCode = record.UserCode
	existing.AgentCode = record.AgentCode
	existing.ChannelCode = record.ChannelCode
	existing.SessionKey = record.SessionKey
	existing.TraceID = record.TraceID
	existing.SpanID = record.SpanID
	existing.ParentSpanID = record.ParentSpanID
	existing.EventType = record.EventType
	existing.Role = record.Role
	existing.Content = record.Content
	existing.TotalTokens = record.TotalTokens
	existing.PromptTokens = record.PromptTokens
	existing.CompletionTokens = record.CompletionTokens
	existing.ReasoningTokens = record.ReasoningTokens
	existing.CachedTokens = record.CachedTokens
	existing.Timestamp = record.Timestamp
	return a.repo.Create(ctx, existing)
}

// Delete 删除对话记录
func (a *RecordServiceAdapter) Delete(ctx context.Context, id uint) error {
	return a.repo.DeleteByID(ctx, id)
}

// GetBySessionKey 根据 SessionKey 获取对话记录
func (a *RecordServiceAdapter) GetBySessionKey(ctx context.Context, sessionKey string, offset int, limit int) ([]models.ConversationRecord, int64, error) {
	records, err := a.repo.FindBySessionKey(ctx, sessionKey, &models.QueryOptions{
		OrderBy: "timestamp",
		Order:   "ASC",
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := a.repo.CountBySessionKey(ctx, sessionKey)
	if err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

// GetByTraceID 根据 TraceID 获取对话记录
func (a *RecordServiceAdapter) GetByTraceID(ctx context.Context, traceID string) ([]models.ConversationRecord, error) {
	return a.repo.FindByTraceID(ctx, traceID)
}
