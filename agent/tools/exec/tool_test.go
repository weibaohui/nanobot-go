package exec

import (
	"context"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
)

// TestTool_Name 测试工具名称
func TestTool_Name(t *testing.T) {
	tool := &Tool{}
	if tool.Name() != "exec" {
		t.Errorf("Name() = %q, 期望 exec", tool.Name())
	}
}

// TestTool_Info 测试工具信息
func TestTool_Info(t *testing.T) {
	tool := &Tool{}
	ctx := context.Background()

	info, err := tool.Info(ctx)
	if err != nil {
		t.Errorf("Info() 返回错误: %v", err)
	}

	if info.Name != "exec" {
		t.Errorf("Info.Name = %q, 期望 exec", info.Name)
	}
}

// TestTool_Run 测试执行工具
func TestTool_Run(t *testing.T) {
	t.Run("执行 echo 命令", func(t *testing.T) {
		tool := &Tool{}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"command": "echo hello"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("Run() 不应该返回空结果")
		}
	})

	t.Run("执行 ls 命令", func(t *testing.T) {
		tool := &Tool{}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"command": "ls /"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("Run() 不应该返回空结果")
		}
	})

	t.Run("执行不存在的命令", func(t *testing.T) {
		tool := &Tool{}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"command": "nonexistent_command_12345"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("Run() 不应该返回空结果")
		}
	})

	t.Run("执行带工作目录的命令", func(t *testing.T) {
		tool := &Tool{WorkingDir: "/tmp"}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"command": "pwd"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("Run() 不应该返回空结果")
		}
	})
}

// TestTool_InvokableRun 测试 InvokableRun 方法
func TestTool_InvokableRun(t *testing.T) {
	tool := &Tool{}
	ctx := context.Background()

	result, err := tool.InvokableRun(ctx, `{"command": "echo test"}`)
	if err != nil {
		t.Errorf("InvokableRun() 返回错误: %v", err)
	}

	if result == "" {
		t.Error("InvokableRun() 不应该返回空结果")
	}
}

// TestTool_InfoParams 测试工具参数信息
func TestTool_InfoParams(t *testing.T) {
	tool := &Tool{}
	ctx := context.Background()

	info, err := tool.Info(ctx)
	if err != nil {
		t.Errorf("Info() 返回错误: %v", err)
	}

	if info.ParamsOneOf == nil {
		t.Error("ParamsOneOf 不应该为 nil")
	}
}

// TestTool_Interface 测试工具接口实现
func TestTool_Interface(t *testing.T) {
	tool := &Tool{}

	var _ interface {
		Name() string
		Info(ctx context.Context) (*schema.ToolInfo, error)
	} = tool
}

// TestTool_Timeout 测试超时设置
func TestTool_Timeout(t *testing.T) {
	tool := &Tool{Timeout: 5}

	if tool.Timeout != 5 {
		t.Errorf("Timeout = %d, 期望 5", tool.Timeout)
	}
}

// TestTool_WorkingDir 测试工作目录设置
func TestTool_WorkingDir(t *testing.T) {
	tool := &Tool{WorkingDir: "/tmp"}

	if tool.WorkingDir != "/tmp" {
		t.Errorf("WorkingDir = %q, 期望 /tmp", tool.WorkingDir)
	}
}

// TestTool_RestrictToWorkspace 测试工作区限制设置
func TestTool_RestrictToWorkspace(t *testing.T) {
	tool := &Tool{RestrictToWorkspace: true}

	if !tool.RestrictToWorkspace {
		t.Error("RestrictToWorkspace 应该为 true")
	}
}

// TestTool_ExecuteWithOutput 测试执行带输出的命令
func TestTool_ExecuteWithOutput(t *testing.T) {
	tool := &Tool{}
	ctx := context.Background()

	result, err := tool.Run(ctx, `{"command": "printf 'line1\nline2\nline3'"}`)
	if err != nil {
		t.Errorf("Run() 返回错误: %v", err)
	}

	if result == "" {
		t.Error("Run() 不应该返回空结果")
	}
}

// TestTool_ExecuteWithExitCode 测试执行带退出码的命令
func TestTool_ExecuteWithExitCode(t *testing.T) {
	tool := &Tool{}
	ctx := context.Background()

	result, err := tool.Run(ctx, `{"command": "exit 1"}`)
	if err != nil {
		t.Errorf("Run() 返回错误: %v", err)
	}

	if result == "" {
		t.Error("Run() 不应该返回空结果")
	}
}

// TestTool_ContextCancellation 测试上下文取消
func TestTool_ContextCancellation(t *testing.T) {
	tool := &Tool{}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := tool.Run(ctx, `{"command": "sleep 10"}`)
	if err != nil {
		t.Errorf("Run() 返回错误: %v", err)
	}

	if result == "" {
		t.Error("Run() 不应该返回空结果")
	}
}

// TestTool_LongOutput 测试长输出截断
func TestTool_LongOutput(t *testing.T) {
	tool := &Tool{}
	ctx := context.Background()

	result, err := tool.Run(ctx, `{"command": "yes | head -n 200"}`)
	if err != nil {
		t.Errorf("Run() 返回错误: %v", err)
	}

	if len(result) > 10050 {
		t.Errorf("结果长度 = %d, 应该被截断到约 10000", len(result))
	}
}
