package websocket

import (
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	// 写入超时
	writeWait = 10 * time.Second
	// 读取超时（用于心跳检测）
	pongWait = 60 * time.Second
	// 心跳间隔
	pingPeriod = (pongWait * 9) / 10
	// 消息缓冲区大小
	maxMessageSize = 10240 // 10KB
)

// Connection 单个 WebSocket 连接封装
type Connection struct {
	id          string
	conn        *websocket.Conn
	userCode    string
	channelCode string
	sessionID   string
	send        chan []byte
	logger      *zap.Logger

	// 生命周期管理
	closeOnce sync.Once
	closeChan chan struct{}
	closed    bool
}

// NewConnection 创建新的连接封装
func NewConnection(conn *websocket.Conn, userCode, channelCode string, logger *zap.Logger) *Connection {
	return &Connection{
		id:          generateConnID(),
		conn:        conn,
		userCode:    userCode,
		channelCode: channelCode,
		send:        make(chan []byte, 256),
		closeChan:   make(chan struct{}),
		logger:      logger.With(zap.String("conn_id", generateConnID())),
	}
}

// ID 返回连接 ID
func (c *Connection) ID() string {
	return c.id
}

// UserCode 返回用户编码
func (c *Connection) UserCode() string {
	return c.userCode
}

// ChannelCode 返回渠道编码
func (c *Connection) ChannelCode() string {
	return c.channelCode
}

// SessionID 返回当前会话 ID
func (c *Connection) SessionID() string {
	return c.sessionID
}

// SetSessionID 设置会话 ID
func (c *Connection) SetSessionID(sessionID string) {
	c.sessionID = sessionID
}

// Send 发送消息到客户端（异步）
func (c *Connection) Send(data []byte) bool {
	select {
	case c.send <- data:
		return true
	case <-c.closeChan:
		return false
	}
}

// Close 关闭连接
func (c *Connection) Close() {
	c.closeOnce.Do(func() {
		c.closed = true
		close(c.closeChan)
		close(c.send)
		c.conn.Close()
		c.logger.Debug("WebSocket 连接已关闭")
	})
}

// IsClosed 检查连接是否已关闭
func (c *Connection) IsClosed() bool {
	return c.closed
}

// ReadPump 读取循环（在 goroutine 中运行）
func (c *Connection) ReadPump(onMessage func(*Connection, []byte), onClose func(*Connection)) {
	defer func() {
		onClose(c)
		c.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Warn("WebSocket 异常关闭", zap.Error(err))
			}
			break
		}

		onMessage(c, message)
	}
}

// WritePump 写入循环（在 goroutine 中运行）
func (c *Connection) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// 通道已关闭
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.logger.Warn("WebSocket 写入失败", zap.Error(err))
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.Warn("WebSocket 心跳失败", zap.Error(err))
				return
			}

		case <-c.closeChan:
			return
		}
	}
}

// generateConnID 生成连接 ID
var connIDCounter uint64
var connIDMutex sync.Mutex

func generateConnID() string {
	connIDMutex.Lock()
	defer connIDMutex.Unlock()
	connIDCounter++
	return fmt.Sprintf("ws_%d_%d", time.Now().UnixMilli(), connIDCounter)
}
