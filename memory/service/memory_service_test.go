package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/weibaohui/nanobot-go/memory/models"
)

// MockStreamMemoryRepository 模拟流水记忆仓储
type MockStreamMemoryRepository struct {
	mock.Mock
}

func (m *MockStreamMemoryRepository) Create(ctx context.Context, memory *models.StreamMemory) error {
	args := m.Called(ctx, memory)
	if args.Get(0) == nil {
		memory.ID = 1 // 模拟自增ID
	}
	return args.Error(0)
}

func (m *MockStreamMemoryRepository) CreateBatch(ctx context.Context, memories []models.StreamMemory) error {
	args := m.Called(ctx, memories)
	return args.Error(0)
}

func (m *MockStreamMemoryRepository) FindByID(ctx context.Context, id uint64) (*models.StreamMemory, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StreamMemory), args.Error(1)
}

func (m *MockStreamMemoryRepository) FindByTraceID(ctx context.Context, traceID string) ([]models.StreamMemory, error) {
	args := m.Called(ctx, traceID)
	return args.Get(0).([]models.StreamMemory), args.Error(1)
}

func (m *MockStreamMemoryRepository) FindBySessionKey(ctx context.Context, sessionKey string, opts *models.QueryOptions) ([]models.StreamMemory, error) {
	args := m.Called(ctx, sessionKey, opts)
	return args.Get(0).([]models.StreamMemory), args.Error(1)
}

func (m *MockStreamMemoryRepository) FindByTimeRange(ctx context.Context, startTime, endTime time.Time, opts *models.QueryOptions) ([]models.StreamMemory, error) {
	args := m.Called(ctx, startTime, endTime, opts)
	return args.Get(0).([]models.StreamMemory), args.Error(1)
}

func (m *MockStreamMemoryRepository) FindUnprocessed(ctx context.Context, before time.Time, limit int) ([]models.StreamMemory, error) {
	args := m.Called(ctx, before, limit)
	return args.Get(0).([]models.StreamMemory), args.Error(1)
}

func (m *MockStreamMemoryRepository) MarkAsProcessed(ctx context.Context, ids []uint64) error {
	args := m.Called(ctx, ids)
	return args.Error(0)
}

func (m *MockStreamMemoryRepository) CountByTimeRange(ctx context.Context, startTime, endTime time.Time) (int64, error) {
	args := m.Called(ctx, startTime, endTime)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockStreamMemoryRepository) CountUnprocessed(ctx context.Context, before time.Time) (int64, error) {
	args := m.Called(ctx, before)
	return args.Get(0).(int64), args.Error(1)
}

// MockLongTermMemoryRepository 模拟长期记忆仓储
type MockLongTermMemoryRepository struct {
	mock.Mock
}

func (m *MockLongTermMemoryRepository) Create(ctx context.Context, memory *models.LongTermMemory) error {
	args := m.Called(ctx, memory)
	return args.Error(0)
}

func (m *MockLongTermMemoryRepository) FindByID(ctx context.Context, id uint64) (*models.LongTermMemory, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.LongTermMemory), args.Error(1)
}

func (m *MockLongTermMemoryRepository) FindByDate(ctx context.Context, date string) (*models.LongTermMemory, error) {
	args := m.Called(ctx, date)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.LongTermMemory), args.Error(1)
}

func (m *MockLongTermMemoryRepository) FindByTimeRange(ctx context.Context, startTime, endTime time.Time, opts *models.QueryOptions) ([]models.LongTermMemory, error) {
	args := m.Called(ctx, startTime, endTime, opts)
	return args.Get(0).([]models.LongTermMemory), args.Error(1)
}

func (m *MockLongTermMemoryRepository) SearchByKeyword(ctx context.Context, keyword string, opts *models.QueryOptions) ([]models.LongTermMemory, error) {
	args := m.Called(ctx, keyword, opts)
	return args.Get(0).([]models.LongTermMemory), args.Error(1)
}

