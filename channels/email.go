package channels

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/weibaohui/nanobot-go/bus"
	"go.uber.org/zap"
)

// EmailChannel 邮件渠道
// 使用 IMAP 接收 + SMTP 发送
type EmailChannel struct {
	*BaseChannel
	config            *EmailConfig
	logger            *zap.Logger
	running           bool
	lastSubjectByChat map[string]string
	lastMessageID     map[string]string
	processedUIDs     map[string]bool
}

// EmailConfig 邮件配置
type EmailConfig struct {
	IMAPHost          string
	IMAPPort          int
	IMAPUsername      string
	IMAPPassword      string
	IMAPUseSSL        bool
	IMAPMailbox       string
	SMTPHost          string
	SMTPPort          int
	SMTPUsername      string
	SMTPPassword      string
	SMTPUseSSL        bool
	SMTPUseTLS        bool
	FromAddress       string
	PollInterval      int
	MarkSeen          bool
	MaxBodyChars      int
	ConsentGranted    bool
	AutoReplyEnabled  bool
	SubjectPrefix     string
	AllowFrom         []string
}

// NewEmailChannel 创建邮件渠道
func NewEmailChannel(config *EmailConfig, messageBus *bus.MessageBus, logger *zap.Logger) *EmailChannel {
	if logger == nil {
		logger = zap.NewNop()
	}
	if config.PollInterval == 0 {
		config.PollInterval = 60
	}
	if config.MaxBodyChars == 0 {
		config.MaxBodyChars = 10000
	}
	return &EmailChannel{
		BaseChannel:       NewBaseChannel("email", messageBus),
		config:            config,
		logger:            logger,
		lastSubjectByChat: make(map[string]string),
		lastMessageID:     make(map[string]string),
		processedUIDs:     make(map[string]bool),
	}
}

// Start 启动邮件渠道
func (c *EmailChannel) Start(ctx context.Context) error {
	if !c.config.ConsentGranted {
		c.logger.Warn("邮件渠道已禁用: consent_granted 为 false")
		return nil
	}

	if !c.validateConfig() {
		return nil
	}

	c.running = true
	c.logger.Info("邮件渠道已启动 (IMAP 轮询模式)")

	// TODO: 实现 IMAP 轮询循环

	return nil
}

// Stop 停止邮件渠道
func (c *EmailChannel) Stop() {
	c.running = false
	c.logger.Info("邮件渠道已停止")
}

// Send 发送邮件
func (c *EmailChannel) Send(msg *bus.OutboundMessage) error {
	if !c.config.ConsentGranted {
		c.logger.Warn("跳过邮件发送: consent_granted 为 false")
		return nil
	}

	forceSend := false
	if msg.Metadata != nil {
		if fs, ok := msg.Metadata["force_send"].(bool); ok {
			forceSend = fs
		}
	}

	if !c.config.AutoReplyEnabled && !forceSend {
		c.logger.Info("跳过自动邮件回复: auto_reply_enabled 为 false")
		return nil
	}

	toAddr := strings.TrimSpace(msg.ChatID)
	if toAddr == "" {
		c.logger.Warn("邮件渠道缺少收件人地址")
		return nil
	}

	// 构建邮件
	baseSubject := c.lastSubjectByChat[toAddr]
	if baseSubject == "" {
		baseSubject = "nanobot 回复"
	}
	subject := c.replySubject(baseSubject)

	// 检查是否有自定义主题
	if msg.Metadata != nil {
		if override, ok := msg.Metadata["subject"].(string); ok && override != "" {
			subject = override
		}
	}

	c.logger.Debug("发送邮件",
		zap.String("to", toAddr),
		zap.String("subject", subject),
	)

	// TODO: 实现 SMTP 发送

	return nil
}

// validateConfig 验证配置
func (c *EmailChannel) validateConfig() bool {
	missing := []string{}
	if c.config.IMAPHost == "" {
		missing = append(missing, "imap_host")
	}
	if c.config.IMAPUsername == "" {
		missing = append(missing, "imap_username")
	}
	if c.config.IMAPPassword == "" {
		missing = append(missing, "imap_password")
	}
	if c.config.SMTPHost == "" {
		missing = append(missing, "smtp_host")
	}
	if c.config.SMTPUsername == "" {
		missing = append(missing, "smtp_username")
	}
	if c.config.SMTPPassword == "" {
		missing = append(missing, "smtp_password")
	}

	if len(missing) > 0 {
		c.logger.Error("邮件渠道配置不完整", zap.Strings("missing", missing))
		return false
	}
	return true
}

// replySubject 生成回复主题
func (c *EmailChannel) replySubject(baseSubject string) string {
	subject := strings.TrimSpace(baseSubject)
	if subject == "" {
		subject = "nanobot 回复"
	}
	prefix := c.config.SubjectPrefix
	if prefix == "" {
		prefix = "Re: "
	}
	if strings.HasPrefix(strings.ToLower(subject), "re:") {
		return subject
	}
	return prefix + subject
}

// fetchNewMessages 获取新消息
func (c *EmailChannel) fetchNewMessages() []map[string]interface{} {
	// TODO: 实现 IMAP 消息获取
	// 搜索 UNSEEN 消息
	return nil
}

// extractTextBody 提取邮件正文
func (c *EmailChannel) extractTextBody(msg interface{}) string {
	// TODO: 实现邮件正文提取
	// 支持 text/plain 和 text/html
	return ""
}

// htmlToText 将 HTML 转换为文本
func htmlToText(html string) string {
	// 替换 <br> 和 </p> 为换行
	text := regexp.MustCompile(`(?i)<\s*br\s*/?>`).ReplaceAllString(html, "\n")
	text = regexp.MustCompile(`(?i)<\s*/\s*p\s*>`).ReplaceAllString(text, "\n")
	// 移除其他 HTML 标签
	text = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(text, "")
	return text
}

// formatIMAPDate 格式化 IMAP 日期
func formatIMAPDate(t time.Time) string {
	months := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun",
		"Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	return strings.Replace(t.Format("02-Jan-2006"), "Jan", months[t.Month()-1], 1)
}
