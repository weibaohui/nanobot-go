package hooks

import (
	"context"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/pkg/bus"
	"go.uber.org/zap"
)

// mockObserver 模拟观察器
type mockObserver struct {
	name        string
	enabled     bool
	eventCount  int
	lastEvent   events.Event
	onEventFunc func(ctx context.Context, event events.Event) error
}

func (m *mockObserver) Name() string  { return m.name }
func (m *mockObserver) Enabled() bool { return m.enabled }
func (m *mockObserver) OnEvent(ctx context.Context, event events.Event) error {
	m.eventCount++
	m.lastEvent = event
	if m.onEventFunc != nil {
		return m.onEventFunc(ctx, event)
	}
	return nil
}

// TestNewHookManager 测试创建 Hook 管理器
func TestNewHookManager(t *testing.T) {
	t.Run("正常创建", func(t *testing.T) {
		logger := zap.NewNop()
		hm := NewHookManager(logger, true)

		if hm == nil {
			t.Fatal("NewHookManager 返回 nil")
		}

		if hm.dispatcher == nil {
			t.Error("dispatcher 不应该为 nil")
		}

		if hm.einoBridge == nil {
			t.Error("einoBridge 不应该为 nil")
		}

		if !hm.enabled {
			t.Error("enabled 应该为 true")
		}
	})

	t.Run("nil logger 使用默认", func(t *testing.T) {
		hm := NewHookManager(nil, true)

		if hm == nil {
			t.Fatal("NewHookManager 返回 nil")
		}

		if hm.logger == nil {
			t.Error("logger 不应该为 nil")
		}
	})

	t.Run("禁用状态", func(t *testing.T) {
		hm := NewHookManager(zap.NewNop(), false)

		if hm.enabled {
			t.Error("enabled 应该为 false")
		}
	})
}

// TestHookManager_Register 测试注册观察器
func TestHookManager_Register(t *testing.T) {
	hm := NewHookManager(zap.NewNop(), true)
	obs := &mockObserver{name: "test-obs", enabled: true}

	hm.Register(obs)

	if hm.Count() != 1 {
		t.Errorf("观察器数量 = %d, 期望 1", hm.Count())
	}

	names := hm.List()
	if len(names) != 1 || names[0] != "test-obs" {
		t.Errorf("观察器列表 = %v, 期望 [test-obs]", names)
	}
}

// TestHookManager_Unregister 测试注销观察器
func TestHookManager_Unregister(t *testing.T) {
	hm := NewHookManager(zap.NewNop(), true)
	hm.Register(&mockObserver{name: "obs1", enabled: true})
	hm.Register(&mockObserver{name: "obs2", enabled: true})

	hm.Unregister("obs1")

	if hm.Count() != 1 {
		t.Errorf("观察器数量 = %d, 期望 1", hm.Count())
	}

	names := hm.List()
	if len(names) != 1 || names[0] != "obs2" {
		t.Errorf("观察器列表 = %v, 期望 [obs2]", names)
	}
}

// TestHookManager_Dispatch 测试事件分发
func TestHookManager_Dispatch(t *testing.T) {
	t.Run("启用时分发事件", func(t *testing.T) {
		hm := NewHookManager(zap.NewNop(), true)
		obs := &mockObserver{name: "test", enabled: true}
		hm.Register(obs)

		msg := &bus.InboundMessage{
			Channel:  "test-channel",
			ChatID:   "chat-1",
			SenderID: "user-1",
			Content:  "Hello",
		}
		event := events.NewMessageReceivedEvent("trace-1", "span-1", "", msg)
		hm.Dispatch(context.Background(), event, "test-channel", "test-session")

		time.Sleep(100 * time.Millisecond)

		if obs.eventCount != 1 {
			t.Errorf("观察器收到事件数 = %d, 期望 1", obs.eventCount)
		}
	})

	t.Run("禁用时不分发事件", func(t *testing.T) {
		hm := NewHookManager(zap.NewNop(), false)
		obs := &mockObserver{name: "test", enabled: true}
		hm.Register(obs)

		msg := &bus.InboundMessage{
			Channel:  "test-channel",
			ChatID:   "chat-1",
			SenderID: "user-1",
			Content:  "Hello",
		}
		event := events.NewMessageReceivedEvent("trace-1", "span-1", "", msg)
		hm.Dispatch(context.Background(), event, "test-channel", "test-session")

		time.Sleep(50 * time.Millisecond)

		if obs.eventCount != 0 {
			t.Errorf("禁用时不应该收到事件，实际 = %d", obs.eventCount)
		}
	})
}

