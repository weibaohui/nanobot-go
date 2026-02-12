package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Message 会话消息
type Message struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
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

// GetHistory 获取消息历史
func (s *Session) GetHistory(maxMessages int) []map[string]any {
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
	sessionsDir string
	cache       map[string]*Session
	mu          sync.RWMutex
}

// NewManager 创建会话管理器
func NewManager(dataDir string) *Manager {
	sessionsDir := filepath.Join(dataDir, "sessions")
	os.MkdirAll(sessionsDir, 0755)
	return &Manager{
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
	safeKey := safeFilename(strings.ReplaceAll(key, ":", "_"))
	return filepath.Join(m.sessionsDir, safeKey+".json")
}

// safeFilename 转换为安全文件名
func safeFilename(name string) string {
	unsafe := "<>:\"/\\|?*"
	for _, char := range unsafe {
		name = strings.ReplaceAll(name, string(char), "_")
	}
	return strings.TrimSpace(name)
}
