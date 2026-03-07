package agent

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/cloudwego/eino/compose"
	"github.com/weibaohui/nanobot-go/bus"
	"go.uber.org/zap"
)

// 初始化时注册中断信息类型（确保 Gob 序列化正确）
func init() {
	gob.Register(&InterruptInfo{})
	gob.Register(&AskUserInterrupt{})
	gob.Register(&PlanApprovalInterrupt{})
	gob.Register(&ToolConfirmInterrupt{})
	gob.Register(&FileOperationInterrupt{})
	gob.Register(&CustomInterrupt{})
}

// InterruptType 中断类型
type InterruptType string

const (
	InterruptTypeAskUser       InterruptType = "ask_user"
	InterruptTypePlanApproval  InterruptType = "plan_approval"
	InterruptTypeToolConfirm   InterruptType = "tool_confirm"
	InterruptTypeFileOperation InterruptType = "file_operation"
	InterruptTypeCustom        InterruptType = "custom"
)

// InterruptStatus 中断状态
type InterruptStatus string

const (
	InterruptStatusPending   InterruptStatus = "pending"
	InterruptStatusResolved  InterruptStatus = "resolved"
	InterruptStatusCancelled InterruptStatus = "cancelled"
	InterruptStatusExpired   InterruptStatus = "expired"
)

// InterruptInfo 中断信息（基础结构）
type InterruptInfo struct {
	CheckpointID         string          `json:"checkpoint_id"`
	OriginalCheckpointID string          `json:"original_checkpoint_id"` // 原始 checkpoint ID，用于恢复
	InterruptID          string          `json:"interrupt_id"`
	Channel              string          `json:"channel"`
	ChatID               string          `json:"chat_id"`
	Question             string          `json:"question"`
	Options              []string        `json:"options,omitempty"`
	SessionKey           string          `json:"session_key"`
	IsAskUser            bool            `json:"is_ask_user"`
	IsPlan               bool            `json:"is_plan"`
	IsSupervisor         bool            `json:"is_supervisor"` // 标记是否来自 Supervisor 模式的中断
	IsMaster             bool            `json:"is_master"`     // 标记是否来自 Master 模式的中断
	Type                 InterruptType   `json:"type"`
	Status               InterruptStatus `json:"status"`
	CreatedAt            time.Time       `json:"created_at"`
	ExpiresAt            *time.Time      `json:"expires_at,omitempty"`
	Priority             int             `json:"priority"`
	Metadata             map[string]any  `json:"metadata,omitempty"`
}

// AskUserInterrupt 用户提问中断
type AskUserInterrupt struct {
	InterruptInfo
	Question     string   `json:"question"`
	Options      []string `json:"options,omitempty"`
	DefaultValue string   `json:"default_value,omitempty"`
	Validation   string   `json:"validation,omitempty"`
}

// PlanApprovalInterrupt 计划审批中断
type PlanApprovalInterrupt struct {
	InterruptInfo
	PlanID      string   `json:"plan_id"`
	PlanContent string   `json:"plan_content"`
	Steps       []string `json:"steps"`
	Requires    []string `json:"requires,omitempty"`
}

// ToolConfirmInterrupt 工具确认中断
type ToolConfirmInterrupt struct {
	InterruptInfo
	ToolName    string         `json:"tool_name"`
	ToolArgs    map[string]any `json:"tool_args"`
	RiskLevel   string         `json:"risk_level"`
	Description string         `json:"description"`
}

// FileOperationInterrupt 文件操作中断
type FileOperationInterrupt struct {
	InterruptInfo
	Operation string `json:"operation"`
	FilePath  string `json:"file_path"`
	Content   string `json:"content,omitempty"`
	Backup    bool   `json:"backup"`
}

// CustomInterrupt 自定义中断
type CustomInterrupt struct {
	InterruptInfo
	CustomType string         `json:"custom_type"`
	Data       map[string]any `json:"data"`
}

// UserResponse 用户响应
type UserResponse struct {
	CheckpointID string         `json:"checkpoint_id"`
	Answer       string         `json:"answer"`
	Approved     bool           `json:"approved,omitempty"`
	ModifiedData map[string]any `json:"modified_data,omitempty"`
	Timestamp    time.Time      `json:"timestamp"`
}

