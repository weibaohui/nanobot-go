package repository

import (
	"context"
	"testing"
	"time"

	"github.com/weibaohui/nanobot-go/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建数据库失败: %v", err)
	}

	if err := db.AutoMigrate(&models.ConversationRecord{}); err != nil {
		t.Fatalf("迁移表失败: %v", err)
	}

	return db
}

func TestNewConversationRecordRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRecordRepository(db)
	if repo == nil {
		t.Error("创建仓储失败")
	}
}

func TestConversationRecordRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRecordRepository(db)

	record := &models.ConversationRecord{
		TraceID:   "trace-1",
		EventType: "prompt_submitted",
		Timestamp: time.Now(),
		SessionKey: "session-1",
		Role:      "user",
		Content:   "测试内容",
	}

	if err := repo.Create(context.Background(), record); err != nil {
		t.Fatalf("创建记录失败: %v", err)
	}

	if record.ID == 0 {
		t.Error("ID 应该被自动填充")
	}
}

func TestConversationRecordRepository_CreateBatch(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRecordRepository(db)

	records := []models.ConversationRecord{
		{TraceID: "trace-1", EventType: "prompt", Timestamp: time.Now(), Role: "user", Content: "内容1"},
		{TraceID: "trace-2", EventType: "prompt", Timestamp: time.Now(), Role: "user", Content: "内容2"},
		{TraceID: "trace-3", EventType: "prompt", Timestamp: time.Now(), Role: "assistant", Content: "内容3"},
	}

	if err := repo.CreateBatch(context.Background(), records); err != nil {
		t.Fatalf("批量创建失败: %v", err)
	}

	count, err := repo.Count(context.Background())
	if err != nil {
		t.Fatalf("统计失败: %v", err)
	}
	if count != 3 {
		t.Errorf("记录数量错误: got %d, want 3", count)
	}
}

func TestConversationRecordRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRecordRepository(db)

	record := &models.ConversationRecord{
		TraceID:   "trace-1",
		EventType: "prompt",
		Timestamp: time.Now(),
		Role:      "user",
		Content:   "测试内容",
	}
	repo.Create(context.Background(), record)

	found, err := repo.FindByID(context.Background(), record.ID)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if found.Content != "测试内容" {
		t.Errorf("内容错误: got %s, want 测试内容", found.Content)
	}

	// 测试不存在的ID
	_, err = repo.FindByID(context.Background(), 99999)
	if err == nil {
		t.Error("不存在的ID应该返回错误")
	}
}

func TestConversationRecordRepository_FindByTraceID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRecordRepository(db)

	// 创建测试数据
	repo.Create(context.Background(), &models.ConversationRecord{
		TraceID:   "trace-1",
		EventType: "prompt",
		Timestamp: time.Now(),
		Role:      "user",
		Content:   "内容1",
	})
	repo.Create(context.Background(), &models.ConversationRecord{
		TraceID:   "trace-1",
		EventType: "response",
		Timestamp: time.Now().Add(time.Second),
		Role:      "assistant",
		Content:   "内容2",
	})
	repo.Create(context.Background(), &models.ConversationRecord{
		TraceID:   "trace-2",
		EventType: "prompt",
		Timestamp: time.Now(),
		Role:      "user",
		Content:   "内容3",
	})

	records, err := repo.FindByTraceID(context.Background(), "trace-1")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("记录数量错误: got %d, want 2", len(records))
	}

	// 验证按时间排序（升序）
	if records[0].Content != "内容1" {
		t.Error("应该按时间升序排列")
	}
}

func TestConversationRecordRepository_FindBySessionKey(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRecordRepository(db)

	// 创建多个会话的数据
	repo.Create(context.Background(), &models.ConversationRecord{
		TraceID:    "trace-1",
		EventType:  "prompt",
		Timestamp:  time.Now(),
		SessionKey: "session-1",
		Role:       "user",
		Content:    "会话1内容",
	})
	repo.Create(context.Background(), &models.ConversationRecord{
		TraceID:    "trace-2",
		EventType:  "prompt",
		Timestamp:  time.Now(),
		SessionKey: "session-2",
		Role:       "user",
		Content:    "会话2内容",
	})

	// 测试无选项查询
	records, err := repo.FindBySessionKey(context.Background(), "session-1", nil)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("记录数量错误: got %d, want 1", len(records))
	}

	// 测试带选项查询
	opts := &models.QueryOptions{
		Roles:   []string{"user"},
		Limit:   10,
		OrderBy: "timestamp",
		Order:   "DESC",
	}
	records, err = repo.FindBySessionKey(context.Background(), "session-1", opts)
	if err != nil {
		t.Fatalf("带选项查询失败: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("带选项记录数量错误: got %d, want 1", len(records))
	}
}

