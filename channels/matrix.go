package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/weibaohui/nanobot-go/bus"
	"github.com/yuin/goldmark"
	"go.uber.org/zap"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

// MatrixChannel Matrix 渠道
// 使用 mautrix/go SDK 连接 Matrix 服务器
type MatrixChannel struct {
	*BaseChannel
	config  *MatrixConfig
	logger  *zap.Logger
	running bool

	// Matrix 客户端
	client *mautrix.Client

	// 同步器
	syncer *mautrix.DefaultSyncer

	// 存储
	store *FileSyncStore

	// 后台任务管理
	bgTasks sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc

	// 忽略自己发送的消息
	botUserID id.UserID

	typingMu     sync.Mutex
	typingCancel map[id.RoomID]context.CancelFunc
}

// MatrixConfig Matrix 配置
type MatrixConfig struct {
	Homeserver string   `json:"homeserver"` // Matrix 服务器地址，如 https://matrix.example.com
	UserID     string   `json:"userId"`     // 用户 ID，如 @nanobot:example.com
	Token      string   `json:"token"`      // 访问令牌
	AllowFrom  []string `json:"allowFrom"`  // 允许的用户白名单
	DataDir    string   `json:"dataDir"`    // 数据存储目录，用于持久化同步状态
}

// NewMatrixChannel 创建 Matrix 渠道
func NewMatrixChannel(config *MatrixConfig, messageBus *bus.MessageBus, logger *zap.Logger) *MatrixChannel {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &MatrixChannel{
		BaseChannel:  NewBaseChannel("matrix", messageBus),
		config:       config,
		logger:       logger,
		typingCancel: make(map[id.RoomID]context.CancelFunc),
	}
}

// Start 启动 Matrix 渠道
func (c *MatrixChannel) Start(ctx context.Context) error {
	if c.config.Homeserver == "" || c.config.UserID == "" || c.config.Token == "" {
		c.logger.Error("Matrix 配置不完整",
			zap.Bool("has_homeserver", c.config.Homeserver != ""),
			zap.Bool("has_userid", c.config.UserID != ""),
			zap.Bool("has_token", c.config.Token != ""),
		)
		return fmt.Errorf("Matrix 配置不完整")
	}

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.running = true

	// 解析用户 ID
	userID := id.UserID(c.config.UserID)
	c.botUserID = userID

	// 创建 Matrix 客户端
	var err error
	c.client, err = mautrix.NewClient(c.config.Homeserver, userID, c.config.Token)
	if err != nil {
		c.logger.Error("创建 Matrix 客户端失败", zap.Error(err))
		return fmt.Errorf("创建 Matrix 客户端失败: %w", err)
	}

	// 创建持久化存储
	storePath := c.getStorePath()
	c.store, err = NewFileSyncStore(storePath, userID)
	if err != nil {
		c.logger.Error("创建 Matrix 存储失败", zap.Error(err))
		return fmt.Errorf("创建 Matrix 存储失败: %w", err)
	}
	c.client.Store = c.store

	// 创建同步器
	c.syncer = mautrix.NewDefaultSyncer()
	c.syncer.ParseEventContent = true
	c.client.Syncer = c.syncer

	// 设置过滤器，只获取新消息，不同步历史状态和成员
	// 这样可以大幅减少启动时的数据量
	c.syncer.FilterJSON = &mautrix.Filter{
		Room: &mautrix.RoomFilter{
			// 时间线过滤器：只获取新消息，不获取历史
			Timeline: &mautrix.FilterPart{
				Limit: 0, // 限制历史消息数量
				Types: []event.Type{
					event.EventMessage,
					event.EventEncrypted,
				},
			},
			// 状态过滤器：不同步房间状态（成员列表、房间名等）
			State: &mautrix.FilterPart{
				Limit:           0,    // 不获取状态事件
				LazyLoadMembers: true, // 只在需要时懒加载成员信息
			},
			// ephemeral 事件（打字状态、已读标记等）
			Ephemeral: &mautrix.FilterPart{
				Limit: 0, // 不获取
			},
			// 账户数据
			AccountData: &mautrix.FilterPart{
				Limit: 0, // 不获取
			},
		},
		// 全局账户数据
		AccountData: &mautrix.FilterPart{
			Limit: 0, // 不获取
		},
		// 在线状态
		Presence: &mautrix.FilterPart{
			Limit: 0, // 不获取
		},
	}

	// 注册忽略旧消息的处理器（机器人只处理启动后收到的消息）
	c.syncer.OnSync(c.client.DontProcessOldEvents)

	// 注册消息事件处理器
	c.syncer.OnEventType(event.EventMessage, c.onMessage)
	c.syncer.OnEventType(event.EventEncrypted, c.onEncryptedMessage)

	// 订阅出站消息
	c.SubscribeOutbound(ctx, func(msg *bus.OutboundMessage) {
		if err := c.Send(msg); err != nil {
			c.logger.Error("发送 Matrix 消息失败", zap.Error(err))
		}
	})

	// 订阅心跳消息
	c.bus.SubscribeOutbound("heartbeat", func(msg *bus.OutboundMessage) error {
		if err := c.Send(msg); err != nil {
			c.logger.Error("发送 Matrix 心跳消息失败", zap.Error(err))
		}
		return nil
	})

	c.logger.Info("Matrix 渠道已启动",
		zap.String("homeserver", c.config.Homeserver),
		zap.String("user_id", string(userID)),
		zap.String("store_path", storePath),
	)

	// 启动同步
	c.bgTasks.Add(1)
	go c.runSync()

	return nil
}

