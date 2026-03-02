package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/weibaohui/nanobot-go/agent/models"
	"github.com/weibaohui/nanobot-go/agent/repository"
)

// setupTestService 创建测试用的 ConversationService
func setupTestService(t *testing.T) (ConversationService, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 自动迁移表结构
	err = db.AutoMigrate(&models.Event{})
	require.NoError(t, err)

	repo := repository.NewEventRepository(db)
	svc := NewConversationService(repo)

	return svc, db
}

// createTestDTO 创建测试用的 ConversationDTO
func createTestDTO(traceID, sessionKey, role, content string) *ConversationDTO {
	return &ConversationDTO{
		TraceID:    traceID,
		EventType:  "test",
		Timestamp:  time.Now(),
		SessionKey: sessionKey,
		Role:       role,
		Content:    content,
		TokenUsage: &TokenUsageDTO{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
			ReasoningTokens:  5,
			CachedTokens:     0,
		},
	}
}

// TestNewConversationService 测试创建 ConversationService
func TestNewConversationService(t *testing.T) {
	svc, _ := setupTestService(t)
	assert.NotNil(t, svc)
}

// TestGetByTraceID 测试根据 TraceID 获取完整对话
func TestGetByTraceID(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	// 创建测试数据
	dtos := []*ConversationDTO{
		createTestDTO("trace-001", "session-001", "user", "Hello"),
		createTestDTO("trace-001", "session-001", "assistant", "Hi"),
	}
	for _, dto := range dtos {
		err := svc.Create(ctx, dto)
		require.NoError(t, err)
	}

	// 查询
	found, err := svc.GetByTraceID(ctx, "trace-001")
	require.NoError(t, err)
	assert.Len(t, found, 2)
	assert.Equal(t, "user", found[0].Role)
	assert.Equal(t, "assistant", found[1].Role)
}

// TestGetByTraceIDEmptyTraceID 测试空 TraceID
func TestGetByTraceIDEmptyTraceID(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	_, err := svc.GetByTraceID(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "traceID cannot be empty")
}

// TestGetByTraceIDNotFound 测试查找不存在的 TraceID
func TestGetByTraceIDNotFound(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	found, err := svc.GetByTraceID(ctx, "non-existent")
	require.NoError(t, err)
	assert.Empty(t, found)
}

// TestListBySessionKey 测试根据 SessionKey 获取对话列表
func TestListBySessionKey(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	// 创建测试数据
	for i := 0; i < 5; i++ {
		dto := createTestDTO("trace-001", "session-001", "user", "Message")
		err := svc.Create(ctx, dto)
		require.NoError(t, err)
	}

	// 查询第一页
	result, err := svc.ListBySessionKey(ctx, "session-001", 1, 3)
	require.NoError(t, err)
	assert.Len(t, result.Conversations, 3)
	assert.Equal(t, int64(5), result.Total)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 3, result.PageSize)

	// 查询第二页
	result, err = svc.ListBySessionKey(ctx, "session-001", 2, 3)
	require.NoError(t, err)
	assert.Len(t, result.Conversations, 2)
	assert.Equal(t, 2, result.Page)
}