func TestConversationRecordRepository_FindByTimeRange(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRecordRepository(db)

	now := time.Now()

	// 创建不同时间的记录
	repo.Create(context.Background(), &models.ConversationRecord{
		TraceID:   "trace-1",
		EventType: "prompt",
		Timestamp: now.Add(-2 * time.Hour),
		Role:      "user",
		Content:   "旧内容",
	})
	repo.Create(context.Background(), &models.ConversationRecord{
		TraceID:   "trace-2",
		EventType: "prompt",
		Timestamp: now,
		Role:      "user",
		Content:   "新内容",
	})

	// 查询最近1小时内的记录
	startTime := now.Add(-1 * time.Hour)
	endTime := now.Add(1 * time.Hour)

	records, err := repo.FindByTimeRange(context.Background(), startTime, endTime, nil)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("记录数量错误: got %d, want 1", len(records))
	}
	if records[0].Content != "新内容" {
		t.Error("应该只返回时间范围内的记录")
	}
}

func TestConversationRecordRepository_FindByTraceIDRoleAndContent(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRecordRepository(db)

	repo.Create(context.Background(), &models.ConversationRecord{
		TraceID:   "trace-1",
		EventType: "prompt",
		Timestamp: time.Now(),
		Role:      "user",
		Content:   "相同内容",
	})
	repo.Create(context.Background(), &models.ConversationRecord{
		TraceID:   "trace-1",
		EventType: "prompt",
		Timestamp: time.Now(),
		Role:      "user",
		Content:   "相同内容",
		TotalTokens: 100,
	})
	repo.Create(context.Background(), &models.ConversationRecord{
		TraceID:   "trace-1",
		EventType: "prompt",
		Timestamp: time.Now(),
		Role:      "assistant",
		Content:   "不同内容",
	})

	records, err := repo.FindByTraceIDRoleAndContent(context.Background(), "trace-1", "user", "相同内容")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("记录数量错误: got %d, want 2", len(records))
	}
}

func TestConversationRecordRepository_CountBySessionKey(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRecordRepository(db)

	repo.Create(context.Background(), &models.ConversationRecord{
		TraceID:    "trace-1",
		EventType:  "prompt",
		Timestamp:  time.Now(),
		SessionKey: "session-1",
		Role:       "user",
		Content:    "内容1",
	})
	repo.Create(context.Background(), &models.ConversationRecord{
		TraceID:    "trace-2",
		EventType:  "prompt",
		Timestamp:  time.Now(),
		SessionKey: "session-1",
		Role:       "assistant",
		Content:    "内容2",
	})
	repo.Create(context.Background(), &models.ConversationRecord{
		TraceID:    "trace-3",
		EventType:  "prompt",
		Timestamp:  time.Now(),
		SessionKey: "session-2",
		Role:       "user",
		Content:    "内容3",
	})

	count, err := repo.CountBySessionKey(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("统计失败: %v", err)
	}
	if count != 2 {
		t.Errorf("数量错误: got %d, want 2", count)
	}
}

func TestConversationRecordRepository_CountByTimeRange(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRecordRepository(db)

	now := time.Now()

	repo.Create(context.Background(), &models.ConversationRecord{
		TraceID:   "trace-1",
		EventType: "prompt",
		Timestamp: now.Add(-2 * time.Hour),
		Role:      "user",
		Content:   "旧内容",
	})
	repo.Create(context.Background(), &models.ConversationRecord{
		TraceID:   "trace-2",
		EventType: "prompt",
		Timestamp: now,
		Role:      "user",
		Content:   "新内容",
	})

	count, err := repo.CountByTimeRange(context.Background(), now.Add(-1*time.Hour), now.Add(1*time.Hour))
	if err != nil {
		t.Fatalf("统计失败: %v", err)
	}
	if count != 1 {
		t.Errorf("数量错误: got %d, want 1", count)
	}
}

func TestConversationRecordRepository_DeleteByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRecordRepository(db)

	record := &models.ConversationRecord{
		TraceID:   "trace-1",
		EventType: "prompt",
		Timestamp: time.Now(),
		Role:      "user",
		Content:   "内容",
	}
	repo.Create(context.Background(), record)

	if err := repo.DeleteByID(context.Background(), record.ID); err != nil {
		t.Fatalf("删除失败: %v", err)
	}

	_, err := repo.FindByID(context.Background(), record.ID)
	if err == nil {
		t.Error("记录应该已被删除")
	}
}

func TestConversationRecordRepository_Count(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRecordRepository(db)

	repo.Create(context.Background(), &models.ConversationRecord{
		TraceID:   "trace-1",
		EventType: "prompt",
		Timestamp: time.Now(),
		Role:      "user",
		Content:   "内容1",
	})
	repo.Create(context.Background(), &models.ConversationRecord{
		TraceID:   "trace-2",
		EventType: "prompt",
		Timestamp: time.Now(),
		Role:      "user",
		Content:   "内容2",
	})

	count, err := repo.Count(context.Background())
	if err != nil {
		t.Fatalf("统计失败: %v", err)
	}
	if count != 2 {
		t.Errorf("数量错误: got %d, want 2", count)
	}
}
