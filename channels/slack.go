package channels

import (
	"context"
	"regexp"
	"strings"

	"github.com/weibaohui/nanobot-go/bus"
	"go.uber.org/zap"
)

// SlackChannel Slack 渠道
// 使用 Socket Mode
type SlackChannel struct {
	*BaseChannel
	config    *SlackConfig
	logger    *zap.Logger
	running   bool
	botUserID string
}

// SlackConfig Slack 配置
type SlackConfig struct {
	BotToken       string
	AppToken       string
	Mode           string
	AllowFrom      []string
	DM             SlackDMConfig
	GroupPolicy    string
	GroupAllowFrom []string
}

// SlackDMConfig Slack 私聊配置
type SlackDMConfig struct {
	Enabled   bool
	Policy    string
	AllowFrom []string
}

// NewSlackChannel 创建 Slack 渠道
func NewSlackChannel(config *SlackConfig, messageBus *bus.MessageBus, logger *zap.Logger) *SlackChannel {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SlackChannel{
		BaseChannel: NewBaseChannel("slack", messageBus),
		config:      config,
		logger:      logger,
	}
}

// Start 启动 Slack 渠道
func (c *SlackChannel) Start(ctx context.Context) error {
	if c.config.BotToken == "" || c.config.AppToken == "" {
		c.logger.Error("Slack bot/app token 未配置")
		return nil
	}

	if c.config.Mode != "socket" {
		c.logger.Error("不支持的 Slack 模式", zap.String("mode", c.config.Mode))
		return nil
	}

	c.running = true
	c.logger.Info("Slack 渠道已启动")

	// TODO: 实现 Slack Socket Mode 连接

	return nil
}

// Stop 停止 Slack 渠道
func (c *SlackChannel) Stop() {
	c.running = false
	c.logger.Info("Slack 渠道已停止")
}

// Send 发送消息
func (c *SlackChannel) Send(msg *bus.OutboundMessage) error {
	// 提取 Slack 元数据
	var threadTS string
	var channelType string
	if msg.Metadata != nil {
		if slack, ok := msg.Metadata["slack"].(map[string]interface{}); ok {
			threadTS, _ = slack["thread_ts"].(string)
			channelType, _ = slack["channel_type"].(string)
		}
	}

	// 非私聊消息使用线程回复
	useThread := threadTS != "" && channelType != "im"

	c.logger.Debug("发送 Slack 消息",
		zap.String("channel", msg.ChatID),
		zap.Bool("use_thread", useThread),
	)

	// TODO: 实现实际的消息发送
	// POST https://slack.com/api/chat.postMessage

	return nil
}

// handleSocketRequest 处理 Socket Mode 请求
func (c *SlackChannel) handleSocketRequest(req map[string]interface{}) {
	reqType, _ := req["type"].(string)
	if reqType != "events_api" {
		return
	}

	// 立即确认
	// TODO: 发送 Socket Mode 响应

	payload, _ := req["payload"].(map[string]interface{})
	event, _ := payload["event"].(map[string]interface{})
	eventType, _ := event["type"].(string)

	// 处理消息或提及事件
	if eventType != "message" && eventType != "app_mention" {
		return
	}

	senderID, _ := event["user"].(string)
	chatID, _ := event["channel"].(string)

	// 忽略机器人/系统消息
	if _, hasSubtype := event["subtype"]; hasSubtype {
		return
	}
	if c.botUserID != "" && senderID == c.botUserID {
		return
	}

	// 避免重复处理：Slack 会同时发送 message 和 app_mention
	text, _ := event["text"].(string)
	if eventType == "message" && c.botUserID != "" && strings.Contains(text, "<@"+c.botUserID+">") {
		return
	}

	channelType, _ := event["channel_type"].(string)

	// 检查权限
	if !c.isAllowed(senderID, chatID, channelType) {
		return
	}

	// 检查是否应该响应
	if channelType != "im" && !c.shouldRespondInChannel(eventType, text, chatID) {
		return
	}

	// 移除机器人提及
	text = c.stripBotMention(text)

	threadTS, _ := event["thread_ts"].(string)
	if threadTS == "" {
		threadTS, _ = event["ts"].(string)
	}

	c.logger.Debug("收到 Slack 消息",
		zap.String("sender", senderID),
		zap.String("channel", chatID),
		zap.String("content", text[:min(50, len(text))]),
	)

	// TODO: 发布到消息总线
}

// isAllowed 检查权限
func (c *SlackChannel) isAllowed(senderID, chatID, channelType string) bool {
	if channelType == "im" {
		if !c.config.DM.Enabled {
			return false
		}
		if c.config.DM.Policy == "allowlist" {
			for _, allowed := range c.config.DM.AllowFrom {
				if senderID == allowed {
					return true
				}
			}
			return false
		}
		return true
	}

	// 群组/频道消息
	if c.config.GroupPolicy == "allowlist" {
		for _, allowed := range c.config.GroupAllowFrom {
			if chatID == allowed {
				return true
			}
		}
		return false
	}
	return true
}

// shouldRespondInChannel 检查是否应该在频道中响应
func (c *SlackChannel) shouldRespondInChannel(eventType, text, chatID string) bool {
	if c.config.GroupPolicy == "open" {
		return true
	}
	if c.config.GroupPolicy == "mention" {
		if eventType == "app_mention" {
			return true
		}
		return c.botUserID != "" && strings.Contains(text, "<@"+c.botUserID+">")
	}
	if c.config.GroupPolicy == "allowlist" {
		for _, allowed := range c.config.GroupAllowFrom {
			if chatID == allowed {
				return true
			}
		}
	}
	return false
}

// stripBotMention 移除机器人提及
func (c *SlackChannel) stripBotMention(text string) string {
	if text == "" || c.botUserID == "" {
		return text
	}
	re := regexp.MustCompile(`<@` + regexp.QuoteMeta(c.botUserID) + `>\s*`)
	return strings.TrimSpace(re.ReplaceAllString(text, ""))
}
