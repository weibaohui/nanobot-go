package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	"github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/larksuite/oapi-sdk-go/v3/ws"
	"github.com/weibaohui/nanobot-go/bus"
	"go.uber.org/zap"
)

// FeishuChannel 飞书渠道
// 使用 WebSocket 长连接接收消息，HTTP API 发送消息
type FeishuChannel struct {
	*BaseChannel
	config  *FeishuConfig
	logger  *zap.Logger
	running bool

	// 飞书客户端
	client *lark.Client

	// WebSocket 客户端
	wsClient *ws.Client

	// 后台任务管理
	bgTasks sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc

	// 消息去重缓存
	processedMsgIDs *syncMap

	// 消息反应缓存: chat_id -> reactionInfo
	reactionCache   map[string]*reactionInfo
	reactionMu      sync.RWMutex

	// 事件处理器
	eventHandler *dispatcher.EventDispatcher
}

// reactionInfo 保存消息反应信息
type reactionInfo struct {
	messageID  string
	reactionID string
}

// syncMap 带大小限制的有序去重缓存
type syncMap struct {
	data    map[string]time.Time
	mu      sync.RWMutex
	maxSize int
}

// newSyncMap 创建新的同步缓存
func newSyncMap(maxSize int) *syncMap {
	return &syncMap{
		data:    make(map[string]time.Time),
		maxSize: maxSize,
	}
}

// add 添加元素，自动清理过期数据
func (m *syncMap) add(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.data[key]; exists {
		return false
	}

	m.data[key] = time.Now()

	// 清理过期数据
	if len(m.data) > m.maxSize {
		// 删除最旧的 20% 数据
		toDelete := int(float64(m.maxSize) * 0.2)
		for k := range m.data {
			if toDelete <= 0 {
				break
			}
			delete(m.data, k)
			toDelete--
		}
	}

	return true
}

// FeishuConfig 飞书配置
type FeishuConfig struct {
	AppID             string   `json:"app_id"`
	AppSecret         string   `json:"app_secret"`
	EncryptKey        string   `json:"encrypt_key"`
	VerificationToken string   `json:"verification_token"`
	AllowFrom         []string `json:"allow_from"`
}

// NewFeishuChannel 创建飞书渠道
func NewFeishuChannel(config *FeishuConfig, messageBus *bus.MessageBus, logger *zap.Logger) *FeishuChannel {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &FeishuChannel{
		BaseChannel:     NewBaseChannel("feishu", messageBus),
		config:          config,
		logger:          logger,
		processedMsgIDs: newSyncMap(1000),
		reactionCache:   make(map[string]*reactionInfo),
	}
}

// Start 启动飞书渠道
func (c *FeishuChannel) Start(ctx context.Context) error {
	if c.config.AppID == "" || c.config.AppSecret == "" {
		c.logger.Error("飞书 app_id 和 app_secret 未配置")
		return fmt.Errorf("飞书配置不完整")
	}

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.running = true

	// 创建飞书客户端（用于发送消息）
	c.client = lark.NewClient(c.config.AppID, c.config.AppSecret)

	// 创建事件处理器
	c.eventHandler = dispatcher.NewEventDispatcher(
		c.config.VerificationToken,
		c.config.EncryptKey,
	).OnP2MessageReceiveV1(c.onMessageReceive)

	// 创建 WebSocket 客户端
	c.wsClient = ws.NewClient(c.config.AppID, c.config.AppSecret,
		ws.WithEventHandler(c.eventHandler),
		ws.WithLogLevel(larkcore.LogLevelInfo),
	)

	// 订阅出站消息
	c.SubscribeOutbound(ctx, func(msg *bus.OutboundMessage) {
		if err := c.Send(msg); err != nil {
			c.logger.Error("发送飞书消息失败", zap.Error(err))
		}
	})

	c.logger.Info("飞书渠道已启动",
		zap.String("app_id", c.config.AppID),
	)

	// 启动 WebSocket 客户端（带重连）
	c.bgTasks.Add(1)
	go c.runWebSocketClient()

	return nil
}

