package dispatcher

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/observer"
	"github.com/weibaohui/nanobot-go/pkg/bus"
	"go.uber.org/zap"
)

// mockObserver 模拟观察器
type mockObserver struct {
	name         string
	enabled      bool
	eventCount   int
	lastEvent    events.Event
	onEventFunc  func(ctx context.Context, event events.Event) error
	mu           sync.Mutex
}

func (m *mockObserver) Name() string {
	return m.name
}

func (m *mockObserver) Enabled() bool {
	return m.enabled
}

func (m *mockObserver) OnEvent(ctx context.Context, event events.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventCount++
	m.lastEvent = event
	if m.onEventFunc != nil {
		return m.onEventFunc(ctx, event)
	}
	return nil
}

func (m *mockObserver) GetEventCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.eventCount
}

func (m *mockObserver) GetLastEvent() events.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastEvent
}

// TestNewDispatcher 测试创建事件分发器
func TestNewDispatcher(t *testing.T) {
	t.Run("正常创建", func(t *testing.T) {
		logger := zap.NewNop()
		d := NewDispatcher(logger)

		if d == nil {
			t.Fatal("NewDispatcher 返回 nil")
		}

		if len(d.observers) != 0 {
			t.Errorf("初始观察器数量 = %d, 期望 0", len(d.observers))
		}
	})

	t.Run("nil logger 使用默认", func(t *testing.T) {
		d := NewDispatcher(nil)

		if d == nil {
			t.Fatal("NewDispatcher 返回 nil")
		}

		if d.logger == nil {
			t.Error("logger 不应该为 nil")
		}
	})
}

// TestDispatcher_Register 测试注册观察器
func TestDispatcher_Register(t *testing.T) {
	d := NewDispatcher(zap.NewNop())

	t.Run("注册单个观察器", func(t *testing.T) {
		obs := &mockObserver{name: "test-observer", enabled: true}
		d.Register(obs)

		if d.Count() != 1 {
			t.Errorf("观察器数量 = %d, 期望 1", d.Count())
		}
	})

	t.Run("注册多个观察器", func(t *testing.T) {
		d2 := NewDispatcher(zap.NewNop())
		obs1 := &mockObserver{name: "observer-1", enabled: true}
		obs2 := &mockObserver{name: "observer-2", enabled: true}

		d2.Register(obs1)
		d2.Register(obs2)

		if d2.Count() != 2 {
			t.Errorf("观察器数量 = %d, 期望 2", d2.Count())
		}
	})

	t.Run("注册 nil 观察器", func(t *testing.T) {
		d2 := NewDispatcher(zap.NewNop())
		d2.Register(nil)

		if d2.Count() != 0 {
			t.Errorf("观察器数量 = %d, 期望 0 (nil 应该被忽略)", d2.Count())
		}
	})

	t.Run("重复注册同名观察器", func(t *testing.T) {
		d2 := NewDispatcher(zap.NewNop())
		obs1 := &mockObserver{name: "same-name", enabled: true}
		obs2 := &mockObserver{name: "same-name", enabled: true}

		d2.Register(obs1)
		d2.Register(obs2)

		if d2.Count() != 2 {
			t.Errorf("观察器数量 = %d, 期望 2 (允许同名)", d2.Count())
		}
	})
}

// TestDispatcher_Unregister 测试注销观察器
func TestDispatcher_Unregister(t *testing.T) {
	t.Run("注销存在的观察器", func(t *testing.T) {
		d := NewDispatcher(zap.NewNop())
		obs := &mockObserver{name: "to-remove", enabled: true}
		d.Register(obs)

		d.Unregister("to-remove")

		if d.Count() != 0 {
			t.Errorf("观察器数量 = %d, 期望 0", d.Count())
		}
	})

	t.Run("注销不存在的观察器", func(t *testing.T) {
		d := NewDispatcher(zap.NewNop())
		obs := &mockObserver{name: "existing", enabled: true}
		d.Register(obs)

		d.Unregister("non-existing")

		if d.Count() != 1 {
			t.Errorf("观察器数量 = %d, 期望 1", d.Count())
		}
	})

	t.Run("注销多个中的某一个", func(t *testing.T) {
		d := NewDispatcher(zap.NewNop())
		d.Register(&mockObserver{name: "obs1", enabled: true})
		d.Register(&mockObserver{name: "obs2", enabled: true})
		d.Register(&mockObserver{name: "obs3", enabled: true})

		d.Unregister("obs2")

		names := d.List()
		if len(names) != 2 {
			t.Errorf("观察器数量 = %d, 期望 2", len(names))
		}

		for _, name := range names {
			if name == "obs2" {
				t.Error("obs2 应该被注销")
			}
		}
	})
}

