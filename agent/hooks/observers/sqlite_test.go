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
	"github.com/weibaohui/nanobot-go/agent/models"
	"github.com/weibaohui/nanobot-go/config"
	"go.uber.org/zap"
)

// setupTestConfig 创建测试配置
func setupTestConfig(t *testing.T) *config.Config {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,
			},
		},
		Database: config.DatabaseConfig{
			Enabled:      true,
			DataDir:      ".data",
			DBName:       "events.db",
			MaxOpenConns: 1,
			MaxIdleConns: 1,
		},
	}
	return cfg
}

func TestNewSQLiteObserver(t *testing.T) {
	cfg := setupTestConfig(t)

	// 创建观察器
	obs, err := NewSQLiteObserver(cfg, zap.NewNop(), nil)
	if err != nil {
		t.Fatalf("创建 SQLiteObserver 失败: %v", err)
	}
	defer obs.Close()

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
	cfg := setupTestConfig(t)
	obs, err := NewSQLiteObserver(cfg, zap.NewNop(), nil)
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

	// 验证数据已插入（使用 ConversationService 查询）
	service := obs.GetConversationService()
	result, err := service.GetByTraceID(context.Background(), "trace-123")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("事件数量错误: got %d, want 1", len(result))
	}

	// 验证 role 和 content
	if result[0].Role != "user" {
		t.Errorf("role 错误: got %s, want user", result[0].Role)
	}
	if result[0].Content != "用户输入内容" {
		t.Errorf("content 错误: got %s, want 用户输入内容", result[0].Content)
	}
}

func TestSQLiteObserver_OnEvent_LLMCallEnd(t *testing.T) {
	cfg := setupTestConfig(t)
	obs, err := NewSQLiteObserver(cfg, zap.NewNop(), nil)
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
	service := obs.GetConversationService()
	result, err := service.GetByTraceID(context.Background(), "trace-123")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("事件数量错误: got %d, want 1", len(result))
	}

	// 验证 role 和 content
	if result[0].Role != "assistant" {
		t.Errorf("role 错误: got %s, want assistant", result[0].Role)
	}
	if result[0].Content != "AI 回复内容" {
		t.Errorf("content 错误: got %s, want AI 回复内容", result[0].Content)
	}
}

func TestSQLiteObserver_OnEvent_LLMCallEnd_WithToolCalls(t *testing.T) {
	cfg := setupTestConfig(t)
	obs, err := NewSQLiteObserver(cfg, zap.NewNop(), nil)
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
	service := obs.GetConversationService()
	result, err := service.GetByTraceID(context.Background(), "trace-123")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("事件数量错误: got %d, want 1", len(result))
	}
	if result[0].Role != "tool" {
		t.Errorf("role 错误: got %s, want tool", result[0].Role)
	}
}

func TestSQLiteObserver_OnEvent_LLMCallEnd_WithTokenUsage(t *testing.T) {
	cfg := setupTestConfig(t)
	obs, err := NewSQLiteObserver(cfg, zap.NewNop(), nil)
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
	service := obs.GetConversationService()
	result, err := service.GetByTraceID(context.Background(), "trace-123")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("事件数量错误: got %d, want 1", len(result))
	}

	if result[0].TokenUsage == nil {
		t.Fatal("TokenUsage 为 nil")
	}

	if result[0].TokenUsage.PromptTokens != 100 {
		t.Errorf("prompt_tokens 错误: got %d, want 100", result[0].TokenUsage.PromptTokens)
	}
	if result[0].TokenUsage.CompletionTokens != 50 {
		t.Errorf("completion_tokens 错误: got %d, want 50", result[0].TokenUsage.CompletionTokens)
	}
	if result[0].TokenUsage.TotalTokens != 150 {
		t.Errorf("total_tokens 错误: got %d, want 150", result[0].TokenUsage.TotalTokens)
	}
	if result[0].TokenUsage.ReasoningTokens != 20 {
		t.Errorf("reasoning_tokens 错误: got %d, want 20", result[0].TokenUsage.ReasoningTokens)
	}
	if result[0].TokenUsage.CachedTokens != 30 {
		t.Errorf("cached_tokens 错误: got %d, want 30", result[0].TokenUsage.CachedTokens)
	}
}

