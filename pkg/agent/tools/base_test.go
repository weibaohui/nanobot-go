package tools

import (
	"testing"
)

// testNamedTool 实现 NamedTool 接口
type testNamedTool struct {
	name string
}

func (m *testNamedTool) Name() string {
	return m.name
}

// testContextSetter 实现 ContextSetter 接口
type testContextSetter struct {
	channel string
	chatID  string
}

func (m *testContextSetter) SetContext(channel, chatID string) {
	m.channel = channel
	m.chatID = chatID
}

// TestNamedTool 测试命名工具接口
func TestNamedTool(t *testing.T) {
	tool := &testNamedTool{name: "test_tool"}

	var _ NamedTool = tool

	if tool.Name() != "test_tool" {
		t.Errorf("Name() = %q, 期望 test_tool", tool.Name())
	}
}

// TestContextSetter 测试上下文设置接口
func TestContextSetter(t *testing.T) {
	setter := &testContextSetter{}

	var _ ContextSetter = setter

	setter.SetContext("websocket", "chat-001")

	if setter.channel != "websocket" {
		t.Errorf("channel = %q, 期望 websocket", setter.channel)
	}

	if setter.chatID != "chat-001" {
		t.Errorf("chatID = %q, 期望 chat-001", setter.chatID)
	}
}

// TestNamedTool_EmptyName 测试空名称
func TestNamedTool_EmptyName(t *testing.T) {
	tool := &testNamedTool{name: ""}

	if tool.Name() != "" {
		t.Errorf("Name() = %q, 期望空字符串", tool.Name())
	}
}

// TestContextSetter_EmptyValues 测试空值设置
func TestContextSetter_EmptyValues(t *testing.T) {
	setter := &testContextSetter{}

	setter.SetContext("", "")

	if setter.channel != "" {
		t.Errorf("channel = %q, 期望空字符串", setter.channel)
	}

	if setter.chatID != "" {
		t.Errorf("chatID = %q, 期望空字符串", setter.chatID)
	}
}

// TestContextSetter_Overwrite 测试覆盖设置
func TestContextSetter_Overwrite(t *testing.T) {
	setter := &testContextSetter{}

	setter.SetContext("channel1", "chat1")
	setter.SetContext("channel2", "chat2")

	if setter.channel != "channel2" {
		t.Errorf("channel = %q, 期望 channel2", setter.channel)
	}

	if setter.chatID != "chat2" {
		t.Errorf("chatID = %q, 期望 chat2", setter.chatID)
	}
}