// getStorePath 获取存储路径
func (c *MatrixChannel) getStorePath() string {
	if c.config.DataDir != "" {
		return filepath.Join(c.config.DataDir, "matrix_sync.json")
	}
	// 默认使用 ~/.nanobot/matrix_sync.json
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".nanobot", "matrix_sync.json")
}

// runSync 运行 Matrix 同步
func (c *MatrixChannel) runSync() {
	defer c.bgTasks.Done()

	for c.running {
		err := c.client.SyncWithContext(c.ctx)
		if err != nil {
			if c.ctx.Err() != nil {
				// 上下文被取消，正常退出
				return
			}
			c.logger.Warn("Matrix 同步错误", zap.Error(err))
		}

		if !c.running {
			break
		}

		// 等待 3 秒后重连
		select {
		case <-time.After(3 * time.Second):
		case <-c.ctx.Done():
			return
		}
	}
}

// startTypingIndicator 启动房间 typing 状态刷新
func (c *MatrixChannel) startTypingIndicator(roomID id.RoomID) {
	if c.client == nil {
		return
	}

	c.typingMu.Lock()
	if c.typingCancel == nil {
		c.typingCancel = make(map[id.RoomID]context.CancelFunc)
	}
	if cancel, ok := c.typingCancel[roomID]; ok {
		cancel()
	}
	typingCtx, cancel := context.WithCancel(context.Background())
	c.typingCancel[roomID] = cancel
	c.typingMu.Unlock()

	c.bgTasks.Add(1)
	go func() {
		defer c.bgTasks.Done()

		typingTimeout := 30 * time.Second
		refreshInterval := 20 * time.Second
		c.sendTypingStatus(roomID, true, typingTimeout)

		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-typingCtx.Done():
				c.sendTypingStatus(roomID, false, 0)
				return
			case <-ticker.C:
				c.sendTypingStatus(roomID, true, typingTimeout)
			}
		}
	}()
}

// stopTypingIndicator 停止房间 typing 状态刷新
func (c *MatrixChannel) stopTypingIndicator(roomID id.RoomID) {
	c.typingMu.Lock()
	cancel, ok := c.typingCancel[roomID]
	if ok {
		delete(c.typingCancel, roomID)
	}
	c.typingMu.Unlock()

	if ok {
		cancel()
	}
}

// stopAllTypingIndicators 停止所有房间 typing 状态刷新
func (c *MatrixChannel) stopAllTypingIndicators() {
	c.typingMu.Lock()
	cancels := make([]context.CancelFunc, 0, len(c.typingCancel))
	for _, cancel := range c.typingCancel {
		cancels = append(cancels, cancel)
	}
	c.typingCancel = make(map[id.RoomID]context.CancelFunc)
	c.typingMu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
}

