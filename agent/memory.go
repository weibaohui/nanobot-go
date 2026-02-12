package agent

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MemoryStore 内存存储系统
type MemoryStore struct {
	workspace  string
	memoryDir  string
	memoryFile string
}

// NewMemoryStore 创建内存存储
func NewMemoryStore(workspace string) *MemoryStore {
	memoryDir := filepath.Join(workspace, "memory")
	os.MkdirAll(memoryDir, 0755)
	return &MemoryStore{
		workspace:  workspace,
		memoryDir:  memoryDir,
		memoryFile: filepath.Join(memoryDir, "MEMORY.md"),
	}
}

// GetTodayFile 获取今日内存文件路径
func (m *MemoryStore) GetTodayFile() string {
	today := time.Now().Format("2006-01-02")
	return filepath.Join(m.memoryDir, today+".md")
}

// ReadToday 读取今日内存笔记
func (m *MemoryStore) ReadToday() string {
	todayFile := m.GetTodayFile()
	data, err := os.ReadFile(todayFile)
	if err != nil {
		return ""
	}
	return string(data)
}

// AppendToday 追加内容到今日内存笔记
func (m *MemoryStore) AppendToday(content string) error {
	todayFile := m.GetTodayFile()

	var existing string
	data, err := os.ReadFile(todayFile)
	if err == nil {
		existing = string(data)
	} else {
		// 新文件添加头部
		today := time.Now().Format("2006-01-02")
		existing = "# " + today + "\n\n"
	}

	newContent := existing + "\n" + content
	return os.WriteFile(todayFile, []byte(newContent), 0644)
}

// ReadLongTerm 读取长期内存
func (m *MemoryStore) ReadLongTerm() string {
	data, err := os.ReadFile(m.memoryFile)
	if err != nil {
		return ""
	}
	return string(data)
}

// WriteLongTerm 写入长期内存
func (m *MemoryStore) WriteLongTerm(content string) error {
	return os.WriteFile(m.memoryFile, []byte(content), 0644)
}

// GetRecentMemories 获取最近 N 天的内存
func (m *MemoryStore) GetRecentMemories(days int) string {
	var memories []string
	today := time.Now()

	for i := 0; i < days; i++ {
		date := today.AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")
		filePath := filepath.Join(m.memoryDir, dateStr+".md")

		data, err := os.ReadFile(filePath)
		if err == nil {
			memories = append(memories, string(data))
		}
	}

	return strings.Join(memories, "\n\n---\n\n")
}

// ListMemoryFiles 列出所有内存文件（按日期排序，最新的在前）
func (m *MemoryStore) ListMemoryFiles() []string {
	files, err := os.ReadDir(m.memoryDir)
	if err != nil {
		return nil
	}

	var result []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".md") && len(f.Name()) == 13 {
			// 格式: YYYY-MM-DD.md
			result = append(result, filepath.Join(m.memoryDir, f.Name()))
		}
	}

	// 按日期排序（最新的在前）
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i] < result[j] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// GetMemoryContext 获取内存上下文
func (m *MemoryStore) GetMemoryContext() string {
	var parts []string

	// 长期内存
	longTerm := m.ReadLongTerm()
	if longTerm != "" {
		parts = append(parts, "## 长期内存\n"+longTerm)
	}

	// 今日笔记
	today := m.ReadToday()
	if today != "" {
		parts = append(parts, "## 今日笔记\n"+today)
	}

	return strings.Join(parts, "\n\n")
}