func (m *MockLongTermMemoryRepository) Update(ctx context.Context, memory *models.LongTermMemory) error {
	args := m.Called(ctx, memory)
	return args.Error(0)
}

func (m *MockLongTermMemoryRepository) DeleteByDate(ctx context.Context, date string) error {
	args := m.Called(ctx, date)
	return args.Error(0)
}

// MockMemorySummarizer 模拟记忆总结器
type MockMemorySummarizer struct {
	mock.Mock
}

func (m *MockMemorySummarizer) SummarizeConversation(ctx context.Context, messages []models.Message) (*models.ConversationSummary, error) {
	args := m.Called(ctx, messages)
	return args.Get(0).(*models.ConversationSummary), args.Error(1)
}

func (m *MockMemorySummarizer) SummarizeToLongTerm(ctx context.Context, streams []models.StreamMemory) (*models.LongTermSummary, error) {
	args := m.Called(ctx, streams)
	return args.Get(0).(*models.LongTermSummary), args.Error(1)
}

func TestMemoryService_WriteMemory(t *testing.T) {
	streamRepo := new(MockStreamMemoryRepository)
	longTermRepo := new(MockLongTermMemoryRepository)
	summarizer := new(MockMemorySummarizer)

	svc := NewMemoryService(streamRepo, longTermRepo, summarizer, true)
	ctx := context.Background()

	t.Run("成功写入", func(t *testing.T) {
		// 幂等检查：返回空列表
		streamRepo.On("FindByTraceID", ctx, "trace-001").Return([]models.StreamMemory{}, nil).Once()
		// 创建记录
		streamRepo.On("Create", ctx, mock.AnythingOfType("*models.StreamMemory")).Return(nil).Once()

		metadata := map[string]interface{}{
			"trace_id":     "trace-001",
			"session_key":  "session-001",
			"channel_type": "matrix",
			"event_type":   "conversation",
		}

		err := svc.WriteMemory(ctx, "测试内容", metadata)
		require.NoError(t, err)
		streamRepo.AssertExpectations(t)
	})

	t.Run("重复 trace_id", func(t *testing.T) {
		// 幂等检查：返回已有记录
		existing := []models.StreamMemory{
			{ID: 1, TraceID: "trace-002", Content: "已有内容"},
		}
		streamRepo.On("FindByTraceID", ctx, "trace-002").Return(existing, nil).Once()

		metadata := map[string]interface{}{
			"trace_id": "trace-002",
		}

		err := svc.WriteMemory(ctx, "测试内容", metadata)
		assert.Equal(t, ErrDuplicateTraceID, err)
	})

	t.Run("未启用", func(t *testing.T) {
		disabledSvc := NewMemoryService(streamRepo, longTermRepo, summarizer, false)
		err := disabledSvc.WriteMemory(ctx, "内容", map[string]interface{}{"trace_id": "test"})
		assert.Equal(t, ErrMemoryDisabled, err)
	})

	t.Run("缺少 trace_id", func(t *testing.T) {
		err := svc.WriteMemory(ctx, "内容", map[string]interface{}{})
		assert.ErrorIs(t, err, ErrInvalidMetadata)
	})
}

func TestMemoryService_SearchMemory(t *testing.T) {
	streamRepo := new(MockStreamMemoryRepository)
	longTermRepo := new(MockLongTermMemoryRepository)
	summarizer := new(MockMemorySummarizer)

	svc := NewMemoryService(streamRepo, longTermRepo, summarizer, true)
	ctx := context.Background()

	t.Run("关键词搜索", func(t *testing.T) {
		// 模拟流水记忆查询
		streams := []models.StreamMemory{
			{ID: 1, TraceID: "trace-001", Content: "包含关键词的内容", CreatedAt: time.Now()},
		}
		streamRepo.On("FindByTimeRange", ctx, mock.Anything, mock.Anything, mock.Anything).Return(streams, nil).Once()

		// 模拟长期记忆搜索
		longTerms := []models.LongTermMemory{
			{ID: 1, Date: "2026-03-02", Summary: "关键词总结", CreatedAt: time.Now()},
		}
		longTermRepo.On("SearchByKeyword", ctx, "关键词", mock.Anything).Return(longTerms, nil).Once()

		filters := models.SearchFilters{
			Limit: 20,
		}

		result, err := svc.SearchMemory(ctx, "关键词", filters)
		require.NoError(t, err)
		assert.Equal(t, 2, result.Total)
		assert.Len(t, result.StreamMemories, 1)
		assert.Len(t, result.LongTermMemories, 1)
	})

	t.Run("未启用", func(t *testing.T) {
		disabledSvc := NewMemoryService(streamRepo, longTermRepo, summarizer, false)
		_, err := disabledSvc.SearchMemory(ctx, "查询", models.SearchFilters{})
		assert.Equal(t, ErrMemoryDisabled, err)
	})
}

