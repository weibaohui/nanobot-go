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

// GetHistory 获取消息历史
func (s *Session) GetHistory(maxMessages int) []map[string]any {

	//TODO 过期机制
	// //超过2小时的消息，认为是过期的，不返回
	// if len(s.Messages) > 0 {
	// 	first := s.Messages[0].Timestamp
	// 	if time.Since(first) > 2*time.Hour {
	// 		return nil
	// 	}
	// }

	//筛选出时间上是2小时之内的消息
	var filteredMessages []Message
	for _, m := range s.Messages {
		if time.Since(m.Timestamp) <= 2*time.Hour {
			filteredMessages = append(filteredMessages, m)
		}
	}

	var messages []Message
	if len(filteredMessages) > maxMessages {
		messages = filteredMessages[len(filteredMessages)-maxMessages:]
	} else {
		messages = filteredMessages
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
	logger      *zap.Logger
	sessionsDir string
	cache       map[string]*Session
	mu          sync.RWMutex
}

// NewManager 创建会话管理器
func NewManager(cfg *config.Config, logger *zap.Logger, dataDir string) *Manager {
	sessionsDir := filepath.Join(dataDir, "sessions")
	os.MkdirAll(sessionsDir, 0755)
	return &Manager{
		cfg:         cfg,
		logger:      logger,
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

// load 从磁盘加载会话（查找最近存在的会话文件）
func (m *Manager) load(key string) *Session {
	// 先查找最近存在的会话文件
	path := m.findLatestSessionFile(key)
	m.logger.Info("findLatestSessionFile", zap.String("key", key), zap.String("path", path))
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		m.logger.Error("ReadFile", zap.String("path", path), zap.Error(err))
		return nil
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		m.logger.Error("Unmarshal", zap.String("path", path), zap.Error(err))
		return nil
	}

	//打印加载的历史消息
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

// Delete 删除会话（删除所有相关的会话文件）
func (m *Manager) Delete(key string) bool {
	m.mu.Lock()
	delete(m.cache, key)
	m.mu.Unlock()

	// 先替换 : 为 _
	keyWithUnderscore := strings.ReplaceAll(key, ":", "_")
	safeKey := safeFilename(keyWithUnderscore)

	// 列出所有匹配的文件
	files, err := os.ReadDir(m.sessionsDir)
	if err != nil {
		return false
	}

	// 删除所有匹配前缀的文件
	prefix := safeKey + "_"
	deleted := false
	for _, f := range files {
		name := f.Name()
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".json") {
			os.Remove(filepath.Join(m.sessionsDir, name))
			deleted = true
		}
	}

	return deleted
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

// getSessionPath 获取会话文件路径（带当天日期）
func (m *Manager) getSessionPath(key string) string {
	now := time.Now()
	date := now.Format("20060102")

	// 先替换 : 为 _
	keyWithUnderscore := strings.ReplaceAll(key, ":", "_")

	// 再处理其他不安全字符（包括单引号和双引号）
	safeKey := safeFilename(keyWithUnderscore)

	return filepath.Join(m.sessionsDir, safeKey+"_"+date+".json")
}

// findLatestSessionFile 查找该 key 最近存在的会话文件
func (m *Manager) findLatestSessionFile(key string) string {
	// 先替换 : 为 _
	keyWithUnderscore := strings.ReplaceAll(key, ":", "_")
	safeKey := safeFilename(keyWithUnderscore)

	// 列出所有匹配的文件
	files, err := os.ReadDir(m.sessionsDir)
	if err != nil {
		return ""
	}

	// 查找匹配前缀的文件
	prefix := safeKey + "_"
	var candidates []string
	for _, f := range files {
		name := f.Name()
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".json") {
			candidates = append(candidates, name)
		}
	}

	if len(candidates) == 0 {
		return ""
	}

	// 按日期降序排序（文件名中包含日期，字符串排序即可）
	sort.Sort(sort.Reverse(sort.StringSlice(candidates)))

	return filepath.Join(m.sessionsDir, candidates[0])
}

// safeFilename 转换为安全文件名
func safeFilename(name string) string {
	unsafe := "<>!:'\"/\\|?*"
	for _, char := range unsafe {
		name = strings.ReplaceAll(name, string(char), "_")
	}
	return strings.TrimSpace(name)
}
