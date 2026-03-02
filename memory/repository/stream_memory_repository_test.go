package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weibaohui/nanobot-go/memory/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 自动迁移表结构
	err = db.AutoMigrate(&models.StreamMemory{}, &models.LongTermMemory{})
	require.NoError(t, err)

	return db
}

func TestStreamMemoryRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewStreamMemoryRepository(db)
	ctx := context.Background()

	memory := &models.StreamMemory{
		TraceID:     "trace-001",
		SessionKey:  "session-001",
		ChannelType: "matrix",
		Content:     "测试对话内容",
		Summary:     "测试总结",
		EventType:   "conversation",
		CreatedAt:   time.Now(),
		Processed:   false,
	}

	err := repo.Create(ctx, memory)
	require.NoError(t, err)
	assert.NotZero(t, memory.ID)
}

func TestStreamMemoryRepository_CreateBatch(t *testing.T) {
	db := setupTestDB(t)
	repo := NewStreamMemoryRepository(db)
	ctx := context.Background()

	memories := []models.StreamMemory{
		{
			TraceID:     "trace-001",
			SessionKey:  "session-001",
			ChannelType: "matrix",
			Content:     "内容1",
			CreatedAt:   time.Now(),
		},
		{
			TraceID:     "trace-002",
			SessionKey:  "session-001",
			ChannelType: "matrix",
			Content:     "内容2",
			CreatedAt:   time.Now(),
		},
	}

	err := repo.CreateBatch(ctx, memories)
	require.NoError(t, err)

	// 验证创建成功
	result, err := repo.FindBySessionKey(ctx, "session-001", nil)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestStreamMemoryRepository_FindByTraceID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewStreamMemoryRepository(db)
	ctx := context.Background()

	// 创建测试数据
	memory := &models.StreamMemory{
		TraceID:     "trace-001",
		SessionKey:  "session-001",
		Content:     "测试内容",
		CreatedAt:   time.Now(),
	}
	err := repo.Create(ctx, memory)
	require.NoError(t, err)

	// 查询
	result, err := repo.FindByTraceID(ctx, "trace-001")
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "测试内容", result[0].Content)

	// 查询不存在的 trace_id
	result, err = repo.FindByTraceID(ctx, "non-existent")
	require.NoError(t, err)
	assert.Len(t, result, 0)
}

func TestStreamMemoryRepository_FindBySessionKey(t *testing.T) {
	db := setupTestDB(t)
	repo := NewStreamMemoryRepository(db)
	ctx := context.Background()

	// 创建测试数据
	now := time.Now()
	memories := []models.StreamMemory{
		{TraceID: "trace-001", SessionKey: "session-001", Content: "内容1", CreatedAt: now.Add(-time.Hour)},
		{TraceID: "trace-002", SessionKey: "session-001", Content: "内容2", CreatedAt: now},
		{TraceID: "trace-003", SessionKey: "session-002", Content: "内容3", CreatedAt: now},
	}
	for i := range memories {
		err := repo.Create(ctx, &memories[i])
		require.NoError(t, err)
	}

	// 测试查询
	result, err := repo.FindBySessionKey(ctx, "session-001", nil)
	require.NoError(t, err)
	assert.Len(t, result, 2)

	// 测试分页
	opts := &models.QueryOptions{Limit: 1, Offset: 0}
	result, err = repo.FindBySessionKey(ctx, "session-001", opts)
	require.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestStreamMemoryRepository_FindByTimeRange(t *testing.T) {
	db := setupTestDB(t)
	repo := NewStreamMemoryRepository(db)
	ctx := context.Background()

	now := time.Now()
	// 创建测试数据
	memories := []models.StreamMemory{
		{TraceID: "trace-001", Content: "昨天", CreatedAt: now.Add(-24 * time.Hour)},
		{TraceID: "trace-002", Content: "今天", CreatedAt: now},
		{TraceID: "trace-003", Content: "明天", CreatedAt: now.Add(24 * time.Hour)},
	}
	for i := range memories {
		err := repo.Create(ctx, &memories[i])
		require.NoError(t, err)
	}

	// 查询今天的记录
	start := now.Add(-12 * time.Hour)
	end := now.Add(12 * time.Hour)
	result, err := repo.FindByTimeRange(ctx, start, end, nil)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "今天", result[0].Content)
}

func TestStreamMemoryRepository_FindUnprocessed(t *testing.T) {
	db := setupTestDB(t)
	repo := NewStreamMemoryRepository(db)
	ctx := context.Background()

	now := time.Now()
	// 创建测试数据
	memories := []models.StreamMemory{
		{TraceID: "trace-001", Content: "已处理", CreatedAt: now.Add(-2 * time.Hour), Processed: true},
		{TraceID: "trace-002", Content: "未处理1", CreatedAt: now.Add(-2 * time.Hour), Processed: false},
		{TraceID: "trace-003", Content: "未处理2", CreatedAt: now.Add(-2 * time.Hour), Processed: false},
		{TraceID: "trace-004", Content: "新的未处理", CreatedAt: now, Processed: false},
	}
	for i := range memories {
		err := repo.Create(ctx, &memories[i])
		require.NoError(t, err)
	}

	// 查询1小时前的未处理记录
	before := now.Add(-1 * time.Hour)
	result, err := repo.FindUnprocessed(ctx, before, 100)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestStreamMemoryRepository_MarkAsProcessed(t *testing.T) {
	db := setupTestDB(t)
	repo := NewStreamMemoryRepository(db)
	ctx := context.Background()

	// 创建测试数据
	memories := []models.StreamMemory{
		{TraceID: "trace-001", Content: "内容1", CreatedAt: time.Now(), Processed: false},
		{TraceID: "trace-002", Content: "内容2", CreatedAt: time.Now(), Processed: false},
	}
	for i := range memories {
		err := repo.Create(ctx, &memories[i])
		require.NoError(t, err)
	}

	// 标记为已处理
	err := repo.MarkAsProcessed(ctx, []uint64{memories[0].ID, memories[1].ID})
	require.NoError(t, err)

	// 验证
	count, err := repo.CountUnprocessed(ctx, time.Now().Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestStreamMemoryRepository_CountByTimeRange(t *testing.T) {
	db := setupTestDB(t)
	repo := NewStreamMemoryRepository(db)
	ctx := context.Background()

	now := time.Now()
	// 创建测试数据
	memories := []models.StreamMemory{
		{TraceID: "trace-001", Content: "内容1", CreatedAt: now},
		{TraceID: "trace-002", Content: "内容2", CreatedAt: now},
		{TraceID: "trace-003", Content: "内容3", CreatedAt: now.Add(24 * time.Hour)},
	}
	for i := range memories {
		err := repo.Create(ctx, &memories[i])
		require.NoError(t, err)
	}

	// 统计今天的记录
	count, err := repo.CountByTimeRange(ctx, now.Add(-time.Hour), now.Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestStreamMemoryRepository_CountUnprocessed(t *testing.T) {
	db := setupTestDB(t)
	repo := NewStreamMemoryRepository(db)
	ctx := context.Background()

	now := time.Now()
	// 创建测试数据
	memories := []models.StreamMemory{
		{TraceID: "trace-001", Content: "已处理", CreatedAt: now.Add(-2 * time.Hour), Processed: true},
		{TraceID: "trace-002", Content: "未处理", CreatedAt: now.Add(-2 * time.Hour), Processed: false},
	}
	for i := range memories {
		err := repo.Create(ctx, &memories[i])
		require.NoError(t, err)
	}

	// 统计未处理记录
	count, err := repo.CountUnprocessed(ctx, now.Add(-time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}
