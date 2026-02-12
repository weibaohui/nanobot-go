package bus

import "time"

// InboundMessage 表示从聊天渠道接收的消息
type InboundMessage struct {
	Channel   string            `json:"channel"`   // telegram, discord, slack, whatsapp
	SenderID  string            `json:"sender_id"` // 用户标识符
	ChatID    string            `json:"chat_id"`   // 聊天/频道标识符
	Content   string            `json:"content"`   // 消息文本
	Timestamp time.Time         `json:"timestamp"` // 时间戳
	Media     []string          `json:"media"`     // 媒体 URL 列表
	Metadata  map[string]any    `json:"metadata"`  // 渠道特定数据
}

// SessionKey 返回会话的唯一标识符
func (m *InboundMessage) SessionKey() string {
	return m.Channel + ":" + m.ChatID
}

// OutboundMessage 表示要发送到聊天渠道的消息
type OutboundMessage struct {
	Channel  string         `json:"channel"`
	ChatID   string         `json:"chat_id"`
	Content  string         `json:"content"`
	ReplyTo  string         `json:"reply_to,omitempty"`
	Media    []string       `json:"media,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// NewInboundMessage 创建一个新的入站消息
func NewInboundMessage(channel, senderID, chatID, content string) *InboundMessage {
	return &InboundMessage{
		Channel:   channel,
		SenderID:  senderID,
		ChatID:    chatID,
		Content:   content,
		Timestamp: time.Now(),
		Media:     []string{},
		Metadata:  make(map[string]any),
	}
}

// NewOutboundMessage 创建一个新的出站消息
func NewOutboundMessage(channel, chatID, content string) *OutboundMessage {
	return &OutboundMessage{
		Channel:  channel,
		ChatID:   chatID,
		Content:  content,
		Media:    []string{},
		Metadata: make(map[string]any),
	}
}
