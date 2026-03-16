package observers

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/observer"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/trace"
	"github.com/weibaohui/nanobot-go/internal/database"
	"github.com/weibaohui/nanobot-go/internal/service/conversation"
	"go.uber.org/zap"
)

func setupTestDeps(t *testing.T) (*database.Client, conversation.Repository, conversation.Service) {
	tmpDir := t.TempDir()

	dbConfig := &database.Config{
		DataDir:      tmpDir,
		DBName:       "test.db",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	}

	dbClient, err := database.NewClient(dbConfig)
	if err != nil {
		t.Fatalf("创建数据库客户端失败: %v", err)
	}

	if err := dbClient.InitSchema(); err != nil {
		dbClient.Close()
		t.Fatalf("初始化表结构失败: %v", err)
	}

	repo := conversation.NewRepository(dbClient.DB())
	convService := conversation.NewService(repo)

	return dbClient, repo, convService
}

func createTestObserver(t *testing.T) (*SQLiteObserver, *database.Client, conversation.Repository, conversation.Service) {
	dbClient, repo, convService := setupTestDeps(t)
	obs := NewSQLiteObserver(zap.NewNop(), nil,
		WithDBClient(dbClient),
		WithDedupRepository(repo),
		WithConversationCreator(convService),
	)
	return obs, dbClient, repo, convService
}

func TestNewSQLiteObserver(t *testing.T) {
	obs, dbClient, _, _ := createTestObserver(t)
	defer dbClient.Close()

	if obs.Name() != "sqlite" {
		t.Errorf("观察器名称错误: got %s, want sqlite", obs.Name())
	}

	if !obs.Enabled() {
		t.Error("观察器应该默认启用")
	}
}

func TestSQLiteObserver_PromptSubmitted(t *testing.T) {
	obs, dbClient, _, convService := createTestObserver(t)
	defer dbClient.Close()

	event := events.NewPromptSubmittedEvent("trace-1", "span-1", "", "用户输入", nil, "session-1")
	if err := obs.OnEvent(context.Background(), event); err != nil {
		t.Fatalf("处理事件失败: %v", err)
	}

	result, err := convService.GetByTraceID(context.Background(), "trace-1")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("记录数量错误: got %d, want 1", len(result))
	}
	if result[0].Role != "user" {
		t.Errorf("role 错误: got %s, want user", result[0].Role)
	}
}

func TestSQLiteObserver_LLMCallEnd(t *testing.T) {
	obs, dbClient, _, convService := createTestObserver(t)
	defer dbClient.Close()

	ctx := trace.WithSessionKey(context.Background(), "session-1")
	event := events.NewLLMCallEndEvent("trace-1", "span-1", "",
		&callbacks.RunInfo{Component: "LLM"},
		&model.CallbackOutput{Message: &schema.Message{Content: "AI回复"}},
		100,
	)

	if err := obs.OnEvent(ctx, event); err != nil {
		t.Fatalf("处理事件失败: %v", err)
	}

	result, err := convService.GetByTraceID(context.Background(), "trace-1")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("记录数量错误: got %d, want 1", len(result))
	}
	if result[0].Role != "assistant" {
		t.Errorf("role 错误: got %s, want assistant", result[0].Role)
	}
}

func TestSQLiteObserver_ToolCompleted(t *testing.T) {
	obs, dbClient, _, convService := createTestObserver(t)
	defer dbClient.Close()

	ctx := trace.WithSessionKey(context.Background(), "session-1")
	event := events.NewToolCompletedEvent("trace-1", "span-1", "", "read_file", "文件内容", true)

	if err := obs.OnEvent(ctx, event); err != nil {
		t.Fatalf("处理事件失败: %v", err)
	}

	result, err := convService.GetByTraceID(context.Background(), "trace-1")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("记录数量错误: got %d, want 1", len(result))
	}
	if result[0].Role != "tool_result" {
		t.Errorf("role 错误: got %s, want tool_result", result[0].Role)
	}
}

func TestSQLiteObserver_TokenUsage(t *testing.T) {
	obs, dbClient, _, convService := createTestObserver(t)
	defer dbClient.Close()

	ctx := trace.WithSessionKey(context.Background(), "session-1")
	event := events.NewLLMCallEndEvent("trace-1", "span-1", "",
		&callbacks.RunInfo{Component: "LLM"},
		&model.CallbackOutput{
			Message: &schema.Message{Content: "AI回复"},
			TokenUsage: &model.TokenUsage{
				PromptTokens:     100,
				CompletionTokens: 50,
				TotalTokens:      150,
			},
		},
		100,
	)

	if err := obs.OnEvent(ctx, event); err != nil {
		t.Fatalf("处理事件失败: %v", err)
	}

	result, err := convService.GetByTraceID(context.Background(), "trace-1")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if result[0].TokenUsage == nil || result[0].TokenUsage.TotalTokens != 150 {
		t.Errorf("TokenUsage 错误")
	}
}

func TestSQLiteObserver_Filter(t *testing.T) {
	_, dbClient, _, _ := createTestObserver(t)
	defer dbClient.Close()

	filter := &observer.ObserverFilter{
		EventTypes: []events.EventType{events.EventPromptSubmitted},
	}
	obsFiltered := NewSQLiteObserver(zap.NewNop(), filter,
		WithDBClient(dbClient),
	)

	if !obsFiltered.ShouldNotify(events.EventPromptSubmitted, "", "") {
		t.Error("过滤器应该允许 prompt_submitted 事件")
	}
	if obsFiltered.ShouldNotify(events.EventLLMCallEnd, "", "") {
		t.Error("过滤器不应该允许 llm_call_end 事件")
	}
}

func TestSQLiteObserver_DatabaseLocation(t *testing.T) {
	tmpDir := t.TempDir()
	dbConfig := &database.Config{
		DataDir:      filepath.Join(tmpDir, ".data"),
		DBName:       "test.db",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	}

	dbClient, err := database.NewClient(dbConfig)
	if err != nil {
		t.Fatalf("创建数据库客户端失败: %v", err)
	}
	defer dbClient.Close()

	if err := dbClient.InitSchema(); err != nil {
		t.Fatalf("初始化表结构失败: %v", err)
	}

	repo := conversation.NewRepository(dbClient.DB())
	convService := conversation.NewService(repo)

	obs := NewSQLiteObserver(zap.NewNop(), nil,
		WithDBClient(dbClient),
		WithDedupRepository(repo),
		WithConversationCreator(convService),
	)

	expectedPath := filepath.Join(tmpDir, ".data", "test.db")
	if obs.GetDBPath() != expectedPath {
		t.Errorf("数据库路径错误: got %s, want %s", obs.GetDBPath(), expectedPath)
	}

	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("数据库文件未创建")
	}
}