// runWebSocketClient 运行 WebSocket 客户端（带重连）
func (c *FeishuChannel) runWebSocketClient() {
	defer c.bgTasks.Done()

	for c.running {
		err := c.wsClient.Start(c.ctx)
		if err != nil {
			if c.ctx.Err() != nil {
				// 上下文被取消，正常退出
				return
			}
			c.logger.Warn("飞书 WebSocket 连接错误", zap.Error(err))
		}

		if !c.running {
			break
		}

		// 等待 5 秒后重连
		select {
		case <-time.After(5 * time.Second):
		case <-c.ctx.Done():
			return
		}
	}
}

// onMessageReceive 处理接收到的消息
func (c *FeishuChannel) onMessageReceive(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	if event == nil || event.Event == nil {
		return nil
	}

	ev := event.Event
	message := ev.Message
	sender := ev.Sender

	if message == nil || sender == nil {
		return nil
	}

	// 消息去重检查
	messageID := *message.MessageId
	if !c.processedMsgIDs.add(messageID) {
		c.logger.Debug("飞书消息重复，忽略", zap.String("message_id", messageID))
		return nil
	}

	// 跳过机器人消息
	if sender.SenderType != nil && *sender.SenderType == "bot" {
		return nil
	}

	// 获取发送者 ID
	senderID := "unknown"
	if sender.SenderId != nil && sender.SenderId.OpenId != nil {
		senderID = *sender.SenderId.OpenId
	}

	// 获取聊天 ID
	chatID := ""
	if message.ChatId != nil {
		chatID = *message.ChatId
	}

	// 获取聊天类型
	chatType := "p2p"
	if message.ChatType != nil {
		chatType = *message.ChatType
	}

	// 获取消息类型
	msgType := ""
	if message.MessageType != nil {
		msgType = *message.MessageType
	}

	// 解析消息内容
	content := c.parseMessageContent(message)
	if content == "" {
		return nil
	}

	// 确定回复目标
	replyTo := chatID
	if chatType == "p2p" {
		replyTo = senderID
	}

	// 添加反应表情表示正在处理，并保存 reaction_id
	// 注意：使用 replyTo 作为 key，因为删除时 msg.ChatID 就是 replyTo
	go c.addReactionAndSave(replyTo, messageID, "OnIt")

	// 检查用户白名单
	if len(c.config.AllowFrom) > 0 {
		allowed := false
		for _, u := range c.config.AllowFrom {
			if senderID == u {
				allowed = true
				break
			}
		}
		if !allowed {
			c.logger.Debug("飞书消息发送者不在白名单中", zap.String("sender", senderID))
			return nil
		}
	}

	c.logger.Info("收到飞书消息",
		zap.String("sender", senderID),
		zap.String("chat_id", chatID),
		zap.String("reply_to", replyTo),
		zap.String("content", content),
	)

	// 发布消息到总线
	c.bus.PublishInbound(&bus.InboundMessage{
		Channel:   "feishu",
		ChatID:    replyTo,
		SenderID:  senderID,
		Content:   content,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"message_id": messageID,
			"chat_type":  chatType,
			"msg_type":   msgType,
			"chat_id":    chatID,
		},
	})

	return nil
}

