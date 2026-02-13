package channels

import (
	"context"

	"github.com/weibaohui/nanobot-go/bus"
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
func (c *BaseChannel) SubscribeOutbound(ctx context.Context, handler func(msg *bus.OutboundMessage)) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := c.bus.ConsumeOutbound(ctx)
				if err != nil {
					continue
				}
				if msg.Channel == c.name {
					handler(msg)
				}
			}
		}
	}()
}

// Manager 渠道管理器
type Manager struct {
	channels map[string]Channel
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
	m.channels[channel.Name()] = channel
}

// Get 获取渠道
func (m *Manager) Get(name string) Channel {
	return m.channels[name]
}

// StartAll 启动所有渠道
func (m *Manager) StartAll(ctx context.Context) error {
	for _, ch := range m.channels {
		if err := ch.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

// StopAll 停止所有渠道
func (m *Manager) StopAll() {
	for _, ch := range m.channels {
		ch.Stop()
	}
}

// List 列出所有渠道名称
func (m *Manager) List() []string {
	var names []string
	for name := range m.channels {
		names = append(names, name)
	}
	return names
}
