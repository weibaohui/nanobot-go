package repository

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weibaohui/nanobot-go/internal/memory/models"
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
		UserCode:  "user-001",
		Date:      time.Now().Format("2006-01-02"),
		Content:   "测试对话内容",
		Summary:   "测试总结",
		SourceIDs: "conv-001",
		Processed: false,
	}

	err := repo.Create(ctx, memory)
	require.NoError(t, err)
	assert.NotZero(t, memory.ID)
}

func TestStreamMemoryRepository_CreateBatch(t *testing.T) {
	db := setupTestDB(t)
	repo := NewStreamMemoryRepository(db)
	ctx := context.Background()

	now := time.Now()
	memories := []models.StreamMemory{
		{
			UserCode:  "user-001",
			Date:      now.Format("2006-01-02"),
			Content:   "内容1",
			SourceIDs: "conv-001",
		},
		{
			UserCode:  "user-001",
			Date:      now.AddDate(0, 0, -1).Format("2006-01-02"),
			Content:   "内容2",
			SourceIDs: "conv-002",
		},
	}

	err := repo.CreateBatch(ctx, memories)
	require.NoError(t, err)

	// 验证创建成功
	result, err := repo.FindByTimeRange(ctx, now.AddDate(0, 0, -2), now.AddDate(0, 0, 1), nil)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestStreamMemoryRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewStreamMemoryRepository(db)
	ctx := context.Background()

	// 创建测试数据
	memory := &models.StreamMemory{
		UserCode:  "user-001",
		Date:      time.Now().Format("2006-01-02"),
		Content:   "测试内容",
		SourceIDs: "conv-001",
	}
	err := repo.Create(ctx, memory)
	require.NoError(t, err)

	// 查询
	result, err := repo.FindByID(ctx, memory.ID)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "测试内容", result.Content)

	// 查询不存在的 ID
	result, err = repo.FindByID(ctx, 9999)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestStreamMemoryRepository_FindByTimeRange(t *testing.T) {
	db := setupTestDB(t)
	repo := NewStreamMemoryRepository(db)
	ctx := context.Background()

	now := time.Now()
	// 创建测试数据
	memories := []models.StreamMemory{
		{UserCode: "user-001", Date: now.AddDate(0, 0, -1).Format("2006-01-02"), Content: "昨天", SourceIDs: "conv-001"},
		{UserCode: "user-001", Date: now.Format("2006-01-02"), Content: "今天", SourceIDs: "conv-002"},
		{UserCode: "user-001", Date: now.AddDate(0, 0, 1).Format("2006-01-02"), Content: "明天", SourceIDs: "conv-003"},
	}
	for i := range memories {
		err := repo.Create(ctx, &memories[i])
		require.NoError(t, err)
	}

	// 查询今天的记录
	start := now.AddDate(0, 0, -1)
	end := now.AddDate(0, 0, 1)
	result, err := repo.FindByTimeRange(ctx, start, end, nil)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestStreamMemoryRepository_FindUnprocessed(t *testing.T) {
	db := setupTestDB(t)
	repo := NewStreamMemoryRepository(db)
	ctx := context.Background()

	now := time.Now()
	// 创建测试数据
	memories := []models.StreamMemory{
		{UserCode: "user-001", Date: now.AddDate(0, 0, -1).Format("2006-01-02"), Content: "已处理", Processed: true, SourceIDs: "conv-001"},
		{UserCode: "user-001", Date: now.AddDate(0, 0, -1).Format("2006-01-02"), Content: "未处理1", Processed: false, SourceIDs: "conv-002"},
		{UserCode: "user-001", Date: now.AddDate(0, 0, -2).Format("2006-01-02"), Content: "未处理2", Processed: false, SourceIDs: "conv-003"},
		{UserCode: "user-001", Date: now.Format("2006-01-02"), Content: "新的未处理", Processed: false, SourceIDs: "conv-004"},
	}
	for i := range memories {
		err := repo.Create(ctx, &memories[i])
		require.NoError(t, err)
	}

	// 查询1小时前的未处理记录
	before := now.Add(-1 * time.Hour)
	result, err := repo.FindUnprocessed(ctx, before, 100, nil)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestStreamMemoryRepository_MarkAsProcessed(t *testing.T) {
	db := setupTestDB(t)
	repo := NewStreamMemoryRepository(db)
	ctx := context.Background()

	// 创建测试数据
	memories := []models.StreamMemory{
		{UserCode: "user-001", Date: time.Now().Format("2006-01-02"), Content: "内容1", Processed: false, SourceIDs: "conv-001"},
		{UserCode: "user-001", Date: time.Now().Format("2006-01-02"), Content: "内容2", Processed: false, SourceIDs: "conv-002"},
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
		{UserCode: "user-001", Date: now.Format("2006-01-02"), Content: "内容1", SourceIDs: "conv-001"},
		{UserCode: "user-001", Date: now.Format("2006-01-02"), Content: "内容2", SourceIDs: "conv-002"},
		{UserCode: "user-001", Date: now.AddDate(0, 0, 1).Format("2006-01-02"), Content: "内容3", SourceIDs: "conv-003"},
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
		{UserCode: "user-001", Date: now.AddDate(0, 0, -1).Format("2006-01-02"), Content: "已处理", Processed: true, SourceIDs: "conv-001"},
		{UserCode: "user-001", Date: now.AddDate(0, 0, -1).Format("2006-01-02"), Content: "未处理", Processed: false, SourceIDs: "conv-002"},
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
