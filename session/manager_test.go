package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/weibaohui/nanobot-go/config"
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

// TestSession_AddMessage 测试添加消息到会话
func TestSession_AddMessage(t *testing.T) {
	session := &Session{
		Key:       "test-session",
		CreatedAt: time.Now(),
		UpdatedAt:  time.Now(),
	}

	session.AddMessage("user", "你好")
	session.AddMessage("assistant", "你好！有什么可以帮助你的吗？")

	if len(session.Messages) != 2 {
		t.Fatalf("消息数量 = %d, 期望 2", len(session.Messages))
	}

	if session.Messages[0].Role != "user" {
		t.Errorf("第一条消息角色 = %q, 期望 user", session.Messages[0].Role)
	}

	if session.Messages[0].Content != "你好" {
		t.Errorf("第一条消息内容 = %q, 期望 你好", session.Messages[0].Content)
	}

	if session.Messages[1].Role != "assistant" {
		t.Errorf("第二条消息角色 = %q, 期望 assistant", session.Messages[1].Role)
	}

	if session.Messages[0].Timestamp.IsZero() {
		t.Error("消息时间戳不应该为零值")
	}
}

// TestSession_AddMessageWithTokenUsage 测试添加带 Token 用量的消息
func TestSession_AddMessageWithTokenUsage(t *testing.T) {
	session := &Session{
		Key:       "test-session",
		CreatedAt: time.Now(),
		UpdatedAt:  time.Now(),
	}

	usage := TokenUsage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	session.AddMessageWithTokenUsage("user", "测试消息", usage)

	if len(session.Messages) != 1 {
		t.Fatalf("消息数量 = %d, 期望 1", len(session.Messages))
	}

	if session.Messages[0].TokenUsage == nil {
		t.Fatal("TokenUsage 不应该为 nil")
	}

	if session.Messages[0].TokenUsage.PromptTokens != 100 {
		t.Errorf("PromptTokens = %d, 期望 100", session.Messages[0].TokenUsage.PromptTokens)
	}

	if session.TokenUsage.PromptTokens != 100 {
		t.Errorf("Session 级别 PromptTokens = %d, 期望 100", session.TokenUsage.PromptTokens)
	}
}

// TestSession_UpdateTokenUsage 测试更新 Session 级别的 Token 用量
func TestSession_UpdateTokenUsage(t *testing.T) {
	session := &Session{
		Key:       "test-session",
		CreatedAt: time.Now(),
		UpdatedAt:  time.Now(),
	}

	session.UpdateTokenUsage(TokenUsage{PromptTokens: 100, CompletionTokens: 50})
	session.UpdateTokenUsage(TokenUsage{PromptTokens: 50, CompletionTokens: 30})

	if session.TokenUsage.PromptTokens != 150 {
		t.Errorf("PromptTokens = %d, 期望 150", session.TokenUsage.PromptTokens)
	}

	if session.TokenUsage.CompletionTokens != 80 {
		t.Errorf("CompletionTokens = %d, 期望 80", session.TokenUsage.CompletionTokens)
	}
}

// TestSession_GetHistory 测试获取消息历史
func TestSession_GetHistory(t *testing.T) {
	t.Run("获取全部消息", func(t *testing.T) {
		session := &Session{
			Key: "test-session",
			Messages: []Message{
				{Role: "user", Content: "消息1", Timestamp: time.Now()},
				{Role: "assistant", Content: "回复1", Timestamp: time.Now()},
				{Role: "user", Content: "消息2", Timestamp: time.Now()},
			},
			CreatedAt: time.Now(),
			UpdatedAt:  time.Now(),
		}

		history := session.GetHistory(10)
		if len(history) != 3 {
			t.Errorf("历史消息数量 = %d, 期望 3", len(history))
		}
	})

	t.Run("限制消息数量", func(t *testing.T) {
		session := &Session{
			Key: "test-session",
			Messages: []Message{
				{Role: "user", Content: "消息1", Timestamp: time.Now()},
				{Role: "assistant", Content: "回复1", Timestamp: time.Now()},
				{Role: "user", Content: "消息2", Timestamp: time.Now()},
				{Role: "assistant", Content: "回复2", Timestamp: time.Now()},
				{Role: "user", Content: "消息3", Timestamp: time.Now()},
			},
			CreatedAt: time.Now(),
			UpdatedAt:  time.Now(),
		}

		history := session.GetHistory(2)
		if len(history) != 2 {
			t.Errorf("历史消息数量 = %d, 期望 2", len(history))
		}

		if history[0]["content"] != "回复2" {
			t.Errorf("第一条消息内容 = %v, 期望 回复2", history[0]["content"])
		}

		if history[1]["content"] != "消息3" {
			t.Errorf("第二条消息内容 = %v, 期望 消息3", history[1]["content"])
		}
	})

	t.Run("空消息列表", func(t *testing.T) {
		session := &Session{
			Key:       "test-session",
			Messages:  []Message{},
			CreatedAt: time.Now(),
			UpdatedAt:  time.Now(),
		}

		history := session.GetHistory(10)
		if len(history) != 0 {
			t.Errorf("历史消息数量 = %d, 期望 0", len(history))
		}
	})
}

