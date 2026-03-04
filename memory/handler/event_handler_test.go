package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/weibaohui/nanobot-go/memory/models"
	"github.com/weibaohui/nanobot-go/memory/service"
	"go.uber.org/zap"
)

// MockMemoryService 模拟内存服务
type MockMemoryService struct {
	mock.Mock
}

func (m *MockMemoryService) WriteMemory(ctx context.Context, content string, metadata map[string]interface{}) error {
	args := m.Called(ctx, content, metadata)
	return args.Error(0)
}

func (m *MockMemoryService) SearchMemory(ctx context.Context, keyword string, filters models.SearchFilters) (*models.SearchResult, error) {
	args := m.Called(ctx, keyword, filters)
	return args.Get(0).(*models.SearchResult), args.Error(1)
}

func (m *MockMemoryService) UpgradeStreamToLongTerm(ctx context.Context, date string) error {
	args := m.Called(ctx, date)
	return args.Error(0)
}

func (m *MockMemoryService) GetUnprocessedCount(ctx context.Context, before time.Time) (int64, error) {
	args := m.Called(ctx, before)
	return args.Get(0).(int64), args.Error(1)
}

// MockConversationService 模拟对话服务
type MockConversationService struct {
	mock.Mock
}

func (m *MockConversationService) GetByTraceID(ctx context.Context, traceID string) ([]interface{}, error) {
	args := m.Called(ctx, traceID)
	return args.Get(0).([]interface{}), args.Error(1)
}

// MockMemorySummarizer 模拟记忆总结器
type MockMemorySummarizer struct {
	mock.Mock
}

func (m *MockMemorySummarizer) SummarizeConversation(ctx context.Context, messages []models.Message) (*models.ConversationSummary, error) {
	args := m.Called(ctx, messages)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ConversationSummary), args.Error(1)
}

func (m *MockMemorySummarizer) SummarizeToLongTerm(ctx context.Context, streams []models.StreamMemory) (*models.LongTermSummary, error) {
	args := m.Called(ctx, streams)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.LongTermSummary), args.Error(1)
}

// TestNewMemoryEventHandler 测试创建事件处理器
func TestNewMemoryEventHandler(t *testing.T) {
	mockMemSvc := new(MockMemoryService)
	mockConvSvc := new(MockConversationService)
	mockSummarizer := new(MockMemorySummarizer)
	logger := zap.NewNop()

	handler := NewMemoryEventHandler(mockMemSvc, mockConvSvc, mockSummarizer, logger, true)

	assert.NotNil(t, handler)
	assert.Equal(t, mockMemSvc, handler.memoryService)
	assert.Equal(t, mockConvSvc, handler.conversationSvc)
	assert.Equal(t, mockSummarizer, handler.summarizer)
	assert.True(t, handler.enabled)
}

