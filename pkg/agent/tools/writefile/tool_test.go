package writefile

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudwego/eino/schema"
)

// TestTool_Name 测试工具名称
func TestTool_Name(t *testing.T) {
	tool := &Tool{}
	if tool.Name() != "write_file" {
		t.Errorf("Name() = %q, 期望 write_file", tool.Name())
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

	if info.Name != "write_file" {
		t.Errorf("Info.Name = %q, 期望 write_file", info.Name)
	}
}

// TestTool_Run 测试执行工具
func TestTool_Run(t *testing.T) {
	t.Run("写入新文件", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		content := "测试文件内容"

		tool := &Tool{AllowedDir: tmpDir}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"path": "`+testFile+`", "content": "`+content+`"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("Run() 不应该返回空结果")
		}

		data, err := os.ReadFile(testFile)
		if err != nil {
			t.Errorf("读取文件失败: %v", err)
		}

		if string(data) != content {
			t.Errorf("文件内容 = %q, 期望 %q", string(data), content)
		}
	})

	t.Run("覆盖已有文件", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("旧内容"), 0644)

		newContent := "新内容"
		tool := &Tool{AllowedDir: tmpDir}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"path": "`+testFile+`", "content": "`+newContent+`"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("Run() 不应该返回空结果")
		}

		data, _ := os.ReadFile(testFile)
		if string(data) != newContent {
			t.Errorf("文件内容 = %q, 期望 %q", string(data), newContent)
		}
	})

	t.Run("创建子目录并写入", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "subdir", "nested", "test.txt")
		content := "嵌套目录文件内容"

		tool := &Tool{AllowedDir: tmpDir}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"path": "`+testFile+`", "content": "`+content+`"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("Run() 不应该返回空结果")
		}

		data, _ := os.ReadFile(testFile)
		if string(data) != content {
			t.Errorf("文件内容 = %q, 期望 %q", string(data), content)
		}
	})

	t.Run("写入空内容", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "empty.txt")

		tool := &Tool{AllowedDir: tmpDir}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"path": "`+testFile+`", "content": ""}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("Run() 不应该返回空结果")
		}

		data, _ := os.ReadFile(testFile)
		if len(data) != 0 {
			t.Errorf("文件内容长度 = %d, 期望 0", len(data))
		}
	})
}

// TestTool_InvokableRun 测试 InvokableRun 方法
func TestTool_InvokableRun(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "测试内容"

	tool := &Tool{AllowedDir: tmpDir}
	ctx := context.Background()

	result, err := tool.InvokableRun(ctx, `{"path": "`+testFile+`", "content": "`+content+`"}`)
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

// TestTool_AllowedDir 测试允许目录
func TestTool_AllowedDir(t *testing.T) {
	tool := &Tool{AllowedDir: "/tmp"}

	if tool.AllowedDir != "/tmp" {
		t.Errorf("AllowedDir = %q, 期望 /tmp", tool.AllowedDir)
	}
}

// TestTool_WriteLargeContent 测试写入大内容
func TestTool_WriteLargeContent(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")

	largeContent := make([]byte, 1024)
	for i := range largeContent {
		largeContent[i] = 'a'
	}

	tool := &Tool{AllowedDir: tmpDir}
	ctx := context.Background()

	result, err := tool.Run(ctx, `{"path": "`+testFile+`", "content": "`+string(largeContent)+`"}`)
	if err != nil {
		t.Errorf("Run() 返回错误: %v", err)
	}

	if result == "" {
		t.Error("Run() 不应该返回空结果")
	}

	data, _ := os.ReadFile(testFile)
	if len(data) != 1024 {
		t.Errorf("文件内容长度 = %d, 期望 1024", len(data))
	}
}