// TestHookManager_Enabled 测试启用状态
func TestHookManager_Enabled(t *testing.T) {
	hm := NewHookManager(zap.NewNop(), true)

	if !hm.Enabled() {
		t.Error("初始状态应该启用")
	}

	hm.SetEnabled(false)
	if hm.Enabled() {
		t.Error("SetEnabled(false) 后应该禁用")
	}

	hm.SetEnabled(true)
	if !hm.Enabled() {
		t.Error("SetEnabled(true) 后应该启用")
	}
}

// TestHookManager_Count 测试观察器计数
func TestHookManager_Count(t *testing.T) {
	hm := NewHookManager(zap.NewNop(), true)

	if hm.Count() != 0 {
		t.Errorf("初始计数 = %d, 期望 0", hm.Count())
	}

	hm.Register(&mockObserver{name: "obs1", enabled: true})
	hm.Register(&mockObserver{name: "obs2", enabled: true})

	if hm.Count() != 2 {
		t.Errorf("注册后计数 = %d, 期望 2", hm.Count())
	}
}

// TestHookManager_List 测试观察器列表
func TestHookManager_List(t *testing.T) {
	hm := NewHookManager(zap.NewNop(), true)

	names := hm.List()
	if len(names) != 0 {
		t.Errorf("空列表长度 = %d, 期望 0", len(names))
	}

	hm.Register(&mockObserver{name: "obs1", enabled: true})
	hm.Register(&mockObserver{name: "obs2", enabled: true})

	names = hm.List()
	if len(names) != 2 {
		t.Errorf("列表长度 = %d, 期望 2", len(names))
	}
}

// TestHookManager_OnMessageReceived 测试消息接收事件
func TestHookManager_OnMessageReceived(t *testing.T) {
	t.Run("启用时触发事件", func(t *testing.T) {
		hm := NewHookManager(zap.NewNop(), true)
		obs := &mockObserver{name: "test", enabled: true}
		hm.Register(obs)

		msg := &bus.InboundMessage{
			Channel:  "dingtalk",
			ChatID:   "chat-1",
			SenderID: "user-1",
			Content:  "Hello",
		}
		hm.OnMessageReceived(context.Background(), msg)

		time.Sleep(100 * time.Millisecond)

		if obs.eventCount != 1 {
			t.Errorf("观察器收到事件数 = %d, 期望 1", obs.eventCount)
		}

		receivedEvent, ok := obs.lastEvent.(*events.MessageReceivedEvent)
		if !ok {
			t.Fatalf("事件类型错误，期望 *MessageReceivedEvent")
		}

		if receivedEvent.Channel != "dingtalk" {
			t.Errorf("事件 Channel = %q, 期望 dingtalk", receivedEvent.Channel)
		}
	})

	t.Run("禁用时不触发", func(t *testing.T) {
		hm := NewHookManager(zap.NewNop(), false)
		obs := &mockObserver{name: "test", enabled: true}
		hm.Register(obs)

		msg := &bus.InboundMessage{Channel: "dingtalk", Content: "Hello"}
		hm.OnMessageReceived(context.Background(), msg)

		time.Sleep(50 * time.Millisecond)

		if obs.eventCount != 0 {
			t.Errorf("禁用时不应该收到事件，实际 = %d", obs.eventCount)
		}
	})
}

// TestHookManager_OnMessageSent 测试消息发送事件
func TestHookManager_OnMessageSent(t *testing.T) {
	hm := NewHookManager(zap.NewNop(), true)
	obs := &mockObserver{name: "test", enabled: true}
	hm.Register(obs)

	msg := &bus.OutboundMessage{
		Channel: "dingtalk",
		ChatID:  "chat-1",
		Content: "Reply",
	}
	hm.OnMessageSent(context.Background(), msg, "dingtalk:chat-1")

	time.Sleep(100 * time.Millisecond)

	if obs.eventCount != 1 {
		t.Errorf("观察器收到事件数 = %d, 期望 1", obs.eventCount)
	}

	_, ok := obs.lastEvent.(*events.MessageSentEvent)
	if !ok {
		t.Fatalf("事件类型错误，期望 *MessageSentEvent")
	}
}

// TestHookManager_OnPromptSubmitted 测试 Prompt 提交事件
func TestHookManager_OnPromptSubmitted(t *testing.T) {
	hm := NewHookManager(zap.NewNop(), true)
	obs := &mockObserver{name: "test", enabled: true}
	hm.Register(obs)

	messages := []*schema.Message{
		{Role: "user", Content: "Hello"},
	}
	hm.OnPromptSubmitted(context.Background(), "Hello", messages, "session-1")

	time.Sleep(100 * time.Millisecond)

	if obs.eventCount != 1 {
		t.Errorf("观察器收到事件数 = %d, 期望 1", obs.eventCount)
	}

	event, ok := obs.lastEvent.(*events.PromptSubmittedEvent)
	if !ok {
		t.Fatalf("事件类型错误，期望 *PromptSubmittedEvent")
	}

	if event.UserInput != "Hello" {
		t.Errorf("UserInput = %q, 期望 Hello", event.UserInput)
	}

	if event.Count != 1 {
		t.Errorf("Count = %d, 期望 1", event.Count)
	}
}

