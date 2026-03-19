package websocket

import (
	"encoding/json"
	"time"

	"github.com/weibaohui/nanobot-go/pkg/bus"
)

// MessageType 消息类型
type MessageType string

const (
	MessageTypePing           MessageType = "ping"            // 心跳请求
	MessageTypePong           MessageType = "pong"            // 心跳响应
	MessageTypeMessage        MessageType = "message"         // 用户消息
	MessageTypeChunk          MessageType = "chunk"           // AI 流式回复片段
	MessageTypeError          MessageType = "error"           // 错误消息
	MessageTypeSystem         MessageType = "system"          // 系统消息
	MessageTypeTaskCreated    MessageType = "task_created"    // 任务创建
	MessageTypeTaskUpdated    MessageType = "task_updated"    // 任务更新
	MessageTypeTaskCompleted  MessageType = "task_completed"  // 任务完成
	MessageTypeTaskLog        MessageType = "task_log"        // 任务日志
)

// Message 通用消息结构
type Message struct {
	Type      MessageType     `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp int64           `json:"timestamp"`
}

// MessagePayload 入站消息负载（用户发送的消息）
type MessagePayload struct {
	Content   string `json:"content"`
	UserCode  string `json:"user_code"`
	SessionID string `json:"session_id,omitempty"`
}

// ChunkPayload 流式响应负载（AI 回复片段）
type ChunkPayload struct {
	Content   string `json:"content"`
	SessionID string `json:"session_id"`
	IsEnd     bool   `json:"is_end"`
}

// ErrorPayload 错误消息负载
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// SystemPayload 系统消息负载
type SystemPayload struct {
	Type      string `json:"type,omitempty"`      // 系统消息类型，如 "session_created"
	SessionID string `json:"session_id,omitempty"`
	Message   string `json:"message,omitempty"`
}

// TaskCreatedPayload 任务创建事件负载
type TaskCreatedPayload struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Work      string `json:"work"`
	Channel   string `json:"channel,omitempty"`
	ChatID    string `json:"chat_id,omitempty"`
	CreatedAt string `json:"created_at"`
	CreatedBy string `json:"created_by,omitempty"`
}

// TaskUpdatedPayload 任务更新事件负载
type TaskUpdatedPayload struct {
	ID        string   `json:"id"`
	Status    string   `json:"status"`
	Result    string   `json:"result,omitempty"`
	Logs      []string `json:"logs,omitempty"`
	UpdatedAt string   `json:"updated_at"`
}

// TaskCompletedPayload 任务完成事件负载
type TaskCompletedPayload struct {
	ID              string   `json:"id"`
	Status          string   `json:"status"`
	Result          string   `json:"result,omitempty"`
	Logs            []string `json:"logs,omitempty"`
	CompletedAt     string   `json:"completed_at"`
	DurationSeconds int      `json:"duration_seconds"`
}

// TaskLogPayload 任务日志事件负载
type TaskLogPayload struct {
	ID        string `json:"id"`
	Log       string `json:"log"`
	Timestamp string `json:"timestamp"`
}

// NewPingMessage 创建心跳请求消息
func NewPingMessage() *Message {
	return &Message{
		Type:      MessageTypePing,
		Timestamp: time.Now().UnixMilli(),
	}
}

// NewPongMessage 创建心跳响应消息
func NewPongMessage() *Message {
	return &Message{
		Type:      MessageTypePong,
		Timestamp: time.Now().UnixMilli(),
	}
}

// NewErrorMessage 创建错误消息
func NewErrorMessage(code, message string) *Message {
	payload, _ := json.Marshal(ErrorPayload{
		Code:    code,
		Message: message,
	})
	return &Message{
		Type:      MessageTypeError,
		Payload:   payload,
		Timestamp: time.Now().UnixMilli(),
	}
}

// NewChunkMessage 创建流式响应消息
func NewChunkMessage(content, sessionID string, isEnd bool) *Message {
	payload, _ := json.Marshal(ChunkPayload{
		Content:   content,
		SessionID: sessionID,
		IsEnd:     isEnd,
	})
	return &Message{
		Type:      MessageTypeChunk,
		Payload:   payload,
		Timestamp: time.Now().UnixMilli(),
	}
}

// NewSystemMessage 创建系统消息
func NewSystemMessage(msgType, sessionID, message string) *Message {
	payload, _ := json.Marshal(SystemPayload{
		Type:      msgType,
		SessionID: sessionID,
		Message:   message,
	})
	return &Message{
		Type:      MessageTypeSystem,
		Payload:   payload,
		Timestamp: time.Now().UnixMilli(),
	}
}

// ToInboundMessage 将 WebSocket 消息转换为 MessageBus 入站消息
func (m *Message) ToInboundMessage(channelCode string) (*bus.InboundMessage, error) {
	if m.Type != MessageTypeMessage {
		return nil, nil
	}

	var payload MessagePayload
	if err := json.Unmarshal(m.Payload, &payload); err != nil {
		return nil, err
	}

	return &bus.InboundMessage{
		Channel:  channelCode,
		SenderID: payload.UserCode,
		ChatID:   payload.SessionID,
		Content:  payload.Content,
		Metadata: map[string]any{
			"websocket":  true,
			"timestamp":  m.Timestamp,
			"session_id": payload.SessionID,
		},
	}, nil
}

// FromStreamChunk 将 MessageBus 流式消息转换为 WebSocket 消息
func FromStreamChunk(chunk *bus.StreamChunk) *Message {
	return NewChunkMessage(chunk.Content, chunk.ChatID, chunk.Done)
}

// FromOutboundMessage 将 MessageBus 出站消息转换为 WebSocket 消息
func FromOutboundMessage(msg *bus.OutboundMessage) *Message {
	return NewChunkMessage(msg.Content, msg.ChatID, true)
}

// NewTaskCreatedMessage 创建任务创建消息
func NewTaskCreatedMessage(payload *TaskCreatedPayload) *Message {
	payloadBytes, _ := json.Marshal(payload)
	return &Message{
		Type:      MessageTypeTaskCreated,
		Payload:   payloadBytes,
		Timestamp: time.Now().UnixMilli(),
	}
}

// NewTaskUpdatedMessage 创建任务更新消息
func NewTaskUpdatedMessage(payload *TaskUpdatedPayload) *Message {
	payloadBytes, _ := json.Marshal(payload)
	return &Message{
		Type:      MessageTypeTaskUpdated,
		Payload:   payloadBytes,
		Timestamp: time.Now().UnixMilli(),
	}
}

// NewTaskCompletedMessage 创建任务完成消息
func NewTaskCompletedMessage(payload *TaskCompletedPayload) *Message {
	payloadBytes, _ := json.Marshal(payload)
	return &Message{
		Type:      MessageTypeTaskCompleted,
		Payload:   payloadBytes,
		Timestamp: time.Now().UnixMilli(),
	}
}

// NewTaskLogMessage 创建任务日志消息
func NewTaskLogMessage(payload *TaskLogPayload) *Message {
	payloadBytes, _ := json.Marshal(payload)
	return &Message{
		Type:      MessageTypeTaskLog,
		Payload:   payloadBytes,
		Timestamp: time.Now().UnixMilli(),
	}
}

// mustMarshal 辅助函数：将对象序列化为 JSON，忽略错误
func mustMarshal(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
