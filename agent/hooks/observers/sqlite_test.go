package observers

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/agent/hooks/observer"
	"go.uber.org/zap"
)

func TestNewSQLiteObserver(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	// 创建观察器
	obs, err := NewSQLiteObserver(tmpDir, zap.NewNop(), nil)
	if err != nil {
		t.Fatalf("创建 SQLiteObserver 失败: %v", err)
	}
	defer obs.Close()

	// 验证数据库文件已创建
	dbPath := filepath.Join(tmpDir, "events.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("数据库文件未创建")
	}

	// 验证观察器名称
	if obs.Name() != "sqlite" {
		t.Errorf("观察器名称错误: got %s, want sqlite", obs.Name())
	}

	// 验证观察器默认启用
	if !obs.Enabled() {
		t.Error("观察器应该默认启用")
	}
}

func TestSQLiteObserver_OnEvent_PromptSubmitted(t *testing.T) {
	tmpDir := t.TempDir()
	obs, err := NewSQLiteObserver(tmpDir, zap.NewNop(), nil)
	if err != nil {
		t.Fatalf("创建 SQLiteObserver 失败: %v", err)
	}
	defer obs.Close()

	// 创建 Prompt 提交事件
	event := events.NewPromptSubmittedEvent(
		"trace-123",
		"span-456",
		"parent-span-789",
		"用户输入内容",
		nil,
		"session-001",
	)

	// 处理事件
	if err := obs.OnEvent(context.Background(), event); err != nil {
		t.Fatalf("处理事件失败: %v", err)
	}

	// 验证数据已插入
	var count int
	row := obs.db.QueryRow("SELECT COUNT(*) FROM events WHERE event_type = ?", "prompt_submitted")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if count != 1 {
		t.Errorf("事件数量错误: got %d, want 1", count)
	}

	// 验证 role 和 content
	var role, content string
	row = obs.db.QueryRow("SELECT role, content FROM events WHERE event_type = ?", "prompt_submitted")
	if err := row.Scan(&role, &content); err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if role != "user" {
		t.Errorf("role 错误: got %s, want user", role)
	}
	if content != "用户输入内容" {
		t.Errorf("content 错误: got %s, want 用户输入内容", content)
	}
}

func TestSQLiteObserver_OnEvent_LLMCallEnd(t *testing.T) {
	tmpDir := t.TempDir()
	obs, err := NewSQLiteObserver(tmpDir, zap.NewNop(), nil)
	if err != nil {
		t.Fatalf("创建 SQLiteObserver 失败: %v", err)
	}
	defer obs.Close()

	// 创建带 session_key 的 context
	ctx := context.WithValue(context.Background(), "session_key", "session-001")

	// 创建 LLM 调用结束事件
	event := events.NewLLMCallEndEvent(
		"trace-123",
		"span-456",
		"parent-span-789",
		&callbacks.RunInfo{Component: "LLM", Name: "test-model"},
		&model.CallbackOutput{
			Message: &schema.Message{
				Content: "AI 回复内容",
			},
		},
		100,
	)

	// 处理事件
	if err := obs.OnEvent(ctx, event); err != nil {
		t.Fatalf("处理事件失败: %v", err)
	}

	// 验证数据已插入
	var count int
	row := obs.db.QueryRow("SELECT COUNT(*) FROM events WHERE event_type = ?", "llm_call_end")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if count != 1 {
		t.Errorf("事件数量错误: got %d, want 1", count)
	}

	// 验证 role 和 content
	var role, content string
	row = obs.db.QueryRow("SELECT role, content FROM events WHERE event_type = ?", "llm_call_end")
	if err := row.Scan(&role, &content); err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if role != "assistant" {
		t.Errorf("role 错误: got %s, want assistant", role)
	}
	if content != "AI 回复内容" {
		t.Errorf("content 错误: got %s, want AI 回复内容", content)
	}
}

func TestSQLiteObserver_OnEvent_LLMCallEnd_WithToolCalls(t *testing.T) {
	tmpDir := t.TempDir()
	obs, err := NewSQLiteObserver(tmpDir, zap.NewNop(), nil)
	if err != nil {
		t.Fatalf("创建 SQLiteObserver 失败: %v", err)
	}
	defer obs.Close()

	// 创建带 session_key 的 context
	ctx := context.WithValue(context.Background(), "session_key", "session-001")

	// 创建带工具调用的 LLM 调用结束事件
	event := events.NewLLMCallEndEvent(
		"trace-123",
		"span-456",
		"parent-span-789",
		&callbacks.RunInfo{Component: "LLM", Name: "test-model"},
		&model.CallbackOutput{
			Message: &schema.Message{
				ToolCalls: []schema.ToolCall{
					{
						Function: schema.FunctionCall{
							Name:      "search",
							Arguments: `{"query": "test"}`,
						},
					},
				},
			},
		},
		100,
	)

	// 处理事件
	if err := obs.OnEvent(ctx, event); err != nil {
		t.Fatalf("处理事件失败: %v", err)
	}

	// 验证 role 为 tool
	var role string
	row := obs.db.QueryRow("SELECT role FROM events WHERE event_type = ?", "llm_call_end")
	if err := row.Scan(&role); err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if role != "tool" {
		t.Errorf("role 错误: got %s, want tool", role)
	}
}

