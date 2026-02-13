package channels

import (
	"context"

	"github.com/weibaohui/nanobot-go/bus"
	"go.uber.org/zap"
)

// QQChannel QQ 渠道
// 使用 botpy SDK
type QQChannel struct {
	*BaseChannel
	config       *QQConfig
	logger       *zap.Logger
	running      bool
	processedIDs map[string]bool
}

// QQConfig QQ 配置
type QQConfig struct {
	AppID    string
	Secret   string
	AllowFrom []string
}

// NewQQChannel 创建 QQ 渠道
func NewQQChannel(config *QQConfig, messageBus *bus.MessageBus, logger *zap.Logger) *QQChannel {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &QQChannel{
		BaseChannel: NewBaseChannel("qq", messageBus),
		config:      config,
		logger:      logger,
		processedIDs: make(map[string]bool),
	}
}

// Start 启动 QQ 渠道
func (c *QQChannel) Start(ctx context.Context) error {
	if c.config.AppID == "" || c.config.Secret == "" {
		c.logger.Error("QQ app_id 和 secret 未配置")
		return nil
	}

	c.running = true
	c.logger.Info("QQ 渠道已启动 (C2C 私聊)")

	// TODO: 实现 qq-botpy SDK 的 WebSocket 连接

	return nil
}

// Stop 停止 QQ 渠道
func (c *QQChannel) Stop() {
	c.running = false
	c.logger.Info("QQ 渠道已停止")
}

// Send 发送消息
func (c *QQChannel) Send(msg *bus.OutboundMessage) error {
	c.logger.Debug("发送 QQ 消息", zap.String("openid", msg.ChatID))

	// TODO: 实现实际的消息发送
	// POST /api/v2/users/{openid}/messages

	return nil
}

// handleMessage 处理接收到的消息
func (c *QQChannel) handleMessage(data map[string]interface{}) {
	// 消息去重
	msgID, _ := data["id"].(string)
	if msgID == "" {
		return
	}
	if c.processedIDs[msgID] {
		return
	}
	c.processedIDs[msgID] = true

	// 限制缓存大小
	if len(c.processedIDs) > 1000 {
		c.processedIDs = make(map[string]bool)
	}

	author, _ := data["author"].(map[string]interface{})
	userID := ""
	if id, ok := author["id"].(string); ok {
		userID = id
	} else if openid, ok := author["user_openid"].(string); ok {
		userID = openid
	}

	content, _ := data["content"].(string)
	if content == "" {
		return
	}

	c.logger.Debug("收到 QQ 消息",
		zap.String("user_id", userID),
		zap.String("content", content[:min(50, len(content))]),
	)

	// TODO: 发布到消息总线
}
