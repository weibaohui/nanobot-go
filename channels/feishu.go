package channels

import (
	"context"
	"encoding/json"

	"github.com/weibaohui/nanobot-go/bus"
	"go.uber.org/zap"
)

// FeishuChannel 飞书渠道
// 使用 WebSocket 长连接
type FeishuChannel struct {
	*BaseChannel
	config  *FeishuConfig
	logger  *zap.Logger
	running bool
}

// FeishuConfig 飞书配置
type FeishuConfig struct {
	AppID             string
	AppSecret         string
	EncryptKey        string
	VerificationToken string
	AllowFrom         []string
}

// NewFeishuChannel 创建飞书渠道
func NewFeishuChannel(config *FeishuConfig, messageBus *bus.MessageBus, logger *zap.Logger) *FeishuChannel {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &FeishuChannel{
		BaseChannel: NewBaseChannel("feishu", messageBus),
		config:      config,
		logger:      logger,
	}
}

// Start 启动飞书渠道
func (c *FeishuChannel) Start(ctx context.Context) error {
	if c.config.AppID == "" || c.config.AppSecret == "" {
		c.logger.Error("飞书 app_id 和 app_secret 未配置")
		return nil
	}

	c.running = true
	c.logger.Info("飞书渠道已启动")

	// TODO: 实现 lark-oapi SDK 的 WebSocket 长连接

	return nil
}

// Stop 停止飞书渠道
func (c *FeishuChannel) Stop() {
	c.running = false
	c.logger.Info("飞书渠道已停止")
}

// Send 发送消息
func (c *FeishuChannel) Send(msg *bus.OutboundMessage) error {
	// 确定接收者类型
	receiveIDType := "open_id"
	if len(msg.ChatID) > 3 && msg.ChatID[:3] == "oc_" {
		receiveIDType = "chat_id"
	}

	// 构建消息卡片
	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"elements": []map[string]interface{}{
			{
				"tag":     "markdown",
				"content": msg.Content,
			},
		},
	}
	content, _ := json.Marshal(card)

	c.logger.Debug("发送飞书消息",
		zap.String("receive_id", msg.ChatID),
		zap.String("receive_id_type", receiveIDType),
		zap.String("content", string(content[:min(100, len(content))])),
	)

	// TODO: 实现实际的消息发送

	return nil
}

// handleMessage 处理接收到的消息
func (c *FeishuChannel) handleMessage(data map[string]interface{}) {
	event, _ := data["event"].(map[string]interface{})
	message, _ := event["message"].(map[string]interface{})
	sender, _ := event["sender"].(map[string]interface{})

	// 跳过机器人消息
	senderType, _ := sender["sender_type"].(string)
	if senderType == "bot" {
		return
	}

	senderInfo, _ := sender["sender_id"].(map[string]interface{})
	senderID, _ := senderInfo["open_id"].(string)
	chatID, _ := message["chat_id"].(string)
	chatType, _ := message["chat_type"].(string)
	msgType, _ := message["message_type"].(string)

	// 解析消息内容
	var content string
	if msgType == "text" {
		var contentData struct {
			Text string `json:"text"`
		}
		if contentBytes, ok := message["content"].(string); ok {
			json.Unmarshal([]byte(contentBytes), &contentData)
			content = contentData.Text
		}
	} else {
		content = "[" + msgType + "]"
	}

	// 群聊回复到群，私聊回复到发送者
	replyTo := chatID
	if chatType == "p2p" {
		replyTo = senderID
	}

	c.logger.Debug("收到飞书消息",
		zap.String("sender", senderID),
		zap.String("chat_id", replyTo),
		zap.String("content", content[:min(50, len(content))]),
	)

	// TODO: 发布到消息总线
}
