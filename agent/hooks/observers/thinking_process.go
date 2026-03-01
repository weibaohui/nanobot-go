package observers

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/weibaohui/nanobot-go/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/agent/hooks/observer"
	"github.com/weibaohui/nanobot-go/agent/hooks/trace"
	"github.com/weibaohui/nanobot-go/bus"
	"github.com/weibaohui/nanobot-go/config"
	"go.uber.org/zap"
)

// ThinkingProcessObserver 思考过程观察器
// 监听 LLM 调用、工具使用等事件，并将这些信息实时发送到 channel
// 让用户能够看到 AI 的思考过程
type ThinkingProcessObserver struct {
	*observer.BaseObserver
	config     *config.ThinkingProcessConfig
	messageBus *bus.MessageBus
	logger     *zap.Logger

	// 会话信息缓存（sessionKey -> chatID）
	// 用于从 sessionKey 获取 chatID
	sessionCache map[string]sessionInfo
	mu           sync.RWMutex
}

// sessionInfo 会话信息
type sessionInfo struct {
	chatID    string
	channel   string
	updatedAt time.Time
}

// NewThinkingProcessObserver 创建思考过程观察器
func NewThinkingProcessObserver(cfg *config.ThinkingProcessConfig, messageBus *bus.MessageBus, logger *zap.Logger, filter *observer.ObserverFilter) *ThinkingProcessObserver {
	if logger == nil {
		logger = zap.NewNop()
	}
	if cfg == nil {
		cfg = &config.ThinkingProcessConfig{Enabled: false}
	}

	return &ThinkingProcessObserver{
		BaseObserver: observer.NewBaseObserver("thinking_process", filter),
		config:       cfg,
		messageBus:   messageBus,
		logger:       logger,
		sessionCache: make(map[string]sessionInfo),
	}
}

// OnEvent 处理事件
func (o *ThinkingProcessObserver) OnEvent(ctx context.Context, event events.Event) error {
	// 检查是否启用
	if !o.config.Enabled {
		return nil
	}

	// 先尝试更新会话缓存（从有会话信息的事件中）
	o.updateSessionCache(event)

	// 检查事件类型是否在监听列表中
	if !o.shouldProcessEvent(event.GetEventType()) {
		return nil
	}

	// 从 context 获取会话信息
	sessionKey := trace.GetSessionKey(ctx)
	channel := trace.GetChannel(ctx)

	// 获取 chatID
	chatID := o.getChatID(event, sessionKey)
	if chatID == "" {
		o.logger.Debug("无法获取 ChatID，跳过思考过程推送",
			zap.String("event_type", string(event.GetEventType())),
			zap.String("session_key", sessionKey),
		)
		return nil
	}

	// 格式化消息
	content := o.formatMessage(event)
	if content == "" {
		return nil
	}

	// 发送消息到 channel
	o.sendThinkingMessage(channel, chatID, content)

	return nil
}

// updateSessionCache 更新会话缓存
func (o *ThinkingProcessObserver) updateSessionCache(event events.Event) {
	switch e := event.(type) {
	case *events.MessageReceivedEvent:
		// 收到消息时，缓存会话信息
		if e.SessionKey != "" && e.ChatID != "" {
			o.mu.Lock()
			o.sessionCache[e.SessionKey] = sessionInfo{
				chatID:    e.ChatID,
				channel:   e.Channel,
				updatedAt: time.Now(),
			}
			o.mu.Unlock()
		}

	case *events.MessageSentEvent:
		// 发送消息时也更新缓存
		if e.SessionKey != "" && e.ChatID != "" {
			o.mu.Lock()
			o.sessionCache[e.SessionKey] = sessionInfo{
				chatID:    e.ChatID,
				channel:   e.Channel,
				updatedAt: time.Now(),
			}
			o.mu.Unlock()
		}
	}
}

// getChatID 获取 ChatID
// 优先从事件中获取，其次从缓存中通过 sessionKey 查找
func (o *ThinkingProcessObserver) getChatID(event events.Event, sessionKey string) string {
	// 首先尝试从事件中直接获取
	switch e := event.(type) {
	case *events.MessageReceivedEvent:
		return e.ChatID
	case *events.MessageSentEvent:
		return e.ChatID
	}

	// 从缓存中通过 sessionKey 查找
	if sessionKey != "" {
		o.mu.RLock()
		info, exists := o.sessionCache[sessionKey]
		o.mu.RUnlock()
		if exists && time.Since(info.updatedAt) < 10*time.Minute {
			return info.chatID
		}
	}

	return ""
}

