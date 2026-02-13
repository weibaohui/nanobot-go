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

	"github.com/weibaohui/nanobot-go/bus"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// DingTalkChannel 钉钉渠道
// 使用 Stream Mode (WebSocket) 接收消息，HTTP API 发送消息
type DingTalkChannel struct {
	*BaseChannel
	config  *DingTalkConfig
	logger  *zap.Logger
	running bool

	// HTTP 客户端
	httpClient *http.Client

	// Access Token 管理
	accessToken string
	tokenExpiry time.Time
	tokenMutex  sync.RWMutex

	// WebSocket 连接
	wsConn      *websocket.Conn
	wsMutex     sync.Mutex
	reconnectCh chan struct{}

	// 后台任务管理
	bgTasks sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
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
		BaseChannel: NewBaseChannel("dingtalk", messageBus),
		config:      config,
		logger:      logger,
		reconnectCh: make(chan struct{}, 1),
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

	c.logger.Info("钉钉渠道已启动", zap.String("client_id", c.config.ClientID))

	// 启动 WebSocket 连接
	c.bgTasks.Add(1)
	go c.runStreamLoop()

	return nil
}

// Stop 停止钉钉渠道
func (c *DingTalkChannel) Stop() {
	c.running = false
	if c.cancel != nil {
		c.cancel()
	}

	// 关闭 WebSocket 连接
	c.wsMutex.Lock()
	if c.wsConn != nil {
		c.wsConn.Close(websocket.StatusNormalClosure, "channel stopping")
		c.wsConn = nil
	}
	c.wsMutex.Unlock()

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
	token, err := c.getAccessToken()
	if err != nil {
		return fmt.Errorf("获取 access token 失败: %w", err)
	}

	// 构建发送请求
	// https://open.dingtalk.com/document/orgapp/robot-batch-send-messages
	url := "https://api.dingtalk.com/v1.0/robot/oToMessages/batchSend"

	reqBody := map[string]interface{}{
		"robotCode": c.config.ClientID,
		"userIds":   []string{msg.ChatID},
		"msgKey":    "sampleMarkdown",
		"msgParam": map[string]interface{}{
			"text":  msg.Content,
			"title": "Nanobot 回复",
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

	c.logger.Debug("钉钉消息已发送", zap.String("chat_id", msg.ChatID))
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

	var result struct {
		AccessToken string `json:"accessToken"`
		ExpireIn    int    `json:"expireIn"`
		Code        string `json:"code"`
		Message     string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Code != "" && result.Code != "0" {
		return "", fmt.Errorf("获取 token 失败: %s", result.Message)
	}

	c.accessToken = result.AccessToken
	// 提前 60 秒过期，确保安全
	c.tokenExpiry = time.Now().Add(time.Duration(result.ExpireIn-60) * time.Second)

	c.logger.Debug("钉钉 access token 已刷新",
		zap.Int("expire_in", result.ExpireIn),
	)

	return c.accessToken, nil
}

// runStreamLoop 运行 Stream 模式循环
func (c *DingTalkChannel) runStreamLoop() {
	defer c.bgTasks.Done()

	for c.running {
		err := c.connectAndListen()
		if err != nil {
			c.logger.Warn("钉钉 Stream 连接错误", zap.Error(err))
		}

		if !c.running {
			break
		}

		// 等待 5 秒后重连
		c.logger.Info("钉钉 Stream 将在 5 秒后重连...")
		select {
		case <-time.After(5 * time.Second):
		case <-c.ctx.Done():
			return
		}
	}
}

// connectAndListen 连接并监听消息
func (c *DingTalkChannel) connectAndListen() error {
	// 获取订阅地址
	subscribeURL, err := c.getSubscribeURL()
	if err != nil {
		return fmt.Errorf("获取订阅地址失败: %w", err)
	}

	c.logger.Debug("钉钉 Stream 订阅地址", zap.String("url", subscribeURL))

	// 建立 WebSocket 连接
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, subscribeURL, nil)
	if err != nil {
		return fmt.Errorf("WebSocket 连接失败: %w", err)
	}
	defer conn.Close(websocket.StatusInternalError, "closing")

	c.wsMutex.Lock()
	c.wsConn = conn
	c.wsMutex.Unlock()

	c.logger.Info("钉钉 Stream 已连接")

	// 监听消息
	for {
		var msg dingTalkStreamMessage
		err := wsjson.Read(c.ctx, conn, &msg)
		if err != nil {
			if c.ctx.Err() != nil {
				return nil // 正常关闭
			}
			return fmt.Errorf("读取消息失败: %w", err)
		}

		// 处理消息
		if err := c.handleStreamMessage(conn, &msg); err != nil {
			c.logger.Warn("处理 Stream 消息失败", zap.Error(err))
		}
	}
}

// getSubscribeURL 获取订阅地址
func (c *DingTalkChannel) getSubscribeURL() (string, error) {
	token, err := c.getAccessToken()
	if err != nil {
		return "", err
	}

	url := "https://api.dingtalk.com/v1.0/gateway/connections/open"

	reqBody := map[string]interface{}{
		"client_id": c.config.ClientID,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(c.ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-acs-dingtalk-access-token", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Endpoint string `json:"endpoint"`
		Token    string `json:"token"`
		Code     string `json:"code"`
		Message  string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Code != "" && result.Code != "0" && result.Code != "success" {
		return "", fmt.Errorf("获取订阅地址失败: %s", result.Message)
	}

	// 构建完整的 WebSocket URL
	return fmt.Sprintf("%s?token=%s", result.Endpoint, result.Token), nil
}

// dingTalkStreamMessage 钉钉 Stream 消息结构
type dingTalkStreamMessage struct {
	Topic string          `json:"topic"`
	Type  string          `json:"type"`
	Data  json.RawMessage `json:"data"`
}

// handleStreamMessage 处理 Stream 消息
func (c *DingTalkChannel) handleStreamMessage(conn *websocket.Conn, msg *dingTalkStreamMessage) error {
	// 发送确认响应
	ack := map[string]string{
		"code":    "200",
		"message": "OK",
	}
	if err := wsjson.Write(c.ctx, conn, ack); err != nil {
		c.logger.Warn("发送 ACK 失败", zap.Error(err))
	}

	// 处理聊天机器人消息
	if msg.Topic == "chatbot" || msg.Type == "CALLBACK" {
		return c.handleChatbotMessage(msg.Data)
	}

	return nil
}

// handleChatbotMessage 处理聊天机器人消息
func (c *DingTalkChannel) handleChatbotMessage(data json.RawMessage) error {
	var msg dingTalkChatbotMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("解析消息失败: %w", err)
	}

	// 提取文本内容
	content := ""
	if msg.Text != nil {
		content = strings.TrimSpace(msg.Text.Content)
	}

	if content == "" {
		c.logger.Debug("收到空消息，忽略")
		return nil
	}

	// 获取发送者信息
	senderID := msg.SenderStaffID
	if senderID == "" {
		senderID = msg.SenderID
	}
	senderName := msg.SenderNick
	if senderName == "" {
		senderName = "Unknown"
	}

	c.logger.Info("收到钉钉消息",
		zap.String("sender", senderName+"("+senderID+")"),
		zap.String("content", content),
	)

	// 发布到消息总线
	inboundMsg := &bus.InboundMessage{
		Channel:  "dingtalk",
		SenderID: senderID,
		ChatID:   senderID, // 私聊场景，chat_id == sender_id
		Content:  content,
		Metadata: map[string]interface{}{
			"sender_name": senderName,
			"message_id":  msg.MessageID,
			"msg_type":    msg.MsgType,
			"platform":    "dingtalk",
		},
	}

	c.bus.PublishInbound(inboundMsg)

	return nil
}

// dingTalkChatbotMessage 钉钉聊天机器人消息结构
type dingTalkChatbotMessage struct {
	MessageID                 string                   `json:"msgId"`
	MsgType                   string                   `json:"msgtype"`
	ConversationID            string                   `json:"conversationId"`
	SenderID                  string                   `json:"senderId"`
	SenderStaffID             string                   `json:"senderStaffId"`
	SenderNick                string                   `json:"senderNick"`
	SenderCorpID              string                   `json:"senderCorpId"`
	SenderDing                string                   `json:"senderDing"`
	Webhook                   string                   `json:"webhook"`
	Text                      *dingTalkChatbotText     `json:"text"`
	RichText                  *dingTalkChatbotRichText `json:"richText"`
	AtUsers                   []dingTalkChatbotAtUser  `json:"atUsers"`
	ChatbotUserID             string                   `json:"chatbotUserId"`
	CreateAt                  int64                    `json:"createAt"`
	SessionWebhook            string                   `json:"sessionWebhook"`
	SessionWebhookExpiredTime int64                    `json:"sessionWebhookExpiredTime"`
}

// dingTalkChatbotText 文本消息内容
type dingTalkChatbotText struct {
	Content string `json:"content"`
}

// dingTalkChatbotRichText 富文本消息内容
type dingTalkChatbotRichText struct {
	RichTextContent string `json:"richTextContent"`
}

// dingTalkChatbotAtUser @用户信息
type dingTalkChatbotAtUser struct {
	DingTalkID string `json:"dingtalkId"`
	StaffID    string `json:"staffId"`
}

// IsAllowed 检查发送者是否被允许
func (c *DingTalkChannel) IsAllowed(senderID string) bool {
	if len(c.config.AllowFrom) == 0 {
		return true
	}
	for _, allowed := range c.config.AllowFrom {
		if senderID == allowed {
			return true
		}
	}
	return false
}
