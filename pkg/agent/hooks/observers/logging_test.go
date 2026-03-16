package observers

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/events"
	hooksobserver "github.com/weibaohui/nanobot-go/pkg/agent/hooks/observer"
	"github.com/weibaohui/nanobot-go/pkg/bus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	zapobserver "go.uber.org/zap/zaptest/observer"
)

// TestIsInterruptError 测试中断错误判断
func TestIsInterruptError(t *testing.T) {
	tests := []struct {
		name     string
		errorMsg string
		expected bool
	}{
		{"Interrupt signal", "Interrupt signal: user stopped", true},
		{"interrupt:", "interrupt: waiting for input", true},
		{"INTERRUPT:", "INTERRUPT: test", true},
		{"普通错误", "some error", false},
		{"空字符串", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isInterruptError(tt.errorMsg)
			if result != tt.expected {
				t.Errorf("isInterruptError(%q) = %v, want %v", tt.errorMsg, result, tt.expected)
			}
		})
	}
}

// TestNewLoggingObserver 测试创建日志观察器
func TestNewLoggingObserver(t *testing.T) {
	logger := zap.NewNop()
	filter := &hooksobserver.ObserverFilter{}

	obs := NewLoggingObserver(logger, filter)

	if obs == nil {
		t.Fatal("NewLoggingObserver 返回 nil")
	}

	if obs.Name() != "logging" {
		t.Errorf("Name() = %q, want logging", obs.Name())
	}

	if !obs.Enabled() {
		t.Error("应该默认启用")
	}
}

// TestNewLoggingObserver_NilLogger 测试 nil logger
func TestNewLoggingObserver_NilLogger(t *testing.T) {
	obs := NewLoggingObserver(nil, nil)

	if obs == nil {
		t.Fatal("NewLoggingObserver 返回 nil")
	}

	if obs.logger == nil {
		t.Error("logger 不应该为 nil")
	}
}

// TestLoggingObserver_OnEvent_MessageReceived 测试消息接收事件
func TestLoggingObserver_OnEvent_MessageReceived(t *testing.T) {
	core, recorded := zapobserver.New(zap.InfoLevel)
	logger := zap.New(core)
	obs := NewLoggingObserver(logger, nil)

	msg := &bus.InboundMessage{
		Channel:  "dingtalk",
		SenderID: "user-1",
		ChatID:   "chat-1",
		Content:  "Hello",
	}
	event := events.NewMessageReceivedEvent("trace-1", "span-1", "", msg)

	err := obs.OnEvent(context.Background(), event)
	if err != nil {
		t.Errorf("OnEvent 返回错误: %v", err)
	}

	entries := recorded.All()
	if len(entries) == 0 {
		t.Error("应该记录日志")
	}

	// 验证日志内容
	found := false
	for _, entry := range entries {
		if strings.Contains(entry.Message, "收到消息") {
			found = true
			break
		}
	}
	if !found {
		t.Error("应该包含'收到消息'日志")
	}
}

// TestLoggingObserver_OnEvent_ToolUsed 测试工具使用事件
func TestLoggingObserver_OnEvent_ToolUsed(t *testing.T) {
	core, recorded := zapobserver.New(zap.InfoLevel)
	logger := zap.New(core)
	obs := NewLoggingObserver(logger, nil)

	event := events.NewToolUsedEvent("trace-1", "span-1", "", "read_file", `{"path": "/tmp/test.txt"}`)

	err := obs.OnEvent(context.Background(), event)
	if err != nil {
		t.Errorf("OnEvent 返回错误: %v", err)
	}

	entries := recorded.All()
	found := false
	for _, entry := range entries {
		if strings.Contains(entry.Message, "使用工具") {
			found = true
			break
		}
	}
	if !found {
		t.Error("应该包含'使用工具'日志")
	}
}

// TestLoggingObserver_OnEvent_ToolError 测试工具错误事件
func TestLoggingObserver_OnEvent_ToolError(t *testing.T) {
	t.Run("普通错误", func(t *testing.T) {
		core, recorded := zapobserver.New(zap.ErrorLevel)
		logger := zap.New(core)
		obs := NewLoggingObserver(logger, nil)

		event := events.NewToolErrorEvent("trace-1", "span-1", "", "exec", "command not found")

		err := obs.OnEvent(context.Background(), event)
		if err != nil {
			t.Errorf("OnEvent 返回错误: %v", err)
		}

		entries := recorded.All()
		found := false
		for _, entry := range entries {
			if entry.Level == zapcore.ErrorLevel && strings.Contains(entry.Message, "工具执行错误") {
				found = true
				break
			}
		}
		if !found {
			t.Error("应该记录错误日志")
		}
	})

	t.Run("中断信号", func(t *testing.T) {
		core, recorded := zapobserver.New(zap.InfoLevel)
		logger := zap.New(core)
		obs := NewLoggingObserver(logger, nil)

		event := events.NewToolErrorEvent("trace-1", "span-1", "", "ask_user", "Interrupt: waiting for input")

		err := obs.OnEvent(context.Background(), event)
		if err != nil {
			t.Errorf("OnEvent 返回错误: %v", err)
		}

		entries := recorded.All()
		found := false
		for _, entry := range entries {
			if entry.Level == zapcore.InfoLevel && strings.Contains(entry.Message, "中断") {
				found = true
				break
			}
		}
		if !found {
			t.Error("中断应该记录为 Info")
		}
	})
}