// InterruptManager 管理中断和恢复
type InterruptManager struct {
	bus              *bus.MessageBus
	logger           *zap.Logger
	checkpoint       compose.CheckPointStore
	pending          map[string]*InterruptInfo // checkpointID -> info
	pendingBySession map[string]*InterruptInfo // sessionKey -> info
	mu               sync.RWMutex
	responseChan     chan *UserResponse

	// 中断处理器注册表
	handlers map[InterruptType]InterruptHandler

	// 中断历史（用于审计和分析）
	history      []*InterruptInfo
	historyMutex sync.RWMutex
	maxHistory   int

	// 配置
	defaultTimeout time.Duration
	maxPending     int
}

// InterruptHandler 中断处理器接口
type InterruptHandler interface {
	// Handle 处理中断
	Handle(ctx context.Context, info *InterruptInfo) (*UserResponse, error)
	// Validate 验证用户响应
	Validate(response *UserResponse) error
	// FormatQuestion 格式化问题
	FormatQuestion(info *InterruptInfo) string
}

// InterruptManagerConfig 中断管理器配置
type InterruptManagerConfig struct {
	Bus            *bus.MessageBus
	Logger         *zap.Logger
	DefaultTimeout time.Duration
	MaxPending     int
	MaxHistory     int
}

// NewInterruptManager 创建中断管理器
func NewInterruptManager(messageBus *bus.MessageBus, logger *zap.Logger) *InterruptManager {
	return NewInterruptManagerWithConfig(&InterruptManagerConfig{
		Bus:            messageBus,
		Logger:         logger,
		DefaultTimeout: 30 * time.Minute,
		MaxPending:     100,
		MaxHistory:     1000,
	})
}

// NewInterruptManagerWithConfig 使用配置创建中断管理器
func NewInterruptManagerWithConfig(cfg *InterruptManagerConfig) *InterruptManager {
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

	mgr := &InterruptManager{
		bus:              cfg.Bus,
		logger:           logger,
		checkpoint:       NewInMemoryCheckpointStore(),
		pending:          make(map[string]*InterruptInfo),
		pendingBySession: make(map[string]*InterruptInfo),
		responseChan:     make(chan *UserResponse, maxPending),
		handlers:         make(map[InterruptType]InterruptHandler),
		history:          make([]*InterruptInfo, 0, maxHistory),
		maxHistory:       maxHistory,
		defaultTimeout:   cfg.DefaultTimeout,
		maxPending:       maxPending,
	}

	// 注册默认处理器
	mgr.registerDefaultHandlers()

	return mgr
}

// registerDefaultHandlers 注册默认处理器
func (m *InterruptManager) registerDefaultHandlers() {
	// AskUser 处理器
	m.handlers[InterruptTypeAskUser] = &AskUserHandler{}

	// PlanApproval 处理器
	m.handlers[InterruptTypePlanApproval] = &PlanApprovalHandler{}

	// ToolConfirm 处理器
	m.handlers[InterruptTypeToolConfirm] = &ToolConfirmHandler{}

	// FileOperation 处理器
	m.handlers[InterruptTypeFileOperation] = &FileOperationHandler{}
}

// GetCheckpointStore 获取 CheckpointStore
func (m *InterruptManager) GetCheckpointStore() compose.CheckPointStore {
	return m.checkpoint
}

// HandleInterrupt 处理中断
func (m *InterruptManager) HandleInterrupt(info *InterruptInfo) {
	// 设置默认值
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
	// 检查是否超过最大待处理数量
	if len(m.pending) >= m.maxPending {
		m.logger.Warn("待处理中断数量已达上限，清理过期中断")
		m.cleanExpiredInterruptsLocked()
	}

	m.pending[info.CheckpointID] = info
	if info.SessionKey != "" {
		m.pendingBySession[info.SessionKey] = info
	}
	m.mu.Unlock()

	// 记录历史
	m.addToHistory(info)

	// 格式化并发送中断消息
	question := m.formatQuestion(info)

	// 发布中断请求
	m.bus.PublishOutbound(bus.NewOutboundMessage(info.Channel, info.ChatID, fmt.Sprintf("❓ %s", question)))

	m.logger.Info("等待用户输入",
		zap.String("checkpoint_id", info.CheckpointID),
		zap.String("type", string(info.Type)),
		zap.String("channel", info.Channel),
		zap.String("chat_id", info.ChatID),
		zap.String("session_key", info.SessionKey),
	)
}