// TestListBySessionKeyEmptySessionKey 测试空 SessionKey
func TestListBySessionKeyEmptySessionKey(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	_, err := svc.ListBySessionKey(ctx, "", 1, 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sessionKey cannot be empty")
}

// TestListBySessionKeyInvalidPage 测试无效页码
func TestListBySessionKeyInvalidPage(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	// 创建测试数据
	dto := createTestDTO("trace-001", "session-001", "user", "Message")
	err := svc.Create(ctx, dto)
	require.NoError(t, err)

	// 页码小于 1，应该默认为 1
	result, err := svc.ListBySessionKey(ctx, "session-001", 0, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Page)
}

// TestListByTimeRange 测试根据时间范围获取对话列表
func TestListByTimeRange(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	now := time.Now()

	// 创建测试数据：旧数据
	oldDTO := createTestDTO("trace-001", "session-001", "user", "Old")
	oldDTO.Timestamp = now.Add(-2 * time.Hour)
	err := svc.Create(ctx, oldDTO)
	require.NoError(t, err)

	// 创建测试数据：新数据
	newDTO := createTestDTO("trace-002", "session-002", "user", "New")
	newDTO.Timestamp = now
	err = svc.Create(ctx, newDTO)
	require.NoError(t, err)

	// 查询最近 30 分钟内的事件
	startTime := now.Add(-30 * time.Minute)
	endTime := now.Add(30 * time.Minute)

	result, err := svc.ListByTimeRange(ctx, startTime, endTime, 1, 10)
	require.NoError(t, err)
	assert.Len(t, result.Conversations, 1)
	assert.Equal(t, "trace-002", result.Conversations[0].TraceID)
}

// TestListRecent 测试获取最近的对话列表
func TestListRecent(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	// 创建测试数据
	for i := 0; i < 5; i++ {
		dto := createTestDTO("trace-001", "session-001", "user", "Message")
		err := svc.Create(ctx, dto)
		require.NoError(t, err)
	}

	// 查询最近的对话
	result, err := svc.ListRecent(ctx, 1, 10)
	require.NoError(t, err)
	assert.Len(t, result.Conversations, 5)
	assert.Equal(t, int64(5), result.Total)
}

// TestCreate 测试创建对话记录
func TestCreate(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	dto := createTestDTO("trace-001", "session-001", "user", "Hello")
	err := svc.Create(ctx, dto)
	require.NoError(t, err)
	assert.NotZero(t, dto.ID)
	assert.NotNil(t, dto.TokenUsage)
}

// TestCreateNilDTO 测试创建空 DTO
func TestCreateNilDTO(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	err := svc.Create(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dto cannot be nil")
}

// TestCreateBatch 测试批量创建对话记录
func TestCreateBatch(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	dtos := []ConversationDTO{
		*createTestDTO("trace-001", "session-001", "user", "Hello"),
		*createTestDTO("trace-002", "session-002", "user", "Hi"),
		*createTestDTO("trace-003", "session-003", "user", "Test"),
	}

	err := svc.CreateBatch(ctx, dtos)
	require.NoError(t, err)

	// 验证数据
	result, err := svc.ListRecent(ctx, 1, 10)
	require.NoError(t, err)
	assert.Len(t, result.Conversations, 3)
}

// TestCreateBatchEmpty 测试批量创建空列表
func TestCreateBatchEmpty(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	err := svc.CreateBatch(ctx, []ConversationDTO{})
	require.NoError(t, err)
}

// TestDTOConversion 测试 DTO 和 Event 之间的转换
func TestDTOConversion(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	original := createTestDTO("trace-001", "session-001", "user", "Hello")
	original.TokenUsage = &TokenUsageDTO{
		PromptTokens:     10,
		CompletionTokens: 20,
		TotalTokens:      30,
		ReasoningTokens:  5,
		CachedTokens:     0,
	}

	// 创建
	err := svc.Create(ctx, original)
	require.NoError(t, err)

	// 查询
	found, err := svc.GetByTraceID(ctx, "trace-001")
	require.NoError(t, err)
	assert.Len(t, found, 1)

	// 验证转换正确
	assert.Equal(t, original.TraceID, found[0].TraceID)
	assert.Equal(t, original.Role, found[0].Role)
	assert.Equal(t, original.Content, found[0].Content)
	assert.NotNil(t, found[0].TokenUsage)
	assert.Equal(t, original.TokenUsage.TotalTokens, found[0].TokenUsage.TotalTokens)
}

// TestPaginationMaxSize 测试分页大小限制
func TestPaginationMaxSize(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	// 创建大量数据
	for i := 0; i < 2000; i++ {
		dto := createTestDTO("trace-001", "session-001", "user", "Message")
		err := svc.Create(ctx, dto)
		require.NoError(t, err)
	}

	// 请求超过最大限制的页面大小
	result, err := svc.ListBySessionKey(ctx, "session-001", 1, 2000)
	require.NoError(t, err)
	assert.Equal(t, 1000, result.PageSize) // 应该被限制为 1000
	assert.Len(t, result.Conversations, 1000)
}