// TestSession_Clear 测试清空会话消息
func TestSession_Clear(t *testing.T) {
	session := &Session{
		Key: "test-session",
		Messages: []Message{
			{Role: "user", Content: "消息1"},
			{Role: "assistant", Content: "回复1"},
		},
		CreatedAt: time.Now(),
		UpdatedAt:  time.Now(),
	}

	session.Clear()

	if len(session.Messages) != 0 {
		t.Errorf("消息数量 = %d, 期望 0", len(session.Messages))
	}
}

// TestNewManager 测试创建会话管理器
func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.DefaultConfig()

	manager := NewManager(cfg, tmpDir)
	if manager == nil {
		t.Fatal("NewManager 返回 nil")
	}

	expectedDir := filepath.Join(tmpDir, "sessions")
	if manager.sessionsDir != expectedDir {
		t.Errorf("sessionsDir = %q, 期望 %q", manager.sessionsDir, expectedDir)
	}

	if manager.cache == nil {
		t.Error("cache 不应该为 nil")
	}
}

// TestManager_GetOrCreate 测试获取或创建会话
func TestManager_GetOrCreate(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.DefaultConfig()
	manager := NewManager(cfg, tmpDir)

	t.Run("创建新会话", func(t *testing.T) {
		session := manager.GetOrCreate("new-session-key")
		if session == nil {
			t.Fatal("GetOrCreate 返回 nil")
		}

		if session.Key != "new-session-key" {
			t.Errorf("Key = %q, 期望 new-session-key", session.Key)
		}

		if session.CreatedAt.IsZero() {
			t.Error("CreatedAt 不应该为零值")
		}
	})

	t.Run("获取已存在的会话", func(t *testing.T) {
		session1 := manager.GetOrCreate("existing-session")
		session1.AddMessage("user", "测试消息")

		session2 := manager.GetOrCreate("existing-session")

		if len(session2.Messages) != 1 {
			t.Errorf("消息数量 = %d, 期望 1", len(session2.Messages))
		}
	})
}

// TestManager_Save 测试保存会话
func TestManager_Save(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.DefaultConfig()
	manager := NewManager(cfg, tmpDir)

	session := &Session{
		Key:       "save-test-session",
		Messages:  []Message{{Role: "user", Content: "测试消息", Timestamp: time.Now()}},
		CreatedAt: time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := manager.Save(session)
	if err != nil {
		t.Fatalf("Save 返回错误: %v", err)
	}

	loaded := manager.load("save-test-session")
	if loaded == nil {
		t.Fatal("加载会话失败")
	}

	if len(loaded.Messages) != 1 {
		t.Errorf("消息数量 = %d, 期望 1", len(loaded.Messages))
	}
}

// TestManager_Delete 测试删除会话
func TestManager_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.DefaultConfig()
	manager := NewManager(cfg, tmpDir)

	t.Run("删除存在的会话", func(t *testing.T) {
		session := &Session{
			Key:       "delete-test-session",
			CreatedAt: time.Now(),
			UpdatedAt:  time.Now(),
		}
		manager.Save(session)

		result := manager.Delete("delete-test-session")
		if !result {
			t.Error("Delete 应该返回 true")
		}

		loaded := manager.load("delete-test-session")
		if loaded != nil {
			t.Error("删除后加载应该返回 nil")
		}
	})

	t.Run("删除不存在的会话", func(t *testing.T) {
		result := manager.Delete("nonexistent-session")
		if result {
			t.Error("删除不存在的会话应该返回 false")
		}
	})
}

