package interrupt

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/cloudwego/eino/compose"
	"github.com/weibaohui/nanobot-go/pkg/bus"
	"go.uber.org/zap"
)

// Manager 管理中断和恢复
type Manager struct {
	bus              *bus.MessageBus
	logger           *zap.Logger
	checkpoint       compose.CheckPointStore
	pending          map[string]*InterruptInfo
	pendingBySession map[string]*InterruptInfo
	mu               sync.RWMutex
	responseChan     chan *UserResponse
	handlers         map[InterruptType]Handler
	history          []*InterruptInfo
	historyMutex     sync.RWMutex
	maxHistory       int
	defaultTimeout   time.Duration
	maxPending       int
}

// ManagerConfig 中断管理器配置
type ManagerConfig struct {
	Bus            *bus.MessageBus
	Logger         *zap.Logger
	DefaultTimeout time.Duration
	MaxPending     int
	MaxHistory     int
}

// NewManager 创建中断管理器
func NewManager(messageBus *bus.MessageBus, logger *zap.Logger) *Manager {
	return NewManagerWithConfig(&ManagerConfig{
		Bus:            messageBus,
		Logger:         logger,
		DefaultTimeout: 30 * time.Minute,
		MaxPending:     100,
		MaxHistory:     1000,
	})
}

// NewManagerWithConfig 使用配置创建中断管理器
func NewManagerWithConfig(cfg *ManagerConfig) *Manager {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	maxPending := cfg.MaxPending
	if maxPending <= 0 {
		maxPending = 100
	}

	maxHistory := cfg.MaxHistory
	if maxHistory <= 0 {
		maxHistory = 1000
	}

	mgr := &Manager{
		bus:              cfg.Bus,
		logger:           logger,
		checkpoint:       NewInMemoryCheckpointStore(),
		pending:          make(map[string]*InterruptInfo),
		pendingBySession: make(map[string]*InterruptInfo),
		responseChan:     make(chan *UserResponse, maxPending),
		handlers:         make(map[InterruptType]Handler),
		history:          make([]*InterruptInfo, 0, maxHistory),
		maxHistory:       maxHistory,
		defaultTimeout:   cfg.DefaultTimeout,
		maxPending:       maxPending,
	}

	mgr.registerDefaultHandlers()
	return mgr
}

func (m *Manager) registerDefaultHandlers() {
	m.handlers[InterruptTypeAskUser] = &AskUserHandler{}
	m.handlers[InterruptTypePlanApproval] = &PlanApprovalHandler{}
	m.handlers[InterruptTypeToolConfirm] = &ToolConfirmHandler{}
	m.handlers[InterruptTypeFileOperation] = &FileOperationHandler{}
}

// GetCheckpointStore 获取 CheckpointStore
func (m *Manager) GetCheckpointStore() compose.CheckPointStore {
	return m.checkpoint
}

// HandleInterrupt 处理中断
func (m *Manager) HandleInterrupt(info *InterruptInfo) {
	if info.Type == "" {
		info.Type = InterruptTypeAskUser
	}
	if info.Status == "" {
		info.Status = InterruptStatusPending
	}
	if info.CreatedAt.IsZero() {
		info.CreatedAt = time.Now()
	}
	if info.ExpiresAt == nil && m.defaultTimeout > 0 {
		expires := info.CreatedAt.Add(m.defaultTimeout)
		info.ExpiresAt = &expires
	}

	m.mu.Lock()
	if len(m.pending) >= m.maxPending {
		m.logger.Warn("待处理中断数量已达上限，清理过期中断")
		m.cleanExpiredInterruptsLocked()
	}

	m.pending[info.CheckpointID] = info
	if info.SessionKey != "" {
		m.pendingBySession[info.SessionKey] = info
	}
	m.mu.Unlock()

	m.addToHistory(info)

	question := m.formatQuestion(info)
	m.bus.PublishOutbound(bus.NewOutboundMessage(info.Channel, info.ChatID, fmt.Sprintf("❓ %s", question)))

	m.logger.Info("等待用户输入",
		zap.String("checkpoint_id", info.CheckpointID),
		zap.String("type", string(info.Type)),
		zap.String("channel", info.Channel),
		zap.String("chat_id", info.ChatID),
		zap.String("session_key", info.SessionKey),
	)
}

func (m *Manager) formatQuestion(info *InterruptInfo) string {
	if handler, ok := m.handlers[info.Type]; ok {
		return handler.FormatQuestion(info)
	}

	question := info.Question
	if len(info.Options) > 0 {
		optionsJSON, _ := json.Marshal(info.Options)
		question += fmt.Sprintf("\n\n选项: %s", string(optionsJSON))
	}
	return question
}

// SubmitUserResponse 提交用户响应
func (m *Manager) SubmitUserResponse(response *UserResponse) error {
	m.mu.RLock()
	info, ok := m.pending[response.CheckpointID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("找不到对应的中断: %s", response.CheckpointID)
	}

	if info.ExpiresAt != nil && time.Now().After(*info.ExpiresAt) {
		m.ClearInterrupt(response.CheckpointID)
		return fmt.Errorf("中断已过期: %s", response.CheckpointID)
	}

	if handler, ok := m.handlers[info.Type]; ok {
		if err := handler.Validate(response); err != nil {
			return fmt.Errorf("响应验证失败: %w", err)
		}
	}

	response.Timestamp = time.Now()

	select {
	case m.responseChan <- response:
		m.logger.Info("用户响应已提交",
			zap.String("checkpoint_id", response.CheckpointID),
			zap.String("answer", response.Answer),
			zap.Bool("approved", response.Approved),
		)
		return nil
	default:
		return fmt.Errorf("响应通道已满")
	}
}

