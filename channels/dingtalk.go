package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/client"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/logger"
	"github.com/weibaohui/nanobot-go/bus"
	"go.uber.org/zap"
)

// sdkLoggerAdapter SDK 日志适配器
type sdkLoggerAdapter struct {
	logger *zap.Logger
}

func (l *sdkLoggerAdapter) Debugf(format string, args ...interface{}) {
	l.logger.Sugar().Debugf(format, args...)
}

func (l *sdkLoggerAdapter) Infof(format string, args ...interface{}) {
	l.logger.Sugar().Infof(format, args...)
}

func (l *sdkLoggerAdapter) Warningf(format string, args ...interface{}) {
	l.logger.Sugar().Warnf(format, args...)
}

func (l *sdkLoggerAdapter) Errorf(format string, args ...interface{}) {
	l.logger.Sugar().Errorf(format, args...)
}

func (l *sdkLoggerAdapter) Fatalf(format string, args ...interface{}) {
	l.logger.Sugar().Fatalf(format, args...)
}

// DingTalkChannel 钉钉渠道
// 使用官方 Stream SDK 接收消息，HTTP API 发送消息
type DingTalkChannel struct {
	*BaseChannel
	config  *DingTalkConfig
	logger  *zap.Logger
	running bool

	// Stream 客户端
	streamClient *client.StreamClient

	// HTTP 客户端
	httpClient *http.Client

	// Access Token 管理
	accessToken string
	tokenExpiry time.Time
	tokenMutex  sync.RWMutex

	// Session Webhook 缓存
	sessionCache map[string]*sessionContext
	sessionMutex sync.RWMutex

	// 后台任务管理
	bgTasks sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

// sessionContext 会话上下文
type sessionContext struct {
	sessionWebhook     string
	expireTime         time.Time
	conversationType   string
	openConversationID string
}

// DingTalkConfig 钉钉配置
type DingTalkConfig struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	AllowFrom    []string `json:"allow_from"`
}

// NewDingTalkChannel 创建钉钉渠道
func NewDingTalkChannel(config *DingTalkConfig, messageBus *bus.MessageBus, logger *zap.Logger) *DingTalkChannel {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &DingTalkChannel{
		BaseChannel:  NewBaseChannel("dingtalk", messageBus),
		config:       config,
		logger:       logger,
		sessionCache: make(map[string]*sessionContext),
	}
}

// Start 启动钉钉渠道
func (c *DingTalkChannel) Start(ctx context.Context) error {
	if c.config.ClientID == "" || c.config.ClientSecret == "" {
		c.logger.Error("钉钉 client_id 和 client_secret 未配置")
		return fmt.Errorf("钉钉配置不完整")
	}

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.running = true

	// 初始化 HTTP 客户端
	c.httpClient = &http.Client{
		Timeout: 30 * time.Second,
	}

	// 设置 SDK 日志
	logger.SetLogger(&sdkLoggerAdapter{logger: c.logger})

	c.logger.Info("钉钉渠道已启动", zap.String("client_id", c.config.ClientID))

	// 创建 Stream 客户端
	c.streamClient = client.NewStreamClient(
		client.WithAppCredential(client.NewAppCredentialConfig(c.config.ClientID, c.config.ClientSecret)),
	)

	// 注册机器人消息回调
	c.streamClient.RegisterChatBotCallbackRouter(c.onChatBotMessage)

	// 订阅出站消息
	c.SubscribeOutbound(ctx, func(msg *bus.OutboundMessage) {
		if err := c.Send(msg); err != nil {
			c.logger.Error("发送钉钉消息失败", zap.Error(err))
		}
	})

	// 启动 Stream 客户端
	c.bgTasks.Add(1)
	go c.runStreamClient()

	return nil
}

// runStreamClient 运行 Stream 客户端
func (c *DingTalkChannel) runStreamClient() {
	defer c.bgTasks.Done()

	for c.running {
		err := c.streamClient.Start(c.ctx)
		if err != nil {
			c.logger.Warn("钉钉 Stream 连接错误", zap.Error(err))
		}

		if !c.running {
			break
		}

		// 等待 5 秒后重连
		// c.logger.Info("钉钉 Stream 将在 5 秒后重连...")
		select {
		case <-time.After(5 * time.Second):
		case <-c.ctx.Done():
			return
		}
	}
}