// TestDispatcher_Dispatch 测试事件分发
func TestDispatcher_Dispatch(t *testing.T) {
	t.Run("分发到单个观察器", func(t *testing.T) {
		d := NewDispatcher(zap.NewNop())
		obs := &mockObserver{name: "test", enabled: true}
		d.Register(obs)

		msg := &bus.InboundMessage{
			Channel:  "test-channel",
			ChatID:   "chat-1",
			SenderID: "user-1",
			Content:  "Hello",
		}
		event := events.NewMessageReceivedEvent("trace-1", "span-1", "", msg)
		d.Dispatch(context.Background(), event, "test-channel", "test-session")

		// 等待异步处理完成
		time.Sleep(100 * time.Millisecond)

		if obs.GetEventCount() != 1 {
			t.Errorf("观察器收到事件数 = %d, 期望 1", obs.GetEventCount())
		}
	})

	t.Run("分发到多个观察器", func(t *testing.T) {
		d := NewDispatcher(zap.NewNop())
		obs1 := &mockObserver{name: "obs1", enabled: true}
		obs2 := &mockObserver{name: "obs2", enabled: true}
		d.Register(obs1)
		d.Register(obs2)

		msg := &bus.InboundMessage{
			Channel:  "test-channel",
			ChatID:   "chat-1",
			SenderID: "user-1",
			Content:  "Hello",
		}
		event := events.NewMessageReceivedEvent("trace-1", "span-1", "", msg)
		d.Dispatch(context.Background(), event, "test-channel", "test-session")

		time.Sleep(100 * time.Millisecond)

		if obs1.GetEventCount() != 1 {
			t.Errorf("obs1 收到事件数 = %d, 期望 1", obs1.GetEventCount())
		}
		if obs2.GetEventCount() != 1 {
			t.Errorf("obs2 收到事件数 = %d, 期望 1", obs2.GetEventCount())
		}
	})

	t.Run("禁用观察器不接收事件", func(t *testing.T) {
		d := NewDispatcher(zap.NewNop())
		enabledObs := &mockObserver{name: "enabled", enabled: true}
		disabledObs := &mockObserver{name: "disabled", enabled: false}
		d.Register(enabledObs)
		d.Register(disabledObs)

		msg := &bus.InboundMessage{
			Channel:  "test-channel",
			ChatID:   "chat-1",
			SenderID: "user-1",
			Content:  "Hello",
		}
		event := events.NewMessageReceivedEvent("trace-1", "span-1", "", msg)
		d.Dispatch(context.Background(), event, "test-channel", "test-session")

		time.Sleep(100 * time.Millisecond)

		if enabledObs.GetEventCount() != 1 {
			t.Errorf("启用观察器收到事件数 = %d, 期望 1", enabledObs.GetEventCount())
		}
		if disabledObs.GetEventCount() != 0 {
			t.Errorf("禁用观察器收到事件数 = %d, 期望 0", disabledObs.GetEventCount())
		}
	})

	t.Run("空观察器列表", func(t *testing.T) {
		d := NewDispatcher(zap.NewNop())
		msg := &bus.InboundMessage{
			Channel:  "test-channel",
			ChatID:   "chat-1",
			SenderID: "user-1",
			Content:  "Hello",
		}
		event := events.NewMessageReceivedEvent("trace-1", "span-1", "", msg)

		// 不应该 panic
		d.Dispatch(context.Background(), event, "test-channel", "test-session")
	})

	t.Run("观察器处理错误不影响其他", func(t *testing.T) {
		d := NewDispatcher(zap.NewNop())
		errObs := &mockObserver{
			name:    "error-obs",
			enabled: true,
			onEventFunc: func(ctx context.Context, event events.Event) error {
				return errors.New("处理错误")
			},
		}
		normalObs := &mockObserver{name: "normal-obs", enabled: true}

		d.Register(errObs)
		d.Register(normalObs)

		msg := &bus.InboundMessage{
			Channel:  "test-channel",
			ChatID:   "chat-1",
			SenderID: "user-1",
			Content:  "Hello",
		}
		event := events.NewMessageReceivedEvent("trace-1", "span-1", "", msg)
		d.Dispatch(context.Background(), event, "test-channel", "test-session")

		time.Sleep(100 * time.Millisecond)

		if normalObs.GetEventCount() != 1 {
			t.Errorf("正常观察器应该收到事件，实际 = %d", normalObs.GetEventCount())
		}
	})
}

