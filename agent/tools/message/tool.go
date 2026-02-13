package message

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/agent/tools/common"
	"github.com/weibaohui/nanobot-go/bus"
)

type Sender interface {
	Send(msg any) error
}

type Tool struct {
	SendCallback   func(msg any) error
	DefaultChannel string
	DefaultChatID  string
}

// Name 返回工具名称
func (t *Tool) Name() string {
	return "message"
}

// Info 返回工具信息
func (t *Tool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.Name(),
		Desc: "发送消息给用户",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"content": {
				Type:     schema.DataType("string"),
				Desc:     "消息内容",
				Required: true,
			},
			"channel": {
				Type: schema.DataType("string"),
				Desc: "目标渠道",
			},
			"chat_id": {
				Type: schema.DataType("string"),
				Desc: "目标聊天ID",
			},
		}),
	}, nil
}

// Run 执行工具逻辑
func (t *Tool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		Content string `json:"content"`
		Channel string `json:"channel"`
		ChatID  string `json:"chat_id"`
	}
	if err := common.DecodeArgs(argumentsInJSON, &args); err != nil {
		return "", err
	}
	channel := args.Channel
	chatID := args.ChatID
	if channel == "" {
		channel = t.DefaultChannel
	}
	if channel == "user" {
		channel = t.DefaultChannel
	}
	if chatID == "" {
		chatID = t.DefaultChatID
	}
	if channel == "" || chatID == "" {
		return "错误: 没有目标渠道或聊天ID", nil
	}
	if t.SendCallback == nil {
		return "错误: 消息发送未配置", nil
	}
	msg := bus.NewOutboundMessage(channel, chatID, args.Content)
	if err := t.SendCallback(msg); err != nil {
		return fmt.Sprintf("错误: 发送消息失败: %s", err), nil
	}
	return fmt.Sprintf("消息已发送到 %s:%s", channel, chatID), nil
}

// InvokableRun 可直接调用的执行入口
func (t *Tool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.Run(ctx, argumentsInJSON, opts...)
}

// SetContext 设置上下文
func (t *Tool) SetContext(channel, chatID string) {
	t.DefaultChannel = channel
	t.DefaultChatID = chatID
}
