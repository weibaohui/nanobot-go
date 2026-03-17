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
	TraceID      string    `json:"trace_id,omitempty"`       // 链路追踪 ID
	SpanID       string    `json:"span_id,omitempty"`        // 跨度 ID
	ParentSpanID string    `json:"parent_span_id,omitempty"` // 父跨度 ID
}

// Session 会话
type Session struct {
	Key       string    `json:"key"`
	Messages  []Message `json:"messages"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// context 相关字段（不序列化）
	cancel context.CancelFunc `json:"-"` // context 取消函数
	ctx    context.Context    `json:"-"` // 当前会话的 context
	mu     sync.RWMutex       `json:"-"` // 保护 context 相关字段
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

// SetContext 设置会话的 context 和 cancel 函数
func (s *Session) SetContext(ctx context.Context, cancel context.CancelFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctx = ctx
	s.cancel = cancel
}

// GetContext 获取当前会话的 context（如果已取消或不存在，返回 nil）
func (s *Session) GetContext() context.Context {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ctx
}

// Cancel 取消当前会话
// 返回 true 表示成功触发取消，false 表示会话不在执行中
func (s *Session) Cancel() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancel == nil {
		return false
	}

	s.cancel()
	s.cancel = nil
	s.ctx = nil
	return true
}

// IsActive 检查会话是否正在活跃处理中
func (s *Session) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cancel != nil
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
func NewManager(cfg *config.Config, logger *zap.Logger, convRepo ConversationRecordRepository) *Manager {
	m := &Manager{
		cfg:      cfg,
		logger:   logger,
		cache:    make(map[string]*Session),
		convRepo: convRepo,
	}
	if convRepo == nil {
		logger.Warn("SessionManager 创建时 ConvRepo 为 nil，历史记录功能将不可用")
	} else {
		logger.Info("SessionManager 创建成功，ConvRepo 已设置")
	}
	return m
}

// GetHistory 从 ConversationRecordRepository 获取会话历史记录
func (m *Manager) GetHistory(ctx context.Context, sessionKey string, maxMessages int) []map[string]any {
	if m.convRepo == nil {
		m.logger.Warn("ConversationRecordRepository not set, returning empty history",
			zap.String("session_key", sessionKey))
		return nil
	}

	m.logger.Debug("开始加载会话历史",
		zap.String("session_key", sessionKey),
		zap.Int("max_messages", maxMessages))

	// 从数据库查询最近的对话记录
	records, err := m.convRepo.FindBySessionKey(ctx, sessionKey, &models.QueryOptions{
		OrderBy: "timestamp",
		Order:   "DESC",
		Limit:   maxMessages * 2,
	})
	if err != nil {
		m.logger.Error("Failed to find conversation by session key",
			zap.String("session_key", sessionKey),
			zap.Error(err))
		return nil
	}

	m.logger.Debug("从数据库查询到对话记录",
		zap.String("session_key", sessionKey),
		zap.Int("total_records", len(records)))

	// 使用所有查询到的记录（移除2小时时间限制，以支持加载历史对话）
	// 注意：查询使用 DESC 排序，最新的在前
	filteredRecords := records

	// 限制消息数量（取最新的 maxMessages 条，即前 maxMessages 条）
	if len(filteredRecords) > maxMessages {
		filteredRecords = filteredRecords[:maxMessages]
	}

	// 反转顺序，使消息按时间正序排列（最旧的在前，最新的在后）
	for i, j := 0, len(filteredRecords)-1; i < j; i, j = i+1, j-1 {
		filteredRecords[i], filteredRecords[j] = filteredRecords[j], filteredRecords[i]
	}

	// 转换为 map 格式
	var history []map[string]any
	for _, record := range filteredRecords {
		history = append(history, map[string]any{
			"role":    record.Role,
			"content": record.Content,
		})
	}

	m.logger.Info("会话历史加载完成",
		zap.String("session_key", sessionKey),
		zap.Int("loaded_messages", len(history)))

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

// CancelSession 取消指定会话
// 返回 true 表示成功触发取消，false 表示会话不存在或不在执行中
func (m *Manager) CancelSession(sessionKey string) bool {
	m.mu.RLock()
	session, ok := m.cache[sessionKey]
	m.mu.RUnlock()

	if !ok {
		return false
	}

	return session.Cancel()
}

// GetSession 获取指定会话（不进行创建）
func (m *Manager) GetSession(sessionKey string) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cache[sessionKey]
}

// IsSessionActive 检查指定会话是否正在活跃处理中
func (m *Manager) IsSessionActive(sessionKey string) bool {
	m.mu.RLock()
	session, ok := m.cache[sessionKey]
	m.mu.RUnlock()

	if !ok {
		return false
	}

	return session.IsActive()
}