// addReactionAndSave 添加反应表情并保存到缓存
func (c *FeishuChannel) addReactionAndSave(chatID, messageID, emojiType string) {
	if c.client == nil || chatID == "" {
		return
	}

	req := larkim.NewCreateMessageReactionReqBuilder().
		MessageId(messageID).
		Body(larkim.NewCreateMessageReactionReqBodyBuilder().
			ReactionType(larkim.NewEmojiBuilder().
				EmojiType(emojiType).
				Build()).
			Build()).
		Build()

	resp, err := c.client.Im.V1.MessageReaction.Create(c.ctx, req)
	if err != nil {
		c.logger.Debug("添加飞书反应表情失败", zap.Error(err))
		return
	}

	if !resp.Success() {
		c.logger.Debug("添加飞书反应表情失败",
			zap.Int("code", resp.Code),
			zap.String("msg", resp.Msg),
		)
		return
	}

	// 保存 reaction_id 到缓存
	if resp.Data != nil && resp.Data.ReactionId != nil {
		c.reactionMu.Lock()
		c.reactionCache[chatID] = &reactionInfo{
			messageID:  messageID,
			reactionID: *resp.Data.ReactionId,
		}
		c.reactionMu.Unlock()
		c.logger.Debug("已保存飞书反应表情",
			zap.String("chat_id", chatID),
			zap.String("reaction_id", *resp.Data.ReactionId),
		)
	}
}

// deleteReactionFromCache 从缓存中删除反应表情
func (c *FeishuChannel) deleteReactionFromCache(chatID string) {
	if c.client == nil || chatID == "" {
		return
	}

	c.reactionMu.RLock()
	info, exists := c.reactionCache[chatID]
	c.reactionMu.RUnlock()

	if !exists {
		return
	}

	req := larkim.NewDeleteMessageReactionReqBuilder().
		MessageId(info.messageID).
		ReactionId(info.reactionID).
		Build()

	resp, err := c.client.Im.V1.MessageReaction.Delete(c.ctx, req)
	if err != nil {
		c.logger.Debug("删除飞书反应表情失败", zap.Error(err))
		return
	}

	if !resp.Success() {
		c.logger.Debug("删除飞书反应表情失败",
			zap.Int("code", resp.Code),
			zap.String("msg", resp.Msg),
		)
		return
	}

	// 从缓存中移除
	c.reactionMu.Lock()
	delete(c.reactionCache, chatID)
	c.reactionMu.Unlock()

	c.logger.Debug("已删除飞书反应表情",
		zap.String("chat_id", chatID),
		zap.String("reaction_id", info.reactionID),
	)
}

// parseMessageContent 解析消息内容
func (c *FeishuChannel) parseMessageContent(message *larkim.EventMessage) string {
	if message == nil || message.MessageType == nil {
		return ""
	}

	msgType := *message.MessageType

	switch msgType {
	case "text":
		if message.Content == nil {
			return ""
		}
		// 解析 JSON 格式的文本内容
		var contentMap map[string]interface{}
		if err := json.Unmarshal([]byte(*message.Content), &contentMap); err == nil {
			if text, ok := contentMap["text"].(string); ok {
				return strings.TrimSpace(text)
			}
		}
		return strings.TrimSpace(*message.Content)
	case "image":
		return "[图片]"
	case "audio":
		return "[语音]"
	case "file":
		return "[文件]"
	case "sticker":
		return "[表情]"
	default:
		return fmt.Sprintf("[%s]", msgType)
	}
}

// Stop 停止飞书渠道
func (c *FeishuChannel) Stop() {
	c.running = false
	if c.cancel != nil {
		c.cancel()
	}

	// WebSocket 客户端会在上下文取消后自动关闭
	// 等待后台任务完成
	c.bgTasks.Wait()

	c.logger.Info("飞书渠道已停止")
}

// Send 发送消息
func (c *FeishuChannel) Send(msg *bus.OutboundMessage) error {
	if c.client == nil {
		return fmt.Errorf("飞书客户端未初始化")
	}

	// 确定 receive_id_type
	receiveIDType := "open_id"
	if strings.HasPrefix(msg.ChatID, "oc_") {
		receiveIDType = "chat_id"
	}

	// 构建卡片消息（支持 Markdown 和表格）
	card := c.buildCard(msg.Content)
	cardJSON, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("序列化卡片失败: %w", err)
	}

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(receiveIDType).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(msg.ChatID).
			MsgType("interactive").
			Content(string(cardJSON)).
			Build()).
		Build()

	resp, err := c.client.Im.V1.Message.Create(c.ctx, req)
	if err != nil {
		return fmt.Errorf("发送飞书消息失败: %w", err)
	}

	if !resp.Success() {
		return fmt.Errorf("发送飞书消息失败: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	c.logger.Debug("飞书消息已发送",
		zap.String("chat_id", msg.ChatID),
	)

	// 发送成功后，删除"正在处理"反应表情
	go c.deleteReactionFromCache(msg.ChatID)

	return nil
}

