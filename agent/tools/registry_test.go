package tools

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// mockTool 用于测试的模拟工具
type mockTool struct {
	name string
	info *schema.ToolInfo
}

func (m *mockTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	if m.info != nil {
		return m.info, nil
	}
	return &schema.ToolInfo{
		Name: m.name,
		Desc: "测试工具",
	}, nil
}

// mockInvokableTool 可调用的模拟工具
type mockInvokableTool struct {
	mockTool
	result string
	err    error
}

func (m *mockInvokableTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return m.result, m.err
}

// mockNamedTool 命名工具
type mockNamedTool struct {
	mockTool
}

func (m *mockNamedTool) Name() string {
	return m.name
}

// TestNewRegistry 测试创建工具注册表
func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	if registry == nil {
		t.Fatal("NewRegistry 返回 nil")
	}

	if registry.tools == nil {
		t.Error("tools 不应该为 nil")
	}
}

// TestRegistry_Register 测试注册工具
func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	t.Run("注册普通工具", func(t *testing.T) {
		tool := &mockTool{name: "test_tool"}
		registry.Register(tool)

		if len(registry.tools) != 1 {
			t.Errorf("tools 长度 = %d, 期望 1", len(registry.tools))
		}
	})

	t.Run("注册命名工具", func(t *testing.T) {
		registry := NewRegistry()
		tool := &mockNamedTool{mockTool: mockTool{name: "named_tool"}}
		registry.Register(tool)

		if registry.tools["named_tool"] == nil {
			t.Error("命名工具应该被注册")
		}
	})

	t.Run("注册空名称工具", func(t *testing.T) {
		registry := NewRegistry()
		tool := &mockTool{name: ""}
		registry.Register(tool)

		if len(registry.tools) != 0 {
			t.Errorf("空名称工具不应该被注册, tools 长度 = %d", len(registry.tools))
		}
	})
}

// TestRegistry_Get 测试获取工具
func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()
	ctx := context.Background()

	tool := &mockTool{name: "get_test"}
	registry.Register(tool)

	t.Run("获取存在的工具", func(t *testing.T) {
		result := registry.Get("get_test")
		if result == nil {
			t.Error("Get 应该返回工具")
		}
	})

	t.Run("获取不存在的工具", func(t *testing.T) {
		result := registry.Get("nonexistent")
		if result != nil {
			t.Error("获取不存在的工具应该返回 nil")
		}
	})

	_ = ctx
}

// TestRegistry_GetTools 测试获取所有工具
func TestRegistry_GetTools(t *testing.T) {
	registry := NewRegistry()

	for i := 0; i < 3; i++ {
		tool := &mockTool{name: "tool_" + string(rune('0'+i))}
		registry.Register(tool)
	}

	tools := registry.GetTools()
	if len(tools) != 3 {
		t.Errorf("GetTools 返回 %d 个工具, 期望 3", len(tools))
	}
}

// TestRegistry_GetToolNames 测试获取所有工具名称
func TestRegistry_GetToolNames(t *testing.T) {
	registry := NewRegistry()
	ctx := context.Background()

	names := []string{"tool_a", "tool_b", "tool_c"}
	for _, name := range names {
		tool := &mockTool{name: name}
		registry.Register(tool)
	}

	result := registry.GetToolNames(ctx)
	if len(result) != 3 {
		t.Errorf("GetToolNames 返回 %d 个名称, 期望 3", len(result))
	}
}

// TestRegistry_GetToolsByNames 测试根据名称获取工具
func TestRegistry_GetToolsByNames(t *testing.T) {
	registry := NewRegistry()

	for _, name := range []string{"tool_a", "tool_b", "tool_c"} {
		tool := &mockTool{name: name}
		registry.Register(tool)
	}

	t.Run("获取指定工具", func(t *testing.T) {
		tools := registry.GetToolsByNames([]string{"tool_a", "tool_c"})
		if len(tools) != 2 {
			t.Errorf("GetToolsByNames 返回 %d 个工具, 期望 2", len(tools))
		}
	})

	t.Run("空名称列表返回所有工具", func(t *testing.T) {
		tools := registry.GetToolsByNames(nil)
		if len(tools) != 3 {
			t.Errorf("GetToolsByNames(nil) 返回 %d 个工具, 期望 3", len(tools))
		}
	})

	t.Run("部分名称不存在", func(t *testing.T) {
		tools := registry.GetToolsByNames([]string{"tool_a", "nonexistent"})
		if len(tools) != 1 {
			t.Errorf("GetToolsByNames 返回 %d 个工具, 期望 1", len(tools))
		}
	})
}

// TestRegistry_Execute 测试执行工具
func TestRegistry_Execute(t *testing.T) {
	registry := NewRegistry()
	ctx := context.Background()

	t.Run("执行存在的工具", func(t *testing.T) {
		tool := &mockInvokableTool{
			mockTool: mockTool{name: "exec_tool"},
			result:   "执行结果",
		}
		registry.Register(tool)

		result, err := registry.Execute(ctx, "exec_tool", map[string]any{"arg": "value"})
		if err != nil {
			t.Errorf("Execute 返回错误: %v", err)
		}

		if result != "执行结果" {
			t.Errorf("结果 = %q, 期望 执行结果", result)
		}
	})

	t.Run("执行不存在的工具", func(t *testing.T) {
		_, err := registry.Execute(ctx, "nonexistent", nil)
		if err == nil {
			t.Error("执行不存在的工具应该返回错误")
		}
	})

	t.Run("执行不可调用的工具", func(t *testing.T) {
		registry := NewRegistry()
		tool := &mockTool{name: "non_invokable"}
		registry.Register(tool)

		_, err := registry.Execute(ctx, "non_invokable", nil)
		if err == nil {
			t.Error("执行不可调用的工具应该返回错误")
		}
	})
}

// TestRegistry_resolveToolName 测试解析工具名称
func TestRegistry_resolveToolName(t *testing.T) {
	registry := NewRegistry()
	ctx := context.Background()

	t.Run("命名工具", func(t *testing.T) {
		tool := &mockNamedTool{mockTool: mockTool{name: "named"}}
		name := registry.resolveToolName(ctx, tool)
		if name != "named" {
			t.Errorf("名称 = %q, 期望 named", name)
		}
	})

	t.Run("普通工具通过 Info 获取名称", func(t *testing.T) {
		tool := &mockTool{name: "info_tool"}
		name := registry.resolveToolName(ctx, tool)
		if name != "info_tool" {
			t.Errorf("名称 = %q, 期望 info_tool", name)
		}
	})
}

// TestRegistry_Concurrency 测试并发安全
func TestRegistry_Concurrency(t *testing.T) {
	registry := NewRegistry()
	ctx := context.Background()

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			tool := &mockTool{name: "concurrent_tool_" + string(rune('0'+id))}
			registry.Register(tool)
			registry.Get("concurrent_tool_" + string(rune('0'+id)))
			registry.GetTools()
			registry.GetToolNames(ctx)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	if len(registry.tools) != 10 {
		t.Errorf("并发注册后 tools 长度 = %d, 期望 10", len(registry.tools))
	}
}

// 确保接口实现
var _ tool.BaseTool = (*mockTool)(nil)
var _ tool.InvokableTool = (*mockInvokableTool)(nil)
var _ NamedTool = (*mockNamedTool)(nil)