// TestManager_ListSessions 测试列出所有会话
func TestManager_ListSessions(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.DefaultConfig()
	manager := NewManager(cfg, tmpDir)

	session1 := &Session{
		Key:       "list-test-1",
		CreatedAt: time.Now().Add(-1 * time.Hour),
		UpdatedAt:  time.Now().Add(-30 * time.Minute),
	}
	session2 := &Session{
		Key:       "list-test-2",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		UpdatedAt:  time.Now(),
	}

	manager.Save(session1)
	manager.Save(session2)

	sessions := manager.ListSessions()

	if len(sessions) != 2 {
		t.Fatalf("会话数量 = %d, 期望 2", len(sessions))
	}

	if sessions[0]["key"] != "list-test-2" {
		t.Errorf("第一个会话 key = %v, 期望 list-test-2 (按更新时间排序)", sessions[0]["key"])
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

// TestManager_GetSessionPath 测试获取会话文件路径
func TestManager_GetSessionPath(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.DefaultConfig()
	manager := NewManager(cfg, tmpDir)

	path := manager.getSessionPath("test:session")

	if !filepath.IsAbs(path) {
		t.Errorf("路径应该是绝对路径: %q", path)
	}

	if filepath.Ext(path) != ".json" {
		t.Errorf("文件扩展名应该是 .json, 路径: %q", path)
	}

	if filepath.Dir(path) != manager.sessionsDir {
		t.Errorf("文件目录 = %q, 期望 %q", filepath.Dir(path), manager.sessionsDir)
	}
}

// TestSession_TokenUsageAccumulation 测试 Token 用量累加
func TestSession_TokenUsageAccumulation(t *testing.T) {
	session := &Session{
		Key:       "test-session",
		CreatedAt: time.Now(),
		UpdatedAt:  time.Now(),
	}

	usage1 := TokenUsage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150}
	usage2 := TokenUsage{PromptTokens: 200, CompletionTokens: 100, TotalTokens: 300}

	session.AddMessageWithTokenUsage("user", "消息1", usage1)
	session.AddMessageWithTokenUsage("assistant", "回复1", usage2)

	if session.TokenUsage.PromptTokens != 300 {
		t.Errorf("累计 PromptTokens = %d, 期望 300", session.TokenUsage.PromptTokens)
	}

	if session.TokenUsage.CompletionTokens != 150 {
		t.Errorf("累计 CompletionTokens = %d, 期望 150", session.TokenUsage.CompletionTokens)
	}

	if session.TokenUsage.TotalTokens != 450 {
		t.Errorf("累计 TotalTokens = %d, 期望 450", session.TokenUsage.TotalTokens)
	}
}

// TestManager_LoadNonexistentSession 测试加载不存在的会话
func TestManager_LoadNonexistentSession(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.DefaultConfig()
	manager := NewManager(cfg, tmpDir)

	session := manager.load("nonexistent-session")
	if session != nil {
		t.Error("加载不存在的会话应该返回 nil")
	}
}

// TestManager_SaveAndLoadWithAllFields 测试保存和加载包含所有字段的会话
func TestManager_SaveAndLoadWithAllFields(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.DefaultConfig()
	manager := NewManager(cfg, tmpDir)

	now := time.Now()
	original := &Session{
		Key: "full-test-session",
		Messages: []Message{
			{
				Role:      "user",
				Content:   "测试消息",
				Timestamp: now,
				TokenUsage: &TokenUsage{
					PromptTokens:     100,
					CompletionTokens: 50,
					TotalTokens:      150,
					ReasoningTokens:  20,
					CachedTokens:     10,
				},
			},
		},
		TokenUsage: TokenUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			ReasoningTokens:  20,
			CachedTokens:     10,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := manager.Save(original)
	if err != nil {
		t.Fatalf("Save 返回错误: %v", err)
	}

	loaded := manager.load("full-test-session")
	if loaded == nil {
		t.Fatal("加载会话失败")
	}

	if loaded.Key != original.Key {
		t.Errorf("Key = %q, 期望 %q", loaded.Key, original.Key)
	}

	if len(loaded.Messages) != 1 {
		t.Fatalf("消息数量 = %d, 期望 1", len(loaded.Messages))
	}

	if loaded.Messages[0].TokenUsage == nil {
		t.Fatal("消息 TokenUsage 不应该为 nil")
	}

	if loaded.Messages[0].TokenUsage.ReasoningTokens != 20 {
		t.Errorf("ReasoningTokens = %d, 期望 20", loaded.Messages[0].TokenUsage.ReasoningTokens)
	}

	if loaded.TokenUsage.CachedTokens != 10 {
		t.Errorf("Session CachedTokens = %d, 期望 10", loaded.TokenUsage.CachedTokens)
	}
}

// TestManager_DataDirCreation 测试数据目录自动创建
func TestManager_DataDirCreation(t *testing.T) {
	tmpDir := t.TempDir()
	nonexistentDir := filepath.Join(tmpDir, "nonexistent", "nested", "dir")

	cfg := config.DefaultConfig()
	manager := NewManager(cfg, nonexistentDir)

	expectedDir := filepath.Join(nonexistentDir, "sessions")
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Errorf("目录 %q 应该被自动创建", expectedDir)
	}

	if manager.sessionsDir != expectedDir {
		t.Errorf("sessionsDir = %q, 期望 %q", manager.sessionsDir, expectedDir)
	}
}
