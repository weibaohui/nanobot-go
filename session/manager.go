package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/weibaohui/nanobot-go/config"
	"go.uber.org/zap"
)

// TokenUsage Token 用量统计
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`     // 输入 token 数量
	CompletionTokens int `json:"completion_tokens"` // 输出 token 数量
	TotalTokens      int `json:"total_tokens"`      // 总 token 数量
	ReasoningTokens  int `json:"reasoning_tokens"`  // 推理 token 数量 (o1 等模型)
	CachedTokens     int `json:"cached_tokens"`     // 缓存 token 数量 (缓存命中)
}

// Add 将另一个 TokenUsage 累加到当前用量
func (t *TokenUsage) Add(other TokenUsage) {
	t.PromptTokens += other.PromptTokens
	t.CompletionTokens += other.CompletionTokens
	t.TotalTokens += other.TotalTokens
	t.ReasoningTokens += other.ReasoningTokens
	t.CachedTokens += other.CachedTokens
}

// Message 会话消息
type Message struct {
	Role         string      `json:"role"`
	Content      string      `json:"content"`
	Timestamp    time.Time   `json:"timestamp"`
	TokenUsage   *TokenUsage `json:"token_usage,omitempty"`    // 该消息对应的 token 用量
	TraceID      string      `json:"trace_id,omitempty"`       // 链路追踪 ID
	SpanID       string      `json:"span_id,omitempty"`        // 跨度 ID
	ParentSpanID string      `json:"parent_span_id,omitempty"` // 父跨度 ID
}

// Session 会话
type Session struct {
	Key        string     `json:"key"`
	Messages   []Message  `json:"messages"`
	TokenUsage TokenUsage `json:"token_usage"` // 累加的 token 用量
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

// AddMessage 添加消息到会话
func (s *Session) AddMessage(role, content string) {
	s.Messages = append(s.Messages, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now().UTC(),
	})
	s.UpdatedAt = time.Now().UTC()
}

// AddMessageWithTrace 添加消息到会话（带链路追踪信息）
func (s *Session) AddMessageWithTrace(role, content, traceID, spanID, parentSpanID string) {
	s.Messages = append(s.Messages, Message{
		Role:         role,
		Content:      content,
		Timestamp:    time.Now().UTC(),
		TraceID:      traceID,
		SpanID:       spanID,
		ParentSpanID: parentSpanID,
	})
	s.UpdatedAt = time.Now().UTC()
}

// AddMessageWithTokenUsage 添加消息到会话并记录 token 用量
func (s *Session) AddMessageWithTokenUsage(role, content string, usage TokenUsage) {
	msg := Message{
		Role:       role,
		Content:    content,
		Timestamp:  time.Now().UTC(),
		TokenUsage: &usage,
	}
	s.Messages = append(s.Messages, msg)

	// 累加到 Session 级别的 TokenUsage
	s.TokenUsage.Add(usage)
	s.UpdatedAt = time.Now().UTC()
}

// AddMessageWithTokenUsageAndTrace 添加消息到会话并记录 token 用量和链路追踪信息
func (s *Session) AddMessageWithTokenUsageAndTrace(role, content string, usage TokenUsage, traceID, spanID, parentSpanID string) {
	msg := Message{
		Role:         role,
		Content:      content,
		Timestamp:    time.Now().UTC(),
		TokenUsage:   &usage,
		TraceID:      traceID,
		SpanID:       spanID,
		ParentSpanID: parentSpanID,
	}
	s.Messages = append(s.Messages, msg)

	// 累加到 Session 级别的 TokenUsage
	s.TokenUsage.Add(usage)
	s.UpdatedAt = time.Now().UTC()
}

// UpdateTokenUsage 更新 Session 级别的累加 token 用量
func (s *Session) UpdateTokenUsage(usage TokenUsage) {
	s.TokenUsage.Add(usage)
	s.UpdatedAt = time.Now().UTC()
}

// GetHistory 获取消息历史
func (s *Session) GetHistory(maxMessages int) []map[string]any {

	//TODO 过期机制
	//超过2小时的消息，认为是过期的，不返回
	if len(s.Messages) > 0 {
		first := s.Messages[0].Timestamp
		if time.Since(first) > 2*time.Hour {
			return nil
		}
	}

	var messages []Message
	if len(s.Messages) > maxMessages {
		messages = s.Messages[len(s.Messages)-maxMessages:]
	} else {
		messages = s.Messages
	}

	var result []map[string]any
	for _, m := range messages {
		result = append(result, map[string]any{
			"role":    m.Role,
			"content": m.Content,
		})
	}
	return result
}

// Clear 清空会话消息
func (s *Session) Clear() {
	s.Messages = nil
	s.UpdatedAt = time.Now().UTC()
}

// Manager 会话管理器
type Manager struct {
	cfg         *config.Config
	sessionsDir string
	cache       map[string]*Session
	mu          sync.RWMutex
}

// NewManager 创建会话管理器
func NewManager(cfg *config.Config, dataDir string) *Manager {
	sessionsDir := filepath.Join(dataDir, "sessions")
	os.MkdirAll(sessionsDir, 0755)
	return &Manager{
		cfg:         cfg,
		sessionsDir: sessionsDir,
		cache:       make(map[string]*Session),
	}
}

// GetOrCreate 获取或创建会话
func (m *Manager) GetOrCreate(key string) *Session {
	m.mu.RLock()
	if session, ok := m.cache[key]; ok {
		m.mu.RUnlock()
		return session
	}
	m.mu.RUnlock()

	// 尝试从磁盘加载
	session := m.load(key)
	if session == nil {
		session = &Session{
			Key:       key,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
	}

	m.mu.Lock()
	m.cache[key] = session
	m.mu.Unlock()

	return session
}

// load 从磁盘加载会话
func (m *Manager) load(key string) *Session {
	path := m.getSessionPath(key)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil
	}

	return &session
}

// Save 保存会话到磁盘
func (m *Manager) Save(session *Session) error {

	path := m.getSessionPath(session.Key)
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.cache[session.Key] = session
	err = os.WriteFile(path, data, 0644)
	m.mu.Unlock()

	return err
}

// Delete 删除会话
func (m *Manager) Delete(key string) bool {
	m.mu.Lock()
	delete(m.cache, key)
	m.mu.Unlock()

	path := m.getSessionPath(key)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	os.Remove(path)
	return true
}

// ListSessions 列出所有会话
func (m *Manager) ListSessions() []map[string]any {
	var sessions []map[string]any

	files, err := os.ReadDir(m.sessionsDir)
	if err != nil {
		return sessions
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".json") {
			path := filepath.Join(m.sessionsDir, f.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			var session Session
			if err := json.Unmarshal(data, &session); err != nil {
				continue
			}

			sessions = append(sessions, map[string]any{
				"key":       session.Key,
				"createdAt": session.CreatedAt,
				"updatedAt": session.UpdatedAt,
				"path":      path,
			})
		}
	}

	// 按更新时间排序（最新的在前）
	sort.Slice(sessions, func(i, j int) bool {
		ti := sessions[i]["updatedAt"].(time.Time)
		tj := sessions[j]["updatedAt"].(time.Time)
		return ti.After(tj)
	})

	return sessions
}

// getSessionPath 获取会话文件路径
func (m *Manager) getSessionPath(key string) string {
	// 使用 UTC 时间避免跨时区问题
	// 历史文件命名：safeKey_YYYYMMDD.json
	now := time.Now().UTC()
	date := now.Format("20060102")

	// 先替换 : 为 _
	keyWithUnderscore := strings.ReplaceAll(key, ":", "_")

	// 再处理其他不安全字符（包括单引号和双引号）
	safeKey := safeFilename(keyWithUnderscore)

	result := filepath.Join(m.sessionsDir, safeKey+"_"+date+".json")

	// 调试：记录生成的文件名
	zap.L().Info("生成 session 文件路径",
		zap.String("original_key", key),
		zap.String("key_with_underscore", keyWithUnderscore),
		zap.String("safe_key", safeKey),
		zap.String("file_path", result),
	)

	return result
}

// safeFilename 转换为安全文件名
func safeFilename(name string) string {
	unsafe := "<>!:'\"/\\|?*"
	for _, char := range unsafe {
		name = strings.ReplaceAll(name, string(char), "_")
	}
	return strings.TrimSpace(name)
}
