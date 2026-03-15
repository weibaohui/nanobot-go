package eino

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/dispatcher"
	"go.uber.org/zap"
)

// TestNewEinoCallbackBridge 测试创建桥接器
func TestNewEinoCallbackBridge(t *testing.T) {
	t.Run("正常创建", func(t *testing.T) {
		disp := dispatcher.NewDispatcher(zap.NewNop())
		logger := zap.NewNop()

		bridge := NewEinoCallbackBridge(disp, logger)

		if bridge == nil {
			t.Fatal("NewEinoCallbackBridge 返回 nil")
		}

		if bridge.dispatcher != disp {
			t.Error("dispatcher 不匹配")
		}

		if bridge.startTimes == nil {
			t.Error("startTimes map 不应该为 nil")
		}
	})

	t.Run("nil logger 使用默认", func(t *testing.T) {
		disp := dispatcher.NewDispatcher(zap.NewNop())
		bridge := NewEinoCallbackBridge(disp, nil)

		if bridge == nil {
			t.Fatal("NewEinoCallbackBridge 返回 nil")
		}

		if bridge.logger == nil {
			t.Error("logger 不应该为 nil")
		}
	})
}

// TestEinoCallbackBridge_Handler 测试获取 Handler
func TestEinoCallbackBridge_Handler(t *testing.T) {
	disp := dispatcher.NewDispatcher(zap.NewNop())
	bridge := NewEinoCallbackBridge(disp, zap.NewNop())

	handler := bridge.Handler()
	if handler == nil {
		t.Fatal("Handler 返回 nil")
	}
}

// TestEinoCallbackBridge_nodeKey 测试节点键生成
func TestEinoCallbackBridge_nodeKey(t *testing.T) {
	bridge := NewEinoCallbackBridge(nil, zap.NewNop())

	info := &callbacks.RunInfo{
		Component: "ChatModel",
		Type:      "model",
		Name:      "gpt-4",
	}

	key := bridge.nodeKey(info)
	expected := "ChatModel:model:gpt-4"

	if key != expected {
		t.Errorf("nodeKey = %q, 期望 %q", key, expected)
	}
}

// TestEinoCallbackBridge_onStart 测试开始回调
func TestEinoCallbackBridge_onStart(t *testing.T) {
	t.Run("处理 ChatModel 开始", func(t *testing.T) {
		disp := dispatcher.NewDispatcher(zap.NewNop())
		bridge := NewEinoCallbackBridge(disp, zap.NewNop())

		info := &callbacks.RunInfo{
			Component: "ChatModel",
			Type:      "model",
			Name:      "gpt-4",
		}

		input := &model.CallbackInput{
			Messages: []*schema.Message{
				{Role: "user", Content: "Hello"},
			},
		}

		ctx := context.Background()
		bridge.onStart(ctx, info, input)

		// 验证 startTimes 记录了时间
		key := bridge.nodeKey(info)
		if _, ok := bridge.startTimes[key]; !ok {
			t.Error("startTimes 应该记录节点开始时间")
		}
	})

	t.Run("处理 Tool 开始", func(t *testing.T) {
		disp := dispatcher.NewDispatcher(zap.NewNop())
		bridge := NewEinoCallbackBridge(disp, zap.NewNop())

		info := &callbacks.RunInfo{
			Component: "Tool",
			Type:      "tool",
			Name:      "read_file",
		}

		input := &tool.CallbackInput{
			ArgumentsInJSON: `{"path": "/tmp/test.txt"}`,
		}

		ctx := context.Background()
		bridge.onStart(ctx, info, input)

		// 验证 startTimes 记录了时间
		key := bridge.nodeKey(info)
		if _, ok := bridge.startTimes[key]; !ok {
			t.Error("startTimes 应该记录节点开始时间")
		}
	})

	t.Run("处理其他组件开始", func(t *testing.T) {
		disp := dispatcher.NewDispatcher(zap.NewNop())
		bridge := NewEinoCallbackBridge(disp, zap.NewNop())

		info := &callbacks.RunInfo{
			Component: "Retriever",
			Type:      "retriever",
			Name:      "default",
		}

		ctx := context.Background()
		bridge.onStart(ctx, info, nil)

		// 验证 startTimes 记录了时间
		key := bridge.nodeKey(info)
		if _, ok := bridge.startTimes[key]; !ok {
			t.Error("startTimes 应该记录节点开始时间")
		}
	})
}

