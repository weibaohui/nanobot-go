package session

import (
	"context"
	"sync"
	"time"

	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/internal/models"
	"go.uber.org/zap"
)

// Message 会话消息
type Message struct {
	Role         string    `json:"role"`
	Content      string    `json:"content"`
	Timestamp    time.Time `json:"timestamp"`
	TraceID      string    `json:"trace_id,omitempty"`        // 链路追踪 ID
	SpanID       string    `json:"span_id,omitempty"`         // 跨度 ID
	ParentSpanID string    `json:"parent_span_id,omitempty"` // 父跨度 ID
}

// Session 会话
type Session struct {
	Key       string    `json:"key"`
	Messages  []Message `json:"messages"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AddMessage 添加消息到会话
func (s *Session) AddMessage(role, content string) {
	s.Messages = append(s.Messages, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
	s.UpdatedAt = time.Now()
}

// AddMessageWithTrace 添加消息到会话（带链路追踪信息）
func (s *Session) AddMessageWithTrace(role, content, traceID, spanID, parentSpanID string) {
	s.Messages = append(s.Messages, Message{
		Role:         role,
		Content:      content,
		Timestamp:    time.Now(),
		TraceID:      traceID,
		SpanID:       spanID,
		ParentSpanID: parentSpanID,
	})
	s.UpdatedAt = time.Now()
}

// Clear 清空会话消息
func (s *Session) Clear() {
	s.Messages = nil
	s.UpdatedAt = time.Now()
}

// ConversationRecordRepository 对话记录仓库接口
type ConversationRecordRepository interface {
	FindBySessionKey(ctx context.Context, sessionKey string, opts *models.QueryOptions) ([]models.ConversationRecord, error)
}

// Manager 会话管理器
type Manager struct {
	cfg      *config.Config
	logger   *zap.Logger
	cache    map[string]*Session
	mu       sync.RWMutex
	convRepo ConversationRecordRepository
}

// NewManager 创建会话管理器
func NewManager(cfg *config.Config, logger *zap.Logger, dataDir string, convRepo ConversationRecordRepository) *Manager {
	return &Manager{
		cfg:      cfg,
		logger:   logger,
		cache:    make(map[string]*Session),
		convRepo: convRepo,
	}
}

// GetHistory 从 ConversationRecordRepository 获取会话历史记录
func (m *Manager) GetHistory(ctx context.Context, sessionKey string, maxMessages int) []map[string]any {
	if m.convRepo == nil {
		m.logger.Warn("ConversationRecordRepository not set, returning empty history")
		return nil
	}

	// 从数据库查询最近的对话记录
	records, err := m.convRepo.FindBySessionKey(ctx, sessionKey, &models.QueryOptions{
		OrderBy: "timestamp",
		Order:   "ASC",
		Limit:   maxMessages * 2,
	})
	if err != nil {
		m.logger.Error("Failed to find conversation by session key",
			zap.String("sessionKey", sessionKey),
			zap.Error(err))
		return nil
	}

	// 筛选出2小时之内的消息
	cutoffTime := time.Now().Add(-2 * time.Hour)
	var filteredRecords []models.ConversationRecord
	for _, record := range records {
		if record.Timestamp.After(cutoffTime) {
			filteredRecords = append(filteredRecords, record)
		}
	}

	// 限制消息数量（取最近的 maxMessages 条）
	if len(filteredRecords) > maxMessages {
		filteredRecords = filteredRecords[len(filteredRecords)-maxMessages:]
	}

	// 转换为 map 格式
	var history []map[string]any
	for _, record := range filteredRecords {
		history = append(history, map[string]any{
			"role":    record.Role,
			"content": record.Content,
		})
	}

	return history
}

// GetOrCreate 获取或创建会话
func (m *Manager) GetOrCreate(key string) *Session {
	m.mu.RLock()
	if session, ok := m.cache[key]; ok {
		m.mu.RUnlock()
		return session
	}
	m.mu.RUnlock()

	// 创建新会话
	session := &Session{
		Key:       key,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	m.mu.Lock()
	m.cache[key] = session
	m.mu.Unlock()

	return session
}