// sendTypingStatus 发送 typing 状态
func (c *MatrixChannel) sendTypingStatus(roomID id.RoomID, typing bool, timeout time.Duration) {
	if c.client == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := c.client.UserTyping(ctx, roomID, typing, timeout); err != nil {
		c.logger.Debug("发送 Matrix typing 状态失败",
			zap.String("room_id", string(roomID)),
			zap.Error(err),
		)
	}
}

// onMessage 处理消息事件
func (c *MatrixChannel) onMessage(ctx context.Context, evt *event.Event) {
	// 忽略自己发送的消息
	if evt.Sender == c.botUserID {
		return
	}

	// 获取消息内容
	content, ok := evt.Content.Parsed.(*event.MessageEventContent)
	if !ok {
		c.logger.Debug("无法解析消息内容")
		return
	}

	// 提取文本内容
	text := content.Body
	if text == "" {
		return
	}

	// 检查用户白名单
	if len(c.config.AllowFrom) > 0 {
		allowed := false
		for _, u := range c.config.AllowFrom {
			if string(evt.Sender) == u {
				allowed = true
				break
			}
		}
		if !allowed {
			c.logger.Debug("消息发送者不在白名单中", zap.String("sender", string(evt.Sender)))
			return
		}
	}

	// 判断是否是群组消息
	// Matrix 房间 ID 以 ! 开头，群组和私聊的判断需要额外查询
	// 这里简化处理：所有房间都按群组处理，除非是 DM（需要额外状态存储）
	chatType := "group"

	c.logger.Info("收到 Matrix 消息",
		zap.String("sender", string(evt.Sender)),
		zap.String("room", string(evt.RoomID)),
		zap.String("content", text),
	)

	c.startTypingIndicator(id.RoomID(evt.RoomID))

	// 发布消息到总线
	c.bus.PublishInbound(&bus.InboundMessage{
		Channel:   "matrix",
		ChatID:    string(evt.RoomID),
		SenderID:  string(evt.Sender),
		Content:   text,
		Timestamp: time.UnixMilli(evt.Timestamp),
		Metadata: map[string]any{
			"event_id":  string(evt.ID),
			"room_id":   string(evt.RoomID),
			"sender":    string(evt.Sender),
			"chat_type": chatType,
			"msg_type":  string(content.MsgType),
		},
	})
}

// onEncryptedMessage 处理加密消息事件
func (c *MatrixChannel) onEncryptedMessage(ctx context.Context, evt *event.Event) {
	// 加密消息需要额外的 Olm/Megolm 加密库支持
	// 这里记录日志并跳过
	c.logger.Debug("收到加密消息，但暂不支持解密",
		zap.String("sender", string(evt.Sender)),
		zap.String("room", string(evt.RoomID)),
	)
}

// Stop 停止 Matrix 渠道
func (c *MatrixChannel) Stop() {
	c.running = false

	// 停止同步
	if c.client != nil {
		c.client.StopSync()
	}

	c.stopAllTypingIndicators()

	if c.cancel != nil {
		c.cancel()
	}

	// 保存同步状态
	if c.store != nil {
		if err := c.store.Save(); err != nil {
			c.logger.Warn("保存 Matrix 同步状态失败", zap.Error(err))
		}
	}

	// 等待后台任务完成
	c.bgTasks.Wait()

	c.logger.Info("Matrix 渠道已停止")
}

// Send 发送消息
// 自动将 Markdown 内容转换为 HTML 格式发送
func (c *MatrixChannel) Send(msg *bus.OutboundMessage) error {
	if c.client == nil {
		return fmt.Errorf("Matrix 客户端未初始化")
	}

	roomID := id.RoomID(msg.ChatID)
	defer c.stopTypingIndicator(roomID)

	// 将 Markdown 转换为 HTML
	htmlContent := markdownToHTML(msg.Content)

	content := &event.MessageEventContent{
		MsgType:       event.MsgText,
		Body:          msg.Content,
		Format:        event.FormatHTML,
		FormattedBody: htmlContent,
	}

	_, err := c.client.SendMessageEvent(c.ctx, roomID, event.EventMessage, content)
	if err != nil {
		return fmt.Errorf("发送消息失败: %w", err)
	}

	c.logger.Debug("Matrix 消息已发送",
		zap.String("room_id", string(roomID)),
	)
	return nil
}