func TestSQLiteObserver_OnEvent_LLMCallEnd_WithTokenUsage(t *testing.T) {
	tmpDir := t.TempDir()
	obs, err := NewSQLiteObserver(tmpDir, zap.NewNop(), nil)
	if err != nil {
		t.Fatalf("创建 SQLiteObserver 失败: %v", err)
	}
	defer obs.Close()

	// 创建带 session_key 的 context
	ctx := context.WithValue(context.Background(), "session_key", "session-001")

	// 创建带 Token Usage 的 LLM 调用结束事件
	event := events.NewLLMCallEndEvent(
		"trace-123",
		"span-456",
		"parent-span-789",
		&callbacks.RunInfo{Component: "LLM", Name: "test-model"},
		&model.CallbackOutput{
			Message: &schema.Message{
				Content: "AI 回复内容",
			},
			TokenUsage: &model.TokenUsage{
				PromptTokens:     100,
				CompletionTokens: 50,
				TotalTokens:      150,
				CompletionTokensDetails: model.CompletionTokensDetails{
					ReasoningTokens: 20,
				},
				PromptTokenDetails: model.PromptTokenDetails{
					CachedTokens: 30,
				},
			},
		},
		100,
	)

	// 处理事件
	if err := obs.OnEvent(ctx, event); err != nil {
		t.Fatalf("处理事件失败: %v", err)
	}

	// 验证 Token Usage 字段
	var promptTokens, completionTokens, totalTokens, reasoningTokens, cachedTokens int
	row := obs.db.QueryRow(`SELECT prompt_tokens, completion_tokens, total_tokens, reasoning_tokens, cached_tokens
		FROM events WHERE event_type = ?`, "llm_call_end")
	if err := row.Scan(&promptTokens, &completionTokens, &totalTokens, &reasoningTokens, &cachedTokens); err != nil {
		t.Fatalf("查询失败: %v", err)
	}

	if promptTokens != 100 {
		t.Errorf("prompt_tokens 错误: got %d, want 100", promptTokens)
	}
	if completionTokens != 50 {
		t.Errorf("completion_tokens 错误: got %d, want 50", completionTokens)
	}
	if totalTokens != 150 {
		t.Errorf("total_tokens 错误: got %d, want 150", totalTokens)
	}
	if reasoningTokens != 20 {
		t.Errorf("reasoning_tokens 错误: got %d, want 20", reasoningTokens)
	}
	if cachedTokens != 30 {
		t.Errorf("cached_tokens 错误: got %d, want 30", cachedTokens)
	}
}

func TestSQLiteObserver_OnEvent_IgnoredEvents(t *testing.T) {
	tmpDir := t.TempDir()
	obs, err := NewSQLiteObserver(tmpDir, zap.NewNop(), nil)
	if err != nil {
		t.Fatalf("创建 SQLiteObserver 失败: %v", err)
	}
	defer obs.Close()

	// 创建工具使用事件（应该被忽略）
	event := events.NewToolUsedEvent(
		"trace-123",
		"span-456",
		"parent-span-789",
		"test_tool",
		`{"arg1": "value1"}`,
	)

	// 处理事件
	if err := obs.OnEvent(context.Background(), event); err != nil {
		t.Fatalf("处理事件失败: %v", err)
	}

	// 验证数据未插入
	var count int
	row := obs.db.QueryRow("SELECT COUNT(*) FROM events")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if count != 0 {
		t.Errorf("不应该存储 tool_used 事件: got %d, want 0", count)
	}
}

func TestSQLiteObserver_Filter(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建带过滤器的观察器
	filter := &observer.ObserverFilter{
		EventTypes: []events.EventType{events.EventPromptSubmitted},
	}
	obs, err := NewSQLiteObserver(tmpDir, zap.NewNop(), filter)
	if err != nil {
		t.Fatalf("创建 SQLiteObserver 失败: %v", err)
	}
	defer obs.Close()

	// 验证过滤器
	if !obs.ShouldNotify(events.EventPromptSubmitted, "", "") {
		t.Error("过滤器应该允许 prompt_submitted 事件")
	}
	if obs.ShouldNotify(events.EventLLMCallEnd, "", "") {
		t.Error("过滤器不应该允许 llm_call_end 事件")
	}
}

func TestSQLiteObserver_ConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()
	obs, err := NewSQLiteObserver(tmpDir, zap.NewNop(), nil)
	if err != nil {
		t.Fatalf("创建 SQLiteObserver 失败: %v", err)
	}
	defer obs.Close()

	// 并发写入测试
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			event := events.NewPromptSubmittedEvent(
				"trace-"+string(rune('0'+id)),
				"span-"+string(rune('0'+id)),
				"parent-span",
				"用户输入"+string(rune('0'+id)),
				nil,
				"session-001",
			)
			if err := obs.OnEvent(context.Background(), event); err != nil {
				t.Errorf("并发写入失败: %v", err)
			}
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证数据已插入
	var count int
	row := obs.db.QueryRow("SELECT COUNT(*) FROM events WHERE event_type = ?", "prompt_submitted")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if count != 10 {
		t.Errorf("事件数量错误: got %d, want 10", count)
	}
}