// formatQuestion 格式化问题
func (m *InterruptManager) formatQuestion(info *InterruptInfo) string {
	// 检查是否有注册的处理器
	if handler, ok := m.handlers[info.Type]; ok {
		return handler.FormatQuestion(info)
	}

	// 默认格式化
	question := info.Question
	if len(info.Options) > 0 {
		optionsJSON, _ := json.Marshal(info.Options)
		question += fmt.Sprintf("\n\n选项: %s", string(optionsJSON))
	}
	return question
}

// SubmitUserResponse 提交用户响应
func (m *InterruptManager) SubmitUserResponse(response *UserResponse) error {
	m.mu.RLock()
	info, ok := m.pending[response.CheckpointID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("找不到对应的中断: %s", response.CheckpointID)
	}

	// 检查中断是否过期
	if info.ExpiresAt != nil && time.Now().After(*info.ExpiresAt) {
		m.ClearInterrupt(response.CheckpointID)
		return fmt.Errorf("中断已过期: %s", response.CheckpointID)
	}

	// 验证响应
	if handler, ok := m.handlers[info.Type]; ok {
		if err := handler.Validate(response); err != nil {
			return fmt.Errorf("响应验证失败: %w", err)
		}
	}

	// 设置时间戳
	response.Timestamp = time.Now()

	// 将响应发送到通道
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
func (m *InterruptManager) WaitForResponse(ctx context.Context, checkpointID string) (*UserResponse, error) {
	for {
		select {
		case resp := <-m.responseChan:
			if resp.CheckpointID == checkpointID {
				// 更新状态
				m.updateInterruptStatus(checkpointID, InterruptStatusResolved)
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

// updateInterruptStatus 更新中断状态
func (m *InterruptManager) updateInterruptStatus(checkpointID string, status InterruptStatus) {
	m.mu.Lock()
	if info, ok := m.pending[checkpointID]; ok {
		info.Status = status
	}
	m.mu.Unlock()
}

// cleanExpiredInterruptsLocked 清理过期中断（调用时已持有锁）
func (m *InterruptManager) cleanExpiredInterruptsLocked() {
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

// addToHistory 添加到历史记录
func (m *InterruptManager) addToHistory(info *InterruptInfo) {
	m.historyMutex.Lock()
	defer m.historyMutex.Unlock()

	if len(m.history) >= m.maxHistory {
		m.history = m.history[1:]
	}
	m.history = append(m.history, info)
}

// GetInterruptHistory 获取中断历史
func (m *InterruptManager) GetInterruptHistory(limit int) []*InterruptInfo {
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
func (m *InterruptManager) GetInterruptStats() map[string]interface{} {
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
func (m *InterruptManager) RegisterHandler(interruptType InterruptType, handler InterruptHandler) {
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

// 默认处理器实现

// AskUserHandler 用户提问处理器
type AskUserHandler struct{}

func (h *AskUserHandler) Handle(ctx context.Context, info *InterruptInfo) (*UserResponse, error) {
	return &UserResponse{
		CheckpointID: info.CheckpointID,
	}, nil
}

func (h *AskUserHandler) Validate(response *UserResponse) error {
	if response.Answer == "" {
		return fmt.Errorf("回答不能为空")
	}
	return nil
}

func (h *AskUserHandler) FormatQuestion(info *InterruptInfo) string {
	question := info.Question
	if len(info.Options) > 0 {
		question += "\n\n可选答案:"
		for i, opt := range info.Options {
			question += fmt.Sprintf("\n%d. %s", i+1, opt)
		}
	}
	return question
}

// PlanApprovalHandler 计划审批处理器
type PlanApprovalHandler struct{}

func (h *PlanApprovalHandler) Handle(ctx context.Context, info *InterruptInfo) (*UserResponse, error) {
	return &UserResponse{
		CheckpointID: info.CheckpointID,
	}, nil
}

func (h *PlanApprovalHandler) Validate(response *UserResponse) error {
	return nil
}

func (h *PlanApprovalHandler) FormatQuestion(info *InterruptInfo) string {
	question := info.Question
	if steps, ok := info.Metadata["steps"].([]string); ok {
		question += "\n\n执行步骤:"
		for i, step := range steps {
			question += fmt.Sprintf("\n%d. %s", i+1, step)
		}
	}
	question += "\n\n请回复 '确认' 或 '批准' 继续，或提出修改意见。"
	return question
}

// ToolConfirmHandler 工具确认处理器
type ToolConfirmHandler struct{}

func (h *ToolConfirmHandler) Handle(ctx context.Context, info *InterruptInfo) (*UserResponse, error) {
	return &UserResponse{
		CheckpointID: info.CheckpointID,
	}, nil
}

func (h *ToolConfirmHandler) Validate(response *UserResponse) error {
	return nil
}

func (h *ToolConfirmHandler) FormatQuestion(info *InterruptInfo) string {
	toolName, _ := info.Metadata["tool_name"].(string)
	riskLevel, _ := info.Metadata["risk_level"].(string)
	toolArgs, _ := info.Metadata["tool_args"].(map[string]any)

	argsJSON, _ := json.MarshalIndent(toolArgs, "", "  ")

	return fmt.Sprintf(`⚠️ 需要确认执行工具

工具名称: %s
风险等级: %s
参数:
%s

请回复 '确认' 或 '批准' 继续，或 '取消' 拒绝执行。`, toolName, riskLevel, string(argsJSON))
}

// FileOperationHandler 文件操作处理器
type FileOperationHandler struct{}

func (h *FileOperationHandler) Handle(ctx context.Context, info *InterruptInfo) (*UserResponse, error) {
	return &UserResponse{
		CheckpointID: info.CheckpointID,
	}, nil
}

func (h *FileOperationHandler) Validate(response *UserResponse) error {
	return nil
}

func (h *FileOperationHandler) FormatQuestion(info *InterruptInfo) string {
	operation, _ := info.Metadata["operation"].(string)
	filePath, _ := info.Metadata["file_path"].(string)

	return fmt.Sprintf(`📁 文件操作确认

操作类型: %s
文件路径: %s

请回复 '确认' 继续，或 '取消' 拒绝操作。`, operation, filePath)
}

// InMemoryCheckpointStore 内存 Checkpoint 存储
type InMemoryCheckpointStore struct {
	mem      map[string]*checkpointEntry
	mu       sync.RWMutex
	maxSize  int
	stopChan chan struct{}
}

// checkpointEntry 带时间戳的 checkpoint 条目
type checkpointEntry struct {
	value     []byte
	createdAt time.Time
}

// NewInMemoryCheckpointStore 创建内存 Checkpoint 存储
func NewInMemoryCheckpointStore() compose.CheckPointStore {
	store := &InMemoryCheckpointStore{
		mem:      make(map[string]*checkpointEntry),
		maxSize:  1000, // 默认最大存储 1000 个 checkpoint
		stopChan: make(chan struct{}),
	}
	// 启动定期清理 goroutine
	go store.cleanupLoop()
	return store
}

// cleanupLoop 定期清理过期的 checkpoint
func (s *InMemoryCheckpointStore) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanExpired()
		case <-s.stopChan:
			return
		}
	}
}

// cleanExpired 清理过期的 checkpoint（超过 1 小时）
func (s *InMemoryCheckpointStore) cleanExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	expireThreshold := time.Now().Add(-1 * time.Hour)
	for key, entry := range s.mem {
		if entry.createdAt.Before(expireThreshold) {
			delete(s.mem, key)
		}
	}
}

// Close 停止清理 goroutine
func (s *InMemoryCheckpointStore) Close() {
	close(s.stopChan)
}

// Delete 删除 checkpoint
func (s *InMemoryCheckpointStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.mem, key)
	return nil
}

// Set 保存 checkpoint
func (s *InMemoryCheckpointStore) Set(ctx context.Context, key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查容量，超过则清理最旧的条目
	if len(s.mem) >= s.maxSize {
		s.evictOldestLocked()
	}

	s.mem[key] = &checkpointEntry{
		value:     value,
		createdAt: time.Now(),
	}
	return nil
}

// Get 获取 checkpoint
func (s *InMemoryCheckpointStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.mem[key]
	if !ok {
		return nil, false, nil
	}
	return entry.value, true, nil
}

// evictOldestLocked 清理最旧的条目（调用时已持有锁）
func (s *InMemoryCheckpointStore) evictOldestLocked() {
	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, entry := range s.mem {
		if first || entry.createdAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.createdAt
			first = false
		}
	}

	if oldestKey != "" {
		delete(s.mem, oldestKey)
	}
}
