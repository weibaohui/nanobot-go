package token_usage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
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

// TokenUsageRecord Token 使用记录
type TokenUsageRecord struct {
	TraceID   string      `json:"trace_id"`  // 链路追踪 ID
	SpanID    string      `json:"span_id"`   // 跨度 ID
	TokenUsage TokenUsage `json:"token_usage"` // Token 使用量
	Timestamp time.Time   `json:"timestamp"`  // 记录时间
}

// TokenUsageManager Token 使用量管理器
type TokenUsageManager struct {
	dataDir    string
	cache      map[string][]TokenUsageRecord // sessionKey -> records
	mu         sync.RWMutex
}

// DataDir 返回数据目录路径
func (m *TokenUsageManager) DataDir() string {
	return m.dataDir
}

// NewTokenUsageManager 创建 Token 使用量管理器
func NewTokenUsageManager(dataDir string) *TokenUsageManager {
	usageDir := filepath.Join(dataDir, "token_usage")
	os.MkdirAll(usageDir, 0755)
	return &TokenUsageManager{
		dataDir: usageDir,
		cache:   make(map[string][]TokenUsageRecord),
	}
}

// AddRecord 添加 Token 使用记录
func (m *TokenUsageManager) AddRecord(sessionKey string, record TokenUsageRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cache[sessionKey] = append(m.cache[sessionKey], record)
	return nil
}

// load 从磁盘加载记录
func (m *TokenUsageManager) load(sessionKey string) []TokenUsageRecord {
	path := m.getUsagePath(sessionKey)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var records []TokenUsageRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil
	}

	return records
}

// Save 保存记录到磁盘
func (m *TokenUsageManager) Save(sessionKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	records, ok := m.cache[sessionKey]
	if !ok || len(records) == 0 {
		return nil
	}

	path := m.getUsagePath(sessionKey)
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// GetRecords 获取指定会话的 Token 使用记录
func (m *TokenUsageManager) GetRecords(sessionKey string) []TokenUsageRecord {
	// 先从缓存获取
	m.mu.RLock()
	records, ok := m.cache[sessionKey]
	m.mu.RUnlock()

	if ok && len(records) > 0 {
		return records
	}

	// 从磁盘加载
	records = m.load(sessionKey)
	if len(records) > 0 {
		// 缓存到内存
		m.mu.Lock()
		m.cache[sessionKey] = records
		m.mu.Unlock()
	}

	return records
}

// GetSummary 获取指定会话的 Token 使用汇总
func (m *TokenUsageManager) GetSummary(sessionKey string) TokenUsage {
	records := m.GetRecords(sessionKey)
	var summary TokenUsage
	for _, r := range records {
		summary.Add(r.TokenUsage)
	}
	return summary
}

// getUsagePath 获取记录文件路径
func (m *TokenUsageManager) getUsagePath(sessionKey string) string {
	now := time.Now()
	date := now.Format("20060102")

	// 替换不安全字符
	safeKey := safeFilename(sessionKey)
	safeKey = strings.ReplaceAll(safeKey, ":", "_")

	return filepath.Join(m.dataDir, safeKey+"_"+date+".json")
}

// safeFilename 转换为安全文件名
func safeFilename(name string) string {
	unsafe := "<>!:'\"/\\|?*"
	for _, char := range unsafe {
		name = strings.ReplaceAll(name, string(char), "_")
	}
	return strings.TrimSpace(name)
}

// ListUsageFiles 列出所有 Token 使用记录文件
func (m *TokenUsageManager) ListUsageFiles() []map[string]any {
	var files []map[string]any

	entries, err := os.ReadDir(m.dataDir)
	if err != nil {
		return files
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		path := filepath.Join(m.dataDir, entry.Name())
		files = append(files, map[string]any{
			"name":    entry.Name(),
			"path":    path,
			"size":    info.Size(),
			"modtime": info.ModTime(),
		})
	}

	// 按修改时间排序（最新的在前）
	sort.Slice(files, func(i, j int) bool {
		ti := files[i]["modtime"].(time.Time)
		tj := files[j]["modtime"].(time.Time)
		return ti.After(tj)
	})

	return files
}