package handler

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"github.com/weibaohui/nanobot-go/memory/models"
	"github.com/weibaohui/nanobot-go/memory/service"
)

// ConversationService 对话服务接口（用于查询完整对话内容）
// 注意：这里使用 interface{} 避免循环导入，实际调用时需类型断言
type ConversationService interface {
	// GetByTraceID 根据 TraceID 获取完整对话
	GetByTraceID(ctx context.Context, traceID string) ([]interface{}, error)
}

// MemoryEventHandler 记忆事件处理器
type MemoryEventHandler struct {
	memoryService     service.MemoryService
	conversationSvc   ConversationService
	summarizer        service.MemorySummarizer
	logger            *zap.Logger
	enabled           bool
}

// NewMemoryEventHandler 创建记忆事件处理器
func NewMemoryEventHandler(
	memoryService service.MemoryService,
	conversationSvc ConversationService,
	summarizer service.MemorySummarizer,
	logger *zap.Logger,
	enabled bool,
) *MemoryEventHandler {
	return &MemoryEventHandler{
		memoryService:   memoryService,
		conversationSvc: conversationSvc,
		summarizer:      summarizer,
		logger:          logger,
		enabled:         enabled,
	}
}

// OnConversationCompleted 处理对话完成事件
// 这是核心的事件处理方法，由事件总线调用
func (h *MemoryEventHandler) OnConversationCompleted(ctx context.Context, event models.ConversationCompletedEvent) error {
	if !h.enabled {
		return nil
	}

	h.logger.Info("处理对话完成事件",
		zap.String("trace_id", event.TraceID),
		zap.String("session_key", event.SessionKey),
		zap.Int("message_count", len(event.Messages)),
	)

	// 1. 幂等检查会在 WriteMemory 中自动处理

	// 2. 如果事件中没有消息，但有 trace_id，尝试从对话服务查询
	messages := event.Messages
	if len(messages) == 0 && event.TraceID != "" {
		// 这里简化处理，实际应该从 conversationSvc 查询
		// 由于循环依赖问题，暂时直接从事件中获取
		h.logger.Warn("事件中无消息内容，跳过总结",
			zap.String("trace_id", event.TraceID),
		)
		// 使用简化内容
		messages = []models.Message{
			{
				Role:      "system",
				Content:   fmt.Sprintf("对话完成: trace_id=%s, session=%s", event.TraceID, event.SessionKey),
				Timestamp: event.EndTime,
			},
		}
	}

	// 3. 生成初步总结
	summary, err := h.summarizer.SummarizeConversation(ctx, messages)
	if err != nil {
		h.logger.Error("生成对话总结失败",
			zap.String("trace_id", event.TraceID),
			zap.Error(err),
		)
		// 总结失败时，使用原始内容作为备选
		summary = &models.ConversationSummary{
			Summary:   fmt.Sprintf("对话完成: %d 条消息", len(messages)),
			KeyPoints: "",
		}
	}

	// 4. 准备元数据
	metadata := map[string]interface{}{
		"trace_id":     event.TraceID,
		"session_key":  event.SessionKey,
		"channel_type": event.ChannelType,
		"event_type":   "conversation_completed",
		"summary":      summary.Summary,
		"key_points":   summary.KeyPoints,
	}

	// 构建内容：总结 + 关键要点
	content := summary.Summary
	if summary.KeyPoints != "" {
		content += "\n\n关键要点:\n" + summary.KeyPoints
	}

	// 5. 写入流水记忆（异步执行，不阻塞）
	go func() {
		writeCtx := context.Background()
		if err := h.memoryService.WriteMemory(writeCtx, content, metadata); err != nil {
			// 幂等错误不算真正的错误
			if err == service.ErrDuplicateTraceID {
				h.logger.Info("流水记忆已存在，跳过写入",
					zap.String("trace_id", event.TraceID),
				)
				return
			}
			h.logger.Error("写入流水记忆失败",
				zap.String("trace_id", event.TraceID),
				zap.Error(err),
			)
			return
		}
		h.logger.Info("流水记忆写入成功",
			zap.String("trace_id", event.TraceID),
		)
	}()

	return nil
}

// HandleAsync 异步处理对话完成事件
// 返回一个立即完成的 channel，实际处理在后台 goroutine 中进行
func (h *MemoryEventHandler) HandleAsync(ctx context.Context, event models.ConversationCompletedEvent) <-chan error {
	errChan := make(chan error, 1)

	go func() {
		defer close(errChan)
		if err := h.OnConversationCompleted(ctx, event); err != nil {
			errChan <- err
		}
	}()

	return errChan
}
