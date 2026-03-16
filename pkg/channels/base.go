package channels

import (
	"context"
	"fmt"
	"sync"

	"github.com/weibaohui/nanobot-go/pkg/bus"
)

// Channel 渠道接口
type Channel interface {
	Name() string
	Start(ctx context.Context) error
	Stop()
}

// BaseChannel 渠道基类
type BaseChannel struct {
	name string
	bus  *bus.MessageBus
}

// NewBaseChannel 创建渠道基类
func NewBaseChannel(name string, messageBus *bus.MessageBus) *BaseChannel {
	return &BaseChannel{
		name: name,
		bus:  messageBus,
	}
}

// Name 返回渠道名称
func (c *BaseChannel) Name() string {
	return c.name
}

// Start 默认启动实现
func (c *BaseChannel) Start(ctx context.Context) error {
	return nil
}

// Stop 默认停止实现
func (c *BaseChannel) Stop() {}

// PublishInbound 发布入站消息
func (c *BaseChannel) PublishInbound(msg *bus.InboundMessage) {
	c.bus.PublishInbound(msg)
}

// SubscribeOutbound 订阅出站消息
// 使用 MessageBus 的订阅机制，确保所有渠道都能收到消息
func (c *BaseChannel) SubscribeOutbound(ctx context.Context, handler func(msg *bus.OutboundMessage)) {
	c.bus.SubscribeOutbound(c.name, func(msg *bus.OutboundMessage) error {
		handler(msg)
		return nil
	})
}

// Manager 渠道管理器
type Manager struct {
	channels map[string]Channel
	mu       sync.RWMutex
	bus      *bus.MessageBus
}

// NewManager 创建渠道管理器
func NewManager(messageBus *bus.MessageBus) *Manager {
	return &Manager{
		channels: make(map[string]Channel),
		bus:      messageBus,
	}
}

// Register 注册渠道
func (m *Manager) Register(channel Channel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels[channel.Name()] = channel
}

// Get 获取渠道
func (m *Manager) Get(name string) Channel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.channels[name]
}

// StartAll 启动所有渠道
func (m *Manager) StartAll(ctx context.Context) error {
	m.mu.RLock()
	channels := make([]Channel, 0, len(m.channels))
	for _, ch := range m.channels {
		channels = append(channels, ch)
	}
	m.mu.RUnlock()

	for _, ch := range channels {
		if err := ch.Start(ctx); err != nil {
			return fmt.Errorf("start channel %s: %w", ch.Name(), err)
		}
	}
	return nil
}

// StopAll 停止所有渠道
func (m *Manager) StopAll() {
	m.mu.RLock()
	channels := make([]Channel, 0, len(m.channels))
	for _, ch := range m.channels {
		channels = append(channels, ch)
	}
	m.mu.RUnlock()

	for _, ch := range channels {
		ch.Stop()
	}
}

// List 列出所有渠道名称
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var names []string
	for name := range m.channels {
		names = append(names, name)
	}
	return names
}