// shouldProcessEvent 检查是否应该处理该事件类型
func (o *ThinkingProcessObserver) shouldProcessEvent(eventType events.EventType) bool {
	// 如果没有配置特定事件，使用默认事件列表
	defaultEvents := []string{
		"tool_used",
		"tool_completed",
		"llm_call_end",
	}

	eventStr := string(eventType)

	// 如果配置了事件列表，使用配置的
	if len(o.config.Events) > 0 {
		for _, e := range o.config.Events {
			if e == eventStr {
				return true
			}
		}
		return false
	}

	// 否则使用默认列表
	for _, e := range defaultEvents {
		if e == eventStr {
			return true
		}
	}
	return false
}

// formatMessage 格式化事件为消息内容
func (o *ThinkingProcessObserver) formatMessage(event events.Event) string {
	switch e := event.(type) {
	case *events.ToolUsedEvent:
		return o.formatToolUsed(e)
	case *events.ToolCompletedEvent:
		return o.formatToolCompleted(e)
	case *events.ToolErrorEvent:
		return o.formatToolError(e)
	case *events.LLMCallEndEvent:
		return o.formatLLMCallEnd(e)
	default:
		return ""
	}
}

// formatToolUsed 格式化工具调用事件
func (o *ThinkingProcessObserver) formatToolUsed(e *events.ToolUsedEvent) string {
	// 简化参数显示
	args := e.ToolArguments
	if len(args) > 100 {
		args = args[:100] + "..."
	}
	return fmt.Sprintf("🔧 **调用工具**: `%s`\n```\n%s\n```", e.ToolName, args)
}

// formatToolCompleted 格式化工具完成事件
func (o *ThinkingProcessObserver) formatToolCompleted(e *events.ToolCompletedEvent) string {
	// 简化响应显示
	resp := e.Response
	if len(resp) > 200 {
		resp = resp[:200] + "..."
	}
	// 清理响应中的多余空白
	resp = strings.TrimSpace(resp)
	if resp == "" {
		resp = "(无输出)"
	}

	status := "✅"
	if !e.Success {
		status = "⚠️"
	}
	return fmt.Sprintf("%s **工具完成**: `%s`\n```\n%s\n```", status, e.ToolName, resp)
}

// formatToolError 格式化工具错误事件
func (o *ThinkingProcessObserver) formatToolError(e *events.ToolErrorEvent) string {
	return fmt.Sprintf("❌ **工具错误**: `%s`\n```\n%s\n```", e.ToolName, e.Error)
}

// formatLLMCallEnd 格式化 LLM 调用结束事件
func (o *ThinkingProcessObserver) formatLLMCallEnd(e *events.LLMCallEndEvent) string {
	// 只在有响应内容时才发送
	if e.ResponseContent == "" {
		return ""
	}

	// 限制内容长度，避免发送过长消息
	content := e.ResponseContent
	if len(content) > 500 {
		content = content[:500] + "..."
	}

	// 如果有工具调用，显示工具调用信息而不是响应内容
	if len(e.ToolCalls) > 0 {
		var toolNames []string
		for _, tc := range e.ToolCalls {
			toolNames = append(toolNames, tc.Function.Name)
		}
		return fmt.Sprintf("🤖 **准备调用工具**: %s", strings.Join(toolNames, ", "))
	}

	return content
}

// sendThinkingMessage 发送思考过程消息
func (o *ThinkingProcessObserver) sendThinkingMessage(channel, chatID, content string) {
	if o.messageBus == nil {
		return
	}

	// 使用 "thinking" 作为特殊 channel 标识
	// 实际发送时使用原始 channel
	msg := &bus.OutboundMessage{
		Channel: channel,
		ChatID:  chatID,
		Content: content,
		Metadata: map[string]any{
			"type":      "thinking_process",
			"timestamp": time.Now().Unix(),
		},
	}

	// 异步发送，避免阻塞主流程
	go func() {
		defer func() {
			if r := recover(); r != nil {
				o.logger.Error("发送思考过程消息 panic",
					zap.Any("recover", r),
				)
			}
		}()
		o.messageBus.PublishOutbound(msg)
	}()
}
