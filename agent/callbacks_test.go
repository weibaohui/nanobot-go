package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
)

// TestNewEinoCallbacks 测试创建回调处理器
func TestNewEinoCallbacks(t *testing.T) {
	t.Run("启用回调", func(t *testing.T) {
		ec := NewEinoCallbacks(true, zap.NewNop())
		if ec == nil {
			t.Fatal("NewEinoCallbacks 返回 nil")
		}

		if !ec.enabled {
			t.Error("回调应该启用")
		}
	})

	t.Run("禁用回调", func(t *testing.T) {
		ec := NewEinoCallbacks(false, zap.NewNop())
		if ec.enabled {
			t.Error("回调应该禁用")
		}
	})

	t.Run("空 logger 使用默认值", func(t *testing.T) {
		ec := NewEinoCallbacks(true, nil)
		if ec.logger == nil {
			t.Error("logger 不应该为 nil")
		}
	})
}

// TestEinoCallbacks_Handler 测试获取 Handler
func TestEinoCallbacks_Handler(t *testing.T) {
	ec := NewEinoCallbacks(true, zap.NewNop())
	handler := ec.Handler()

	if handler == nil {
		t.Error("Handler 不应该返回 nil")
	}
}

// TestEinoCallbacks_onStart 测试开始回调
func TestEinoCallbacks_onStart(t *testing.T) {
	t.Run("启用回调", func(t *testing.T) {
		ec := NewEinoCallbacks(true, zap.NewNop())
		ctx := context.Background()
		info := &callbacks.RunInfo{
			Component: "ChatModel",
			Type:      "test",
			Name:      "test-model",
		}

		result := ec.onStart(ctx, info, nil)
		if result == nil {
			t.Error("onStart 应该返回 context")
		}

		if len(ec.startTimes) != 1 {
			t.Errorf("startTimes 长度 = %d, 期望 1", len(ec.startTimes))
		}
	})

	t.Run("禁用回调", func(t *testing.T) {
		ec := NewEinoCallbacks(false, zap.NewNop())
		ctx := context.Background()
		info := &callbacks.RunInfo{
			Component: "ChatModel",
		}

		result := ec.onStart(ctx, info, nil)
		if result == nil {
			t.Error("onStart 应该返回 context")
		}

		if len(ec.startTimes) != 0 {
			t.Errorf("禁用时 startTimes 长度应该为 0, 得到 %d", len(ec.startTimes))
		}
	})
}

// TestEinoCallbacks_onEnd 测试结束回调
func TestEinoCallbacks_onEnd(t *testing.T) {
	ec := NewEinoCallbacks(true, zap.NewNop())
	ctx := context.Background()
	info := &callbacks.RunInfo{
		Component: "ChatModel",
		Type:      "test",
		Name:      "test-model",
	}

	ec.onStart(ctx, info, nil)
	result := ec.onEnd(ctx, info, nil)

	if result == nil {
		t.Error("onEnd 应该返回 context")
	}

	if len(ec.startTimes) != 0 {
		t.Errorf("onEnd 后 startTimes 长度应该为 0, 得到 %d", len(ec.startTimes))
	}
}

// TestEinoCallbacks_onError 测试错误回调
func TestEinoCallbacks_onError(t *testing.T) {
	t.Run("普通错误", func(t *testing.T) {
		ec := NewEinoCallbacks(true, zap.NewNop())
		ctx := context.Background()
		info := &callbacks.RunInfo{
			Component: "ChatModel",
			Type:      "test",
			Name:      "test-model",
		}

		ec.onStart(ctx, info, nil)
		result := ec.onError(ctx, info, errors.New("测试错误"))

		if result == nil {
			t.Error("onError 应该返回 context")
		}
	})

	t.Run("中断错误", func(t *testing.T) {
		ec := NewEinoCallbacks(true, zap.NewNop())
		ctx := context.Background()
		info := &callbacks.RunInfo{
			Component: "ChatModel",
		}

		ec.onStart(ctx, info, nil)
		result := ec.onError(ctx, info, errors.New("INTERRUPT: 用户中断"))

		if result == nil {
			t.Error("onError 应该返回 context")
		}
	})
}

// TestIsInterruptError 测试检查中断错误
func TestIsInterruptError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil 错误", nil, false},
		{"INTERRUPT: 前缀", errors.New("INTERRUPT: 用户中断"), true},
		{"interrupt signal", errors.New("interrupt signal: received"), true},
		{"interrupt happened", errors.New("some interrupt happened here"), true},
		{"普通错误", errors.New("普通错误"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isInterruptError(tt.err)
			if result != tt.expected {
				t.Errorf("isInterruptError() = %v, 期望 %v", result, tt.expected)
			}
		})
	}
}

// TestEinoCallbacks_nodeKey 测试生成节点键
func TestEinoCallbacks_nodeKey(t *testing.T) {
	ec := NewEinoCallbacks(true, zap.NewNop())
	info := &callbacks.RunInfo{
		Component: "ChatModel",
		Type:      "openai",
		Name:      "gpt-4",
	}

	key := ec.nodeKey(info)
	expected := "ChatModel:openai:gpt-4"

	if key != expected {
		t.Errorf("nodeKey() = %q, 期望 %q", key, expected)
	}
}