// markdownToHTML 将 Markdown 转换为 HTML
func markdownToHTML(md string) string {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(md), &buf); err != nil {
		return md
	}
	return buf.String()
}

// SendNotice 发送通知消息（不会产生通知提醒）
func (c *MatrixChannel) SendNotice(msg *bus.OutboundMessage) error {
	if c.client == nil {
		return fmt.Errorf("Matrix 客户端未初始化")
	}

	roomID := id.RoomID(msg.ChatID)
	defer c.stopTypingIndicator(roomID)

	// 发送通知消息
	_, err := c.client.SendNotice(c.ctx, roomID, msg.Content)
	if err != nil {
		return fmt.Errorf("发送通知失败: %w", err)
	}

	c.logger.Debug("Matrix 通知已发送",
		zap.String("room_id", string(roomID)),
	)
	return nil
}

// SendReply 发送回复消息
func (c *MatrixChannel) SendReply(msg *bus.OutboundMessage, replyToEventID id.EventID) error {
	if c.client == nil {
		return fmt.Errorf("Matrix 客户端未初始化")
	}

	roomID := id.RoomID(msg.ChatID)
	defer c.stopTypingIndicator(roomID)

	// 构建包含回复的消息内容
	content := &event.MessageEventContent{
		MsgType: event.MsgText,
		Body:    msg.Content,
		RelatesTo: &event.RelatesTo{
			InReplyTo: &event.InReplyTo{
				EventID: replyToEventID,
			},
		},
	}

	_, err := c.client.SendMessageEvent(c.ctx, roomID, event.EventMessage, content)
	if err != nil {
		return fmt.Errorf("发送回复失败: %w", err)
	}

	c.logger.Debug("Matrix 回复已发送",
		zap.String("room_id", string(roomID)),
		zap.String("reply_to", string(replyToEventID)),
	)
	return nil
}

// SendHTML 发送 HTML 格式消息
// plainText 是纯文本回退内容，htmlContent 是 HTML 格式内容
func (c *MatrixChannel) SendHTML(msg *bus.OutboundMessage, htmlContent string) error {
	if c.client == nil {
		return fmt.Errorf("Matrix 客户端未初始化")
	}

	roomID := id.RoomID(msg.ChatID)
	defer c.stopTypingIndicator(roomID)

	content := &event.MessageEventContent{
		MsgType:       event.MsgText,
		Body:          msg.Content,
		Format:        event.FormatHTML,
		FormattedBody: htmlContent,
	}

	_, err := c.client.SendMessageEvent(c.ctx, roomID, event.EventMessage, content)
	if err != nil {
		return fmt.Errorf("发送 HTML 消息失败: %w", err)
	}

	c.logger.Debug("Matrix HTML 消息已发送",
		zap.String("room_id", string(roomID)),
	)
	return nil
}

// JoinRoom 加入房间
func (c *MatrixChannel) JoinRoom(roomIDOrAlias string, serverNames []string) error {
	if c.client == nil {
		return fmt.Errorf("Matrix 客户端未初始化")
	}

	_, err := c.client.JoinRoom(c.ctx, roomIDOrAlias, &mautrix.ReqJoinRoom{
		Via: serverNames,
	})
	if err != nil {
		return fmt.Errorf("加入房间失败: %w", err)
	}

	c.logger.Info("已加入 Matrix 房间", zap.String("room", roomIDOrAlias))
	return nil
}

// LeaveRoom 离开房间
func (c *MatrixChannel) LeaveRoom(roomID id.RoomID) error {
	if c.client == nil {
		return fmt.Errorf("Matrix 客户端未初始化")
	}

	_, err := c.client.LeaveRoom(c.ctx, roomID)
	if err != nil {
		return fmt.Errorf("离开房间失败: %w", err)
	}

	c.logger.Info("已离开 Matrix 房间", zap.String("room_id", string(roomID)))
	return nil
}

