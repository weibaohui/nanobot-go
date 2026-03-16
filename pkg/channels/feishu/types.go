package feishu

import (
	"context"
	"sync"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/larksuite/oapi-sdk-go/v3/ws"
	"github.com/weibaohui/nanobot-go/pkg/bus"
	"go.uber.org/zap"
)

// Config 飞书配置
type Config struct {
	AppID             string   `json:"app_id"`
	AppSecret         string   `json:"app_secret"`
	EncryptKey        string   `json:"encrypt_key"`
	VerificationToken string   `json:"verification_token"`
	AllowFrom         []string `json:"allow_from"`
	ChannelID         uint     `json:"channel_id"` // 数据库中的渠道ID
}

// Channel 飞书渠道
// 使用 WebSocket 长连接接收消息，HTTP API 发送消息
type Channel struct {
	bus     *bus.MessageBus
	name    string
	config  *Config
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

	// 消息反应缓存: message_id -> reactionInfo
	reactionCache map[string]*reactionInfo
	reactionMu    sync.RWMutex

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

// MessageEvent 飞书消息事件包装
type MessageEvent struct {
	Message *larkim.EventMessage
	Sender  *larkim.EventSender
	ChatID  string
	ChatType string
	MsgType string
	Content string
	MessageID string
	SenderID string
}