// WaitForResponse 等待用户响应
func (m *Manager) WaitForResponse(ctx context.Context, checkpointID string) (*UserResponse, error) {
	for {
		select {
		case resp := <-m.responseChan:
			if resp.CheckpointID == checkpointID {
				m.updateInterruptStatus(checkpointID, InterruptStatusResolved)
				m.ClearInterrupt(checkpointID)
				return resp, nil
			}
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
func (m *Manager) CancelInterrupt(checkpointID string) {
	m.updateInterruptStatus(checkpointID, InterruptStatusCancelled)
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
func (m *Manager) GetPendingInterrupt(sessionKey string) *InterruptInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pendingBySession[sessionKey]
}

// ClearInterrupt 清除已处理的中断
func (m *Manager) ClearInterrupt(checkpointID string) {
	m.mu.Lock()
	if info, ok := m.pending[checkpointID]; ok {
		delete(m.pending, checkpointID)
		if info.SessionKey != "" {
			delete(m.pendingBySession, info.SessionKey)
		}
	}
	m.mu.Unlock()
}

func (m *Manager) updateInterruptStatus(checkpointID string, status InterruptStatus) {
	m.mu.Lock()
	if info, ok := m.pending[checkpointID]; ok {
		info.Status = status
	}
	m.mu.Unlock()
}

func (m *Manager) cleanExpiredInterruptsLocked() {
	now := time.Now()
	for id, info := range m.pending {
		if info.ExpiresAt != nil && now.After(*info.ExpiresAt) {
			info.Status = InterruptStatusExpired
			delete(m.pending, id)
			if info.SessionKey != "" {
				delete(m.pendingBySession, info.SessionKey)
			}
			m.logger.Info("清理过期中断",
				zap.String("checkpoint_id", id),
			)
		}
	}
}

func (m *Manager) addToHistory(info *InterruptInfo) {
	m.historyMutex.Lock()
	defer m.historyMutex.Unlock()

	if len(m.history) >= m.maxHistory {
		m.history = m.history[1:]
	}
	m.history = append(m.history, info)
}

// GetInterruptHistory 获取中断历史
func (m *Manager) GetInterruptHistory(limit int) []*InterruptInfo {
	m.historyMutex.RLock()
	defer m.historyMutex.RUnlock()

	if limit <= 0 || limit > len(m.history) {
		limit = len(m.history)
	}

	start := len(m.history) - limit
	if start < 0 {
		start = 0
	}

	result := make([]*InterruptInfo, limit)
	copy(result, m.history[start:])
	return result
}

// GetInterruptStats 获取中断统计
func (m *Manager) GetInterruptStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.historyMutex.RLock()
	defer m.historyMutex.RUnlock()

	stats := map[string]interface{}{
		"pending_count": len(m.pending),
		"history_count": len(m.history),
		"by_type":       make(map[string]int),
		"by_status":     make(map[string]int),
	}

	typeStats := stats["by_type"].(map[string]int)
	statusStats := stats["by_status"].(map[string]int)

	for _, info := range m.history {
		typeStats[string(info.Type)]++
		statusStats[string(info.Status)]++
	}

	return stats
}

// RegisterHandler 注册中断处理器
func (m *Manager) RegisterHandler(interruptType InterruptType, handler Handler) {
	m.handlers[interruptType] = handler
	m.logger.Info("注册中断处理器",
		zap.String("type", string(interruptType)),
	)
}

// CreateAskUserInterrupt 创建用户提问中断
func CreateAskUserInterrupt(checkpointID, interruptID, channel, chatID, sessionKey, question string, options []string) *InterruptInfo {
	return &InterruptInfo{
		CheckpointID: checkpointID,
		InterruptID:  interruptID,
		Channel:      channel,
		ChatID:       chatID,
		SessionKey:   sessionKey,
		Question:     question,
		Options:      options,
		Type:         InterruptTypeAskUser,
		Status:       InterruptStatusPending,
		CreatedAt:    time.Now(),
		Priority:     10,
	}
}

// CreatePlanApprovalInterrupt 创建计划审批中断
func CreatePlanApprovalInterrupt(checkpointID, interruptID, channel, chatID, sessionKey, planID, planContent string, steps []string) *InterruptInfo {
	return &InterruptInfo{
		CheckpointID: checkpointID,
		InterruptID:  interruptID,
		Channel:      channel,
		ChatID:       chatID,
		SessionKey:   sessionKey,
		Question:     "请审批以下计划",
		Type:         InterruptTypePlanApproval,
		Status:       InterruptStatusPending,
		CreatedAt:    time.Now(),
		Priority:     20,
		Metadata: map[string]any{
			"plan_id":      planID,
			"plan_content": planContent,
			"steps":        steps,
		},
	}
}

// CreateToolConfirmInterrupt 创建工具确认中断
func CreateToolConfirmInterrupt(checkpointID, interruptID, channel, chatID, sessionKey, toolName string, toolArgs map[string]any, riskLevel string) *InterruptInfo {
	return &InterruptInfo{
		CheckpointID: checkpointID,
		InterruptID:  interruptID,
		Channel:      channel,
		ChatID:       chatID,
		SessionKey:   sessionKey,
		Question:     fmt.Sprintf("确认执行工具: %s", toolName),
		Type:         InterruptTypeToolConfirm,
		Status:       InterruptStatusPending,
		CreatedAt:    time.Now(),
		Priority:     30,
		Metadata: map[string]any{
			"tool_name":  toolName,
			"tool_args":  toolArgs,
			"risk_level": riskLevel,
		},
	}
}
