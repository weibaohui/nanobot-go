package channels

import (
	"context"
	"encoding/json"

	"github.com/weibaohui/nanobot-go/bus"
	"go.uber.org/zap"
)

// DiscordChannel Discord 渠道
// 使用 Gateway WebSocket 连接
type DiscordChannel struct {
	*BaseChannel
	config     *DiscordConfig
	logger     *zap.Logger
	running    bool
	botUserID  string
	typingStop map[string]chan struct{}
}

// DiscordConfig Discord 配置
type DiscordConfig struct {
	Token      string
	AppToken   string
	GatewayURL string
	Intents    int
	AllowFrom  []string
}

// NewDiscordChannel 创建 Discord 渠道
func NewDiscordChannel(config *DiscordConfig, messageBus *bus.MessageBus, logger *zap.Logger) *DiscordChannel {
	if logger == nil {
		logger = zap.NewNop()
	}
	if config.GatewayURL == "" {
		config.GatewayURL = "wss://gateway.discord.gg/?v=10&encoding=json"
	}
	return &DiscordChannel{
		BaseChannel: NewBaseChannel("discord", messageBus),
		config:      config,
		logger:      logger,
		typingStop:  make(map[string]chan struct{}),
	}
}

// Start 启动 Discord 渠道
func (c *DiscordChannel) Start(ctx context.Context) error {
	if c.config.Token == "" {
		c.logger.Error("Discord bot token 未配置")
		return nil
	}

	c.running = true
	c.logger.Info("Discord 渠道已启动")

	// TODO: 实现 Discord Gateway WebSocket 连接

	return nil
}

// Stop 停止 Discord 渠道
func (c *DiscordChannel) Stop() {
	c.running = false
	for _, stop := range c.typingStop {
		close(stop)
	}
	c.logger.Info("Discord 渠道已停止")
}

// Send 发送消息
func (c *DiscordChannel) Send(msg *bus.OutboundMessage) error {
	// 停止输入指示器
	if stop, ok := c.typingStop[msg.ChatID]; ok {
		close(stop)
		delete(c.typingStop, msg.ChatID)
	}

	// TODO: 实现通过 Discord REST API 发送消息
	// POST https://discord.com/api/v10/channels/{channel_id}/messages

	c.logger.Debug("发送 Discord 消息", zap.String("channel_id", msg.ChatID))
	return nil
}

// handleGatewayMessage 处理 Gateway 消息
func (c *DiscordChannel) handleGatewayMessage(raw string) {
	var data struct {
		Op   int                    `json:"op"`
		Type string                 `json:"t"`
		Seq  int                    `json:"s"`
		D    map[string]interface{} `json:"d"`
	}

	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		c.logger.Warn("无效的 Gateway 消息")
		return
	}

	switch data.Op {
	case 10: // HELLO
		interval, _ := data.D["heartbeat_interval"].(float64)
		c.logger.Debug("Discord Gateway HELLO", zap.Float64("heartbeat_interval", interval))
		// TODO: 开始心跳循环
	case 0: // Dispatch
		switch data.Type {
		case "READY":
			user, _ := data.D["user"].(map[string]interface{})
			c.botUserID, _ = user["id"].(string)
			c.logger.Info("Discord Gateway 已就绪", zap.String("bot_user_id", c.botUserID))
		case "MESSAGE_CREATE":
			c.handleMessageCreate(data.D)
		}
	case 7: // RECONNECT
		c.logger.Info("Discord Gateway 请求重连")
	case 9: // INVALID_SESSION
		c.logger.Warn("Discord Gateway 会话无效")
	}
}

// handleMessageCreate 处理消息创建事件
func (c *DiscordChannel) handleMessageCreate(data map[string]interface{}) {
	author, _ := data["author"].(map[string]interface{})
	if bot, _ := author["bot"].(bool); bot {
		return
	}

	senderID, _ := author["id"].(string)
	channelID, _ := data["channel_id"].(string)
	content, _ := data["content"].(string)

	c.logger.Debug("收到 Discord 消息",
		zap.String("sender", senderID),
		zap.String("channel", channelID),
		zap.String("content", content[:min(50, len(content))]),
	)

	// TODO: 发布到消息总线
}
