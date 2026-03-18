package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/weibaohui/nanobot-go/pkg/bus"
	"go.uber.org/zap"
)

// upgrader WebSocket 升级器
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// 允许所有来源，生产环境应限制为特定域名
		return true
	},
}

// Channel WebSocket 渠道实现
type Channel struct {
	name          string
	config        *Config
	bus           *bus.MessageBus
	logger        *zap.Logger
	connManager   *ConnectionManager

	// 生命周期
	ctx           context.Context
	cancel        context.CancelFunc
	running       bool
	wg            sync.WaitGroup

	// 存储连接与用户代码的映射，用于处理出站消息
	connUserMap   map[string]string // connID -> userCode
	mu            sync.RWMutex
}

// NewChannel 创建 WebSocket 渠道
func NewChannel(config *Config, messageBus *bus.MessageBus, logger *zap.Logger) *Channel {
	if logger == nil {
		logger = zap.NewNop()
	}

	channelName := "websocket"
	if config.ChannelCode != "" {
		channelName = fmt.Sprintf("websocket_%s", config.ChannelCode)
	}

	return &Channel{
		name:        channelName,
		config:      config,
		bus:         messageBus,
		logger:      logger.With(zap.String("channel", channelName)),
		connManager: NewConnectionManager(config.ChannelCode, logger),
		connUserMap: make(map[string]string),
	}
}

// Name 返回渠道名称
func (c *Channel) Name() string {
	return c.name
}

// Config 返回渠道配置
func (c *Channel) Config() *Config {
	return c.config
}

// Start 启动 WebSocket 渠道
func (c *Channel) Start(ctx context.Context) error {
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.running = true

	// 订阅出站消息
	c.subscribeOutbound()

	c.logger.Info("WebSocket 渠道已启动",
		zap.String("channel_code", c.config.ChannelCode),
	)

	return nil
}

// Stop 停止 WebSocket 渠道
func (c *Channel) Stop() {
	c.running = false
	if c.cancel != nil {
		c.cancel()
	}

	// 关闭所有连接
	c.connManager.CloseAll()

	// 等待所有 goroutine 完成
	c.wg.Wait()

	c.logger.Info("WebSocket 渠道已停止")
}

// HandleWebSocket 处理 WebSocket 连接升级
func (c *Channel) HandleWebSocket(w http.ResponseWriter, r *http.Request, userCode string) {
	// 升级 HTTP 连接为 WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		c.logger.Warn("WebSocket 升级失败", zap.Error(err))
		return
	}

	// 创建连接封装
	connection := NewConnection(conn, userCode, c.config.ChannelCode, c.logger)

	// 记录连接与用户的关系
	c.mu.Lock()
	c.connUserMap[connection.ID()] = userCode
	c.mu.Unlock()

	// 添加到管理器
	c.connManager.Add(connection)

	// 启动读写 goroutine
	c.wg.Add(2)
	go func() {
		defer c.wg.Done()
		connection.ReadPump(c.onMessage, c.onClose)
	}()
	go func() {
		defer c.wg.Done()
		connection.WritePump()
	}()

	// 发送连接成功消息
	sysMsg := NewSystemMessage("connected", "", "连接成功")
	data, _ := json.Marshal(sysMsg)
	connection.Send(data)

	c.logger.Info("新的 WebSocket 连接",
		zap.String("conn_id", connection.ID()),
		zap.String("user_code", userCode),
	)
}

// HandleWebSocketWithGin 适配 Gin 框架的 WebSocket 处理器
func (c *Channel) HandleWebSocketWithGin(ctx *gin.Context, userCode string) {
	c.HandleWebSocket(ctx.Writer, ctx.Request, userCode)
}

// onMessage 处理收到的消息
func (c *Channel) onMessage(conn *Connection, data []byte) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		c.logger.Warn("解析消息失败", zap.Error(err), zap.String("data", string(data)))
		c.sendError(conn, "INVALID_MESSAGE", "消息格式错误")
		return
	}

	// 处理心跳
	if msg.Type == MessageTypePing {
		pong := NewPongMessage()
		pongData, _ := json.Marshal(pong)
		conn.Send(pongData)
		return
	}

	// 处理用户消息
	if msg.Type == MessageTypeMessage {
		c.handleUserMessage(conn, &msg)
		return
	}

	c.logger.Debug("收到未知类型消息", zap.String("type", string(msg.Type)))
}

// onClose 处理连接关闭
func (c *Channel) onClose(conn *Connection) {
	// 从用户映射中移除
	c.mu.Lock()
	delete(c.connUserMap, conn.ID())
	c.mu.Unlock()

	// 从管理器中移除
	c.connManager.Remove(conn.ID())
}