// TestMemoryEventHandler_OnConversationCompleted 测试处理对话完成事件
func TestMemoryEventHandler_OnConversationCompleted(t *testing.T) {
	t.Run("成功处理事件", func(t *testing.T) {
		mockMemSvc := new(MockMemoryService)
		mockConvSvc := new(MockConversationService)
		mockSummarizer := new(MockMemorySummarizer)
		logger := zap.NewNop()
		handler := NewMemoryEventHandler(mockMemSvc, mockConvSvc, mockSummarizer, logger, true)

		event := models.ConversationCompletedEvent{
			TraceID:     "trace-001",
			SessionKey:  "session-001",
			ChannelType: "dingtalk",
			Messages: []models.Message{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there"},
			},
			EndTime: time.Now(),
		}

		summary := &models.ConversationSummary{
			Summary:   "用户问候",
			KeyPoints: "1. 用户说 Hello\n2. 助手回复 Hi",
		}

		mockSummarizer.On("SummarizeConversation", mock.Anything, event.Messages).Return(summary, nil).Once()
		mockMemSvc.On("WriteMemory", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err := handler.OnConversationCompleted(context.Background(), event)

		assert.NoError(t, err)
		time.Sleep(200 * time.Millisecond) // 等待异步写入完成

		mockSummarizer.AssertExpectations(t)
		mockMemSvc.AssertExpectations(t)
	})

	t.Run("禁用时不处理", func(t *testing.T) {
		mockMemSvc := new(MockMemoryService)
		mockConvSvc := new(MockConversationService)
		mockSummarizer := new(MockMemorySummarizer)
		logger := zap.NewNop()
		handler := NewMemoryEventHandler(mockMemSvc, mockConvSvc, mockSummarizer, logger, false)

		event := models.ConversationCompletedEvent{
			TraceID:    "trace-001",
			SessionKey: "session-001",
		}

		err := handler.OnConversationCompleted(context.Background(), event)

		assert.NoError(t, err)
		mockSummarizer.AssertNotCalled(t, "SummarizeConversation")
		mockMemSvc.AssertNotCalled(t, "WriteMemory")
	})

	t.Run("事件中没有消息", func(t *testing.T) {
		mockMemSvc := new(MockMemoryService)
		mockConvSvc := new(MockConversationService)
		mockSummarizer := new(MockMemorySummarizer)
		logger := zap.NewNop()
		handler := NewMemoryEventHandler(mockMemSvc, mockConvSvc, mockSummarizer, logger, true)

		event := models.ConversationCompletedEvent{
			TraceID:     "trace-001",
			SessionKey:  "session-001",
			ChannelType: "dingtalk",
			Messages:    []models.Message{},
			EndTime:     time.Now(),
		}

		// 没有消息时会使用默认消息
		summary := &models.ConversationSummary{
			Summary: "对话完成: 0 条消息",
		}

		mockSummarizer.On("SummarizeConversation", mock.Anything, mock.Anything).Return(summary, nil).Once()
		mockMemSvc.On("WriteMemory", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err := handler.OnConversationCompleted(context.Background(), event)

		assert.NoError(t, err)
		time.Sleep(200 * time.Millisecond)

		mockSummarizer.AssertExpectations(t)
	})

	t.Run("总结失败时使用默认内容", func(t *testing.T) {
		mockMemSvc := new(MockMemoryService)
		mockConvSvc := new(MockConversationService)
		mockSummarizer := new(MockMemorySummarizer)
		logger := zap.NewNop()
		handler := NewMemoryEventHandler(mockMemSvc, mockConvSvc, mockSummarizer, logger, true)

		event := models.ConversationCompletedEvent{
			TraceID:     "trace-001",
			SessionKey:  "session-001",
			ChannelType: "dingtalk",
			Messages: []models.Message{
				{Role: "user", Content: "Hello"},
			},
			EndTime: time.Now(),
		}

		// 模拟总结失败
		mockSummarizer.On("SummarizeConversation", mock.Anything, event.Messages).
			Return(nil, errors.New("summarization failed")).Once()
		mockMemSvc.On("WriteMemory", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err := handler.OnConversationCompleted(context.Background(), event)

		assert.NoError(t, err)
		time.Sleep(200 * time.Millisecond)

		mockSummarizer.AssertExpectations(t)
		mockMemSvc.AssertExpectations(t)
	})

	t.Run("写入内存时处理重复错误", func(t *testing.T) {
		mockMemSvc := new(MockMemoryService)
		mockConvSvc := new(MockConversationService)
		mockSummarizer := new(MockMemorySummarizer)
		logger := zap.NewNop()
		handler := NewMemoryEventHandler(mockMemSvc, mockConvSvc, mockSummarizer, logger, true)

		event := models.ConversationCompletedEvent{
			TraceID:     "trace-001",
			SessionKey:  "session-001",
			ChannelType: "dingtalk",
			Messages: []models.Message{
				{Role: "user", Content: "Hello"},
			},
			EndTime: time.Now(),
		}

		summary := &models.ConversationSummary{
			Summary:   "用户问候",
			KeyPoints: "要点",
		}

		// 模拟返回重复错误
		mockSummarizer.On("SummarizeConversation", mock.Anything, event.Messages).Return(summary, nil).Once()
		mockMemSvc.On("WriteMemory", mock.Anything, mock.Anything, mock.Anything).
			Return(service.ErrDuplicateTraceID).Once()

		err := handler.OnConversationCompleted(context.Background(), event)

		assert.NoError(t, err)
		time.Sleep(200 * time.Millisecond)

		mockSummarizer.AssertExpectations(t)
		mockMemSvc.AssertExpectations(t)
	})

	t.Run("构建内容包含关键要点", func(t *testing.T) {
		mockMemSvc := new(MockMemoryService)
		mockConvSvc := new(MockConversationService)
		mockSummarizer := new(MockMemorySummarizer)
		logger := zap.NewNop()
		handler := NewMemoryEventHandler(mockMemSvc, mockConvSvc, mockSummarizer, logger, true)

		event := models.ConversationCompletedEvent{
			TraceID:     "trace-001",
			SessionKey:  "session-001",
			ChannelType: "dingtalk",
			Messages: []models.Message{
				{Role: "user", Content: "Hello"},
			},
			EndTime: time.Now(),
		}

		summary := &models.ConversationSummary{
			Summary:   "对话总结",
			KeyPoints: "1. 要点一\n2. 要点二",
		}

		mockSummarizer.On("SummarizeConversation", mock.Anything, event.Messages).Return(summary, nil).Once()

		// 验证写入的内容包含关键要点
		mockMemSvc.On("WriteMemory", mock.Anything,
			mock.MatchedBy(func(content string) bool {
				return content == "对话总结\n\n关键要点:\n1. 要点一\n2. 要点二"
			}),
			mock.Anything,
		).Return(nil).Once()

		err := handler.OnConversationCompleted(context.Background(), event)

		assert.NoError(t, err)
		time.Sleep(200 * time.Millisecond)

		mockMemSvc.AssertExpectations(t)
	})
}

// TestMemoryEventHandler_HandleAsync 测试异步处理
func TestMemoryEventHandler_HandleAsync(t *testing.T) {
	mockMemSvc := new(MockMemoryService)
	mockConvSvc := new(MockConversationService)
	mockSummarizer := new(MockMemorySummarizer)
	logger := zap.NewNop()
	handler := NewMemoryEventHandler(mockMemSvc, mockConvSvc, mockSummarizer, logger, true)

	event := models.ConversationCompletedEvent{
		TraceID:     "trace-001",
		SessionKey:  "session-001",
		ChannelType: "dingtalk",
		Messages: []models.Message{
			{Role: "user", Content: "Hello"},
		},
		EndTime: time.Now(),
	}

	summary := &models.ConversationSummary{
		Summary: "测试总结",
	}

	mockSummarizer.On("SummarizeConversation", mock.Anything, event.Messages).Return(summary, nil).Once()
	mockMemSvc.On("WriteMemory", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	// 异步处理
	errChan := handler.HandleAsync(context.Background(), event)

	// 等待处理完成
	err := <-errChan
	assert.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	mockSummarizer.AssertExpectations(t)
	mockMemSvc.AssertExpectations(t)
}

// TestMemoryEventHandler_HandleAsync_WithError 测试异步处理错误
func TestMemoryEventHandler_HandleAsync_WithError(t *testing.T) {
	mockMemSvc := new(MockMemoryService)
	mockConvSvc := new(MockConversationService)
	mockSummarizer := new(MockMemorySummarizer)
	logger := zap.NewNop()
	handler := NewMemoryEventHandler(mockMemSvc, mockConvSvc, mockSummarizer, logger, true)

	event := models.ConversationCompletedEvent{
		TraceID:     "trace-001",
		SessionKey:  "session-001",
		ChannelType: "dingtalk",
		Messages:    []models.Message{},
		EndTime:     time.Now(),
	}

	expectedErr := errors.New("summarization failed")
	mockSummarizer.On("SummarizeConversation", mock.Anything, mock.Anything).
		Return(nil, expectedErr).Once()
	mockMemSvc.On("WriteMemory", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	errChan := handler.HandleAsync(context.Background(), event)

	// OnConversationCompleted 在 summarizer 失败时不会返回错误，而是使用默认内容
	err := <-errChan
	assert.NoError(t, err)

	time.Sleep(200 * time.Millisecond)
}

// TestMemoryEventHandler_Metadata 测试元数据构建
func TestMemoryEventHandler_Metadata(t *testing.T) {
	mockMemSvc := new(MockMemoryService)
	mockConvSvc := new(MockConversationService)
	mockSummarizer := new(MockMemorySummarizer)
	logger := zap.NewNop()
	handler := NewMemoryEventHandler(mockMemSvc, mockConvSvc, mockSummarizer, logger, true)

	event := models.ConversationCompletedEvent{
		TraceID:     "trace-abc-123",
		SessionKey:  "dingtalk:chat-123",
		ChannelType: "dingtalk",
		Messages: []models.Message{
			{Role: "user", Content: "Test"},
		},
		EndTime: time.Now(),
	}

	summary := &models.ConversationSummary{
		Summary:   "测试",
		KeyPoints: "要点",
	}

	mockSummarizer.On("SummarizeConversation", mock.Anything, event.Messages).Return(summary, nil).Once()

	// 验证元数据正确性
	mockMemSvc.On("WriteMemory", mock.Anything, mock.Anything,
		mock.MatchedBy(func(metadata map[string]interface{}) bool {
			return metadata["trace_id"] == "trace-abc-123" &&
				metadata["session_key"] == "dingtalk:chat-123" &&
				metadata["channel_type"] == "dingtalk" &&
				metadata["event_type"] == "conversation_completed" &&
				metadata["summary"] == "测试" &&
				metadata["key_points"] == "要点"
		}),
	).Return(nil).Once()

	err := handler.OnConversationCompleted(context.Background(), event)

	assert.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	mockMemSvc.AssertExpectations(t)
}
