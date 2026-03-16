package bus

import (
	"testing"
	"time"
)

// TestInboundMessage_SessionKey 测试入站消息的 SessionKey 方法
func TestInboundMessage_SessionKey(t *testing.T) {
	tests := []struct {
		name     string
		msg      *InboundMessage
		expected string
	}{
		{
			name: "基本组合",
			msg: &InboundMessage{
				Channel: "telegram",
				ChatID:  "chat123",
			},
			expected: "telegram:chat123",
		},
		{
			name: "WebSocket渠道",
			msg: &InboundMessage{
				Channel: "websocket",
				ChatID:  "user456",
			},
			expected: "websocket:user456",
		},
		{
			name: "空值组合",
			msg: &InboundMessage{
				Channel: "",
				ChatID:  "",
			},
			expected: ":",
		},
		{
			name: "包含冒号的ChatID",
			msg: &InboundMessage{
				Channel: "matrix",
				ChatID:  "room:server",
			},
			expected: "matrix:room:server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.msg.SessionKey()
			if result != tt.expected {
				t.Errorf("SessionKey() = %q, 期望 %q", result, tt.expected)
			}
		})
	}
}

// TestNewInboundMessage 测试创建入站消息
func TestNewInboundMessage(t *testing.T) {
	msg := NewInboundMessage("telegram", "user123", "chat456", "Hello World")

	if msg.Channel != "telegram" {
		t.Errorf("Channel = %q, 期望 telegram", msg.Channel)
	}

	if msg.SenderID != "user123" {
		t.Errorf("SenderID = %q, 期望 user123", msg.SenderID)
	}

	if msg.ChatID != "chat456" {
		t.Errorf("ChatID = %q, 期望 chat456", msg.ChatID)
	}

	if msg.Content != "Hello World" {
		t.Errorf("Content = %q, 期望 Hello World", msg.Content)
	}

	if msg.Timestamp.IsZero() {
		t.Error("Timestamp 不应该为零值")
	}

	if msg.Media == nil {
		t.Error("Media 不应该为 nil")
	}

	if msg.Metadata == nil {
		t.Error("Metadata 不应该为 nil")
	}
}

// TestNewOutboundMessage 测试创建出站消息
func TestNewOutboundMessage(t *testing.T) {
	msg := NewOutboundMessage("telegram", "chat456", "Reply message")

	if msg.Channel != "telegram" {
		t.Errorf("Channel = %q, 期望 telegram", msg.Channel)
	}

	if msg.ChatID != "chat456" {
		t.Errorf("ChatID = %q, 期望 chat456", msg.ChatID)
	}

	if msg.Content != "Reply message" {
		t.Errorf("Content = %q, 期望 Reply message", msg.Content)
	}

	if msg.Media == nil {
		t.Error("Media 不应该为 nil")
	}

	if msg.Metadata == nil {
		t.Error("Metadata 不应该为 nil")
	}
}

// TestNewStreamChunk 测试创建流式片段
func TestNewStreamChunk(t *testing.T) {
	chunk := NewStreamChunk("websocket", "chat123", "delta text", "accumulated text", false)

	if chunk.Channel != "websocket" {
		t.Errorf("Channel = %q, 期望 websocket", chunk.Channel)
	}

	if chunk.ChatID != "chat123" {
		t.Errorf("ChatID = %q, 期望 chat123", chunk.ChatID)
	}

	if chunk.Delta != "delta text" {
		t.Errorf("Delta = %q, 期望 delta text", chunk.Delta)
	}

	if chunk.Content != "accumulated text" {
		t.Errorf("Content = %q, 期望 accumulated text", chunk.Content)
	}

	if chunk.Done != false {
		t.Errorf("Done = %v, 期望 false", chunk.Done)
	}
}

// TestNewStreamChunk_Done 测试流式片段完成状态
func TestNewStreamChunk_Done(t *testing.T) {
	chunk := NewStreamChunk("websocket", "chat123", "", "final text", true)

	if !chunk.Done {
		t.Error("Done 应该为 true")
	}
}

// TestInboundMessage_Fields 测试入站消息的所有字段
func TestInboundMessage_Fields(t *testing.T) {
	now := time.Now()
	msg := &InboundMessage{
		Channel:   "dingtalk",
		SenderID:  "sender001",
		ChatID:    "chat001",
		Content:   "测试消息",
		Timestamp: now,
		Media:     []string{"http://example.com/image.jpg"},
		Metadata: map[string]any{
			"key": "value",
		},
	}

	if msg.Channel != "dingtalk" {
		t.Errorf("Channel = %q, 期望 dingtalk", msg.Channel)
	}

	if len(msg.Media) != 1 {
		t.Errorf("Media 长度 = %d, 期望 1", len(msg.Media))
	}

	if msg.Metadata["key"] != "value" {
		t.Errorf("Metadata[key] = %v, 期望 value", msg.Metadata["key"])
	}
}

// TestOutboundMessage_Fields 测试出站消息的所有字段
func TestOutboundMessage_Fields(t *testing.T) {
	msg := &OutboundMessage{
		Channel: "matrix",
		ChatID:  "room001",
		Content: "回复内容",
		ReplyTo: "msg001",
		Media:   []string{"http://example.com/file.pdf"},
		Metadata: map[string]any{
			"priority": "high",
		},
	}

	if msg.ReplyTo != "msg001" {
		t.Errorf("ReplyTo = %q, 期望 msg001", msg.ReplyTo)
	}

	if len(msg.Media) != 1 {
		t.Errorf("Media 长度 = %d, 期望 1", len(msg.Media))
	}
}

// TestInterruptRequest 测试中断请求结构
func TestInterruptRequest(t *testing.T) {
	req := &InterruptRequest{
		Channel:      "websocket",
		ChatID:       "chat001",
		CheckpointID: "checkpoint001",
		InterruptID:  "interrupt001",
		Question:     "请选择操作",
		Options:      []string{"选项A", "选项B"},
	}

	if req.Channel != "websocket" {
		t.Errorf("Channel = %q, 期望 websocket", req.Channel)
	}

	if len(req.Options) != 2 {
		t.Errorf("Options 长度 = %d, 期望 2", len(req.Options))
	}
}

// TestInterruptResponse 测试中断响应结构
func TestInterruptResponse(t *testing.T) {
	resp := &InterruptResponse{
		Channel:      "websocket",
		ChatID:       "chat001",
		CheckpointID: "checkpoint001",
		InterruptID:  "interrupt001",
		Answer:       "选项A",
	}

	if resp.Answer != "选项A" {
		t.Errorf("Answer = %q, 期望 选项A", resp.Answer)
	}
}
