package websocket

import (
	"sync"

	"go.uber.org/zap"
)

// ConnectionManager WebSocket 连接管理器
type ConnectionManager struct {
	connections map[string]*Connection // key: connID
	userIndex   map[string][]string    // user_code -> []connID
	channelCode string
	logger      *zap.Logger
	mu          sync.RWMutex
}

// NewConnectionManager 创建连接管理器
func NewConnectionManager(channelCode string, logger *zap.Logger) *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]*Connection),
		userIndex:   make(map[string][]string),
		channelCode: channelCode,
		logger:      logger.With(zap.String("channel_code", channelCode)),
	}
}

// Add 添加新连接
func (m *ConnectionManager) Add(conn *Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.connections[conn.ID()] = conn
	m.userIndex[conn.UserCode()] = append(m.userIndex[conn.UserCode()], conn.ID())

	m.logger.Info("WebSocket 连接已添加",
		zap.String("conn_id", conn.ID()),
		zap.String("user_code", conn.UserCode()),
		zap.Int("total_connections", len(m.connections)),
	)
}

// Remove 移除连接
func (m *ConnectionManager) Remove(connID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, exists := m.connections[connID]
	if !exists {
		return
	}

	// 从用户索引中移除
	userCode := conn.UserCode()
	if ids, ok := m.userIndex[userCode]; ok {
		newIDs := make([]string, 0, len(ids)-1)
		for _, id := range ids {
			if id != connID {
				newIDs = append(newIDs, id)
			}
		}
		if len(newIDs) == 0 {
			delete(m.userIndex, userCode)
		} else {
			m.userIndex[userCode] = newIDs
		}
	}

	// 从连接池中移除
	delete(m.connections, connID)

	m.logger.Info("WebSocket 连接已移除",
		zap.String("conn_id", connID),
		zap.String("user_code", userCode),
		zap.Int("total_connections", len(m.connections)),
	)
}

// Get 获取指定连接
func (m *ConnectionManager) Get(connID string) *Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connections[connID]
}

// SendToConnection 向指定连接发送消息
func (m *ConnectionManager) SendToConnection(connID string, data []byte) bool {
	m.mu.RLock()
	conn, exists := m.connections[connID]
	m.mu.RUnlock()

	if !exists {
		return false
	}
	return conn.Send(data)
}

// SendToUser 向指定用户的所有连接发送消息
func (m *ConnectionManager) SendToUser(userCode string, data []byte) int {
	m.mu.RLock()
	connIDs, exists := m.userIndex[userCode]
	m.mu.RUnlock()

	if !exists {
		return 0
	}

	sent := 0
	for _, connID := range connIDs {
		if m.SendToConnection(connID, data) {
			sent++
		}
	}
	return sent
}

// Broadcast 广播消息给所有连接
func (m *ConnectionManager) Broadcast(data []byte) int {
	m.mu.RLock()
	connections := make([]*Connection, 0, len(m.connections))
	for _, conn := range m.connections {
		connections = append(connections, conn)
	}
	m.mu.RUnlock()

	sent := 0
	for _, conn := range connections {
		if conn.Send(data) {
			sent++
		}
	}
	return sent
}

// GetUserConnections 获取指定用户的所有连接
func (m *ConnectionManager) GetUserConnections(userCode string) []*Connection {
	m.mu.RLock()
	connIDs, exists := m.userIndex[userCode]
	m.mu.RUnlock()

	if !exists {
		return nil
	}

	connections := make([]*Connection, 0, len(connIDs))
	for _, connID := range connIDs {
		if conn := m.Get(connID); conn != nil {
			connections = append(connections, conn)
		}
	}
	return connections
}

// Count 返回当前连接总数
func (m *ConnectionManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connections)
}

// CountByUser 返回指定用户的连接数
func (m *ConnectionManager) CountByUser(userCode string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.userIndex[userCode])
}

// CloseAll 关闭所有连接
func (m *ConnectionManager) CloseAll() {
	m.mu.RLock()
	connections := make([]*Connection, 0, len(m.connections))
	for _, conn := range m.connections {
		connections = append(connections, conn)
	}
	m.mu.RUnlock()

	for _, conn := range connections {
		conn.Close()
	}
}
