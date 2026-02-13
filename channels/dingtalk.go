package channels

import (
	"context"
	"encoding/json"
	"time"

	"github.com/weibaohui/nanobot-go/bus"
	"go.uber.org/zap"
)

// DingTalkChannel 钉钉渠道
// 使用 Stream Mode (WebSocket)
type DingTalkChannel struct {
	*BaseChannel
	config      *DingTalkConfig
	logger      *zap.Logger
	running     bool
	accessToken string
	tokenExpiry time.Time
}

// DingTalkConfig 钉钉配置
type DingTalkConfig struct {
	ClientID     string
	ClientSecret string
	AllowFrom    []string
}

// NewDingTalkChannel 创建钉钉渠道
func NewDingTalkChannel(config *DingTalkConfig, messageBus *bus.MessageBus, logger *zap.Logger) *DingTalkChannel {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &DingTalkChannel{
		BaseChannel: NewBaseChannel("dingtalk", messageBus),
		config:      config,
		logger:      logger,
	}
}

// Start 启动钉钉渠道
func (c *DingTalkChannel) Start(ctx context.Context) error {
	if c.config.ClientID == "" || c.config.ClientSecret == "" {
		c.logger.Error("钉钉 client_id 和 client_secret 未配置")
		return nil
	}

	c.running = true
	c.logger.Info("钉钉渠道已启动")

	// TODO: 实现 dingtalk-stream SDK 的 Stream Mode 连接

	return nil
}

// Stop 停止钉钉渠道
func (c *DingTalkChannel) Stop() {
	c.running = false
	c.logger.Info("钉钉渠道已停止")
}

// Send 发送消息
func (c *DingTalkChannel) Send(msg *bus.OutboundMessage) error {
	token, err := c.getAccessToken()
	if err != nil {
		return err
	}

	c.logger.Debug("发送钉钉消息",
		zap.String("user_id", msg.ChatID),
		zap.String("token", token[:min(10, len(token))]+"..."),
	)

	// TODO: 实现实际的 HTTP 请求发送
	// POST https://api.dingtalk.com/v1.0/robot/oToMessages/batchSend

	return nil
}

// getAccessToken 获取或刷新 Access Token
func (c *DingTalkChannel) getAccessToken() (string, error) {
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}

	// TODO: 实现 token 获取
	// POST https://api.dingtalk.com/v1.0/oauth2/accessToken
	// {"appKey": clientID, "appSecret": clientSecret}

	return "", nil
}

// handleMessage 处理接收到的消息
func (c *DingTalkChannel) handleMessage(data map[string]interface{}) {
	// 解析消息
	textContent, _ := data["text"].(map[string]interface{})
	content, _ := textContent["content"].(string)
	senderID, _ := data["senderStaffId"].(string)
	if senderID == "" {
		senderID, _ = data["senderId"].(string)
	}
	senderName, _ := data["senderNick"].(string)

	c.logger.Debug("收到钉钉消息",
		zap.String("sender", senderName+"("+senderID+")"),
		zap.String("content", content[:min(50, len(content))]),
	)

	// TODO: 发布到消息总线
}

// DingTalkHandler 钉钉消息处理器
type DingTalkHandler struct {
	channel *DingTalkChannel
}

// Process 处理消息
func (h *DingTalkHandler) Process(message map[string]interface{}) (int, string) {
	chatbotMsg, _ := message["data"].(map[string]interface{})

	// 提取文本内容
	content := ""
	if text, ok := chatbotMsg["text"].(map[string]interface{}); ok {
		content, _ = text["content"].(string)
	}

	if content == "" {
		return 0, "OK"
	}

	senderID, _ := chatbotMsg["senderStaffId"].(string)
	if senderID == "" {
		senderID, _ = chatbotMsg["senderId"].(string)
	}
	senderName, _ := chatbotMsg["senderNick"].(string)

	h.channel.logger.Info("收到钉钉消息",
		zap.String("sender", senderName+"("+senderID+")"),
		zap.String("content", content),
	)

	// 异步处理消息
	go h.channel.handleMessage(chatbotMsg)

	return 0, "OK"
}

// MarshalJSON 实现 JSON 序列化
func (h *DingTalkHandler) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{})
}