// GetRoomMembers 获取房间成员
func (c *MatrixChannel) GetRoomMembers(roomID id.RoomID) ([]id.UserID, error) {
	if c.client == nil {
		return nil, fmt.Errorf("Matrix 客户端未初始化")
	}

	resp, err := c.client.JoinedMembers(c.ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("获取房间成员失败: %w", err)
	}

	var members []id.UserID
	for userID := range resp.Joined {
		members = append(members, userID)
	}

	return members, nil
}

// GetDisplayName 获取用户显示名称
func (c *MatrixChannel) GetDisplayName(userID id.UserID) (string, error) {
	if c.client == nil {
		return "", fmt.Errorf("Matrix 客户端未初始化")
	}

	resp, err := c.client.GetDisplayName(c.ctx, userID)
	if err != nil {
		return "", fmt.Errorf("获取显示名称失败: %w", err)
	}

	return resp.DisplayName, nil
}

// SetPresence 设置在线状态
func (c *MatrixChannel) SetPresence(presence event.Presence, statusMsg string) error {
	if c.client == nil {
		return fmt.Errorf("Matrix 客户端未初始化")
	}

	err := c.client.SetPresence(c.ctx, mautrix.ReqPresence{
		Presence:  presence,
		StatusMsg: statusMsg,
	})
	if err != nil {
		return fmt.Errorf("设置在线状态失败: %w", err)
	}

	c.logger.Debug("Matrix 在线状态已设置",
		zap.String("presence", string(presence)),
	)
	return nil
}

// ========== FileSyncStore 文件持久化存储 ==========

// FileSyncStore 基于文件的同步状态存储
type FileSyncStore struct {
	filePath string
	userID   id.UserID

	mu        sync.RWMutex
	filterID  string
	nextBatch string
}

// syncData 用于 JSON 序列化的数据结构
type syncData struct {
	FilterID  string `json:"filter_id"`
	NextBatch string `json:"next_batch"`
}

// NewFileSyncStore 创建文件同步存储
func NewFileSyncStore(filePath string, userID id.UserID) (*FileSyncStore, error) {
	store := &FileSyncStore{
		filePath: filePath,
		userID:   userID,
	}

	// 尝试加载现有数据
	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return store, nil
}

// load 从文件加载数据
func (s *FileSyncStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var sd syncData
	if err := json.Unmarshal(data, &sd); err != nil {
		return err
	}

	s.mu.Lock()
	s.filterID = sd.FilterID
	s.nextBatch = sd.NextBatch
	s.mu.Unlock()

	return nil
}

// Save 保存数据到文件
func (s *FileSyncStore) Save() error {
	s.mu.RLock()
	sd := syncData{
		FilterID:  s.filterID,
		NextBatch: s.nextBatch,
	}
	s.mu.RUnlock()

	data, err := json.MarshalIndent(sd, "", "  ")
	if err != nil {
		return err
	}

	// 确保目录存在
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
}

// LoadFilterID 实现 mautrix.SyncStore 接口
func (s *FileSyncStore) LoadFilterID(ctx context.Context, userID id.UserID) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.filterID, nil
}

// SaveFilterID 实现 mautrix.SyncStore 接口
func (s *FileSyncStore) SaveFilterID(ctx context.Context, userID id.UserID, filterID string) error {
	s.mu.Lock()
	s.filterID = filterID
	s.mu.Unlock()
	return s.Save()
}

// LoadNextBatch 实现 mautrix.SyncStore 接口
func (s *FileSyncStore) LoadNextBatch(ctx context.Context, userID id.UserID) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nextBatch, nil
}

// SaveNextBatch 实现 mautrix.SyncStore 接口
func (s *FileSyncStore) SaveNextBatch(ctx context.Context, userID id.UserID, nextBatchToken string) error {
	s.mu.Lock()
	s.nextBatch = nextBatchToken
	s.mu.Unlock()
	return s.Save()
}
