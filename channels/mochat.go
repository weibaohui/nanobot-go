package channels

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/weibaohui/nanobot-go/bus"
	"go.uber.org/zap"
)

// MochatChannel Mochat 渠道
// 使用 Socket.IO 连接，支持 HTTP 轮询回退
type MochatChannel struct {
	*BaseChannel
	config    *MochatConfig
	logger    *zap.Logger
	running   bool
	connected bool

	stateDir      string
	sessionCursor map[string]int64
	sessionSet    map[string]bool
	panelSet      map[string]bool
	autoDiscover  struct {
		sessions bool
		panels   bool
	}
	coldSessions      map[string]bool
	sessionByConverse map[string]string

	seenSet     map[string]map[string]bool
	seenQueue   map[string][]string
	delayStates map[string]*DelayState
	targetLocks map[string]*sync.Mutex

	fallbackMode bool
}

// MochatConfig Mochat 配置
type MochatConfig struct {
	ClawToken               string
	BaseURL                 string
	SocketURL               string
	SocketPath              string
	SocketDisableMsgpack    bool
	SocketReconnectDelay    int
	SocketMaxReconnectDelay int
	SocketConnectTimeout    int
	MaxRetryAttempts        int
	Sessions                []string
	Panels                  []string
	WatchLimit              int
	RefreshInterval         int
	Mention                 MochatMentionConfig
	Groups                  map[string]MochatGroupConfig
	AllowFrom               []string
}

// MochatMentionConfig Mochat 提及配置
type MochatMentionConfig struct {
	RequireInGroups bool
}

// MochatGroupConfig Mochat 群组配置
type MochatGroupConfig struct {
	RequireMention bool
}

// DelayState 延迟消息状态
type DelayState struct {
	entries []MochatBufferedEntry
	lock    sync.Mutex
	timer   *time.Timer
}

// MochatBufferedEntry 缓冲的消息条目
type MochatBufferedEntry struct {
	RawBody        string
	Author         string
	SenderName     string
	SenderUsername string
	Timestamp      int64
	MessageID      string
	GroupID        string
}

// MochatTarget 目标解析结果
type MochatTarget struct {
	ID      string
	IsPanel bool
}

// NewMochatChannel 创建 Mochat 渠道
func NewMochatChannel(config *MochatConfig, messageBus *bus.MessageBus, logger *zap.Logger) *MochatChannel {
	if logger == nil {
		logger = zap.NewNop()
	}
	if config.WatchLimit == 0 {
		config.WatchLimit = 50
	}
	if config.RefreshInterval == 0 {
		config.RefreshInterval = 30000
	}
	return &MochatChannel{
		BaseChannel:       NewBaseChannel("mochat", messageBus),
		config:            config,
		logger:            logger,
		sessionCursor:     make(map[string]int64),
		sessionSet:        make(map[string]bool),
		panelSet:          make(map[string]bool),
		coldSessions:      make(map[string]bool),
		sessionByConverse: make(map[string]string),
		seenSet:           make(map[string]map[string]bool),
		seenQueue:         make(map[string][]string),
		delayStates:       make(map[string]*DelayState),
		targetLocks:       make(map[string]*sync.Mutex),
	}
}

// Start 启动 Mochat 渠道
func (c *MochatChannel) Start(ctx context.Context) error {
	if c.config.ClawToken == "" {
		c.logger.Error("Mochat claw_token 未配置")
		return nil
	}

	c.running = true
	c.seedTargetsFromConfig()

	c.logger.Info("Mochat 渠道已启动")

	// TODO: 实现 Socket.IO 连接
	// TODO: 实现目标刷新循环

	return nil
}

// Stop 停止 Mochat 渠道
func (c *MochatChannel) Stop() {
	c.running = false
	c.connected = false
	c.logger.Info("Mochat 渠道已停止")
}

// Send 发送消息
func (c *MochatChannel) Send(msg *bus.OutboundMessage) error {
	if c.config.ClawToken == "" {
		c.logger.Warn("Mochat claw_token 缺失，跳过发送")
		return nil
	}

	content := strings.TrimSpace(msg.Content)
	if content == "" {
		return nil
	}

	target := c.resolveMochatTarget(msg.ChatID)
	if target.ID == "" {
		c.logger.Warn("Mochat 发送目标为空")
		return nil
	}

	isPanel := target.IsPanel || c.panelSet[target.ID]

	c.logger.Debug("发送 Mochat 消息",
		zap.String("target", target.ID),
		zap.Bool("is_panel", isPanel),
	)

	// TODO: 实现 API 发送

	return nil
}

// seedTargetsFromConfig 从配置初始化目标
func (c *MochatChannel) seedTargetsFromConfig() {
	sessions, autoSessions := c.normalizeIDList(c.config.Sessions)
	panels, autoPanels := c.normalizeIDList(c.config.Panels)

	for _, sid := range sessions {
		c.sessionSet[sid] = true
		if _, ok := c.sessionCursor[sid]; !ok {
			c.coldSessions[sid] = true
		}
	}
	for _, pid := range panels {
		c.panelSet[pid] = true
	}

	c.autoDiscover.sessions = autoSessions
	c.autoDiscover.panels = autoPanels
}

// normalizeIDList 规范化 ID 列表
func (c *MochatChannel) normalizeIDList(values []string) ([]string, bool) {
	cleaned := []string{}
	hasWildcard := false
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if v == "*" {
			hasWildcard = true
			continue
		}
		cleaned = append(cleaned, v)
	}
	return cleaned, hasWildcard
}

// resolveMochatTarget 解析目标
func (c *MochatChannel) resolveMochatTarget(raw string) MochatTarget {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return MochatTarget{}
	}

	lowered := strings.ToLower(trimmed)
	cleaned := trimmed
	forcedPanel := false

	for _, prefix := range []string{"mochat:", "group:", "channel:", "panel:"} {
		if strings.HasPrefix(lowered, prefix) {
			cleaned = strings.TrimSpace(trimmed[len(prefix):])
			forcedPanel = prefix != "mochat:"
			break
		}
	}

	if cleaned == "" {
		return MochatTarget{}
	}

	isPanel := forcedPanel || !strings.HasPrefix(cleaned, "session_")
	return MochatTarget{ID: cleaned, IsPanel: isPanel}
}

// handleWatchPayload 处理监听负载
func (c *MochatChannel) handleWatchPayload(payload map[string]interface{}, kind string) {
	// TODO: 实现消息处理
}

// handleNotifyChatMessage 处理聊天消息通知
func (c *MochatChannel) handleNotifyChatMessage(payload interface{}) {
	// TODO: 实现消息处理
}

// normalizeMochatContent 规范化内容
func normalizeMochatContent(content interface{}) string {
	if content == nil {
		return ""
	}
	if s, ok := content.(string); ok {
		return strings.TrimSpace(s)
	}
	data, _ := json.Marshal(content)
	return string(data)
}

// extractMentionIDs 提取提及 ID
func extractMentionIDs(value interface{}) []string {
	if value == nil {
		return nil
	}
	list, ok := value.([]interface{})
	if !ok {
		return nil
	}
	ids := []string{}
	for _, item := range list {
		if s, ok := item.(string); ok && s != "" {
			ids = append(ids, s)
		} else if m, ok := item.(map[string]interface{}); ok {
			for _, key := range []string{"id", "userId", "_id"} {
				if id, ok := m[key].(string); ok && id != "" {
					ids = append(ids, id)
					break
				}
			}
		}
	}
	return ids
}
