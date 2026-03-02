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

	records, err := repo.FindByTraceID(context.Background(), "trace-1")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("记录数量错误: got %d, want 2", len(records))
	}
}

func TestConversationRecordRepository_FindBySessionKey(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRecordRepository(db)

	repo.Create(context.Background(), &models.ConversationRecord{
		TraceID:    "trace-1",
		EventType:  "prompt",
		Timestamp:  time.Now(),
		SessionKey: "session-1",
		Role:       "user",
		Content:    "内容",
	})

	records, err := repo.FindBySessionKey(context.Background(), "session-1", nil)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("记录数量错误: got %d, want 1", len(records))
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

	records, err := repo.FindByTraceIDRoleAndContent(context.Background(), "trace-1", "user", "相同内容")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("记录数量错误: got %d, want 2", len(records))
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