// TestHookManager_OnSystemPromptBuilt 测试系统 Prompt 生成事件
func TestHookManager_OnSystemPromptBuilt(t *testing.T) {
	hm := NewHookManager(zap.NewNop(), true)
	obs := &mockObserver{name: "test", enabled: true}
	hm.Register(obs)

	systemPrompt := "You are a helpful assistant"
	hm.OnSystemPromptBuilt(context.Background(), systemPrompt)

	time.Sleep(100 * time.Millisecond)

	if obs.eventCount != 1 {
		t.Errorf("观察器收到事件数 = %d, 期望 1", obs.eventCount)
	}

	event, ok := obs.lastEvent.(*events.SystemPromptBuiltEvent)
	if !ok {
		t.Fatalf("事件类型错误，期望 *SystemPromptBuiltEvent")
	}

	if event.SystemPrompt != systemPrompt {
		t.Errorf("SystemPrompt = %q, 期望 %q", event.SystemPrompt, systemPrompt)
	}

	if event.Length != len(systemPrompt) {
		t.Errorf("Length = %d, 期望 %d", event.Length, len(systemPrompt))
	}
}

// TestHookManager_OnToolUsed 测试工具使用事件
func TestHookManager_OnToolUsed(t *testing.T) {
	hm := NewHookManager(zap.NewNop(), true)
	obs := &mockObserver{name: "test", enabled: true}
	hm.Register(obs)

	toolName := "read_file"
	toolArgs := `{"path": "/tmp/test.txt"}`
	hm.OnToolUsed(context.Background(), toolName, toolArgs)

	time.Sleep(100 * time.Millisecond)

	if obs.eventCount != 1 {
		t.Errorf("观察器收到事件数 = %d, 期望 1", obs.eventCount)
	}

	event, ok := obs.lastEvent.(*events.ToolUsedEvent)
	if !ok {
		t.Fatalf("事件类型错误，期望 *ToolUsedEvent")
	}

	if event.ToolName != toolName {
		t.Errorf("ToolName = %q, 期望 %q", event.ToolName, toolName)
	}

	if event.ToolArguments != toolArgs {
		t.Errorf("ToolArguments = %q, 期望 %q", event.ToolArguments, toolArgs)
	}
}

// TestHookManager_OnToolCompleted 测试工具完成事件
func TestHookManager_OnToolCompleted(t *testing.T) {
	hm := NewHookManager(zap.NewNop(), true)
	obs := &mockObserver{name: "test", enabled: true}
	hm.Register(obs)

	hm.OnToolCompleted(context.Background(), "read_file", "file content", true)

	time.Sleep(100 * time.Millisecond)

	if obs.eventCount != 1 {
		t.Errorf("观察器收到事件数 = %d, 期望 1", obs.eventCount)
	}

	event, ok := obs.lastEvent.(*events.ToolCompletedEvent)
	if !ok {
		t.Fatalf("事件类型错误，期望 *ToolCompletedEvent")
	}

	if event.ToolName != "read_file" {
		t.Errorf("ToolName = %q, 期望 read_file", event.ToolName)
	}

	if !event.Success {
		t.Error("Success 应该为 true")
	}
}

// TestHookManager_OnToolError 测试工具错误事件
func TestHookManager_OnToolError(t *testing.T) {
	hm := NewHookManager(zap.NewNop(), true)
	obs := &mockObserver{name: "test", enabled: true}
	hm.Register(obs)

	hm.OnToolError(context.Background(), "exec", "command not found")

	time.Sleep(100 * time.Millisecond)

	if obs.eventCount != 1 {
		t.Errorf("观察器收到事件数 = %d, 期望 1", obs.eventCount)
	}

	event, ok := obs.lastEvent.(*events.ToolErrorEvent)
	if !ok {
		t.Fatalf("事件类型错误，期望 *ToolErrorEvent")
	}

	if event.ToolName != "exec" {
		t.Errorf("ToolName = %q, 期望 exec", event.ToolName)
	}

	if event.Error != "command not found" {
		t.Errorf("Error = %q, 期望 command not found", event.Error)
	}
}

