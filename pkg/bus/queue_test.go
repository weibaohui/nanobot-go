package bus

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

// TestNewMessageBus 测试创建消息总线
func TestNewMessageBus(t *testing.T) {
	t.Run("使用nil logger", func(t *testing.T) {
		bus := NewMessageBus(nil)
		if bus == nil {
			t.Fatal("NewMessageBus 返回 nil")
		}
		if bus.logger == nil {
			t.Error("logger 不应该为 nil")
		}
	})

	t.Run("使用自定义logger", func(t *testing.T) {
		logger := zap.NewNop()
		bus := NewMessageBus(logger)
		if bus.logger != logger {
			t.Error("logger 应该是传入的 logger")
		}
	})
}

// TestMessageBus_PublishConsumeInbound 测试入站消息的发布和消费
func TestMessageBus_PublishConsumeInbound(t *testing.T) {
	bus := NewMessageBus(nil)
	ctx := context.Background()

	msg := NewInboundMessage("test", "user1", "chat1", "hello")

	go func() {
		bus.PublishInbound(msg)
	}()

	consumed, err := bus.ConsumeInbound(ctx)
	if err != nil {
		t.Fatalf("ConsumeInbound 返回错误: %v", err)
	}

	if consumed.Channel != "test" {
		t.Errorf("Channel = %q, 期望 test", consumed.Channel)
	}

	if consumed.Content != "hello" {
		t.Errorf("Content = %q, 期望 hello", consumed.Content)
	}
}

// TestMessageBus_ConsumeInbound_ContextCancel 测试上下文取消时的消费
func TestMessageBus_ConsumeInbound_ContextCancel(t *testing.T) {
	bus := NewMessageBus(nil)
	ctx, cancel := context.WithCancel(context.Background())

	var err error
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		_, err = bus.ConsumeInbound(ctx)
	}()

	cancel()
	wg.Wait()

	if err != context.Canceled {
		t.Errorf("错误 = %v, 期望 context.Canceled", err)
	}
}

// TestMessageBus_PublishConsumeOutbound 测试出站消息的发布和消费
func TestMessageBus_PublishConsumeOutbound(t *testing.T) {
	bus := NewMessageBus(nil)
	ctx := context.Background()

	msg := NewOutboundMessage("test", "chat1", "reply")

	go func() {
		bus.PublishOutbound(msg)
	}()

	consumed, err := bus.ConsumeOutbound(ctx)
	if err != nil {
		t.Fatalf("ConsumeOutbound 返回错误: %v", err)
	}

	if consumed.Channel != "test" {
		t.Errorf("Channel = %q, 期望 test", consumed.Channel)
	}

	if consumed.Content != "reply" {
		t.Errorf("Content = %q, 期望 reply", consumed.Content)
	}
}

// TestMessageBus_PublishStream 测试流式消息发布
func TestMessageBus_PublishStream(t *testing.T) {
	bus := NewMessageBus(nil)

	chunk := NewStreamChunk("test", "chat1", "delta", "content", false)

	bus.PublishStream(chunk)

	if bus.StreamSize() != 1 {
		t.Errorf("StreamSize = %d, 期望 1", bus.StreamSize())
	}
}

// TestMessageBus_PublishStream_FullChannel 测试流式消息通道满时丢弃
func TestMessageBus_PublishStream_FullChannel(t *testing.T) {
	bus := NewMessageBus(nil)

	for i := 0; i < 1100; i++ {
		chunk := NewStreamChunk("test", "chat1", "delta", "content", false)
		bus.PublishStream(chunk)
	}

	if bus.StreamSize() > 1000 {
		t.Errorf("StreamSize = %d, 应该不超过 1000", bus.StreamSize())
	}
}

// TestMessageBus_SubscribeOutbound 测试出站消息订阅
func TestMessageBus_SubscribeOutbound(t *testing.T) {
	bus := NewMessageBus(nil)

	var receivedMsg *OutboundMessage
	var mu sync.Mutex

	bus.SubscribeOutbound("test", func(msg *OutboundMessage) error {
		mu.Lock()
		defer mu.Unlock()
		receivedMsg = msg
		return nil
	})

	ctx := context.Background()
	bus.StartDispatcher(ctx)
	defer bus.Stop()

	msg := NewOutboundMessage("test", "chat1", "hello")
	bus.PublishOutbound(msg)

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if receivedMsg == nil {
		t.Fatal("未收到消息")
	}

	if receivedMsg.Content != "hello" {
		t.Errorf("Content = %q, 期望 hello", receivedMsg.Content)
	}
}