func TestSQLiteObserver_OnEvent_ToolCompleted(t *testing.T) {
	cfg := setupTestConfig(t)
	obs, err := NewSQLiteObserver(cfg, zap.NewNop(), nil)
	if err != nil {
		t.Fatalf("创建 SQLiteObserver 失败: %v", err)
	}
	defer obs.Close()

	// 创建带 session_key 的 context
	ctx := context.WithValue(context.Background(), "session_key", "session-001")

	// 创建工具完成事件
	event := events.NewToolCompletedEvent(
		"trace-123",
		"span-456",
		"parent-span-789",
		"read_file",
		"文件内容读取成功",
		true,
	)

	// 处理事件
	if err := obs.OnEvent(ctx, event); err != nil {
		t.Fatalf("处理事件失败: %v", err)
	}

	// 验证数据已插入
	service := obs.GetConversationService()
	result, err := service.GetByTraceID(context.Background(), "trace-123")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("事件数量错误: got %d, want 1", len(result))
	}

	// 验证 role 和 content
	if result[0].Role != "tool_result" {
		t.Errorf("role 错误: got %s, want tool_result", result[0].Role)
	}
	if result[0].Content != "read_file: 文件内容读取成功" {
		t.Errorf("content 错误: got %s", result[0].Content)
	}
}

func TestSQLiteObserver_OnEvent_IgnoredEvents(t *testing.T) {
	cfg := setupTestConfig(t)
	obs, err := NewSQLiteObserver(cfg, zap.NewNop(), nil)
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

	// 验证数据未插入（使用直接数据库查询）
	var count int64
	if err := obs.GetDBClient().DB().Model(&models.Event{}).Count(&count).Error; err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if count != 0 {
		t.Errorf("不应该存储 tool_used 事件: got %d, want 0", count)
	}
}

func TestSQLiteObserver_Filter(t *testing.T) {
	cfg := setupTestConfig(t)

	// 创建带过滤器的观察器
	filter := &observer.ObserverFilter{
		EventTypes: []events.EventType{events.EventPromptSubmitted},
	}
	obs, err := NewSQLiteObserver(cfg, zap.NewNop(), filter)
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
	cfg := setupTestConfig(t)
	obs, err := NewSQLiteObserver(cfg, zap.NewNop(), nil)
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
	var count int64
	if err := obs.GetDBClient().DB().
		Model(&models.Event{}).
		Where("event_type = ?", "prompt_submitted").
		Count(&count).Error; err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if count != 10 {
		t.Errorf("事件数量错误: got %d, want 10", count)
	}
}

func TestSQLiteObserver_Deduplication(t *testing.T) {
	cfg := setupTestConfig(t)
	obs, err := NewSQLiteObserver(cfg, zap.NewNop(), nil)
	if err != nil {
		t.Fatalf("创建 SQLiteObserver 失败: %v", err)
	}
	defer obs.Close()

	ctx := context.WithValue(context.Background(), "session_key", "session-001")

	// 第一次插入：无 TokenUsage
	event1 := events.NewLLMCallEndEvent(
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
	if err := obs.OnEvent(ctx, event1); err != nil {
		t.Fatalf("处理事件1失败: %v", err)
	}

	// 第二次插入：有 TokenUsage（相同 traceID、role、content）
	event2 := events.NewLLMCallEndEvent(
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
			},
		},
		100,
	)
	if err := obs.OnEvent(ctx, event2); err != nil {
		t.Fatalf("处理事件2失败: %v", err)
	}

	// 验证只有一条记录（去重）
	var count int64
	if err := obs.GetDBClient().DB().
		Model(&models.Event{}).
		Where("trace_id = ?", "trace-123").
		Count(&count).Error; err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if count != 1 {
		t.Errorf("去重失败，记录数量: got %d, want 1", count)
	}

	// 验证保留的是有 TokenUsage 的记录
	var record models.Event
	if err := obs.GetDBClient().DB().
		Where("trace_id = ?", "trace-123").
		First(&record).Error; err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if record.TotalTokens != 150 {
		t.Errorf("应保留有 TokenUsage 的记录: got total_tokens=%d, want 150", record.TotalTokens)
	}
}

func TestSQLiteObserver_DatabaseLocation(t *testing.T) {
	cfg := setupTestConfig(t)
	obs, err := NewSQLiteObserver(cfg, zap.NewNop(), nil)
	if err != nil {
		t.Fatalf("创建 SQLiteObserver 失败: %v", err)
	}
	defer obs.Close()

	// 验证数据库文件位于 workspace/.data 目录下
	expectedPath := filepath.Join(cfg.GetWorkspacePath(), ".data", "events.db")
	actualPath := obs.GetDBPath()

	if actualPath != expectedPath {
		t.Errorf("数据库路径错误: got %s, want %s", actualPath, expectedPath)
	}

	// 验证数据库文件已创建
	if _, err := os.Stat(actualPath); os.IsNotExist(err) {
		t.Error("数据库文件未创建")
	}
}