package events

import (
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/pkg/bus"
)

// MessageReceivedEvent 收到消息事件
type MessageReceivedEvent struct {
	*BaseEvent
	Message    *bus.InboundMessage `json:"message"`     // 原始消息
	Preview    string              `json:"preview"`     // 内容预览
	SenderID   string              `json:"sender_id"`   // 发送者 ID
	ChatID     string              `json:"chat_id"`     // 聊天 ID
	Channel    string              `json:"channel"`     // 渠道名称
	SessionKey string              `json:"session_key"` // 会话键
}

// NewMessageReceivedEvent 创建收到消息事件
func NewMessageReceivedEvent(traceID, spanID, parentSpanID string, msg *bus.InboundMessage) *MessageReceivedEvent {
	preview := msg.Content
	if len(preview) > 100 {
		preview = preview[:100] + "..."
	}

	return &MessageReceivedEvent{
		BaseEvent:  NewBaseEvent(traceID, spanID, parentSpanID, EventMessageReceived),
		Message:    msg,
		Preview:    preview,
		SenderID:   msg.SenderID,
		ChatID:     msg.ChatID,
		Channel:    msg.Channel,
		SessionKey: msg.SessionKey(),
	}
}

// MessageSentEvent 发送消息事件
type MessageSentEvent struct {
	*BaseEvent
	Message    *bus.OutboundMessage `json:"message"`     // 输出消息
	Content    string               `json:"content"`     // 消息内容
	Preview    string               `json:"preview"`     // 内容预览
	Channel    string               `json:"channel"`     // 渠道名称
	ChatID     string               `json:"chat_id"`     // 聊天 ID
	SessionKey string               `json:"session_key"` // 会话键
}

// NewMessageSentEvent 创建发送消息事件
func NewMessageSentEvent(traceID, spanID, parentSpanID string, msg *bus.OutboundMessage, sessionKey string) *MessageSentEvent {
	preview := msg.Content
	if len(preview) > 100 {
		preview = preview[:100] + "..."
	}

	return &MessageSentEvent{
		BaseEvent:  NewBaseEvent(traceID, spanID, parentSpanID, EventMessageSent),
		Message:    msg,
		Content:    msg.Content,
		Preview:    preview,
		Channel:    msg.Channel,
		ChatID:     msg.ChatID,
		SessionKey: sessionKey,
	}
}

// PromptSubmittedEvent 提交 Prompt 事件
type PromptSubmittedEvent struct {
	*BaseEvent
	UserInput  string            `json:"user_input"`  // 用户输入
	Messages   []*schema.Message `json:"messages"`    // 完整消息列表
	Count      int               `json:"count"`       // 消息数量
	SessionKey string            `json:"session_key"` // 会话键
}

// NewPromptSubmittedEvent 创建提交 Prompt 事件
func NewPromptSubmittedEvent(traceID, spanID, parentSpanID string, userInput string, messages []*schema.Message, sessionKey string) *PromptSubmittedEvent {
	return &PromptSubmittedEvent{
		BaseEvent:  NewBaseEvent(traceID, spanID, parentSpanID, EventPromptSubmitted),
		UserInput:  userInput,
		Messages:   messages,
		Count:      len(messages),
		SessionKey: sessionKey,
	}
}

// SystemPromptBuiltEvent 生成系统 Prompt 事件
type SystemPromptBuiltEvent struct {
	*BaseEvent
	SystemPrompt string `json:"system_prompt"` // 系统提示词内容
	Length       int    `json:"length"`        // 提示词长度
}

// NewSystemPromptBuiltEvent 创建生成系统 Prompt 事件
func NewSystemPromptBuiltEvent(traceID, spanID, parentSpanID string, systemPrompt string) *SystemPromptBuiltEvent {
	return &SystemPromptBuiltEvent{
		BaseEvent:    NewBaseEvent(traceID, spanID, parentSpanID, EventSystemPromptBuilt),
		SystemPrompt: systemPrompt,
		Length:       len(systemPrompt),
	}
}