// TestEinoCallbackBridge_onEnd 测试结束回调
func TestEinoCallbackBridge_onEnd(t *testing.T) {
	t.Run("处理 ChatModel 结束", func(t *testing.T) {
		disp := dispatcher.NewDispatcher(zap.NewNop())
		bridge := NewEinoCallbackBridge(disp, zap.NewNop())

		info := &callbacks.RunInfo{
			Component: "ChatModel",
			Type:      "model",
			Name:      "gpt-4",
		}

		// 先记录开始时间
		key := bridge.nodeKey(info)
		bridge.startTimes[key] = time.Now().Add(-100 * time.Millisecond)

		output := &model.CallbackOutput{
			Message: &schema.Message{
				Role:    "assistant",
				Content: "Hello!",
			},
			TokenUsage: &model.TokenUsage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}

		ctx := context.Background()
		bridge.onEnd(ctx, info, output)

		// 验证 startTimes 被清理
		if _, ok := bridge.startTimes[key]; ok {
			t.Error("startTimes 应该清理已完成的节点")
		}
	})

	t.Run("处理 Tool 结束", func(t *testing.T) {
		disp := dispatcher.NewDispatcher(zap.NewNop())
		bridge := NewEinoCallbackBridge(disp, zap.NewNop())

		info := &callbacks.RunInfo{
			Component: "Tool",
			Type:      "tool",
			Name:      "list_dir",
		}

		key := bridge.nodeKey(info)
		bridge.startTimes[key] = time.Now().Add(-50 * time.Millisecond)

		output := &tool.CallbackOutput{
			Response: "file1.txt\nfile2.txt",
		}

		ctx := context.Background()
		bridge.onEnd(ctx, info, output)

		// 验证 startTimes 被清理
		if _, ok := bridge.startTimes[key]; ok {
			t.Error("startTimes 应该清理已完成的节点")
		}
	})

	t.Run("无开始时间记录", func(t *testing.T) {
		disp := dispatcher.NewDispatcher(zap.NewNop())
		bridge := NewEinoCallbackBridge(disp, zap.NewNop())

		info := &callbacks.RunInfo{
			Component: "ChatModel",
			Type:      "model",
			Name:      "gpt-4",
		}

		output := &model.CallbackOutput{
			Message: &schema.Message{
				Role:    "assistant",
				Content: "Hello!",
			},
		}

		ctx := context.Background()
		// 不应该 panic
		bridge.onEnd(ctx, info, output)
	})
}

// TestEinoCallbackBridge_onError 测试错误回调
func TestEinoCallbackBridge_onError(t *testing.T) {
	t.Run("处理 ChatModel 错误", func(t *testing.T) {
		disp := dispatcher.NewDispatcher(zap.NewNop())
		bridge := NewEinoCallbackBridge(disp, zap.NewNop())

		info := &callbacks.RunInfo{
			Component: "ChatModel",
			Type:      "model",
			Name:      "gpt-4",
		}

		key := bridge.nodeKey(info)
		bridge.startTimes[key] = time.Now().Add(-100 * time.Millisecond)

		testErr := errors.New("API rate limit exceeded")

		ctx := context.Background()
		bridge.onError(ctx, info, testErr)

		// 验证 startTimes 被清理
		if _, ok := bridge.startTimes[key]; ok {
			t.Error("startTimes 应该清理已完成的节点")
		}
	})

	t.Run("处理 Tool 错误", func(t *testing.T) {
		disp := dispatcher.NewDispatcher(zap.NewNop())
		bridge := NewEinoCallbackBridge(disp, zap.NewNop())

		info := &callbacks.RunInfo{
			Component: "Tool",
			Type:      "tool",
			Name:      "exec",
		}

		key := bridge.nodeKey(info)
		bridge.startTimes[key] = time.Now().Add(-50 * time.Millisecond)

		testErr := errors.New("command not found")

		ctx := context.Background()
		bridge.onError(ctx, info, testErr)

		// 验证 startTimes 被清理
		if _, ok := bridge.startTimes[key]; ok {
			t.Error("startTimes 应该清理已完成的节点")
		}
	})
}