// TestEinoCallbacks_logModelInput 测试记录模型输入
func TestEinoCallbacks_logModelInput(t *testing.T) {
	ec := NewEinoCallbacks(true, zap.NewNop())
	info := &callbacks.RunInfo{
		Component: "ChatModel",
	}

	t.Run("nil 输入", func(t *testing.T) {
		ec.logModelInput(nil, info)
	})

	t.Run("有效输入", func(t *testing.T) {
		input := &model.CallbackInput{
			Messages: []*schema.Message{
				{Role: schema.System, Content: "系统提示"},
				{Role: schema.User, Content: "用户消息"},
			},
			Tools: []*schema.ToolInfo{
				{Name: "tool1"},
				{Name: "tool2"},
			},
		}
		ec.logModelInput(input, info)
	})
}

// TestEinoCallbacks_logModelOutput 测试记录模型输出
func TestEinoCallbacks_logModelOutput(t *testing.T) {
	ec := NewEinoCallbacks(true, zap.NewNop())
	info := &callbacks.RunInfo{
		Component: "ChatModel",
	}

	t.Run("nil 输出", func(t *testing.T) {
		ec.logModelOutput(nil, info)
	})

	t.Run("有效输出", func(t *testing.T) {
		output := &model.CallbackOutput{
			Message: &schema.Message{
				Role:    schema.Assistant,
				Content: "助手回复",
				ToolCalls: []schema.ToolCall{
					{
						Type: "function",
						Function: schema.FunctionCall{
							Name:      "test_tool",
							Arguments: "{}",
						},
					},
				},
			},
		}
		ec.logModelOutput(output, info)
	})
}

// TestEinoCallbacks_logToolInput 测试记录工具输入
func TestEinoCallbacks_logToolInput(t *testing.T) {
	ec := NewEinoCallbacks(true, zap.NewNop())
	info := &callbacks.RunInfo{
		Component: "Tool",
	}

	t.Run("nil 输入", func(t *testing.T) {
		ec.logToolInput(nil, info)
	})

	t.Run("有效输入", func(t *testing.T) {
		input := &tool.CallbackInput{
			ArgumentsInJSON: `{"arg1": "value1"}`,
		}
		ec.logToolInput(input, info)
	})
}

// TestEinoCallbacks_logToolOutput 测试记录工具输出
func TestEinoCallbacks_logToolOutput(t *testing.T) {
	ec := NewEinoCallbacks(true, zap.NewNop())
	info := &callbacks.RunInfo{
		Component: "Tool",
	}

	t.Run("nil 输出", func(t *testing.T) {
		ec.logToolOutput(nil, info)
	})

	t.Run("有效输出", func(t *testing.T) {
		output := &tool.CallbackOutput{
			Response: "工具执行结果",
		}
		ec.logToolOutput(output, info)
	})
}

// TestEinoCallbacks_logGenericInput 测试记录通用输入
func TestEinoCallbacks_logGenericInput(t *testing.T) {
	ec := NewEinoCallbacks(true, zap.NewNop())
	info := &callbacks.RunInfo{
		Component: "Unknown",
	}

	ec.logGenericInput("test input", info)
}

// TestEinoCallbacks_logGenericOutput 测试记录通用输出
func TestEinoCallbacks_logGenericOutput(t *testing.T) {
	ec := NewEinoCallbacks(true, zap.NewNop())
	info := &callbacks.RunInfo{
		Component: "Unknown",
	}

	ec.logGenericOutput("test output", info)
}

// TestRegisterGlobalCallbacks 测试注册全局回调
func TestRegisterGlobalCallbacks(t *testing.T) {
	t.Run("nil 回调", func(t *testing.T) {
		RegisterGlobalCallbacks(nil)
	})

	t.Run("禁用回调", func(t *testing.T) {
		ec := NewEinoCallbacks(false, zap.NewNop())
		RegisterGlobalCallbacks(ec)
	})

	t.Run("启用回调", func(t *testing.T) {
		ec := NewEinoCallbacks(true, zap.NewNop())
		RegisterGlobalCallbacks(ec)
	})
}

// TestEinoCallbacks_CallSequence 测试调用序列
func TestEinoCallbacks_CallSequence(t *testing.T) {
	ec := NewEinoCallbacks(true, zap.NewNop())
	ctx := context.Background()
	info := &callbacks.RunInfo{
		Component: "ChatModel",
	}

	ec.onStart(ctx, info, nil)
	if ec.callSequence != 1 {
		t.Errorf("callSequence = %d, 期望 1", ec.callSequence)
	}

	ec.onStart(ctx, info, nil)
	if ec.callSequence != 2 {
		t.Errorf("callSequence = %d, 期望 2", ec.callSequence)
	}
}
