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
	Role       string      `json:"role"`
	Content    string      `json:"content"`
	Timestamp  time.Time   `json:"timestamp"`
	TokenUsage *TokenUsage `json:"token_usage,omitempty"` // 该消息对应的 token 用量
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
		Timestamp: time.Now(),
	})
	s.UpdatedAt = time.Now()
}

// AddMessageWithTokenUsage 添加消息到会话并记录 token 用量
func (s *Session) AddMessageWithTokenUsage(role, content string, usage TokenUsage) {
	msg := Message{
		Role:       role,
		Content:    content,
		Timestamp:  time.Now(),
		TokenUsage: &usage,
	}
	s.Messages = append(s.Messages, msg)

	// 累加到 Session 级别的 TokenUsage
	s.TokenUsage.Add(usage)
	s.UpdatedAt = time.Now()
}

// UpdateTokenUsage 更新 Session 级别的累加 token 用量
func (s *Session) UpdateTokenUsage(usage TokenUsage) {
	s.TokenUsage.Add(usage)
	s.UpdatedAt = time.Now()
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
	s.UpdatedAt = time.Now()
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
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
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
	m.mu.Unlock()

	return os.WriteFile(path, data, 0644)
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

	//TODO 做成一个配置项
	// 以每天 6 点为界，6 点前的会话归档到历史文件
	// 历史文件命名：safeKey_YYYYMMDD.json
	now := time.Now()
	// 如果当前时间在 0~6 点之间，则归档到前一天
	if now.Hour() < 6 {
		now = now.AddDate(0, 0, -1)
	}
	date := now.Format("20060102")
	safeKey := safeFilename(strings.ReplaceAll(key, ":", "_"))
	return filepath.Join(m.sessionsDir, safeKey+"_"+date+".json")
}

// safeFilename 转换为安全文件名
func safeFilename(name string) string {
	unsafe := "<>:\"/\\|?*"
	for _, char := range unsafe {
		name = strings.ReplaceAll(name, string(char), "_")
	}
	return strings.TrimSpace(name)
}
