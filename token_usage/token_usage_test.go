package token_usage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestTokenUsage_Add 测试 Token 用量累加
func TestTokenUsage_Add(t *testing.T) {
	tests := []struct {
		name     string
		initial  TokenUsage
		add      TokenUsage
		expected TokenUsage
	}{
		{
			name:     "基本累加",
			initial:  TokenUsage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150},
			add:      TokenUsage{PromptTokens: 200, CompletionTokens: 100, TotalTokens: 300},
			expected: TokenUsage{PromptTokens: 300, CompletionTokens: 150, TotalTokens: 450},
		},
		{
			name:     "累加零值",
			initial:  TokenUsage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150},
			add:      TokenUsage{},
			expected: TokenUsage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150},
		},
		{
			name:     "累加推理和缓存token",
			initial:  TokenUsage{ReasoningTokens: 50, CachedTokens: 20},
			add:      TokenUsage{ReasoningTokens: 30, CachedTokens: 10},
			expected: TokenUsage{ReasoningTokens: 80, CachedTokens: 30},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.initial.Add(tt.add)

			if tt.initial.PromptTokens != tt.expected.PromptTokens {
				t.Errorf("PromptTokens = %d, 期望 %d", tt.initial.PromptTokens, tt.expected.PromptTokens)
			}

			if tt.initial.CompletionTokens != tt.expected.CompletionTokens {
				t.Errorf("CompletionTokens = %d, 期望 %d", tt.initial.CompletionTokens, tt.expected.CompletionTokens)
			}

			if tt.initial.TotalTokens != tt.expected.TotalTokens {
				t.Errorf("TotalTokens = %d, 期望 %d", tt.initial.TotalTokens, tt.expected.TotalTokens)
			}

			if tt.initial.ReasoningTokens != tt.expected.ReasoningTokens {
				t.Errorf("ReasoningTokens = %d, 期望 %d", tt.initial.ReasoningTokens, tt.expected.ReasoningTokens)
			}

			if tt.initial.CachedTokens != tt.expected.CachedTokens {
				t.Errorf("CachedTokens = %d, 期望 %d", tt.initial.CachedTokens, tt.expected.CachedTokens)
			}
		})
	}
}

// TestNewTokenUsageManager 测试创建 Token 使用量管理器
func TestNewTokenUsageManager(t *testing.T) {
	tmpDir := t.TempDir()

	manager := NewTokenUsageManager(tmpDir)
	if manager == nil {
		t.Fatal("NewTokenUsageManager 返回 nil")
	}

	expectedDir := filepath.Join(tmpDir, "token_usage")
	if manager.dataDir != expectedDir {
		t.Errorf("dataDir = %q, 期望 %q", manager.dataDir, expectedDir)
	}

	if manager.cache == nil {
		t.Error("cache 不应该为 nil")
	}

	// 检查目录是否创建
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Errorf("目录 %q 应该被创建", expectedDir)
	}
}

// TestTokenUsageManager_AddRecord 测试添加记录
func TestTokenUsageManager_AddRecord(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewTokenUsageManager(tmpDir)

	record := TokenUsageRecord{
		TraceID:   "trace-123",
		SpanID:    "span-456",
		TokenUsage: TokenUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
		Timestamp: time.Now(),
	}

	err := manager.AddRecord("session-key", record)
	if err != nil {
		t.Fatalf("AddRecord 返回错误: %v", err)
	}

	records := manager.GetRecords("session-key")
	if len(records) != 1 {
		t.Fatalf("记录数量 = %d, 期望 1", len(records))
	}

	if records[0].TraceID != "trace-123" {
		t.Errorf("TraceID = %q, 期望 trace-123", records[0].TraceID)
	}
}

// TestTokenUsageManager_GetSummary 测试获取汇总
func TestTokenUsageManager_GetSummary(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewTokenUsageManager(tmpDir)

	// 添加多条记录
	record1 := TokenUsageRecord{
		TraceID:   "trace-1",
		TokenUsage: TokenUsage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150},
	}
	record2 := TokenUsageRecord{
		TraceID:   "trace-2",
		TokenUsage: TokenUsage{PromptTokens: 200, CompletionTokens: 100, TotalTokens: 300},
	}

	manager.AddRecord("session-key", record1)
	manager.AddRecord("session-key", record2)

	summary := manager.GetSummary("session-key")

	if summary.PromptTokens != 300 {
		t.Errorf("汇总 PromptTokens = %d, 期望 300", summary.PromptTokens)
	}

	if summary.CompletionTokens != 150 {
		t.Errorf("汇总 CompletionTokens = %d, 期望 150", summary.CompletionTokens)
	}

	if summary.TotalTokens != 450 {
		t.Errorf("汇总 TotalTokens = %d, 期望 450", summary.TotalTokens)
	}
}

// TestTokenUsageManager_SaveAndLoad 测试保存和加载
func TestTokenUsageManager_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewTokenUsageManager(tmpDir)

	record := TokenUsageRecord{
		TraceID:   "trace-123",
		SpanID:    "span-456",
		TokenUsage: TokenUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
		Timestamp: time.Now(),
	}

	manager.AddRecord("test-session", record)
	err := manager.Save("test-session")
	if err != nil {
		t.Fatalf("Save 返回错误: %v", err)
	}

	// 创建新的管理器，从磁盘加载
	newManager := NewTokenUsageManager(tmpDir)
	records := newManager.load("test-session")

	if len(records) != 1 {
		t.Fatalf("记录数量 = %d, 期望 1", len(records))
	}

	if records[0].TokenUsage.PromptTokens != 100 {
		t.Errorf("PromptTokens = %d, 期望 100", records[0].TokenUsage.PromptTokens)
	}
}

// TestSafeFilename 测试安全文件名转换
func TestSafeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"正常字符", "normal_name", "normal_name"},
		{"包含冒号", "test:name", "test_name"},
		{"包含多个不安全字符", "test<>:\"/\\|?*name", "test_________name"},
		{"前后空格", "  test  ", "test"},
		{"空字符串", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := safeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("safeFilename(%q) = %q, 期望 %q", tt.input, result, tt.expected)
			}
		})
	}
}