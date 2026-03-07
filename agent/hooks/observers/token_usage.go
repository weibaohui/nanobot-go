package observers

import (
	"context"

	"github.com/weibaohui/nanobot-go/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/agent/hooks/observer"
	"github.com/weibaohui/nanobot-go/agent/hooks/trace"
	"github.com/weibaohui/nanobot-go/token_usage"
	"go.uber.org/zap"
)

// TokenUsageObserver Token 使用量观察器
// 监听 LLM 调用事件，记录 Token 使用情况到独立文件
type TokenUsageObserver struct {
	*observer.BaseObserver
	usageManager *token_usage.TokenUsageManager
	logger       *zap.Logger
}

// NewTokenUsageObserver 创建 Token 使用量观察器
func NewTokenUsageObserver(usageManager *token_usage.TokenUsageManager, logger *zap.Logger, filter *observer.ObserverFilter) *TokenUsageObserver {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &TokenUsageObserver{
		BaseObserver: observer.NewBaseObserver("token_usage", filter),
		usageManager: usageManager,
		logger:       logger,
	}
}

// OnEvent 处理事件
func (tuo *TokenUsageObserver) OnEvent(ctx context.Context, event events.Event) error {
	switch event.GetEventType() {
	case events.EventLLMCallEnd:
		tuo.handleLLMCallEnd(ctx, event)
	}
	return nil
}

// handleLLMCallEnd 处理 LLM 调用结束事件
// 记录 Token 使用情况
func (tuo *TokenUsageObserver) handleLLMCallEnd(ctx context.Context, event events.Event) {
	e, ok := event.(*events.LLMCallEndEvent)
	if !ok {
		return
	}

	// 如果没有 TokenUsage 信息，跳过
	if e.TokenUsage == nil {
		return
	}

	// 从 context 获取 sessionKey（使用 trace.GetSessionKey）
	sessionKey := trace.GetSessionKey(ctx)
	// 如果 context 中没有 sessionKey，使用 trace_id
	if sessionKey == "" {
		sessionKey = e.GetTraceID()
	}

	record := token_usage.TokenUsageRecord{
		TraceID: e.GetTraceID(),
		SpanID:  e.GetSpanID(),
		TokenUsage: token_usage.TokenUsage{
			PromptTokens:     e.TokenUsage.PromptTokens,
			CompletionTokens: e.TokenUsage.CompletionTokens,
			TotalTokens:      e.TokenUsage.TotalTokens,
			ReasoningTokens:  e.TokenUsage.CompletionTokensDetails.ReasoningTokens,
			CachedTokens:     e.TokenUsage.PromptTokenDetails.CachedTokens,
		},
		Timestamp: event.GetTimestamp(),
	}

	// 添加到管理器
	if err := tuo.usageManager.AddRecord(sessionKey, record); err != nil {
		tuo.logger.Error("添加 Token 使用记录失败",
			zap.String("session_key", sessionKey),
			zap.Error(err),
		)
		return
	}

	// 立即保存到磁盘
	if err := tuo.usageManager.Save(sessionKey); err != nil {
		tuo.logger.Error("保存 Token 使用记录失败",
			zap.String("session_key", sessionKey),
			zap.Error(err),
		)
	}
}