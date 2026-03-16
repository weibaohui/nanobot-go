package feishu

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/weibaohui/nanobot-go/pkg/bus"
	"go.uber.org/zap"
)

// messageHandler 消息处理器
type messageHandler struct {
	channel *Channel
}

// newMessageHandler 创建消息处理器
func newMessageHandler(channel *Channel) *messageHandler {
	return &messageHandler{channel: channel}
}

// onMessageReceive 处理接收到的消息
func (h *messageHandler) onMessageReceive(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	c := h.channel
	c.logger.Info("飞书回调触发")
	if event == nil || event.Event == nil {
		c.logger.Info("飞书事件为空")
		return nil
	}

	ev := event.Event
	message := ev.Message
	sender := ev.Sender

	if message == nil || sender == nil {
		c.logger.Info("飞书消息或发送者为空")
		return nil
	}

	// 消息去重检查
	messageID := *message.MessageId
	if !c.processedMsgIDs.add(messageID) {
		c.logger.Info("飞书消息重复，忽略", zap.String("message_id", messageID))
		return nil
	}

	// 跳过机器人消息
	if sender.SenderType != nil && *sender.SenderType == "bot" {
		return nil
	}

	// 获取发送者 ID
	senderID := "unknown"
	if sender.SenderId != nil && sender.SenderId.OpenId != nil {
		senderID = *sender.SenderId.OpenId
	}

	// 获取聊天 ID
	chatID := ""
	if message.ChatId != nil {
		chatID = *message.ChatId
	}

	// 获取聊天类型
	chatType := "p2p"
	if message.ChatType != nil {
		chatType = *message.ChatType
	}

	// 获取消息类型
	msgType := ""
	if message.MessageType != nil {
		msgType = *message.MessageType
	}

	// 解析消息内容
	content := h.parseMessageContent(message)
	if content == "" {
		return nil
	}

	// 确定回复目标
	replyTo := chatID
	if chatType == "p2p" {
		replyTo = senderID
	}

	// 添加反应表情表示正在处理，并保存 reaction_id
	// 注意：使用 messageID 作为 key，支持同一聊天的多条消息
	go c.addReactionAndSave(messageID, "OnIt")

	// 检查用户白名单
	if len(c.config.AllowFrom) > 0 {
		allowed := false
		for _, u := range c.config.AllowFrom {
			if senderID == u {
				allowed = true
				break
			}
		}
		if !allowed {
			c.logger.Debug("飞书消息发送者不在白名单中", zap.String("sender", senderID))
			return nil
		}
	}

	c.logger.Info("收到飞书消息",
		zap.String("sender", senderID),
		zap.String("chat_id", chatID),
		zap.String("reply_to", replyTo),
		zap.String("content", content),
	)

	// 发布消息到总线
	// 在 Metadata 中记录 app_id，用于后续消息路由
	// 记录 channel_id 用于加载渠道绑定的 Agent 配置
	c.bus.PublishInbound(&bus.InboundMessage{
		Channel:   "feishu",
		ChatID:    replyTo,
		SenderID:  senderID,
		Content:   content,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"message_id": messageID,
			"chat_type":  chatType,
			"msg_type":   msgType,
			"chat_id":    chatID,
			"app_id":     c.config.AppID,     // 记录 app_id 用于消息路由
			"channel_id": c.config.ChannelID, // 记录 channel_id 用于加载 Agent 配置
		},
	})

	return nil
}

// onReactionCreated 处理消息表情反应事件（仅记录日志，不处理业务逻辑）
func (h *messageHandler) onReactionCreated(ctx context.Context, event *larkim.P2MessageReactionCreatedV1) error {
	c := h.channel
	// 忽略表情反应事件，仅记录调试日志
	if event != nil && event.Event != nil {
		ev := event.Event
		emojiType := ""
		if ev.ReactionType != nil && ev.ReactionType.EmojiType != nil {
			emojiType = *ev.ReactionType.EmojiType
		}
		c.logger.Debug("收到飞书表情反应创建事件",
			zap.String("message_id", *ev.MessageId),
			zap.String("emoji", emojiType),
		)
	}
	return nil
}

// onReactionDeleted 处理消息表情反应删除事件（仅记录日志，不处理业务逻辑）
func (h *messageHandler) onReactionDeleted(ctx context.Context, event *larkim.P2MessageReactionDeletedV1) error {
	c := h.channel
	// 忽略表情反应删除事件，仅记录调试日志
	if event != nil && event.Event != nil {
		ev := event.Event
		emojiType := ""
		if ev.ReactionType != nil && ev.ReactionType.EmojiType != nil {
			emojiType = *ev.ReactionType.EmojiType
		}
		c.logger.Debug("收到飞书表情反应删除事件",
			zap.String("message_id", *ev.MessageId),
			zap.String("emoji", emojiType),
		)
	}
	return nil
}

// parseMessageContent 解析消息内容
func (h *messageHandler) parseMessageContent(message *larkim.EventMessage) string {
	if message == nil || message.MessageType == nil {
		return ""
	}

	msgType := *message.MessageType

	switch msgType {
	case "text":
		if message.Content == nil {
			return ""
		}
		// 解析 JSON 格式的文本内容
		var contentMap map[string]interface{}
		if err := json.Unmarshal([]byte(*message.Content), &contentMap); err == nil {
			if text, ok := contentMap["text"].(string); ok {
				return strings.TrimSpace(text)
			}
		}
		return strings.TrimSpace(*message.Content)
	case "image":
		return "[图片]"
	case "audio":
		return "[语音]"
	case "file":
		return "[文件]"
	case "sticker":
		return "[表情]"
	default:
		return "[" + msgType + "]"
	}
}
