package channels

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/weibaohui/nanobot-go/bus"
	"go.uber.org/zap"
)

// WhatsAppChannel WhatsApp 渠道
// 通过 Node.js 桥接连接，使用 WebSocket 通信
type WhatsAppChannel struct {
	*BaseChannel
	config    *WhatsAppConfig
	logger    *zap.Logger
	running   bool
	connected bool
}

// WhatsAppConfig WhatsApp 配置
type WhatsAppConfig struct {
	BridgeURL string
	AllowFrom []string
}

// NewWhatsAppChannel 创建 WhatsApp 渠道
func NewWhatsAppChannel(config *WhatsAppConfig, messageBus *bus.MessageBus, logger *zap.Logger) *WhatsAppChannel {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &WhatsAppChannel{
		BaseChannel: NewBaseChannel("whatsapp", messageBus),
		config:      config,
		logger:      logger,
	}
}

// Start 启动 WhatsApp 渠道
func (c *WhatsAppChannel) Start(ctx context.Context) error {
	if c.config.BridgeURL == "" {
		c.logger.Error("WhatsApp bridge URL 未配置")
		return nil
	}

	c.running = true
	c.logger.Info("WhatsApp 渠道已启动", zap.String("bridge_url", c.config.BridgeURL))

	// TODO: 实现 WebSocket 连接到 Node.js 桥接
	// 需要导入: "nhooyr.io/websocket"

	return nil
}

// Stop 停止 WhatsApp 渠道
func (c *WhatsAppChannel) Stop() {
	c.running = false
	c.connected = false
	c.logger.Info("WhatsApp 渠道已停止")
}

// Send 发送消息
func (c *WhatsAppChannel) Send(msg *bus.OutboundMessage) error {
	if !c.connected {
		c.logger.Warn("WhatsApp 桥接未连接")
		return nil
	}

	// TODO: 实现实际的消息发送
	payload := map[string]interface{}{
		"type": "send",
		"to":   msg.ChatID,
		"text": msg.Content,
	}
	data, _ := json.Marshal(payload)
	c.logger.Debug("发送 WhatsApp 消息", zap.String("payload", string(data)))

	return nil
}

// handleBridgeMessage 处理来自桥接的消息
func (c *WhatsAppChannel) handleBridgeMessage(raw string) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		c.logger.Warn("无效的 JSON 消息", zap.String("raw", raw[:min(100, len(raw))]))
		return
	}

	msgType, _ := data["type"].(string)
	switch msgType {
	case "message":
		c.handleIncomingMessage(data)
	case "status":
		status, _ := data["status"].(string)
		c.logger.Info("WhatsApp 状态", zap.String("status", status))
		c.connected = status == "connected"
	case "qr":
		c.logger.Info("请在桥接终端扫描二维码连接 WhatsApp")
	case "error":
		errMsg, _ := data["error"].(string)
		c.logger.Error("WhatsApp 桥接错误", zap.String("error", errMsg))
	}
}

// handleIncomingMessage 处理接收到的消息
func (c *WhatsAppChannel) handleIncomingMessage(data map[string]interface{}) {
	sender, _ := data["sender"].(string)
	content, _ := data["content"].(string)

	// 提取发送者 ID
	senderID := sender
	if idx := strings.Index(sender, "@"); idx > 0 {
		senderID = sender[:idx]
	}

	c.logger.Debug("收到 WhatsApp 消息",
		zap.String("sender", senderID),
		zap.String("content", content[:min(50, len(content))]),
	)

	// TODO: 发布到消息总线
}