// onChatBotMessage 处理机器人消息
func (c *DingTalkChannel) onChatBotMessage(ctx context.Context, data *chatbot.BotCallbackDataModel) ([]byte, error) {
	// 提取文本内容
	content := strings.TrimSpace(data.Text.Content)

	if content == "" {
		c.logger.Debug("收到空消息，忽略")
		return []byte(""), nil
	}

	senderID := data.SenderStaffId
	if senderID == "" {
		senderID = data.SenderId
	}
	senderName := data.SenderNick
	if senderName == "" {
		senderName = "Unknown"
	}

	// 缓存 session webhook 用于回复
	if data.SessionWebhook != "" {
		c.sessionMutex.Lock()
		c.sessionCache[data.ConversationId] = &sessionContext{
			sessionWebhook:     data.SessionWebhook,
			expireTime:         time.Now().Add(time.Duration(data.SessionWebhookExpiredTime) * time.Second),
			conversationType:   data.ConversationType,
			openConversationID: data.ConversationId,
		}
		c.sessionMutex.Unlock()
	}

	c.logger.Info("收到钉钉消息",
		zap.String("sender", senderName),
		zap.String("sender_id", senderID),
		zap.String("content", content),
	)

	// 发布消息到总线
	chatType := "direct"
	if data.ConversationType == "2" {
		chatType = "group"
	}

	c.bus.PublishInbound(&bus.InboundMessage{
		Channel:   "dingtalk",
		ChatID:    data.ConversationId,
		SenderID:  senderID,
		Content:   content,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"msg_id":          data.MsgId,
			"session_webhook": data.SessionWebhook,
			"conversation_id": data.ConversationId,
			"sender_name":     senderName,
			"chat_type":       chatType,
		},
	})

	return []byte(""), nil
}

// Stop 停止钉钉渠道
func (c *DingTalkChannel) Stop() {
	c.running = false
	if c.cancel != nil {
		c.cancel()
	}

	// 关闭 Stream 客户端
	if c.streamClient != nil {
		c.streamClient.Close()
	}

	// 等待后台任务完成
	c.bgTasks.Wait()

	// 关闭 HTTP 客户端
	if c.httpClient != nil {
		c.httpClient.CloseIdleConnections()
	}

	c.logger.Info("钉钉渠道已停止")
}

// Send 发送消息
func (c *DingTalkChannel) Send(msg *bus.OutboundMessage) error {
	// 优先使用 session webhook 回复
	c.sessionMutex.RLock()
	session, ok := c.sessionCache[msg.ChatID]
	c.sessionMutex.RUnlock()

	if ok && session.sessionWebhook != "" && time.Now().Before(session.expireTime) {
		return c.replyViaWebhook(session.sessionWebhook, msg.Content)
	}

	// 回退到 API 发送
	return c.sendViaAPI(msg.ChatID, msg.Content)
}

// replyViaWebhook 通过 Webhook 回复消息
func (c *DingTalkChannel) replyViaWebhook(webhookURL, content string) error {
	reqBody := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": content,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(c.ctx, "POST", webhookURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("发送失败: status=%d", resp.StatusCode)
	}

	c.logger.Debug("钉钉消息已通过 Webhook 发送")
	return nil
}

// sendViaAPI 通过 API 发送消息
func (c *DingTalkChannel) sendViaAPI(userID, content string) error {
	token, err := c.getAccessToken()
	if err != nil {
		return fmt.Errorf("获取 access token 失败: %w", err)
	}

	// 构建发送请求
	// https://open.dingtalk.com/document/orgapp/robot-batch-send-messages
	url := "https://api.dingtalk.com/v1.0/robot/oToMessages/batchSend"

	reqBody := map[string]interface{}{
		"robotCode": c.config.ClientID,
		"userIds":   []string{userID},
		"msgKey":    "sampleText",
		"msgParam": map[string]interface{}{
			"content": content,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(c.ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-acs-dingtalk-access-token", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("钉钉发送消息失败",
			zap.Int("status", resp.StatusCode),
			zap.String("response", string(respBody)),
		)
		return fmt.Errorf("发送失败: %s", string(respBody))
	}

	c.logger.Debug("钉钉消息已发送", zap.String("user_id", userID))
	return nil
}

// getAccessToken 获取或刷新 Access Token
func (c *DingTalkChannel) getAccessToken() (string, error) {
	c.tokenMutex.RLock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		token := c.accessToken
		c.tokenMutex.RUnlock()
		return token, nil
	}
	c.tokenMutex.RUnlock()

	c.tokenMutex.Lock()
	defer c.tokenMutex.Unlock()

	// 双重检查
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}

	// 获取新 token
	url := "https://api.dingtalk.com/v1.0/oauth2/accessToken"
	reqBody := map[string]string{
		"appKey":    c.config.ClientID,
		"appSecret": c.config.ClientSecret,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(c.ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 读取响应体用于调试
	respBody, _ := io.ReadAll(resp.Body)
	c.logger.Debug("钉钉 Token API 响应",
		zap.Int("status", resp.StatusCode),
		zap.String("body", string(respBody)),
	)

	var result struct {
		AccessToken string `json:"accessToken"`
		ExpireIn    int    `json:"expireIn"`
		Code        string `json:"code"`
		Message     string `json:"message"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("解析 token 响应失败: %w", err)
	}

	if result.Code != "" && result.Code != "0" {
		return "", fmt.Errorf("获取 token 失败: code=%s, message=%s", result.Code, result.Message)
	}

	if result.AccessToken == "" {
		return "", fmt.Errorf("获取 token 失败: accessToken 为空, 响应: %s", string(respBody))
	}

	c.accessToken = result.AccessToken
	// 提前 60 秒过期，确保安全
	c.tokenExpiry = time.Now().Add(time.Duration(result.ExpireIn-60) * time.Second)

	c.logger.Debug("钉钉 access token 已刷新",
		zap.Int("expire_in", result.ExpireIn),
	)

	return c.accessToken, nil
}
