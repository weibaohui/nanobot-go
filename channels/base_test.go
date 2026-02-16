package channels

import (
	"context"
	"testing"

	"github.com/weibaohui/nanobot-go/bus"
	"go.uber.org/zap"
)

// TestNewBaseChannel 测试创建渠道基类
func TestNewBaseChannel(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())
	channel := NewBaseChannel("test-channel", messageBus)

	if channel == nil {
		t.Fatal("NewBaseChannel 返回 nil")
	}

	if channel.Name() != "test-channel" {
		t.Errorf("Name() = %q, 期望 test-channel", channel.Name())
	}
}

// TestBaseChannel_Name 测试获取渠道名称
func TestBaseChannel_Name(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())
	channel := NewBaseChannel("websocket", messageBus)

	if channel.Name() != "websocket" {
		t.Errorf("Name() = %q, 期望 websocket", channel.Name())
	}
}

// TestBaseChannel_Start 测试默认启动实现
func TestBaseChannel_Start(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())
	channel := NewBaseChannel("test", messageBus)

	ctx := context.Background()
	err := channel.Start(ctx)
	if err != nil {
		t.Errorf("Start() 返回错误: %v", err)
	}
}

// TestBaseChannel_Stop 测试默认停止实现
func TestBaseChannel_Stop(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())
	channel := NewBaseChannel("test", messageBus)

	channel.Stop()
}

// TestBaseChannel_PublishInbound 测试发布入站消息
func TestBaseChannel_PublishInbound(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())
	channel := NewBaseChannel("test", messageBus)

	msg := bus.NewInboundMessage("test", "user1", "chat1", "hello")
	channel.PublishInbound(msg)

	if messageBus.InboundSize() != 1 {
		t.Errorf("InboundSize = %d, 期望 1", messageBus.InboundSize())
	}
}

// TestNewManager 测试创建渠道管理器
func TestNewManager(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())
	manager := NewManager(messageBus)

	if manager == nil {
		t.Fatal("NewManager 返回 nil")
	}

	if manager.channels == nil {
		t.Error("channels 不应该为 nil")
	}
}

// TestManager_Register 测试注册渠道
func TestManager_Register(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())
	manager := NewManager(messageBus)

	channel := NewBaseChannel("test-channel", messageBus)
	manager.Register(channel)

	if len(manager.channels) != 1 {
		t.Errorf("channels 长度 = %d, 期望 1", len(manager.channels))
	}
}

// TestManager_Get 测试获取渠道
func TestManager_Get(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())
	manager := NewManager(messageBus)

	channel := NewBaseChannel("test-channel", messageBus)
	manager.Register(channel)

	retrieved := manager.Get("test-channel")
	if retrieved == nil {
		t.Fatal("Get 返回 nil")
	}

	if retrieved.Name() != "test-channel" {
		t.Errorf("Name() = %q, 期望 test-channel", retrieved.Name())
	}

	// 测试获取不存在的渠道
	notFound := manager.Get("nonexistent")
	if notFound != nil {
		t.Error("获取不存在的渠道应该返回 nil")
	}
}

// TestManager_StartAll 测试启动所有渠道
func TestManager_StartAll(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())
	manager := NewManager(messageBus)

	channel1 := NewBaseChannel("channel1", messageBus)
	channel2 := NewBaseChannel("channel2", messageBus)

	manager.Register(channel1)
	manager.Register(channel2)

	ctx := context.Background()
	err := manager.StartAll(ctx)
	if err != nil {
		t.Errorf("StartAll 返回错误: %v", err)
	}
}

// TestManager_StopAll 测试停止所有渠道
func TestManager_StopAll(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())
	manager := NewManager(messageBus)

	channel1 := NewBaseChannel("channel1", messageBus)
	channel2 := NewBaseChannel("channel2", messageBus)

	manager.Register(channel1)
	manager.Register(channel2)

	manager.StopAll()
}

// TestManager_List 测试列出所有渠道名称
func TestManager_List(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())
	manager := NewManager(messageBus)

	channel1 := NewBaseChannel("channel1", messageBus)
	channel2 := NewBaseChannel("channel2", messageBus)
	channel3 := NewBaseChannel("channel3", messageBus)

	manager.Register(channel1)
	manager.Register(channel2)
	manager.Register(channel3)

	names := manager.List()
	if len(names) != 3 {
		t.Errorf("List 返回 %d 个名称, 期望 3", len(names))
	}
}

// TestManager_List_Empty 测试空管理器列表
func TestManager_List_Empty(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())
	manager := NewManager(messageBus)

	names := manager.List()
	if len(names) != 0 {
		t.Errorf("List 返回 %d 个名称, 期望 0", len(names))
	}
}

// mockChannel 用于测试的模拟渠道
type mockChannel struct {
	name      string
	startErr  error
	started   bool
	stopped   bool
}

func (m *mockChannel) Name() string {
	return m.name
}

func (m *mockChannel) Start(ctx context.Context) error {
	m.started = true
	return m.startErr
}

func (m *mockChannel) Stop() {
	m.stopped = true
}

// TestManager_StartAll_WithError 测试启动时遇到错误
func TestManager_StartAll_WithError(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())
	manager := NewManager(messageBus)

	// 注册一个会返回错误的渠道
	errorChannel := &mockChannel{
		name:     "error-channel",
		startErr: context.Canceled,
	}
	manager.Register(errorChannel)

	ctx := context.Background()
	err := manager.StartAll(ctx)
	if err == nil {
		t.Error("StartAll 应该返回错误")
	}
}

// TestManager_MultipleOperations 测试多个操作组合
func TestManager_MultipleOperations(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())
	manager := NewManager(messageBus)

	// 注册多个不同名称的渠道
	names := []string{"channel1", "channel2", "channel3", "channel4", "channel5"}
	for _, name := range names {
		channel := NewBaseChannel(name, messageBus)
		manager.Register(channel)
	}

	// 列出渠道
	listNames := manager.List()
	if len(listNames) != 5 {
		t.Errorf("List 返回 %d 个名称, 期望 5", len(listNames))
	}

	// 启动所有
	ctx := context.Background()
	if err := manager.StartAll(ctx); err != nil {
		t.Errorf("StartAll 返回错误: %v", err)
	}

	// 停止所有
	manager.StopAll()
}
