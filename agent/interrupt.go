package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/cloudwego/eino/compose"
	"github.com/weibaohui/nanobot-go/bus"
	"go.uber.org/zap"
)

// InterruptManager 管理中断和恢复
type InterruptManager struct {
	bus              *bus.MessageBus
	logger           *zap.Logger
	checkpoint       compose.CheckPointStore
	pending          map[string]*InterruptInfo // checkpointID -> info
	pendingBySession map[string]*InterruptInfo // sessionKey -> info
	mu               sync.RWMutex
	responseChan     chan *UserResponse
}

// InterruptInfo 中断信息
type InterruptInfo struct {
	CheckpointID string   `json:"checkpoint_id"`
	InterruptID  string   `json:"interrupt_id"`
	Channel      string   `json:"channel"`
	ChatID       string   `json:"chat_id"`
	Question     string   `json:"question"`
	Options      []string `json:"options,omitempty"`
	SessionKey   string   `json:"session_key"`
	IsAskUser    bool     `json:"is_ask_user"`
	IsPlan       bool     `json:"is_plan"`
}

// UserResponse 用户响应
type UserResponse struct {
	CheckpointID string `json:"checkpoint_id"`
	Answer       string `json:"answer"`
}

// NewInterruptManager 创建中断管理器
func NewInterruptManager(messageBus *bus.MessageBus, logger *zap.Logger) *InterruptManager {
	return &InterruptManager{
		bus:              messageBus,
		logger:           logger,
		checkpoint:       NewInMemoryCheckpointStore(),
		pending:          make(map[string]*InterruptInfo),
		pendingBySession: make(map[string]*InterruptInfo),
		responseChan:     make(chan *UserResponse, 100),
	}
}

// GetCheckpointStore 获取 CheckpointStore
func (m *InterruptManager) GetCheckpointStore() compose.CheckPointStore {
	return m.checkpoint
}

// HandleInterrupt 处理中断
func (m *InterruptManager) HandleInterrupt(info *InterruptInfo) {
	m.mu.Lock()
	m.pending[info.CheckpointID] = info
	if info.SessionKey != "" {
		m.pendingBySession[info.SessionKey] = info
	}
	m.mu.Unlock()

	// 发送中断消息到用户
	question := info.Question
	if len(info.Options) > 0 {
		optionsJSON, _ := json.Marshal(info.Options)
		question += fmt.Sprintf("\n\n选项: %s", string(optionsJSON))
	}

	// 发布中断请求
	m.bus.PublishOutbound(bus.NewOutboundMessage(info.Channel, info.ChatID, fmt.Sprintf("❓ %s", question)))

	m.logger.Info("等待用户输入",
		zap.String("checkpoint_id", info.CheckpointID),
		zap.String("channel", info.Channel),
		zap.String("chat_id", info.ChatID),
		zap.String("session_key", info.SessionKey),
	)
}

// SubmitUserResponse 提交用户响应
func (m *InterruptManager) SubmitUserResponse(response *UserResponse) error {
	m.mu.RLock()
	_, ok := m.pending[response.CheckpointID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("找不到对应的中断: %s", response.CheckpointID)
	}

	// 将响应发送到通道
	select {
	case m.responseChan <- response:
		m.logger.Info("用户响应已提交",
			zap.String("checkpoint_id", response.CheckpointID),
			zap.String("answer", response.Answer),
		)
		return nil
	default:
		return fmt.Errorf("响应通道已满")
	}
}

// WaitForResponse 等待用户响应
func (m *InterruptManager) WaitForResponse(ctx context.Context, checkpointID string) (*UserResponse, error) {
	for {
		select {
		case resp := <-m.responseChan:
			if resp.CheckpointID == checkpointID {
				// 清理
				m.ClearInterrupt(checkpointID)
				return resp, nil
			}
			// 不是目标响应，放回通道
			select {
			case m.responseChan <- resp:
			default:
				m.logger.Warn("无法将非目标响应放回通道")
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// CancelInterrupt 取消中断
func (m *InterruptManager) CancelInterrupt(checkpointID string) {
	m.mu.Lock()
	if info, ok := m.pending[checkpointID]; ok {
		delete(m.pending, checkpointID)
		if info.SessionKey != "" {
			delete(m.pendingBySession, info.SessionKey)
		}
	}
	m.mu.Unlock()
}

// GetPendingInterrupt 获取指定会话的待处理中断
func (m *InterruptManager) GetPendingInterrupt(sessionKey string) *InterruptInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pendingBySession[sessionKey]
}

// ClearInterrupt 清除已处理的中断
func (m *InterruptManager) ClearInterrupt(checkpointID string) {
	m.mu.Lock()
	if info, ok := m.pending[checkpointID]; ok {
		delete(m.pending, checkpointID)
		if info.SessionKey != "" {
			delete(m.pendingBySession, info.SessionKey)
		}
	}
	m.mu.Unlock()
}

// InMemoryCheckpointStore 内存 Checkpoint 存储
type InMemoryCheckpointStore struct {
	mem map[string][]byte
	mu  sync.RWMutex
}

// NewInMemoryCheckpointStore 创建内存 Checkpoint 存储
func NewInMemoryCheckpointStore() compose.CheckPointStore {
	return &InMemoryCheckpointStore{
		mem: make(map[string][]byte),
	}
}

// Set 保存 checkpoint
func (s *InMemoryCheckpointStore) Set(ctx context.Context, key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mem[key] = value
	return nil
}

// Get 获取 checkpoint
func (s *InMemoryCheckpointStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.mem[key]
	return v, ok, nil
}
