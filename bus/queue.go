package bus

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// OutboundCallback 是出站消息的回调函数类型
type OutboundCallback func(msg *OutboundMessage) error

// StreamCallback 是流式消息的回调函数类型
type StreamCallback func(chunk *StreamChunk) error

// MessageBus 是解耦渠道和代理核心的异步消息总线
type MessageBus struct {
	inbound             chan *InboundMessage
	outbound            chan *OutboundMessage
	stream              chan *StreamChunk
	outboundSubscribers map[string][]OutboundCallback
	streamSubscribers   map[string][]StreamCallback
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
		stream:              make(chan *StreamChunk, 1000), // 流式消息需要更大的缓冲
		outboundSubscribers: make(map[string][]OutboundCallback),
		streamSubscribers:   make(map[string][]StreamCallback),
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

// PublishStream 发布流式消息片段
func (b *MessageBus) PublishStream(chunk *StreamChunk) {
	select {
	case b.stream <- chunk:
	default:
		b.logger.Warn("流式消息通道已满，丢弃片段",
			zap.String("channel", chunk.Channel),
			zap.String("chat_id", chunk.ChatID),
		)
	}
}

// SubscribeOutbound 订阅特定渠道的出站消息
func (b *MessageBus) SubscribeOutbound(channel string, callback OutboundCallback) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.outboundSubscribers[channel] = append(b.outboundSubscribers[channel], callback)
	b.logger.Info("订阅出站消息", zap.String("channel", channel), zap.Int("当前订阅者数量", len(b.outboundSubscribers[channel])))
}

// SubscribeStream 订阅特定渠道的流式消息
func (b *MessageBus) SubscribeStream(channel string, callback StreamCallback) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.streamSubscribers[channel] = append(b.streamSubscribers[channel], callback)
}

// StartDispatcher 启动出站消息分发器
func (b *MessageBus) StartDispatcher(ctx context.Context) {
	b.running = true
	go b.dispatchLoop(ctx)
	go b.streamDispatchLoop(ctx)
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

// streamDispatchLoop 分发流式消息给订阅的渠道
func (b *MessageBus) streamDispatchLoop(ctx context.Context) {
	for b.running {
		select {
		case chunk := <-b.stream:
			b.dispatchStreamToSubscribers(chunk)
		case <-ctx.Done():
			b.running = false
			return
		case <-time.After(100 * time.Millisecond):
			// 继续循环
		}
	}
}

// dispatchToSubscribers 将消息分发给订阅者
func (b *MessageBus) dispatchToSubscribers(msg *OutboundMessage) {
	b.mu.RLock()
	subscribers := b.outboundSubscribers[msg.Channel]
	b.mu.RUnlock()

	b.logger.Info("分发出站消息", zap.String("channel", msg.Channel), zap.Int("订阅者数量", len(subscribers)))

	for _, callback := range subscribers {
		if err := callback(msg); err != nil {
			b.logger.Error("分发消息到渠道失败",
				zap.String("channel", msg.Channel),
				zap.Error(err),
			)
		}
	}
}

// dispatchStreamToSubscribers 将流式消息分发给订阅者
func (b *MessageBus) dispatchStreamToSubscribers(chunk *StreamChunk) {
	b.mu.RLock()
	subscribers := b.streamSubscribers[chunk.Channel]
	b.mu.RUnlock()

	for _, callback := range subscribers {
		if err := callback(chunk); err != nil {
			b.logger.Error("分发流式消息到渠道失败",
				zap.String("channel", chunk.Channel),
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

// StreamSize 返回待处理的流式消息数量
func (b *MessageBus) StreamSize() int {
	return len(b.stream)
}
