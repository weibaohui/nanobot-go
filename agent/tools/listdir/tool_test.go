package listdir

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
	if tool.Name() != "list_dir" {
		t.Errorf("Name() = %q, 期望 list_dir", tool.Name())
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

	if info.Name != "list_dir" {
		t.Errorf("Info.Name = %q, 期望 list_dir", info.Name)
	}
}

// TestTool_Run 测试执行工具
func TestTool_Run(t *testing.T) {
	t.Run("列出非空目录", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content"), 0644)
		os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

		tool := &Tool{AllowedDir: tmpDir}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"path": "`+tmpDir+`"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("Run() 不应该返回空结果")
		}
	})

	t.Run("列出空目录", func(t *testing.T) {
		tmpDir := t.TempDir()

		tool := &Tool{AllowedDir: tmpDir}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"path": "`+tmpDir+`"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result != "目录为空" {
			t.Errorf("Run() = %q, 期望 目录为空", result)
		}
	})

	t.Run("列出不存在目录", func(t *testing.T) {
		tool := &Tool{AllowedDir: "/tmp"}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"path": "/tmp/nonexistent_dir_12345"}`)
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
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("content"), 0644)

	tool := &Tool{AllowedDir: tmpDir}
	ctx := context.Background()

	result, err := tool.InvokableRun(ctx, `{"path": "`+tmpDir+`"}`)
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

// TestTool_ListSubdirectory 测试列出子目录
func TestTool_ListSubdirectory(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("content"), 0644)

	tool := &Tool{AllowedDir: tmpDir}
	ctx := context.Background()

	result, err := tool.Run(ctx, `{"path": "`+subDir+`"}`)
	if err != nil {
		t.Errorf("Run() 返回错误: %v", err)
	}

	if result == "" {
		t.Error("Run() 不应该返回空结果")
	}
}

// TestTool_ListWithFilesAndDirs 测试列出文件和目录混合
func TestTool_ListWithFilesAndDirs(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("content"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "dir1"), 0755)
	os.Mkdir(filepath.Join(tmpDir, "dir2"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "another.txt"), []byte("content"), 0644)

	tool := &Tool{AllowedDir: tmpDir}
	ctx := context.Background()

	result, err := tool.Run(ctx, `{"path": "`+tmpDir+`"}`)
	if err != nil {
		t.Errorf("Run() 返回错误: %v", err)
	}

	if result == "" {
		t.Error("Run() 不应该返回空结果")
	}
}