// TestEinoCallbackBridge_handleModelStart 测试模型开始处理
func TestEinoCallbackBridge_handleModelStart(t *testing.T) {
	disp := dispatcher.NewDispatcher(zap.NewNop())
	bridge := NewEinoCallbackBridge(disp, zap.NewNop())

	info := &callbacks.RunInfo{
		Component: "ChatModel",
		Type:      "model",
		Name:      "claude-3",
	}

	toolInfo := &schema.ToolInfo{
		Name: "read_file",
	}

	input := &model.CallbackInput{
		Messages: []*schema.Message{
			{Role: "system", Content: "You are helpful"},
			{Role: "user", Content: "Hello"},
		},
		Tools: []*schema.ToolInfo{toolInfo},
		Config: &model.Config{
			Model:       "claude-3-opus",
			MaxTokens:   1000,
			Temperature: 0.7,
			TopP:        0.9,
		},
	}

	ctx := context.Background()
	bridge.handleModelStart(ctx, "trace-1", "span-1", "parent-1", info, input, "matrix", "session-1")

	// 验证分发了事件
	// 由于是异步的，我们无法直接验证，但应该不会 panic
}

// TestEinoCallbackBridge_handleModelEnd 测试模型结束处理
func TestEinoCallbackBridge_handleModelEnd(t *testing.T) {
	disp := dispatcher.NewDispatcher(zap.NewNop())
	bridge := NewEinoCallbackBridge(disp, zap.NewNop())

	info := &callbacks.RunInfo{
		Component: "ChatModel",
		Type:      "model",
		Name:      "gpt-4",
	}

	output := &model.CallbackOutput{
		Message: &schema.Message{
			Role:    "assistant",
			Content: "The answer is 42",
			ToolCalls: []schema.ToolCall{
				{
					ID:       "call-1",
					Function: schema.FunctionCall{Name: "calculator", Arguments: `{"expression": "6*7"}`},
				},
			},
		},
		TokenUsage: &model.TokenUsage{
			PromptTokens:     20,
			CompletionTokens: 10,
			TotalTokens:      30,
		},
	}

	ctx := context.Background()
	bridge.handleModelEnd(ctx, "trace-1", "span-1", "parent-1", info, output, 150, "dingtalk", "session-1")

	// 验证分发了事件
	// 由于是异步的，我们无法直接验证，但应该不会 panic
}

// TestEinoCallbackBridge_handleToolStart 测试工具开始处理
func TestEinoCallbackBridge_handleToolStart(t *testing.T) {
	disp := dispatcher.NewDispatcher(zap.NewNop())
	bridge := NewEinoCallbackBridge(disp, zap.NewNop())

	info := &callbacks.RunInfo{
		Component: "Tool",
		Type:      "tool",
		Name:      "write_file",
	}

	input := &tool.CallbackInput{
		ArgumentsInJSON: `{"path": "/tmp/test.txt", "content": "hello"}`,
	}

	ctx := context.Background()
	bridge.handleToolStart(ctx, "trace-1", "span-1", "parent-1", info, input, "matrix", "session-1")

	// 验证分发了事件
	// 由于是异步的，我们无法直接验证，但应该不会 panic
}

// TestEinoCallbackBridge_handleToolEnd 测试工具结束处理
func TestEinoCallbackBridge_handleToolEnd(t *testing.T) {
	disp := dispatcher.NewDispatcher(zap.NewNop())
	bridge := NewEinoCallbackBridge(disp, zap.NewNop())

	info := &callbacks.RunInfo{
		Component: "Tool",
		Type:      "tool",
		Name:      "list_dir",
	}

	output := &tool.CallbackOutput{
		Response: "file1.txt\nfile2.txt",
	}

	ctx := context.Background()
	bridge.handleToolEnd(ctx, "trace-1", "span-1", "parent-1", info, output, 50, "matrix", "session-1")

	// 验证分发了事件
	// 由于是异步的，我们无法直接验证，但应该不会 panic
}

// TestEinoCallbackBridge_handleToolError 测试工具错误处理
func TestEinoCallbackBridge_handleToolError(t *testing.T) {
	disp := dispatcher.NewDispatcher(zap.NewNop())
	bridge := NewEinoCallbackBridge(disp, zap.NewNop())

	info := &callbacks.RunInfo{
		Component: "Tool",
		Type:      "tool",
		Name:      "exec",
	}

	testErr := errors.New("permission denied")

	ctx := context.Background()
	bridge.handleToolError(ctx, "trace-1", "span-1", "parent-1", info, testErr, "matrix", "session-1")

	// 验证分发了事件
	// 由于是异步的，我们无法直接验证，但应该不会 panic
}
