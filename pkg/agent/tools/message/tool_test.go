package message

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/bus"
)

// mockSendCallback 模拟发送回调
func mockSendCallback(msg any) error {
	return nil
}

// mockSendCallbackError 模拟发送回调错误
func mockSendCallbackError(msg any) error {
	return errors.New("发送失败")
}

// TestTool_Name 测试工具名称
func TestTool_Name(t *testing.T) {
	tool := &Tool{}
	if tool.Name() != "message" {
		t.Errorf("Name() = %q, 期望 message", tool.Name())
	}
}

// TestTool_Info 测试工具信息
func TestTool_Info(t *testing.T) {
	tool := &Tool{}
	ctx := context.Background()

	info, err := tool.Info(ctx)
	if err != nil {
		t.Errorf("Info() 返回错误: %v", err)
	}

	if info.Name != "message" {
		t.Errorf("Info.Name = %q, 期望 message", info.Name)
	}
}

// TestTool_Run 测试执行工具
func TestTool_Run(t *testing.T) {
	t.Run("正常发送", func(t *testing.T) {
		tool := &Tool{
			SendCallback:   mockSendCallback,
			DefaultChannel: "websocket",
			DefaultChatID:  "chat-001",
		}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"content": "测试消息"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result != "消息已发送到 websocket:chat-001" {
			t.Errorf("Run() = %q, 期望 消息已发送到 websocket:chat-001", result)
		}
	})

	t.Run("指定渠道和聊天ID", func(t *testing.T) {
		tool := &Tool{
			SendCallback:   mockSendCallback,
			DefaultChannel: "websocket",
			DefaultChatID:  "chat-001",
		}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"content": "测试消息", "channel": "dingtalk", "chat_id": "chat-002"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result != "消息已发送到 dingtalk:chat-002" {
			t.Errorf("Run() = %q, 期望 消息已发送到 dingtalk:chat-002", result)
		}
	})

	t.Run("无默认渠道", func(t *testing.T) {
		tool := &Tool{
			SendCallback: mockSendCallback,
		}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"content": "测试消息"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result != "错误: 没有目标渠道或聊天ID" {
			t.Errorf("Run() = %q, 期望 错误: 没有目标渠道或聊天ID", result)
		}
	})

	t.Run("无发送回调", func(t *testing.T) {
		tool := &Tool{
			DefaultChannel: "websocket",
			DefaultChatID:  "chat-001",
		}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"content": "测试消息"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result != "错误: 消息发送未配置" {
			t.Errorf("Run() = %q, 期望 错误: 消息发送未配置", result)
		}
	})

	t.Run("发送失败", func(t *testing.T) {
		tool := &Tool{
			SendCallback:   mockSendCallbackError,
			DefaultChannel: "websocket",
			DefaultChatID:  "chat-001",
		}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"content": "测试消息"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result != "错误: 发送消息失败: 发送失败" {
			t.Errorf("Run() = %q, 期望 错误: 发送消息失败: 发送失败", result)
		}
	})

	t.Run("channel 为 user 时使用默认渠道", func(t *testing.T) {
		tool := &Tool{
			SendCallback:   mockSendCallback,
			DefaultChannel: "websocket",
			DefaultChatID:  "chat-001",
		}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"content": "测试消息", "channel": "user"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result != "消息已发送到 websocket:chat-001" {
			t.Errorf("Run() = %q, 期望 消息已发送到 websocket:chat-001", result)
		}
	})
}

// TestTool_SetContext 测试设置上下文
func TestTool_SetContext(t *testing.T) {
	tool := &Tool{}
	tool.SetContext("websocket", "chat-001")

	if tool.DefaultChannel != "websocket" {
		t.Errorf("DefaultChannel = %q, 期望 websocket", tool.DefaultChannel)
	}

	if tool.DefaultChatID != "chat-001" {
		t.Errorf("DefaultChatID = %q, 期望 chat-001", tool.DefaultChatID)
	}
}

// TestTool_InvokableRun 测试 InvokableRun 方法
func TestTool_InvokableRun(t *testing.T) {
	tool := &Tool{
		SendCallback:   mockSendCallback,
		DefaultChannel: "websocket",
		DefaultChatID:  "chat-001",
	}
	ctx := context.Background()

	result, err := tool.InvokableRun(ctx, `{"content": "测试消息"}`)
	if err != nil {
		t.Errorf("InvokableRun() 返回错误: %v", err)
	}

	if result == "" {
		t.Error("InvokableRun() 不应该返回空结果")
	}
}

// TestSender_Interface 测试 Sender 接口
func TestSender_Interface(t *testing.T) {
	var _ Sender = (*mockSender)(nil)
}

// mockSender 模拟发送器
type mockSender struct{}

func (m *mockSender) Send(msg any) error {
	return nil
}

// TestBusOutboundMessage 测试总线消息
func TestBusOutboundMessage(t *testing.T) {
	msg := bus.NewOutboundMessage("websocket", "chat-001", "测试消息")

	if msg.Channel != "websocket" {
		t.Errorf("Channel = %q, 期望 websocket", msg.Channel)
	}

	if msg.ChatID != "chat-001" {
		t.Errorf("ChatID = %q, 期望 chat-001", msg.ChatID)
	}

	if msg.Content != "测试消息" {
		t.Errorf("Content = %q, 期望 测试消息", msg.Content)
	}
}

// TestTool_InfoParams 测试工具参数信息
func TestTool_InfoParams(t *testing.T) {
	tool := &Tool{}
	ctx := context.Background()

	info, err := tool.Info(ctx)
	if err != nil {
		t.Errorf("Info() 返回错误: %v", err)
	}

	if info.ParamsOneOf == nil {
		t.Error("ParamsOneOf 不应该为 nil")
	}
}

// TestTool_Interface 测试工具接口实现
func TestTool_Interface(t *testing.T) {
	tool := &Tool{}

	var _ interface {
		Name() string
		Info(ctx context.Context) (*schema.ToolInfo, error)
	} = tool
}