// TestHookManager_OnSkillLookup 测试技能查找事件
func TestHookManager_OnSkillLookup(t *testing.T) {
	hm := NewHookManager(zap.NewNop(), true)
	obs := &mockObserver{name: "test", enabled: true}
	hm.Register(obs)

	hm.OnSkillLookup(context.Background(), "read-file", true, "builtin", "/path/to/skill")

	time.Sleep(100 * time.Millisecond)

	if obs.eventCount != 1 {
		t.Errorf("观察器收到事件数 = %d, 期望 1", obs.eventCount)
	}

	event, ok := obs.lastEvent.(*events.SkillLookupEvent)
	if !ok {
		t.Fatalf("事件类型错误，期望 *SkillLookupEvent")
	}

	if event.SkillName != "read-file" {
		t.Errorf("SkillName = %q, 期望 read-file", event.SkillName)
	}

	if !event.Found {
		t.Error("Found 应该为 true")
	}

	if event.Source != "builtin" {
		t.Errorf("Source = %q, 期望 builtin", event.Source)
	}
}

// TestHookManager_OnSkillUsed 测试技能使用事件
func TestHookManager_OnSkillUsed(t *testing.T) {
	hm := NewHookManager(zap.NewNop(), true)
	obs := &mockObserver{name: "test", enabled: true}
	hm.Register(obs)

	hm.OnSkillUsed(context.Background(), "read-file", 150)

	time.Sleep(100 * time.Millisecond)

	if obs.eventCount != 1 {
		t.Errorf("观察器收到事件数 = %d, 期望 1", obs.eventCount)
	}

	event, ok := obs.lastEvent.(*events.SkillUsedEvent)
	if !ok {
		t.Fatalf("事件类型错误，期望 *SkillUsedEvent")
	}

	if event.SkillName != "read-file" {
		t.Errorf("SkillName = %q, 期望 read-file", event.SkillName)
	}

	if event.SkillLength != 150 {
		t.Errorf("SkillLength = %d, 期望 150", event.SkillLength)
	}
}

// TestHookManager_EinoHandler 测试获取 Eino Handler
func TestHookManager_EinoHandler(t *testing.T) {
	hm := NewHookManager(zap.NewNop(), true)

	handler := hm.EinoHandler()
	if handler == nil {
		t.Error("EinoHandler 不应该返回 nil")
	}
}

// TestHookManager_EinoHandlerForGlobal 测试获取全局 Handler
func TestHookManager_EinoHandlerForGlobal(t *testing.T) {
	t.Run("启用时返回 Handler", func(t *testing.T) {
		hm := NewHookManager(zap.NewNop(), true)
		handler := hm.EinoHandlerForGlobal()

		if handler == nil {
			t.Error("EinoHandlerForGlobal 不应该返回 nil")
		}
	})

	t.Run("禁用时返回 nil", func(t *testing.T) {
		hm := NewHookManager(zap.NewNop(), false)
		handler := hm.EinoHandlerForGlobal()

		if handler != nil {
			t.Error("禁用时 EinoHandlerForGlobal 应该返回 nil")
		}
	})
}

// TestHookManager_GetStats 测试统计信息
func TestHookManager_GetStats(t *testing.T) {
	hm := NewHookManager(zap.NewNop(), true)
	hm.Register(&mockObserver{name: "obs1", enabled: true})
	hm.Register(&mockObserver{name: "obs2", enabled: true})

	stats := hm.GetStats()

	if stats == nil {
		t.Fatal("GetStats 返回 nil")
	}

	if stats.ObserverCount != 2 {
		t.Errorf("ObserverCount = %d, 期望 2", stats.ObserverCount)
	}

	if !stats.Enabled {
		t.Error("Enabled 应该为 true")
	}
}

// TestTraceFunctions 测试 Trace 相关函数
func TestTraceFunctions(t *testing.T) {
	t.Run("WithTraceID 和 GetTraceID", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithTraceID(ctx, "trace-123")

		traceID := GetTraceID(ctx)
		if traceID != "trace-123" {
			t.Errorf("GetTraceID = %q, 期望 trace-123", traceID)
		}
	})

	t.Run("NewTraceID 生成唯一 ID", func(t *testing.T) {
		id1 := NewTraceID()
		id2 := NewTraceID()

		if id1 == "" {
			t.Error("NewTraceID 不应该返回空字符串")
		}

		if id1 == id2 {
			t.Error("NewTraceID 应该生成不同的 ID")
		}
	})

	t.Run("MustGetTraceID 无 TraceID 返回空", func(t *testing.T) {
		ctx := context.Background()
		traceID := MustGetTraceID(ctx)

		if traceID != "" {
			t.Errorf("MustGetTraceID = %q, 期望空字符串", traceID)
		}
	})
}
