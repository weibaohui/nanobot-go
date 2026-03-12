package agent

import (
	"context"
	"fmt"

	"github.com/weibaohui/nanobot-go/agent/hooks/trace"
	"github.com/weibaohui/nanobot-go/bus"
	"go.uber.org/zap"
)

// Run 运行代理循环
func (l *Loop) Run(ctx context.Context) error {
	l.running = true
	l.logger.Info("消息监听循环处理功能已启动")

	for l.running {
		// 等待消息
		msg, err := l.bus.ConsumeInbound(ctx)
		if err != nil {
			if err == context.DeadlineExceeded {
				continue
			}
			if err == context.Canceled {
				return nil
			}
			return err
		}

		// 处理消息
		if err := l.processMessage(ctx, msg); err != nil {
			l.logger.Error("处理消息失败", zap.Error(err))
			outMsg := bus.NewOutboundMessage(msg.Channel, msg.ChatID, fmt.Sprintf("抱歉，我遇到了错误: %s", err))
			// 传递原始消息的 message_id
			if msg.Metadata != nil {
				if msgID, ok := msg.Metadata["message_id"].(string); ok {
					outMsg.Metadata["reply_to_message_id"] = msgID
				}
			}
			l.bus.PublishOutbound(outMsg)
		}
	}

	return nil
}

// processMessage 处理单条消息
func (l *Loop) processMessage(ctx context.Context, msg *bus.InboundMessage) error {
	preview := msg.Content
	if len(preview) > 80 {
		preview = preview[:80] + "..."
	}
	l.logger.Info("处理消息",
		zap.String("渠道", msg.Channel),
		zap.String("发送者", msg.SenderID),
		zap.String("内容", preview),
	)

	// 为每条消息创建根 span，建立完整的调用链
	ctx = trace.WithTraceID(ctx, trace.NewTraceID())
	ctx = trace.WithSpanID(ctx, trace.NewSpanID())
	// 根 span 没有 parentSpanID

	// 注入会话信息到 context，用于事件分发时获取
	sessionKey := msg.SessionKey()
	ctx = trace.WithSessionInfo(ctx, sessionKey, msg.Channel)

	// 触发收到消息事件
	if l.hookManager != nil {
		l.hookManager.OnMessageReceived(ctx, msg)
	}

	// 使用 Master Agent 处理消息（包括中断恢复和正常处理）
	if l.masterAgent == nil {
		return fmt.Errorf("Master Agent 未初始化，无法处理消息")
	}
	l.logger.Info("使用 Master Agent 处理消息")
	response, err := l.masterAgent.Process(ctx, msg)

	if err != nil {
		// 检查是否是中断
		if IsInterruptError(err) {
			return nil
		}
		// 非中断错误：如果 response 包含错误信息（由 interruptible 构造），直接发送
		// 否则构造默认错误消息
		outMsg := bus.NewOutboundMessage(msg.Channel, msg.ChatID, response)
		if response != "" {
			l.logger.Error("Master Agent 处理失败", zap.Error(err), zap.String("response", response))
		} else {
			l.logger.Error("Master Agent 处理失败", zap.Error(err))
			outMsg.Content = fmt.Sprintf("抱歉，处理消息时遇到错误: %v", err)
		}
		// 传递原始消息的 metadata 用于渠道特定功能
		if msg.Metadata != nil {
			// 复制 message_id 用于删除反应表情等功能
			if msgID, ok := msg.Metadata["message_id"].(string); ok {
				outMsg.Metadata["reply_to_message_id"] = msgID
			}
			// 复制 app_id 用于飞书多渠道路由
			if appID, ok := msg.Metadata["app_id"].(string); ok {
				outMsg.Metadata["app_id"] = appID
			}
		}
		l.bus.PublishOutbound(outMsg)
		return nil
	}

	// 发布响应
	outMsg := bus.NewOutboundMessage(msg.Channel, msg.ChatID, response)
	// 传递原始消息的 metadata 用于渠道特定功能
	if msg.Metadata != nil {
		// 复制 message_id 用于删除反应表情等功能
		if msgID, ok := msg.Metadata["message_id"].(string); ok {
			outMsg.Metadata["reply_to_message_id"] = msgID
		}
		// 复制 app_id 用于飞书多渠道路由
		if appID, ok := msg.Metadata["app_id"].(string); ok {
			outMsg.Metadata["app_id"] = appID
		}
	}
	l.bus.PublishOutbound(outMsg)
	return nil
}
