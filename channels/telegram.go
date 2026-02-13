package channels

import (
	"context"
	"regexp"
	"strings"

	"github.com/weibaohui/nanobot-go/bus"
	"go.uber.org/zap"
)

// TelegramChannel Telegram 渠道
// 需要外部依赖: github.com/go-telegram-bot-api/telegram-bot-api
type TelegramChannel struct {
	*BaseChannel
	config     *TelegramConfig
	logger     *zap.Logger
	running    bool
	chatIDs    map[string]int64
	typingStop map[string]chan struct{}
}

// TelegramConfig Telegram 配置
type TelegramConfig struct {
	Token     string
	Proxy     string
	AllowFrom []string
}

// NewTelegramChannel 创建 Telegram 渠道
func NewTelegramChannel(config *TelegramConfig, messageBus *bus.MessageBus, logger *zap.Logger) *TelegramChannel {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &TelegramChannel{
		BaseChannel: NewBaseChannel("telegram", messageBus),
		config:      config,
		logger:      logger,
		chatIDs:     make(map[string]int64),
		typingStop:  make(map[string]chan struct{}),
	}
}

// Start 启动 Telegram 渠道
func (c *TelegramChannel) Start(ctx context.Context) error {
	if c.config.Token == "" {
		c.logger.Error("Telegram bot token 未配置")
		return nil
	}

	c.running = true
	c.logger.Info("Telegram 渠道已启动")

	// TODO: 实现实际的 Telegram Bot API 连接
	// 需要导入: "github.com/go-telegram-bot-api/telegram-bot-api"
	// bot, err := tgbotapi.NewBotAPI(c.config.Token)

	return nil
}

// Stop 停止 Telegram 渠道
func (c *TelegramChannel) Stop() {
	c.running = false
	for _, stop := range c.typingStop {
		close(stop)
	}
	c.logger.Info("Telegram 渠道已停止")
}

// Send 发送消息
func (c *TelegramChannel) Send(msg *bus.OutboundMessage) error {
	// 停止输入指示器
	if stop, ok := c.typingStop[msg.ChatID]; ok {
		close(stop)
		delete(c.typingStop, msg.ChatID)
	}

	// TODO: 实现实际的消息发送
	// chatID, err := strconv.ParseInt(msg.ChatID, 10, 64)
	// tgMsg := tgbotapi.NewMessage(chatID, markdownToTelegramHTML(msg.Content))
	// tgMsg.ParseMode = "HTML"
	// _, err = c.bot.Send(tgMsg)

	c.logger.Debug("发送 Telegram 消息", zap.String("chat_id", msg.ChatID))
	return nil
}

// markdownToTelegramHTML 将 Markdown 转换为 Telegram 安全的 HTML
func markdownToTelegramHTML(text string) string {
	if text == "" {
		return ""
	}

	// 保护代码块
	codeBlocks := []string{}
	codeBlockRe := regexp.MustCompile("```[\\w]*\n?([\\s\\S]*?)```")
	text = codeBlockRe.ReplaceAllStringFunc(text, func(m string) string {
		matches := codeBlockRe.FindStringSubmatch(m)
		if len(matches) > 1 {
			codeBlocks = append(codeBlocks, matches[1])
			return "\x00CB" + strings.Repeat(" ", len(codeBlocks)-1) + "\x00"
		}
		return m
	})

	// 保护行内代码
	inlineCodes := []string{}
	inlineRe := regexp.MustCompile("`([^`]+)`")
	text = inlineRe.ReplaceAllStringFunc(text, func(m string) string {
		matches := inlineRe.FindStringSubmatch(m)
		if len(matches) > 1 {
			inlineCodes = append(inlineCodes, matches[1])
			return "\x00IC" + strings.Repeat(" ", len(inlineCodes)-1) + "\x00"
		}
		return m
	})

	// 标题处理
	headerRe := regexp.MustCompile("(?m)^#{1,6}\\s+(.+)$")
	text = headerRe.ReplaceAllString(text, "$1")

	// 引用处理
	quoteRe := regexp.MustCompile("(?m)^>\\s*(.*)$")
	text = quoteRe.ReplaceAllString(text, "$1")

	// HTML 转义
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")

	// 链接处理
	linkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	text = linkRe.ReplaceAllString(text, `<a href="$2">$1</a>`)

	// 粗体处理
	boldRe := regexp.MustCompile(`\*\*(.+?)\*\*`)
	text = boldRe.ReplaceAllString(text, "<b>$1</b>")

	// 斜体处理
	italicRe := regexp.MustCompile(`(?<![a-zA-Z0-9])_([^_]+)_(?![a-zA-Z0-9])`)
	text = italicRe.ReplaceAllString(text, "<i>$1</i>")

	// 列表处理
	listRe := regexp.MustCompile("(?m)^[-*]\\s+")
	text = listRe.ReplaceAllString(text, "• ")

	// 恢复行内代码
	for i, code := range inlineCodes {
		escaped := strings.ReplaceAll(code, "&", "&amp;")
		escaped = strings.ReplaceAll(escaped, "<", "&lt;")
		escaped = strings.ReplaceAll(escaped, ">", "&gt;")
		text = strings.Replace(text, "\x00IC"+strings.Repeat(" ", i)+"\x00", "<code>"+escaped+"</code>", 1)
	}

	// 恢复代码块
	for i, code := range codeBlocks {
		escaped := strings.ReplaceAll(code, "&", "&amp;")
		escaped = strings.ReplaceAll(escaped, "<", "&lt;")
		escaped = strings.ReplaceAll(escaped, ">", "&gt;")
		text = strings.Replace(text, "\x00CB"+strings.Repeat(" ", i)+"\x00", "<pre><code>"+escaped+"</code></pre>", 1)
	}

	return text
}

// IsAllowed 检查发送者是否被允许
func (c *TelegramChannel) IsAllowed(senderID string) bool {
	if len(c.config.AllowFrom) == 0 {
		return true
	}
	for _, allowed := range c.config.AllowFrom {
		if senderID == allowed {
			return true
		}
		// 支持格式: "id|username"
		if strings.Contains(senderID, "|") {
			for _, part := range strings.Split(senderID, "|") {
				if part == allowed {
					return true
				}
			}
		}
	}
	return false
}