// TestLoggingObserver_OnEvent_LLMCallEnd 测试 LLM 调用结束
func TestLoggingObserver_OnEvent_LLMCallEnd(t *testing.T) {
	core, recorded := zapobserver.New(zap.InfoLevel)
	logger := zap.New(core)
	obs := NewLoggingObserver(logger, nil)

	event := events.NewLLMCallEndEvent("trace-1", "span-1", "",
		&callbacks.RunInfo{Component: "LLM"},
		&model.CallbackOutput{
			Message: &schema.Message{Content: "AI response"},
			TokenUsage: &model.TokenUsage{
				PromptTokens:     100,
				CompletionTokens: 50,
				TotalTokens:      150,
			},
		},
		200,
	)

	err := obs.OnEvent(context.Background(), event)
	if err != nil {
		t.Errorf("OnEvent 返回错误: %v", err)
	}

	entries := recorded.All()
	found := false
	for _, entry := range entries {
		if strings.Contains(entry.Message, "LLM 调用结束") {
			// 检查是否包含 token usage
			for _, field := range entry.Context {
				if field.Key == "token_usage" {
					found = true
					break
				}
			}
		}
	}
	if !found {
		t.Error("应该记录 LLM 调用结束和 token usage")
	}
}

// TestLoggingObserver_OnEvent_LLMCallError 测试 LLM 调用错误
func TestLoggingObserver_OnEvent_LLMCallError(t *testing.T) {
	t.Run("普通错误", func(t *testing.T) {
		core, recorded := zapobserver.New(zap.ErrorLevel)
		logger := zap.New(core)
		obs := NewLoggingObserver(logger, nil)

		event := events.NewLLMCallErrorEvent("trace-1", "span-1", "",
			&callbacks.RunInfo{Component: "LLM"},
			errors.New("API error"),
			0,
		)

		err := obs.OnEvent(context.Background(), event)
		if err != nil {
			t.Errorf("OnEvent 返回错误: %v", err)
		}

		entries := recorded.All()
		found := false
		for _, entry := range entries {
			if entry.Level == zapcore.ErrorLevel {
				found = true
				break
			}
		}
		if !found {
			t.Error("应该记录错误日志")
		}
	})

	t.Run("中断错误", func(t *testing.T) {
		core, recorded := zapobserver.New(zap.InfoLevel)
		logger := zap.New(core)
		obs := NewLoggingObserver(logger, nil)

		event := events.NewLLMCallErrorEvent("trace-1", "span-1", "",
			&callbacks.RunInfo{Component: "LLM"},
			errors.New("Interrupt signal: user stopped"),
			0,
		)

		err := obs.OnEvent(context.Background(), event)
		if err != nil {
			t.Errorf("OnEvent 返回错误: %v", err)
		}

		entries := recorded.All()
		found := false
		for _, entry := range entries {
			if entry.Level == zapcore.InfoLevel && strings.Contains(entry.Message, "中断") {
				found = true
				break
			}
		}
		if !found {
			t.Error("中断应该记录为 Info")
		}
	})
}

// TestLoggingObserver_OnEvent_UnknownEvent 测试未知事件类型
func TestLoggingObserver_OnEvent_UnknownEvent(t *testing.T) {
	core, recorded := zapobserver.New(zap.DebugLevel)
	logger := zap.New(core)
	obs := NewLoggingObserver(logger, nil)

	// 创建一个自定义事件类型（通过直接构造 BaseEvent）
	type customEvent struct {
		*events.BaseEvent
	}

	event := &customEvent{
		BaseEvent: events.NewBaseEvent("trace-1", "span-1", "", events.EventType("custom_event")),
	}

	err := obs.OnEvent(context.Background(), event)
	if err != nil {
		t.Errorf("OnEvent 返回错误: %v", err)
	}

	entries := recorded.All()
	found := false
	for _, entry := range entries {
		if strings.Contains(entry.Message, "未知事件") {
			found = true
			break
		}
	}
	if !found {
		t.Error("应该记录未知事件类型日志")
	}
}