// TestDispatcher_List 测试列出观察器
func TestDispatcher_List(t *testing.T) {
	d := NewDispatcher(zap.NewNop())

	t.Run("空列表", func(t *testing.T) {
		names := d.List()
		if len(names) != 0 {
			t.Errorf("空列表长度 = %d, 期望 0", len(names))
		}
	})

	t.Run("列出所有观察器", func(t *testing.T) {
		d.Register(&mockObserver{name: "obs1", enabled: true})
		d.Register(&mockObserver{name: "obs2", enabled: true})
		d.Register(&mockObserver{name: "obs3", enabled: true})

		names := d.List()
		if len(names) != 3 {
			t.Errorf("列表长度 = %d, 期望 3", len(names))
		}

		// 检查是否包含所有观察器
		nameMap := make(map[string]bool)
		for _, name := range names {
			nameMap[name] = true
		}

		for _, expected := range []string{"obs1", "obs2", "obs3"} {
			if !nameMap[expected] {
				t.Errorf("列表中缺少 %s", expected)
			}
		}
	})
}

// TestDispatcher_Count 测试观察器计数
func TestDispatcher_Count(t *testing.T) {
	d := NewDispatcher(zap.NewNop())

	if d.Count() != 0 {
		t.Errorf("初始计数 = %d, 期望 0", d.Count())
	}

	d.Register(&mockObserver{name: "obs1", enabled: true})
	if d.Count() != 1 {
		t.Errorf("注册后计数 = %d, 期望 1", d.Count())
	}

	d.Register(&mockObserver{name: "obs2", enabled: true})
	if d.Count() != 2 {
		t.Errorf("注册后计数 = %d, 期望 2", d.Count())
	}

	d.Unregister("obs1")
	if d.Count() != 1 {
		t.Errorf("注销后计数 = %d, 期望 1", d.Count())
	}
}

// TestDispatcher_Concurrent 测试并发安全
func TestDispatcher_Concurrent(t *testing.T) {
	d := NewDispatcher(zap.NewNop())
	obs := &mockObserver{name: "concurrent", enabled: true}
	d.Register(obs)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			msg := &bus.InboundMessage{
				Channel:  "test-channel",
				ChatID:   "chat-1",
				SenderID: "user-1",
				Content:  "Hello",
			}
			event := events.NewMessageReceivedEvent(
				"trace-"+string(rune('0'+i%10)),
				"span-"+string(rune('0'+i%10)),
				"",
				msg,
			)
			d.Dispatch(context.Background(), event, "channel", "session")
		}(i)
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	if obs.GetEventCount() != 100 {
		t.Errorf("并发事件数 = %d, 期望 100", obs.GetEventCount())
	}
}