// TestMessageBus_SubscribeStream 测试流式消息订阅
func TestMessageBus_SubscribeStream(t *testing.T) {
	bus := NewMessageBus(nil)

	var receivedChunks []*StreamChunk
	var mu sync.Mutex

	bus.SubscribeStream("test", func(chunk *StreamChunk) error {
		mu.Lock()
		defer mu.Unlock()
		receivedChunks = append(receivedChunks, chunk)
		return nil
	})

	ctx := context.Background()
	bus.StartDispatcher(ctx)
	defer bus.Stop()

	chunk1 := NewStreamChunk("test", "chat1", "delta1", "content1", false)
	chunk2 := NewStreamChunk("test", "chat1", "delta2", "content2", true)

	bus.PublishStream(chunk1)
	bus.PublishStream(chunk2)

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(receivedChunks) != 2 {
		t.Fatalf("收到 %d 个片段, 期望 2", len(receivedChunks))
	}

	if receivedChunks[0].Delta != "delta1" {
		t.Errorf("第一个片段 Delta = %q, 期望 delta1", receivedChunks[0].Delta)
	}
}

// TestMessageBus_InboundSize 测试入站消息队列大小
func TestMessageBus_InboundSize(t *testing.T) {
	bus := NewMessageBus(nil)

	if bus.InboundSize() != 0 {
		t.Errorf("初始 InboundSize = %d, 期望 0", bus.InboundSize())
	}

	bus.PublishInbound(NewInboundMessage("test", "user", "chat", "msg1"))
	bus.PublishInbound(NewInboundMessage("test", "user", "chat", "msg2"))

	if bus.InboundSize() != 2 {
		t.Errorf("InboundSize = %d, 期望 2", bus.InboundSize())
	}
}

// TestMessageBus_OutboundSize 测试出站消息队列大小
func TestMessageBus_OutboundSize(t *testing.T) {
	bus := NewMessageBus(nil)

	if bus.OutboundSize() != 0 {
		t.Errorf("初始 OutboundSize = %d, 期望 0", bus.OutboundSize())
	}

	bus.PublishOutbound(NewOutboundMessage("test", "chat", "msg1"))
	bus.PublishOutbound(NewOutboundMessage("test", "chat", "msg2"))

	if bus.OutboundSize() != 2 {
		t.Errorf("OutboundSize = %d, 期望 2", bus.OutboundSize())
	}
}

// TestMessageBus_Stop 测试停止消息总线
func TestMessageBus_Stop(t *testing.T) {
	bus := NewMessageBus(nil)

	ctx := context.Background()
	bus.StartDispatcher(ctx)

	if !bus.running {
		t.Error("消息总线应该正在运行")
	}

	bus.Stop()

	if bus.running {
		t.Error("消息总线应该已停止")
	}
}

// TestMessageBus_MultipleSubscribers 测试多个订阅者
func TestMessageBus_MultipleSubscribers(t *testing.T) {
	bus := NewMessageBus(nil)

	var count int
	var mu sync.Mutex

	bus.SubscribeOutbound("test", func(msg *OutboundMessage) error {
		mu.Lock()
		defer mu.Unlock()
		count++
		return nil
	})

	bus.SubscribeOutbound("test", func(msg *OutboundMessage) error {
		mu.Lock()
		defer mu.Unlock()
		count++
		return nil
	})

	ctx := context.Background()
	bus.StartDispatcher(ctx)
	defer bus.Stop()

	bus.PublishOutbound(NewOutboundMessage("test", "chat", "msg"))

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if count != 2 {
		t.Errorf("订阅者调用次数 = %d, 期望 2", count)
	}
}

// TestMessageBus_SubscriberError 测试订阅者返回错误
func TestMessageBus_SubscriberError(t *testing.T) {
	bus := NewMessageBus(zap.NewNop())

	var called bool
	var mu sync.Mutex

	bus.SubscribeOutbound("test", func(msg *OutboundMessage) error {
		return nil
	})

	bus.SubscribeOutbound("test", func(msg *OutboundMessage) error {
		mu.Lock()
		defer mu.Unlock()
		called = true
		return nil
	})

	ctx := context.Background()
	bus.StartDispatcher(ctx)
	defer bus.Stop()

	bus.PublishOutbound(NewOutboundMessage("test", "chat", "msg"))

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if !called {
		t.Error("第二个订阅者应该被调用")
	}
}
