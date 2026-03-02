package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/weibaohui/nanobot-go/agent/models"
)

// setupTestDB 创建测试数据库连接
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 自动迁移表结构
	err = db.AutoMigrate(&models.Event{})
	require.NoError(t, err)

	return db
}

// createTestEvent 创建测试事件
func createTestEvent(traceID, sessionKey, role, content string) models.Event {
	return models.Event{
		TraceID:          traceID,
		EventType:        "test",
		Timestamp:        time.Now(),
		SessionKey:       sessionKey,
		Role:             role,
		Content:          content,
		PromptTokens:     10,
		CompletionTokens: 20,
		TotalTokens:      30,
		ReasoningTokens:  5,
		CachedTokens:     0,
	}
}

// TestNewEventRepository 测试创建 EventRepository
func TestNewEventRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEventRepository(db)
	assert.NotNil(t, repo)
}

// TestCreate 测试创建事件
func TestCreate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEventRepository(db)
	ctx := context.Background()

	event := createTestEvent("trace-001", "session-001", "user", "Hello")
	err := repo.Create(ctx, &event)
	require.NoError(t, err)
	assert.NotZero(t, event.ID)
}

// TestCreateBatch 测试批量创建事件
func TestCreateBatch(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEventRepository(db)
	ctx := context.Background()

	events := []models.Event{
		createTestEvent("trace-001", "session-001", "user", "Hello"),
		createTestEvent("trace-001", "session-001", "assistant", "Hi"),
		createTestEvent("trace-002", "session-002", "user", "Test"),
	}

	err := repo.CreateBatch(ctx, events)
	require.NoError(t, err)

	// 验证数据
	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

// TestFindByID 测试根据 ID 查找事件
func TestFindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEventRepository(db)
	ctx := context.Background()

	// 创建测试数据
	event := createTestEvent("trace-001", "session-001", "user", "Hello")
	err := repo.Create(ctx, &event)
	require.NoError(t, err)

	// 查询
	found, err := repo.FindByID(ctx, event.ID)
	require.NoError(t, err)
	assert.Equal(t, event.ID, found.ID)
	assert.Equal(t, event.TraceID, found.TraceID)
	assert.Equal(t, event.SessionKey, found.SessionKey)
}

// TestFindByIDNotFound 测试查找不存在的 ID
func TestFindByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEventRepository(db)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, 999)
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}

// TestFindByTraceID 测试根据 TraceID 查找事件
func TestFindByTraceID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEventRepository(db)
	ctx := context.Background()

	// 创建测试数据
	events := []models.Event{
		createTestEvent("trace-001", "session-001", "user", "Hello"),
		createTestEvent("trace-001", "session-001", "assistant", "Hi"),
		createTestEvent("trace-002", "session-002", "user", "Test"),
	}
	for i := range events {
		err := repo.Create(ctx, &events[i])
		require.NoError(t, err)
	}

	// 查询
	found, err := repo.FindByTraceID(ctx, "trace-001")
	require.NoError(t, err)
	assert.Len(t, found, 2)
	assert.Equal(t, "trace-001", found[0].TraceID)
}

// TestFindBySessionKey 测试根据 SessionKey 查找事件
func TestFindBySessionKey(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEventRepository(db)
	ctx := context.Background()

	// 创建测试数据
	events := []models.Event{
		createTestEvent("trace-001", "session-001", "user", "Hello"),
		createTestEvent("trace-002", "session-001", "assistant", "Hi"),
		createTestEvent("trace-003", "session-002", "user", "Test"),
	}
	for i := range events {
		err := repo.Create(ctx, &events[i])
		require.NoError(t, err)
	}

	// 查询
	found, err := repo.FindBySessionKey(ctx, "session-001", nil)
	require.NoError(t, err)
	assert.Len(t, found, 2)
}