// buildCard 构建飞书卡片消息
func (c *FeishuChannel) buildCard(content string) map[string]interface{} {
	elements := c.buildCardElements(content)
	return map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"elements": elements,
	}
}

// tableRegex 匹配 Markdown 表格的正则表达式
var tableRegex = regexp.MustCompile(`(?m)((?:^[ \t]*\|.+\|[ \t]*\n)(?:^[ \t]*\|[-:\s|]+\|[ \t]*\n)(?:^[ \t]*\|.+\|[ \t]*\n?)+)`)

// buildCardElements 构建卡片元素（支持 Markdown 和表格）
func (c *FeishuChannel) buildCardElements(content string) []interface{} {
	var elements []interface{}
	lastEnd := 0

	// 查找所有表格
	matches := tableRegex.FindAllStringIndex(content, -1)
	for _, match := range matches {
		// 表格前的 Markdown 内容
		before := strings.TrimSpace(content[lastEnd:match[0]])
		if before != "" {
			elements = append(elements, map[string]interface{}{
				"tag":     "markdown",
				"content": before,
			})
		}

		// 解析表格
		table := c.parseMarkdownTable(content[match[0]:match[1]])
		if table != nil {
			elements = append(elements, table)
		} else {
			elements = append(elements, map[string]interface{}{
				"tag":     "markdown",
				"content": content[match[0]:match[1]],
			})
		}

		lastEnd = match[1]
	}

	// 剩余内容
	remaining := strings.TrimSpace(content[lastEnd:])
	if remaining != "" {
		elements = append(elements, map[string]interface{}{
			"tag":     "markdown",
			"content": remaining,
		})
	}

	if len(elements) == 0 {
		elements = append(elements, map[string]interface{}{
			"tag":     "markdown",
			"content": content,
		})
	}

	return elements
}

// parseMarkdownTable 解析 Markdown 表格为飞书表格元素
func (c *FeishuChannel) parseMarkdownTable(tableText string) interface{} {
	lines := strings.Split(strings.TrimSpace(tableText), "\n")
	if len(lines) < 3 {
		return nil
	}

	// 过滤空行
	var validLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			validLines = append(validLines, line)
		}
	}

	if len(validLines) < 3 {
		return nil
	}

	// 解析表头
	headers := c.splitTableRow(validLines[0])
	if len(headers) == 0 {
		return nil
	}

	// 跳过分隔行（第2行）
	// 解析数据行
	var rows []map[string]interface{}
	for i := 2; i < len(validLines); i++ {
		cells := c.splitTableRow(validLines[i])
		row := make(map[string]interface{})
		for j := range headers {
			key := fmt.Sprintf("c%d", j)
			if j < len(cells) {
				row[key] = cells[j]
			} else {
				row[key] = ""
			}
		}
		rows = append(rows, row)
	}

	// 构建列定义
	var columns []map[string]interface{}
	for i, h := range headers {
		columns = append(columns, map[string]interface{}{
			"tag":          "column",
			"name":         fmt.Sprintf("c%d", i),
			"display_name": h,
			"width":        "auto",
		})
	}

	return map[string]interface{}{
		"tag":       "table",
		"page_size": len(rows) + 1,
		"columns":   columns,
		"rows":      rows,
	}
}

// splitTableRow 分割表格行
func (c *FeishuChannel) splitTableRow(row string) []string {
	row = strings.TrimSpace(row)
	row = strings.Trim(row, "|")
	cells := strings.Split(row, "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}
