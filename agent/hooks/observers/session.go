package observers

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/weibaohui/nanobot-go/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/agent/hooks/observer"
	"github.com/weibaohui/nanobot-go/session"
	"go.uber.org/zap"
)

// SessionObserver 会话观察器
// 监听消息事件，自动保存到会话
type SessionObserver struct {
	*observer.BaseObserver
	sessions *session.Manager
	logger   *zap.Logger

	// 去重：记录最近保存的消息（基于 role + content + span_id + trace_id 组合去重）
	recentMessages map[string]map[string]struct{} // sessionKey -> messageKey -> struct{}
	mu             sync.RWMutex
}

// NewSessionObserver 创建会话观察器
func NewSessionObserver(sessions *session.Manager, logger *zap.Logger, filter *observer.ObserverFilter) *SessionObserver {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SessionObserver{
		BaseObserver:   observer.NewBaseObserver("session", filter),
		sessions:       sessions,
		logger:         logger,
		recentMessages: make(map[string]map[string]struct{}),
	}
}

// OnEvent 处理事件
func (so *SessionObserver) OnEvent(ctx context.Context, event events.Event) error {
	switch event.GetEventType() {
	case events.EventPromptSubmitted:
		so.handlePromptSubmitted(ctx, event)
	case events.EventLLMCallEnd:
		so.handleLLMCallEnd(ctx, event)
	}
	return nil
}

// isDuplicate 检查是否是重复消息（基于 role + content + span_id + trace_id）
func (so *SessionObserver) isDuplicate(sessionKey, role, content, traceID, spanID string) bool {
	messageKey := so.buildMessageKey(role, content, traceID, spanID)

	so.mu.RLock()
	messages, exists := so.recentMessages[sessionKey]
	so.mu.RUnlock()

	if !exists {
		return false
	}

	so.mu.RLock()
	_, exists = messages[messageKey]
	so.mu.RUnlock()

	return exists
}

// buildMessageKey 构建消息唯一键（基于 role + content + span_id + trace_id）
func (so *SessionObserver) buildMessageKey(role, content, traceID, spanID string) string {
	return role + "|" + content + "|" + traceID + "|" + spanID
}

// recordMessage 记录消息用于去重（基于 role + content + span_id + trace_id）
func (so *SessionObserver) recordMessage(sessionKey, role, content, traceID, spanID string) {
	messageKey := so.buildMessageKey(role, content, traceID, spanID)

	so.mu.Lock()
	if so.recentMessages[sessionKey] == nil {
		so.recentMessages[sessionKey] = make(map[string]struct{})
	}
	so.recentMessages[sessionKey][messageKey] = struct{}{}
	so.mu.Unlock()
}

// handlePromptSubmitted 处理 Prompt 提交事件
// 保存用户消息到会话
func (so *SessionObserver) handlePromptSubmitted(_ context.Context, event events.Event) {
	e, ok := event.(*events.PromptSubmittedEvent)
	if !ok {
		return
	}

	if e.UserInput == "" || e.SessionKey == "" {
		return
	}

	// 去重检查（基于 role + content + span_id + trace_id）
	if so.isDuplicate(e.SessionKey, "user", e.UserInput, e.GetTraceID(), e.GetSpanID()) {
		return
	}

	// 立即记录消息用于去重（防止并发导致重复）
	so.recordMessage(e.SessionKey, "user", e.UserInput, e.GetTraceID(), e.GetSpanID())

	sess := so.sessions.GetOrCreate(e.SessionKey)
	sess.AddMessageWithTrace("user", e.UserInput, e.GetTraceID(), e.GetSpanID(), e.GetParentSpanID())
	if err := so.sessions.Save(sess); err != nil {
		so.logger.Error("保存用户消息到会话失败",
			zap.String("session_key", e.SessionKey),
			zap.Error(err),
		)
		return
	}
}

// handleLLMCallEnd 处理 LLM 调用结束事件
// 保存助手回复到会话
func (so *SessionObserver) handleLLMCallEnd(ctx context.Context, event events.Event) {
	e, ok := event.(*events.LLMCallEndEvent)
	if !ok {
		return
	}

	// 从 context 获取 sessionKey
	sessionKey := getCtxSessionKey(ctx)
	if sessionKey == "" {
		return
	}

	content := e.ResponseContent
	role := "assistant"

	// 检查是否有工具调用
	if len(e.ToolCalls) > 0 {
		role = "tool"
		// 拼接所有工具调用信息
		var toolCallsStr strings.Builder
		for _, tc := range e.ToolCalls {
			fmt.Fprintf(&toolCallsStr, "%s(%s) ", tc.Function.Name, tc.Function.Arguments)
		}
		content = toolCallsStr.String()
	}

	// 空内容不保存
	if content == "" {
		return
	}

	// 去重检查（基于 role + content + span_id + trace_id）
	if so.isDuplicate(sessionKey, role, content, e.GetTraceID(), e.GetSpanID()) {
		// so.logger.Debug("跳过重复消息（相同 role+content+trace_id+span_id）",
		// 	zap.String("session_key", sessionKey),
		// 	zap.String("role", role),
		// 	zap.String("trace_id", e.GetTraceID()),
		// 	zap.String("span_id", e.GetSpanID()),
		// 	zap.String("content_preview", func() string {
		// 		if len(content) > 50 {
		// 			return content[:50] + "..."
		// 		}
		// 		return content
		// 	}()),
		// )
		return
	}

	// 立即记录消息用于去重（防止并发导致重复）
	so.recordMessage(sessionKey, role, content, e.GetTraceID(), e.GetSpanID())

	sess := so.sessions.GetOrCreate(sessionKey)

	// 如果有 TokenUsage，使用 AddMessageWithTokenUsageAndTrace（记录到消息级别并累加到 session 级别）
	if e.TokenUsage != nil {
		tokenUsage := session.TokenUsage{
			PromptTokens:     e.TokenUsage.PromptTokens,
			CompletionTokens: e.TokenUsage.CompletionTokens,
			TotalTokens:      e.TokenUsage.TotalTokens,
			ReasoningTokens:  0,
			CachedTokens:     0,
		}
		sess.AddMessageWithTokenUsageAndTrace(role, content, tokenUsage, e.GetTraceID(), e.GetSpanID(), e.GetParentSpanID())
	} else {
		// 没有 TokenUsage，使用 AddMessageWithTrace
		sess.AddMessageWithTrace(role, content, e.GetTraceID(), e.GetSpanID(), e.GetParentSpanID())
	}

	if err := so.sessions.Save(sess); err != nil {
		so.logger.Error("保存助手消息到会话失败",
			zap.String("session_key", sessionKey),
			zap.Error(err),
		)
		return
	}
}

// getCtxSessionKey 从上下文获取 sessionKey
// 这里需要一个约定：将 sessionKey 存储在 context 中
func getCtxSessionKey(ctx context.Context) string {
	if sessionKey, ok := ctx.Value("session_key").(string); ok {
		return sessionKey
	}
	return ""
}