func TestMemoryService_UpgradeStreamToLongTerm(t *testing.T) {
	streamRepo := new(MockStreamMemoryRepository)
	longTermRepo := new(MockLongTermMemoryRepository)
	summarizer := new(MockMemorySummarizer)

	svc := NewMemoryService(streamRepo, longTermRepo, summarizer, true)
	ctx := context.Background()

	t.Run("成功升级", func(t *testing.T) {
		date := "2026-03-01"
		targetDate, _ := time.Parse("2006-01-02", date)
		startOfDay := targetDate
		endOfDay := targetDate.AddDate(0, 0, 1).Add(-time.Nanosecond)

		// 模拟查询流水记忆
		streams := []models.StreamMemory{
			{ID: 1, TraceID: "trace-001", Content: "内容1", CreatedAt: targetDate.Add(time.Hour)},
			{ID: 2, TraceID: "trace-002", Content: "内容2", CreatedAt: targetDate.Add(2 * time.Hour)},
		}
		streamRepo.On("FindByTimeRange", ctx, startOfDay, endOfDay, mock.Anything).Return(streams, nil).Once()

		// 模拟总结
		summary := &models.LongTermSummary{
			WhatHappened: "发生了一些事情",
			Conclusion:   "结论是...",
			Value:        "价值是...",
			Highlights:   []string{"事件1", "事件2"},
		}
		summarizer.On("SummarizeToLongTerm", ctx, streams).Return(summary, nil).Once()

		// 检查是否存在长期记忆
		longTermRepo.On("FindByDate", ctx, date).Return(nil, nil).Once()

		// 创建长期记忆
		longTermRepo.On("Create", ctx, mock.AnythingOfType("*models.LongTermMemory")).Return(nil).Once()

		// 标记为已处理
		streamRepo.On("MarkAsProcessed", ctx, []uint64{1, 2}).Return(nil).Once()

		err := svc.UpgradeStreamToLongTerm(ctx, date)
		require.NoError(t, err)
		streamRepo.AssertExpectations(t)
		longTermRepo.AssertExpectations(t)
	})

	t.Run("日期格式无效", func(t *testing.T) {
		err := svc.UpgradeStreamToLongTerm(ctx, "invalid-date")
		assert.ErrorIs(t, err, ErrInvalidMetadata)
	})

	t.Run("未启用", func(t *testing.T) {
		disabledSvc := NewMemoryService(streamRepo, longTermRepo, summarizer, false)
		err := disabledSvc.UpgradeStreamToLongTerm(ctx, "2026-03-01")
		assert.Equal(t, ErrMemoryDisabled, err)
	})
}

func TestMemoryService_GetUnprocessedCount(t *testing.T) {
	streamRepo := new(MockStreamMemoryRepository)
	longTermRepo := new(MockLongTermMemoryRepository)
	summarizer := new(MockMemorySummarizer)

	svc := NewMemoryService(streamRepo, longTermRepo, summarizer, true)
	ctx := context.Background()

	streamRepo.On("CountUnprocessed", ctx, mock.Anything).Return(int64(5), nil).Once()

	count, err := svc.GetUnprocessedCount(ctx, time.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
}
