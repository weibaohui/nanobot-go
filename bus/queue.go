package bus

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// OutboundCallback 是出站消息的回调函数类型
type OutboundCallback func(msg *OutboundMessage) error

// MessageBus 是解耦渠道和代理核心的异步消息总线
type MessageBus struct {
	inbound             chan *InboundMessage
	outbound            chan *OutboundMessage
	outboundSubscribers map[string][]OutboundCallback
	mu                  sync.RWMutex
	running             bool
	logger              *zap.Logger
}

// NewMessageBus 创建一个新的消息总线
func NewMessageBus(logger *zap.Logger) *MessageBus {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &MessageBus{
		inbound:             make(chan *InboundMessage, 100),
		outbound:            make(chan *OutboundMessage, 100),
		outboundSubscribers: make(map[string][]OutboundCallback),
		logger:              logger,
	}
}

// PublishInbound 从渠道向代理发布消息
func (b *MessageBus) PublishInbound(msg *InboundMessage) {
	b.inbound <- msg
}

// ConsumeInbound 消费下一条入站消息（阻塞直到可用）
func (b *MessageBus) ConsumeInbound(ctx context.Context) (*InboundMessage, error) {
	select {
	case msg := <-b.inbound:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// PublishOutbound 从代理向渠道发布响应
func (b *MessageBus) PublishOutbound(msg *OutboundMessage) {
	b.outbound <- msg
}

// ConsumeOutbound 消费下一条出站消息（阻塞直到可用）
func (b *MessageBus) ConsumeOutbound(ctx context.Context) (*OutboundMessage, error) {
	select {
	case msg := <-b.outbound:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// SubscribeOutbound 订阅特定渠道的出站消息
func (b *MessageBus) SubscribeOutbound(channel string, callback OutboundCallback) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.outboundSubscribers[channel] = append(b.outboundSubscribers[channel], callback)
}

// StartDispatcher 启动出站消息分发器
func (b *MessageBus) StartDispatcher(ctx context.Context) {
	b.running = true
	go b.dispatchLoop(ctx)
}

// dispatchLoop 分发出站消息给订阅的渠道
func (b *MessageBus) dispatchLoop(ctx context.Context) {
	for b.running {
		select {
		case msg := <-b.outbound:
			b.dispatchToSubscribers(msg)
		case <-ctx.Done():
			b.running = false
			return
		case <-time.After(1 * time.Second):
			// 继续循环
		}
	}
}

// dispatchToSubscribers 将消息分发给订阅者
func (b *MessageBus) dispatchToSubscribers(msg *OutboundMessage) {
	b.mu.RLock()
	subscribers := b.outboundSubscribers[msg.Channel]
	b.mu.RUnlock()

	for _, callback := range subscribers {
		if err := callback(msg); err != nil {
			b.logger.Error("分发消息到渠道失败",
				zap.String("channel", msg.Channel),
				zap.Error(err),
			)
		}
	}
}

// Stop 停止分发器循环
func (b *MessageBus) Stop() {
	b.running = false
}

// InboundSize 返回待处理的入站消息数量
func (b *MessageBus) InboundSize() int {
	return len(b.inbound)
}

// OutboundSize 返回待处理的出站消息数量
func (b *MessageBus) OutboundSize() int {
	return len(b.outbound)
}
