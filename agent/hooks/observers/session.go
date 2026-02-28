package observers

import (
	"context"
	"crypto/md5"
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

	// 去重：记录最近保存的消息（sessionKey -> messageHash -> timestamp）
	// 1分钟内的相同消息不会被重复保存
	recentMessages map[string]map[string]time.Time
	mu           sync.RWMutex
}

// NewSessionObserver 创建会话观察器
func NewSessionObserver(sessions *session.Manager, logger *zap.Logger, filter *observer.ObserverFilter) *SessionObserver {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SessionObserver{
		BaseObserver:  observer.NewBaseObserver("session", filter),
		sessions:      sessions,
		logger:        logger,
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

// isDuplicate 检查是否是重复消息
func (so *SessionObserver) isDuplicate(sessionKey, content string) bool {
	hash := so.hashContent(content)

	so.mu.RLock()
	sessionMsgs, exists := so.recentMessages[sessionKey]
	so.mu.RUnlock()

	if !exists {
		return false
	}

	so.mu.RLock()
	timestamp, exists := sessionMsgs[hash]
	so.mu.RUnlock()

	if !exists {
		return false
	}

	// 1分钟内的消息认为是重复的
	return time.Since(timestamp) < time.Minute
}

// recordMessage 记录消息用于去重
func (so *SessionObserver) recordMessage(sessionKey, content string) {
	hash := so.hashContent(content)

	so.mu.Lock()
	if so.recentMessages[sessionKey] == nil {
		so.recentMessages[sessionKey] = make(map[string]time.Time)
	}
	so.recentMessages[sessionKey][hash] = time.Now()

	// 清理1分钟前的记录
	so.cleanupOldMessages(sessionKey)
	so.mu.Unlock()
}

// hashContent 计算消息内容的哈希值
func (so *SessionObserver) hashContent(content string) string {
	h := md5.New()
	h.Write([]byte(content))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// cleanupOldMessages 清理1分钟前的记录
func (so *SessionObserver) cleanupOldMessages(sessionKey string) {
	cutoff := time.Now().Add(-time.Minute)
	for hash, timestamp := range so.recentMessages[sessionKey] {
		if timestamp.Before(cutoff) {
			delete(so.recentMessages[sessionKey], hash)
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

	// 去重检查
	if so.isDuplicate(e.SessionKey, e.UserInput) {
		return
	}

	sess := so.sessions.GetOrCreate(e.SessionKey)
	sess.AddMessage("user", e.UserInput)
	if err := so.sessions.Save(sess); err != nil {
		so.logger.Error("保存用户消息到会话失败",
			zap.String("session_key", e.SessionKey),
			zap.Error(err),
		)
		return
	}

	// 记录用于去重
	so.recordMessage(e.SessionKey, e.UserInput)
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

	// 去重检查
	if so.isDuplicate(sessionKey, content) {
		so.logger.Debug("跳过重复消息",
			zap.String("session_key", sessionKey),
			zap.String("role", role),
			zap.String("content_preview", func() string {
				if len(content) > 50 {
					return content[:50] + "..."
				}
				return content
			}()),
		)
		return
	}

	sess := so.sessions.GetOrCreate(sessionKey)
	sess.AddMessage(role, content)

	// 如果有 TokenUsage，更新到 session
	if e.TokenUsage != nil {
		tokenUsage := session.TokenUsage{
			PromptTokens:     e.TokenUsage.PromptTokens,
			CompletionTokens: e.TokenUsage.CompletionTokens,
			TotalTokens:      e.TokenUsage.TotalTokens,
			ReasoningTokens:  0, // model.TokenUsage 没有此字段
			CachedTokens:     0, // model.TokenUsage 没有此字段
		}
		sess.UpdateTokenUsage(tokenUsage)
	}

	if err := so.sessions.Save(sess); err != nil {
		so.logger.Error("保存助手消息到会话失败",
			zap.String("session_key", sessionKey),
			zap.Error(err),
		)
		return
	}

	// 记录用于去重
	so.recordMessage(sessionKey, content)
}

// getCtxSessionKey 从上下文获取 sessionKey
// 这里需要一个约定：将 sessionKey 存储在 context 中
func getCtxSessionKey(ctx context.Context) string {
	if sessionKey, ok := ctx.Value("session_key").(string); ok {
		return sessionKey
	}
	return ""
}