// TestObserverFilter_ShouldNotify 测试观察器过滤器
func TestObserverFilter_ShouldNotify(t *testing.T) {
	tests := []struct {
		name       string
		filter     *observer.ObserverFilter
		eventType  events.EventType
		channel    string
		sessionKey string
		expected   bool
	}{
		{
			name:       "空过滤器允许所有",
			filter:     &observer.ObserverFilter{},
			eventType:  events.EventMessageReceived,
			channel:    "any",
			sessionKey: "any",
			expected:   true,
		},
		{
			name: "匹配事件类型",
			filter: &observer.ObserverFilter{
				EventTypes: []events.EventType{events.EventMessageReceived, events.EventMessageSent},
			},
			eventType:  events.EventMessageReceived,
			channel:    "any",
			sessionKey: "any",
			expected:   true,
		},
		{
			name: "不匹配事件类型",
			filter: &observer.ObserverFilter{
				EventTypes: []events.EventType{events.EventMessageReceived},
			},
			eventType:  events.EventMessageSent,
			channel:    "any",
			sessionKey: "any",
			expected:   false,
		},
		{
			name: "匹配渠道",
			filter: &observer.ObserverFilter{
				Channels: []string{"dingtalk", "matrix"},
			},
			eventType:  events.EventMessageReceived,
			channel:    "dingtalk",
			sessionKey: "any",
			expected:   true,
		},
		{
			name: "不匹配渠道",
			filter: &observer.ObserverFilter{
				Channels: []string{"dingtalk"},
			},
			eventType:  events.EventMessageReceived,
			channel:    "matrix",
			sessionKey: "any",
			expected:   false,
		},
		{
			name: "空渠道字段不匹配",
			filter: &observer.ObserverFilter{
				Channels: []string{"dingtalk"},
			},
			eventType:  events.EventMessageReceived,
			channel:    "",
			sessionKey: "any",
			expected:   true,
		},
		{
			name: "匹配会话",
			filter: &observer.ObserverFilter{
				SessionKeys: []string{"session-1", "session-2"},
			},
			eventType:  events.EventMessageReceived,
			channel:    "any",
			sessionKey: "session-1",
			expected:   true,
		},
		{
			name: "组合条件全部匹配",
			filter: &observer.ObserverFilter{
				EventTypes:  []events.EventType{events.EventMessageReceived},
				Channels:    []string{"dingtalk"},
				SessionKeys: []string{"session-1"},
			},
			eventType:  events.EventMessageReceived,
			channel:    "dingtalk",
			sessionKey: "session-1",
			expected:   true,
		},
		{
			name: "组合条件部分匹配",
			filter: &observer.ObserverFilter{
				EventTypes: []events.EventType{events.EventMessageReceived},
				Channels:   []string{"dingtalk"},
			},
			eventType:  events.EventMessageReceived,
			channel:    "matrix",
			sessionKey: "session-1",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.ShouldNotify(tt.eventType, tt.channel, tt.sessionKey)
			if result != tt.expected {
				t.Errorf("ShouldNotify() = %v, 期望 %v", result, tt.expected)
			}
		})
	}
}

// TestBaseObserver 测试基础观察器
func TestBaseObserver(t *testing.T) {
	t.Run("创建基础观察器", func(t *testing.T) {
		filter := &observer.ObserverFilter{Channels: []string{"test"}}
		obs := observer.NewBaseObserver("test-obs", filter)

		if obs.Name() != "test-obs" {
			t.Errorf("Name() = %q, 期望 test-obs", obs.Name())
		}

		if !obs.Enabled() {
			t.Error("新创建的观察器应该启用")
		}

		if obs.Filter() != filter {
			t.Error("Filter() 应该返回创建时的过滤器")
		}
	})

	t.Run("nil 过滤器使用空过滤器", func(t *testing.T) {
		obs := observer.NewBaseObserver("test-obs", nil)

		if obs.Filter() == nil {
			t.Error("nil 过滤器应该被替换为空过滤器")
		}
	})

	t.Run("设置启用状态", func(t *testing.T) {
		obs := observer.NewBaseObserver("test-obs", nil)

		obs.SetEnabled(false)
		if obs.Enabled() {
			t.Error("SetEnabled(false) 后应该禁用")
		}

		obs.SetEnabled(true)
		if !obs.Enabled() {
			t.Error("SetEnabled(true) 后应该启用")
		}
	})

	t.Run("ShouldNotify 使用过滤器", func(t *testing.T) {
		filter := &observer.ObserverFilter{Channels: []string{"dingtalk"}}
		obs := observer.NewBaseObserver("test-obs", filter)

		if !obs.ShouldNotify(events.EventMessageReceived, "dingtalk", "session-1") {
			t.Error("应该匹配 dingtalk 渠道")
		}

		if obs.ShouldNotify(events.EventMessageReceived, "matrix", "session-1") {
			t.Error("不应该匹配 matrix 渠道")
		}
	})

	t.Run("默认 OnEvent 返回 nil", func(t *testing.T) {
		obs := observer.NewBaseObserver("test-obs", nil)
		err := obs.OnEvent(context.Background(), nil)
		if err != nil {
			t.Errorf("默认 OnEvent 应该返回 nil, 实际 = %v", err)
		}
	})
}
