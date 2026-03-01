package observers

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

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

	// 去重：记录最近保存的消息（基于 role + content + 秒级时间戳）
	recentMessages map[string]map[string]time.Time // sessionKey -> messageKey -> timestamp
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
		recentMessages: make(map[string]map[string]time.Time),
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

// isDuplicate 检查是否是重复消息（基于 role + content + 秒级时间戳）
func (so *SessionObserver) isDuplicate(sessionKey, role, content string, timestamp time.Time) bool {
	messageKey := so.buildMessageKey(role, content)

	so.mu.RLock()
	messages, exists := so.recentMessages[sessionKey]
	so.mu.RUnlock()

	if !exists {
		return false
	}

	so.mu.RLock()
	lastTime, exists := messages[messageKey]
	so.mu.RUnlock()

	if !exists {
		return false
	}

	// 如果在1秒内（包含），认为是重复
	return timestamp.Sub(lastTime) < time.Second
}

// buildMessageKey 构建消息唯一键（基于 role + content）
func (so *SessionObserver) buildMessageKey(role, content string) string {
	return role + "|" + content
}

// recordMessage 记录消息用于去重（基于 role + content + 秒级时间戳）
func (so *SessionObserver) recordMessage(sessionKey, role, content string, timestamp time.Time) {
	messageKey := so.buildMessageKey(role, content)

	so.mu.Lock()
	defer so.mu.Unlock()

	if so.recentMessages[sessionKey] == nil {
		so.recentMessages[sessionKey] = make(map[string]time.Time)
	}

	// 记录消息时间
	so.recentMessages[sessionKey][messageKey] = timestamp

	// 清理超过1秒的记录
	so.cleanupExpiredMessages(sessionKey, timestamp)
}

// cleanupExpiredMessages 清理超过1秒的消息记录
func (so *SessionObserver) cleanupExpiredMessages(sessionKey string, currentTime time.Time) {
	messages, exists := so.recentMessages[sessionKey]
	if !exists {
		return
	}

	for key, ts := range messages {
		if currentTime.Sub(ts) >= time.Second {
			delete(messages, key)
		}
	}
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

	timestamp := event.GetTimestamp()

	// 去重检查（基于 role + content + 秒级时间戳）
	if so.isDuplicate(e.SessionKey, "user", e.UserInput, timestamp) {
		return
	}

	// 立即记录消息用于去重（防止并发导致重复）
	so.recordMessage(e.SessionKey, "user", e.UserInput, timestamp)

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

	// 去重检查（基于 role + content + 秒级时间戳）
	if so.isDuplicate(sessionKey, role, content, event.GetTimestamp()) {
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
	so.recordMessage(sessionKey, role, content, event.GetTimestamp())

	sess := so.sessions.GetOrCreate(sessionKey)

	// 只记录对话内容，不记录 TokenUsage
	sess.AddMessageWithTrace(role, content, e.GetTraceID(), e.GetSpanID(), e.GetParentSpanID())

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