// TestFindBySessionKeyWithPagination 测试带分页的 SessionKey 查询
func TestFindBySessionKeyWithPagination(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEventRepository(db)
	ctx := context.Background()

	// 创建 5 条测试数据
	for i := 0; i < 5; i++ {
		event := createTestEvent("trace-001", "session-001", "user", "Message")
		err := repo.Create(ctx, &event)
		require.NoError(t, err)
	}

	// 第一页：2 条
	found, err := repo.FindBySessionKey(ctx, "session-001", &QueryOptions{
		Limit:  2,
		Offset: 0,
	})
	require.NoError(t, err)
	assert.Len(t, found, 2)

	// 第二页：2 条
	found, err = repo.FindBySessionKey(ctx, "session-001", &QueryOptions{
		Limit:  2,
		Offset: 2,
	})
	require.NoError(t, err)
	assert.Len(t, found, 2)

	// 第三页：1 条
	found, err = repo.FindBySessionKey(ctx, "session-001", &QueryOptions{
		Limit:  2,
		Offset: 4,
	})
	require.NoError(t, err)
	assert.Len(t, found, 1)
}

// TestFindByTimeRange 测试根据时间范围查找事件
func TestFindByTimeRange(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEventRepository(db)
	ctx := context.Background()

	now := time.Now()
	past := now.Add(-1 * time.Hour)

	// 创建测试数据
	event1 := createTestEvent("trace-001", "session-001", "user", "Old")
	event1.Timestamp = past
	err := repo.Create(ctx, &event1)
	require.NoError(t, err)

	event2 := createTestEvent("trace-002", "session-002", "user", "New")
	event2.Timestamp = now
	err = repo.Create(ctx, &event2)
	require.NoError(t, err)

	// 查询最近 30 分钟的事件
	startTime := now.Add(-30 * time.Minute)
	endTime := now.Add(30 * time.Minute)

	found, err := repo.FindByTimeRange(ctx, startTime, endTime, nil)
	require.NoError(t, err)
	assert.Len(t, found, 1)
	assert.Equal(t, "trace-002", found[0].TraceID)
}

// TestCountBySessionKey 测试统计 SessionKey 下的事件数量
func TestCountBySessionKey(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEventRepository(db)
	ctx := context.Background()

	// 创建测试数据
	events := []models.Event{
		createTestEvent("trace-001", "session-001", "user", "Hello"),
		createTestEvent("trace-002", "session-001", "assistant", "Hi"),
		createTestEvent("trace-003", "session-002", "user", "Test"),
	}
	for i := range events {
		err := repo.Create(ctx, &events[i])
		require.NoError(t, err)
	}

	// 统计
	count, err := repo.CountBySessionKey(ctx, "session-001")
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	count, err = repo.CountBySessionKey(ctx, "session-002")
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	count, err = repo.CountBySessionKey(ctx, "non-existent")
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

// TestCountByTimeRange 测试统计时间范围内的事件数量
func TestCountByTimeRange(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEventRepository(db)
	ctx := context.Background()

	now := time.Now()

	// 创建测试数据 - 使用固定时间确保测试可重复
	events := []models.Event{
		createTestEvent("trace-001", "session-001", "user", "Test4"),
		createTestEvent("trace-002", "session-001", "user", "Test3"),
		createTestEvent("trace-003", "session-001", "user", "Test2"),
		createTestEvent("trace-004", "session-001", "user", "Test1"),
		createTestEvent("trace-005", "session-001", "user", "Test0"),
	}
	// 设置固定的时间戳
	events[0].Timestamp = now.Add(-4 * time.Minute)
	events[1].Timestamp = now.Add(-3 * time.Minute)
	events[2].Timestamp = now.Add(-2 * time.Minute)
	events[3].Timestamp = now.Add(-1 * time.Minute)
	events[4].Timestamp = now

	for i := range events {
		err := repo.Create(ctx, &events[i])
		require.NoError(t, err)
	}

	// 统计最近 3 分钟内的事件（不包括 3 分钟前）
	startTime := now.Add(-3 * time.Minute).Add(1 * time.Nanosecond) // 不包含边界
	endTime := now.Add(1 * time.Minute)

	count, err := repo.CountByTimeRange(ctx, startTime, endTime)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count) // 0, 1, 2 分钟前
}

// TestCount 测试统计事件总数
func TestCount(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEventRepository(db)
	ctx := context.Background()

	// 创建测试数据
	for i := 0; i < 3; i++ {
		event := createTestEvent("trace-001", "session-001", "user", "Test")
		err := repo.Create(ctx, &event)
		require.NoError(t, err)
	}

	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}