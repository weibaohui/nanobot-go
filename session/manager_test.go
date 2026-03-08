package session

import (
	"context"
	"testing"
	"time"

	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/internal/models"
	"go.uber.org/zap"
)

// TestSession_AddMessage 测试添加消息到会话
func TestSession_AddMessage(t *testing.T) {
	session := &Session{
		Key:       "test-session",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
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

// TestSession_Clear 测试清空会话消息
func TestSession_Clear(t *testing.T) {
	session := &Session{
		Key: "test-session",
		Messages: []Message{
			{Role: "user", Content: "消息1"},
			{Role: "assistant", Content: "回复1"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
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
	logger := zap.NewNop()

	manager := NewManager(cfg, logger, tmpDir, nil)
	if manager == nil {
		t.Fatal("NewManager 返回 nil")
	}

	if manager.cache == nil {
		t.Error("cache 不应该为 nil")
	}
}

// TestManager_GetOrCreate 测试获取或创建会话
func TestManager_GetOrCreate(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.DefaultConfig()
	manager := NewManager(cfg, zap.NewNop(), tmpDir, nil)

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

// TestSession_AddMessageWithTrace 测试添加带链路追踪信息的消息
func TestSession_AddMessageWithTrace(t *testing.T) {
	session := &Session{
		Key:       "test-session",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	session.AddMessageWithTrace("user", "测试消息", "trace-123", "span-456", "parent-span-789")

	if len(session.Messages) != 1 {
		t.Fatalf("消息数量 = %d, 期望 1", len(session.Messages))
	}

	if session.Messages[0].TraceID != "trace-123" {
		t.Errorf("TraceID = %q, 期望 trace-123", session.Messages[0].TraceID)
	}

	if session.Messages[0].SpanID != "span-456" {
		t.Errorf("SpanID = %q, 期望 span-456", session.Messages[0].SpanID)
	}

	if session.Messages[0].ParentSpanID != "parent-span-789" {
		t.Errorf("ParentSpanID = %q, 期望 parent-span-789", session.Messages[0].ParentSpanID)
	}
}

// mockConvRepo 用于测试的模拟对话记录仓库
type mockConvRepo struct {
	records []models.ConversationRecord
}

func (m *mockConvRepo) FindBySessionKey(ctx context.Context, sessionKey string, opts *models.QueryOptions) ([]models.ConversationRecord, error) {
	var result []models.ConversationRecord
	for _, r := range m.records {
		if r.SessionKey == sessionKey {
			result = append(result, r)
		}
	}
	return result, nil
}

// TestManager_GetHistory 测试从仓库获取历史记录
func TestManager_GetHistory(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.DefaultConfig()

	// 创建模拟仓库数据
	now := time.Now()
	mockRepo := &mockConvRepo{
		records: []models.ConversationRecord{
			{
				SessionKey: "test-session",
				Role:       "user",
				Content:    "你好",
				Timestamp:  now.Add(-5 * time.Minute),
			},
			{
				SessionKey: "test-session",
				Role:       "assistant",
				Content:    "你好！有什么可以帮助你的？",
				Timestamp:  now.Add(-4 * time.Minute),
			},
		},
	}

	manager := NewManager(cfg, zap.NewNop(), tmpDir, mockRepo)

	history := manager.GetHistory(context.Background(), "test-session", 10)

	if len(history) != 2 {
		t.Fatalf("历史记录数量 = %d, 期望 2", len(history))
	}

	if history[0]["role"] != "user" {
		t.Errorf("第一条记录角色 = %v, 期望 user", history[0]["role"])
	}

	if history[1]["role"] != "assistant" {
		t.Errorf("第二条记录角色 = %v, 期望 assistant", history[1]["role"])
	}
}

// TestManager_GetHistory_NoRepo 测试没有仓库时返回空历史
func TestManager_GetHistory_NoRepo(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.DefaultConfig()
	manager := NewManager(cfg, zap.NewNop(), tmpDir, nil)

	history := manager.GetHistory(context.Background(), "test-session", 10)

	if history != nil {
		t.Errorf("没有仓库时应该返回 nil, 得到 %v", history)
	}
}

// TestManager_GetHistory_FilterByTime 测试时间过滤
func TestManager_GetHistory_FilterByTime(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.DefaultConfig()

	// 创建模拟仓库数据，包含旧记录
	now := time.Now()
	mockRepo := &mockConvRepo{
		records: []models.ConversationRecord{
			{
				SessionKey: "test-session",
				Role:       "user",
				Content:    "3小时前的消息",
				Timestamp:  now.Add(-3 * time.Hour), // 超过2小时，应该被过滤
			},
			{
				SessionKey: "test-session",
				Role:       "user",
				Content:    "1小时前的消息",
				Timestamp:  now.Add(-1 * time.Hour), // 在2小时内，应该保留
			},
		},
	}

	manager := NewManager(cfg, zap.NewNop(), tmpDir, mockRepo)

	history := manager.GetHistory(context.Background(), "test-session", 10)

	if len(history) != 1 {
		t.Fatalf("历史记录数量 = %d, 期望 1 (只保留2小时内的)", len(history))
	}

	if history[0]["content"] != "1小时前的消息" {
		t.Errorf("内容 = %v, 期望 '1小时前的消息'", history[0]["content"])
	}
}