// handleUserMessage 处理用户发送的消息
func (c *Channel) handleUserMessage(conn *Connection, msg *Message) {
	// 解析消息
	var payload MessagePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendError(conn, "INVALID_PAYLOAD", "消息负载格式错误")
		return
	}

	// 使用连接的用户代码（从 JWT 获取）
	payload.UserCode = conn.UserCode()

	// 如果前端没有传递 session_id，使用连接已有的或生成新的
	if payload.SessionID == "" {
		payload.SessionID = conn.SessionID()
		if payload.SessionID == "" {
			// 生成新的 session ID：使用连接 ID 作为唯一标识
			payload.SessionID = conn.ID()
		}
	}

	// 更新消息的 payload
	payloadBytes, _ := json.Marshal(payload)
	msg.Payload = payloadBytes

	// 转换为入站消息
	inbound, err := msg.ToInboundMessage(c.config.ChannelCode)
	if err != nil {
		c.logger.Warn("转换入站消息失败", zap.Error(err))
		c.sendError(conn, "INTERNAL_ERROR", "消息处理失败")
		return
	}

	// 保存会话 ID 到连接（用于后续消息路由）
	conn.SetSessionID(inbound.ChatID)

	// 添加到元数据，用于后续路由回此连接
	inbound.Metadata["conn_id"] = conn.ID()
	inbound.Metadata["user_code"] = conn.UserCode()
	inbound.Metadata["channel_id"] = c.config.ChannelID

	// 发布到消息总线
	c.bus.PublishInbound(inbound)

	c.logger.Debug("消息已发布到总线",
		zap.String("user_code", inbound.SenderID),
		zap.String("session_id", inbound.ChatID),
	)
}

// subscribeOutbound 订阅出站消息
func (c *Channel) subscribeOutbound() {
	// 订阅流式消息（用于实时响应）
	// 注意：使用 ChannelCode 而不是 c.name，因为消息总线分发时使用的是 ChannelCode
	c.bus.SubscribeStream(c.config.ChannelCode, func(chunk *bus.StreamChunk) error {
		// 检查消息是否属于当前渠道
		if chunk.Channel != c.config.ChannelCode {
			return nil
		}

		// 转换为 WebSocket 消息
		wsMsg := FromStreamChunk(chunk)
		data, err := json.Marshal(wsMsg)
		if err != nil {
			c.logger.Error("序列化流式消息失败", zap.Error(err))
			return err
		}

		// 流式消息目前缺少 user_code，需要额外存储映射
		// 临时方案：尝试用 ChatID 查找，如果失败则广播
		sent := c.connManager.SendToUser(chunk.ChatID, data)
		if sent == 0 {
			// 可能是 ChatID 与 userCode 不匹配，广播给所有连接
			c.logger.Debug("流式消息未找到对应用户，尝试广播",
				zap.String("chat_id", chunk.ChatID),
			)
			c.connManager.Broadcast(data)
		}
		return nil
	})

	// 订阅普通出站消息
	// 注意：使用 ChannelCode 而不是 c.name，因为消息总线分发时使用的是 ChannelCode
	c.bus.SubscribeOutbound(c.config.ChannelCode, func(msg *bus.OutboundMessage) error {
		// 检查消息是否属于当前渠道
		if msg.Channel != c.config.ChannelCode {
			return nil
		}

		// 转换为 WebSocket 消息
		wsMsg := FromOutboundMessage(msg)
		data, err := json.Marshal(wsMsg)
		if err != nil {
			c.logger.Error("序列化消息失败", zap.Error(err))
			return err
		}

		// 从 metadata 获取 user_code，用于路由到正确的连接
		userCode := ""
		if msg.Metadata != nil {
			if code, ok := msg.Metadata["user_code"].(string); ok {
				userCode = code
			}
		}

		// 发送到目标用户
		if userCode != "" {
			sent := c.connManager.SendToUser(userCode, data)
			c.logger.Debug("发送出站消息",
				zap.String("user_code", userCode),
				zap.Int("sent_count", sent),
			)
		} else {
			// 没有 user_code，广播给所有连接
			c.logger.Warn("出站消息缺少 user_code，将广播给所有连接",
				zap.String("chat_id", msg.ChatID),
			)
			c.connManager.Broadcast(data)
		}

		return nil
	})
}

// sendError 发送错误消息到指定连接
func (c *Channel) sendError(conn *Connection, code, message string) {
	errMsg := NewErrorMessage(code, message)
	data, _ := json.Marshal(errMsg)
	conn.Send(data)
}

// GetConnectionManager 获取连接管理器
func (c *Channel) GetConnectionManager() *ConnectionManager {
	return c.connManager
}

// IsRunning 检查渠道是否运行中
func (c *Channel) IsRunning() bool {
	return c.running
}
