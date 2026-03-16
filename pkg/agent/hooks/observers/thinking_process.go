package observers

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/observer"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/trace"
	"github.com/weibaohui/nanobot-go/pkg/bus"
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
	chatID                string
	channel               string
	enableThinkingProcess bool
	updatedAt             time.Time
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
	// 检查是否启用 - 优先从 context 获取 Agent 级别的设置
	enabled := trace.GetEnableThinkingProcess(ctx)
	sessionKey := trace.GetSessionKey(ctx)

	// 如果 context 中没有设置，尝试从 sessionCache 获取
	if !enabled && sessionKey != "" {
		o.mu.RLock()
		if info, exists := o.sessionCache[sessionKey]; exists {
			enabled = info.enableThinkingProcess
		}
		o.mu.RUnlock()
	}

	// 如果还是没有，使用全局配置作为后备
	if !enabled {
		enabled = o.config.Enabled
	}

	// 先尝试更新会话缓存（从有会话信息的事件中）
	// 注意：缓存更新要在 enabled 检查之前，确保即使思考过程未启用，缓存也能被更新
	o.updateSessionCache(ctx, event)

	if !enabled {
		return nil
	}

	// 检查事件类型是否在监听列表中
	if !o.shouldProcessEvent(event.GetEventType()) {
		return nil
	}

	// 从 context 获取会话信息（sessionKey 已在上面获取）
	channel := trace.GetChannel(ctx)

	// 获取 chatID 和 channel（优先从缓存获取完整的会话信息）
	chatID, cachedChannel := o.getSessionInfo(event, sessionKey)
	if chatID == "" {
		o.logger.Info("[ThinkingProcess] 无法获取 ChatID，跳过",
			zap.String("event_type", string(event.GetEventType())),
			zap.String("session_key", sessionKey),
		)
		return nil
	}

	o.logger.Info("[ThinkingProcess] 处理事件",
		zap.String("event_type", string(event.GetEventType())),
		zap.String("session_key", sessionKey),
		zap.String("chat_id", chatID),
		zap.String("channel", channel),
	)

	// 如果 context 中没有 channel，使用缓存的 channel
	if channel == "" && cachedChannel != "" {
		channel = cachedChannel
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
func (o *ThinkingProcessObserver) updateSessionCache(ctx context.Context, event events.Event) {
	switch e := event.(type) {
	case *events.MessageReceivedEvent:
		// 收到消息时，缓存会话信息
		if e.SessionKey != "" && e.ChatID != "" {
			o.mu.Lock()
			o.sessionCache[e.SessionKey] = sessionInfo{
				chatID:                e.ChatID,
				channel:               e.Channel,
				enableThinkingProcess: trace.GetEnableThinkingProcess(ctx),
				updatedAt:             time.Now(),
			}
			o.mu.Unlock()
		}

	case *events.MessageSentEvent:
		// 发送消息时也更新缓存
		if e.SessionKey != "" && e.ChatID != "" {
			o.mu.Lock()
			o.sessionCache[e.SessionKey] = sessionInfo{
				chatID:                e.ChatID,
				channel:               e.Channel,
				enableThinkingProcess: trace.GetEnableThinkingProcess(ctx),
				updatedAt:             time.Now(),
			}
			o.mu.Unlock()
		}

	case *events.PromptSubmittedEvent:
		// Prompt 提交时也更新缓存（如果有 sessionKey）
		if e.SessionKey != "" {
			o.mu.RLock()
			info, exists := o.sessionCache[e.SessionKey]
			o.mu.RUnlock()
			if exists {
				o.mu.Lock()
				o.sessionCache[e.SessionKey] = sessionInfo{
					chatID:                info.chatID,
					channel:               info.channel,
					enableThinkingProcess: trace.GetEnableThinkingProcess(ctx),
					updatedAt:             time.Now(),
				}
				o.mu.Unlock()
			}
		}
	}
}

// getSessionInfo 获取会话信息 (chatID, channel)
// 优先从事件中获取，其次从缓存中通过 sessionKey 查找
func (o *ThinkingProcessObserver) getSessionInfo(event events.Event, sessionKey string) (string, string) {
	// 首先尝试从事件中直接获取
	switch e := event.(type) {
	case *events.MessageReceivedEvent:
		return e.ChatID, e.Channel
	case *events.MessageSentEvent:
		return e.ChatID, e.Channel
	}

	// 从缓存中通过 sessionKey 查找
	if sessionKey != "" {
		o.mu.RLock()
		info, exists := o.sessionCache[sessionKey]
		o.mu.RUnlock()
		if exists && time.Since(info.updatedAt) < 30*time.Minute {
			return info.chatID, info.channel
		}
	}

	return "", ""
}

// shouldProcessEvent 检查是否应该处理该事件类型
func (o *ThinkingProcessObserver) shouldProcessEvent(eventType events.EventType) bool {
	// 默认监听所有思考和工具相关事件（使用常量）
	defaultEvents := []events.EventType{
		events.EventLLMCallStart,    // LLM 开始思考
		events.EventLLMCallEnd,      // LLM 思考完成
		events.EventLLMCallError,    // LLM 调用错误
		events.EventToolUsed,        // 工具开始执行
		events.EventToolCompleted,   // 工具执行完成
		events.EventToolError,       // 工具执行错误
		events.EventToolCall,        // 工具调用
		events.EventComponentStart,  // 组件开始
		events.EventComponentEnd,    // 组件完成
		events.EventComponentError,  // 组件错误
	}

	// 如果配置了事件列表，使用配置的
	if len(o.config.Events) > 0 {
		for _, e := range o.config.Events {
			if events.EventType(e) == eventType {
				return true
			}
		}
		return false
	}

	// 否则使用默认列表
	for _, e := range defaultEvents {
		if e == eventType {
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
	case *events.LLMCallStartEvent:
		return o.formatLLMCallStart(e)
	case *events.LLMCallEndEvent:
		return o.formatLLMCallEnd(e)
	case *events.LLMCallErrorEvent:
		return o.formatLLMCallError(e)
	case *events.ComponentStartEvent:
		return o.formatComponentStart(e)
	case *events.ComponentEndEvent:
		return o.formatComponentEnd(e)
	case *events.ComponentErrorEvent:
		return o.formatComponentError(e)
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

// formatLLMCallStart 格式化 LLM 调用开始事件
// 返回空字符串，不显示"开始思考"提示，避免与最终回复重复
func (o *ThinkingProcessObserver) formatLLMCallStart(e *events.LLMCallStartEvent) string {
	return ""
}

// formatLLMCallEnd 格式化 LLM 调用结束事件
// 只显示工具调用决定，不显示普通回复内容（避免与最终回复重复）
func (o *ThinkingProcessObserver) formatLLMCallEnd(e *events.LLMCallEndEvent) string {
	// 如果有工具调用，显示工具调用信息
	if len(e.ToolCalls) > 0 {
		var toolNames []string
		for _, tc := range e.ToolCalls {
			toolNames = append(toolNames, tc.Function.Name)
		}
		return fmt.Sprintf("🤖 **决定调用工具**: %s", strings.Join(toolNames, ", "))
	}

	// 普通回复内容不显示，由正常的消息回复发送
	return ""
}

// formatLLMCallError 格式化 LLM 调用错误事件
func (o *ThinkingProcessObserver) formatLLMCallError(e *events.LLMCallErrorEvent) string {
	return fmt.Sprintf("❌ **AI 调用错误**: %s", e.Error)
}

// formatComponentStart 格式化组件开始事件
func (o *ThinkingProcessObserver) formatComponentStart(e *events.ComponentStartEvent) string {
	return fmt.Sprintf("▶️ **开始**: %s", e.Name)
}

// formatComponentEnd 格式化组件结束事件
func (o *ThinkingProcessObserver) formatComponentEnd(e *events.ComponentEndEvent) string {
	return fmt.Sprintf("✅ **完成**: %s (%dms)", e.Name, e.DurationMs)
}

// formatComponentError 格式化组件错误事件
func (o *ThinkingProcessObserver) formatComponentError(e *events.ComponentErrorEvent) string {
	return fmt.Sprintf("❌ **错误**: %s - %s", e.Name, e.Error)
}

// sendThinkingMessage 发送思考过程消息
func (o *ThinkingProcessObserver) sendThinkingMessage(channel, chatID, content string) {
	if o.messageBus == nil {
		return
	}

	contentPreview := content
	if len(contentPreview) > 100 {
		contentPreview = contentPreview[:100] + "..."
	}
	o.logger.Info("[ThinkingProcess] 发送思考消息",
		zap.String("channel", channel),
		zap.String("chat_id", chatID),
		zap.String("content_preview", contentPreview),
	)

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
