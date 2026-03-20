package api

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// TaskWebSocketHandler Task WebSocket 处理器
// 提供独立的 WebSocket 连接用于 Task 实时通知，不依赖于 channel_code
type TaskWebSocketHandler struct {
	logger    *zap.Logger
	upgrader  websocket.Upgrader
	clients   map[string]*TaskWebSocketClient // userCode -> client
	clientsMu sync.RWMutex
}

// TaskWebSocketClient Task WebSocket 客户端
type TaskWebSocketClient struct {
	userCode string
	conn     *websocket.Conn
	send     chan []byte
}

// NewTaskWebSocketHandler 创建 Task WebSocket 处理器
func NewTaskWebSocketHandler(logger *zap.Logger) *TaskWebSocketHandler {
	return &TaskWebSocketHandler{
		logger: logger.With(zap.String("component", "task_websocket")),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源（开发环境）
			},
		},
		clients: make(map[string]*TaskWebSocketClient),
	}
}

// Handle 处理 Task WebSocket 连接请求
// GET /ws/tasks?token=xxx
func (h *TaskWebSocketHandler) Handle(c *gin.Context) {
	// 从 URL 参数获取 token
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少 token 参数"})
		return
	}

	claims, err := ParseToken(token)
	if err != nil {
		h.logger.Warn("解析 token 失败", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的 token"})
		return
	}

	userCode := claims.Username
	if userCode == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无法获取用户信息"})
		return
	}

	h.logger.Info("Task WebSocket 连接请求",
		zap.String("user_code", userCode),
	)

	// 升级 HTTP 连接为 WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("WebSocket 升级失败", zap.Error(err))
		return
	}

	// 创建客户端
	client := &TaskWebSocketClient{
		userCode: userCode,
		conn:     conn,
		send:     make(chan []byte, 256),
	}

	// 注册客户端
	h.clientsMu.Lock()
	// 如果该用户已有连接，先关闭旧连接
	if oldClient, exists := h.clients[userCode]; exists {
		close(oldClient.send)
		oldClient.conn.Close()
	}
	h.clients[userCode] = client
	h.clientsMu.Unlock()

	h.logger.Info("Task WebSocket 连接已建立",
		zap.String("user_code", userCode),
	)

	// 启动读写 goroutine
	go h.writePump(client)
	go h.readPump(client)
}

// readPump 处理从客户端读取消息
func (h *TaskWebSocketHandler) readPump(client *TaskWebSocketClient) {
	defer func() {
		h.unregister(client)
		client.conn.Close()
	}()

	// 设置读取限制
	client.conn.SetReadLimit(512 * 1024) // 512KB

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Warn("WebSocket 读取错误", zap.Error(err))
			}
			break
		}

		// 处理 ping/pong
		msgStr := string(message)
		if msgStr == `"ping"` || msgStr == `ping` {
			client.send <- []byte(`{"type":"pong"}`)
		}
	}
}

// writePump 处理向客户端写入消息
func (h *TaskWebSocketHandler) writePump(client *TaskWebSocketClient) {
	defer func() {
		client.conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.send:
			if !ok {
				// 通道已关闭
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				h.logger.Warn("WebSocket 写入错误", zap.Error(err))
				return
			}
		}
	}
}

// unregister 注销客户端
func (h *TaskWebSocketHandler) unregister(client *TaskWebSocketClient) {
	h.clientsMu.Lock()
	if c, exists := h.clients[client.userCode]; exists && c == client {
		delete(h.clients, client.userCode)
		close(client.send)
		h.logger.Info("Task WebSocket 连接已断开",
			zap.String("user_code", client.userCode),
		)
	}
	h.clientsMu.Unlock()
}

// Broadcast 向所有连接的客户端广播消息
func (h *TaskWebSocketHandler) Broadcast(message []byte) {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()

	for userCode, client := range h.clients {
		select {
		case client.send <- message:
			// 发送成功
		default:
			// 通道已满，关闭连接
			h.logger.Warn("客户端发送通道已满，关闭连接",
				zap.String("user_code", userCode),
			)
			close(client.send)
		}
	}
}

// BroadcastToUser 向特定用户发送消息
func (h *TaskWebSocketHandler) BroadcastToUser(userCode string, message []byte) {
	h.clientsMu.RLock()
	client, exists := h.clients[userCode]
	h.clientsMu.RUnlock()

	if !exists {
		return
	}

	select {
	case client.send <- message:
	default:
		h.logger.Warn("客户端发送通道已满",
			zap.String("user_code", userCode),
		)
	}
}

// GetClientCount 获取当前连接数
func (h *TaskWebSocketHandler) GetClientCount() int {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	return len(h.clients)
}

// BroadcastTaskEvent 广播任务事件
// 这是一个便捷方法，用于从其他模块调用
func (h *TaskWebSocketHandler) BroadcastTaskEvent(eventType string, payload interface{}) {
	// 使用简单的 JSON 序列化
	// 实际实现中应该使用更完整的结构
	h.logger.Debug("广播任务事件",
		zap.String("event_type", eventType),
	)
}
