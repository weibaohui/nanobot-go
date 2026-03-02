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

func setupLongTermTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.LongTermMemory{})
	require.NoError(t, err)

	return db
}

func TestLongTermMemoryRepository_Create(t *testing.T) {
	db := setupLongTermTestDB(t)
	repo := NewLongTermMemoryRepository(db)
	ctx := context.Background()

	memory := &models.LongTermMemory{
		Date:         "2026-03-02",
		Summary:      "今日总结",
		WhatHappened: "发生了一些事情",
		Conclusion:   "结论是...",
		Value:        "价值是...",
		Highlights:   `["事件1", "事件2"]`,
		SourceIDs:    "1,2,3",
	}

	err := repo.Create(ctx, memory)
	require.NoError(t, err)
	assert.NotZero(t, memory.ID)
	assert.NotZero(t, memory.CreatedAt)
	assert.NotZero(t, memory.UpdatedAt)
}

func TestLongTermMemoryRepository_FindByID(t *testing.T) {
	db := setupLongTermTestDB(t)
	repo := NewLongTermMemoryRepository(db)
	ctx := context.Background()

	memory := &models.LongTermMemory{
		Date:    "2026-03-02",
		Summary: "测试总结",
	}
	err := repo.Create(ctx, memory)
	require.NoError(t, err)

	// 查询存在的记录
	result, err := repo.FindByID(ctx, memory.ID)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "测试总结", result.Summary)

	// 查询不存在的记录
	result, err = repo.FindByID(ctx, 9999)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestLongTermMemoryRepository_FindByDate(t *testing.T) {
	db := setupLongTermTestDB(t)
	repo := NewLongTermMemoryRepository(db)
	ctx := context.Background()

	memory := &models.LongTermMemory{
		Date:    "2026-03-02",
		Summary: "3月2日总结",
	}
	err := repo.Create(ctx, memory)
	require.NoError(t, err)

	// 查询存在的日期
	result, err := repo.FindByDate(ctx, "2026-03-02")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "3月2日总结", result.Summary)

	// 查询不存在的日期
	result, err = repo.FindByDate(ctx, "2026-03-03")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestLongTermMemoryRepository_FindByTimeRange(t *testing.T) {
	db := setupLongTermTestDB(t)
	repo := NewLongTermMemoryRepository(db)
	ctx := context.Background()

	memories := []models.LongTermMemory{
		{Date: "2026-03-01", Summary: "3月1日"},
		{Date: "2026-03-02", Summary: "3月2日"},
		{Date: "2026-03-03", Summary: "3月3日"},
	}
	for i := range memories {
		err := repo.Create(ctx, &memories[i])
		require.NoError(t, err)
	}

	// 查询时间范围
	startTime := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 3, 2, 23, 59, 59, 0, time.UTC)
	result, err := repo.FindByTimeRange(ctx, startTime, endTime, nil)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestLongTermMemoryRepository_SearchByKeyword(t *testing.T) {
	db := setupLongTermTestDB(t)
	repo := NewLongTermMemoryRepository(db)
	ctx := context.Background()

	memories := []models.LongTermMemory{
		{Date: "2026-03-01", Summary: "关于项目的讨论", WhatHappened: "讨论了项目进展"},
		{Date: "2026-03-02", Summary: "代码审查", WhatHappened: "审查了新功能代码"},
		{Date: "2026-03-03", Summary: "项目总结", Conclusion: "项目成功上线"},
	}
	for i := range memories {
		err := repo.Create(ctx, &memories[i])
		require.NoError(t, err)
	}

	// 关键词搜索
	result, err := repo.SearchByKeyword(ctx, "项目", nil)
	require.NoError(t, err)
	assert.Len(t, result, 2) // 应该匹配 "关于项目的讨论" 和 "项目总结"

	// 搜索另一个关键词
	result, err = repo.SearchByKeyword(ctx, "代码", nil)
	require.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestLongTermMemoryRepository_Update(t *testing.T) {
	db := setupLongTermTestDB(t)
	repo := NewLongTermMemoryRepository(db)
	ctx := context.Background()

	memory := &models.LongTermMemory{
		Date:    "2026-03-02",
		Summary: "原始总结",
	}
	err := repo.Create(ctx, memory)
	require.NoError(t, err)

	// 更新
	memory.Summary = "更新后的总结"
	err = repo.Update(ctx, memory)
	require.NoError(t, err)

	// 验证
	result, err := repo.FindByID(ctx, memory.ID)
	require.NoError(t, err)
	assert.Equal(t, "更新后的总结", result.Summary)
	assert.True(t, result.UpdatedAt.After(result.CreatedAt))
}

func TestLongTermMemoryRepository_DeleteByDate(t *testing.T) {
	db := setupLongTermTestDB(t)
	repo := NewLongTermMemoryRepository(db)
	ctx := context.Background()

	memories := []models.LongTermMemory{
		{Date: "2026-03-01", Summary: "3月1日"},
		{Date: "2026-03-02", Summary: "3月2日"},
	}
	for i := range memories {
		err := repo.Create(ctx, &memories[i])
		require.NoError(t, err)
	}

	// 删除指定日期的记录
	err := repo.DeleteByDate(ctx, "2026-03-01")
	require.NoError(t, err)

	// 验证
	result, err := repo.FindByDate(ctx, "2026-03-01")
	require.NoError(t, err)
	assert.Nil(t, result)

	result, err = repo.FindByDate(ctx, "2026-03-02")
	require.NoError(t, err)
	assert.NotNil(t, result)
}
