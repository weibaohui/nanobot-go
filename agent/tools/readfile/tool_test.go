package readfile

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
	if tool.Name() != "read_file" {
		t.Errorf("Name() = %q, 期望 read_file", tool.Name())
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

	if info.Name != "read_file" {
		t.Errorf("Info.Name = %q, 期望 read_file", info.Name)
	}
}

// TestTool_Run 测试执行工具
func TestTool_Run(t *testing.T) {
	t.Run("读取存在的文件", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		content := "测试文件内容"
		os.WriteFile(testFile, []byte(content), 0644)

		tool := &Tool{AllowedDir: tmpDir}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"path": "`+testFile+`"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result != content {
			t.Errorf("Run() = %q, 期望 %q", result, content)
		}
	})

	t.Run("读取不存在的文件", func(t *testing.T) {
		tool := &Tool{AllowedDir: "/tmp"}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"path": "/tmp/nonexistent_file_12345.txt"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("Run() 不应该返回空结果")
		}
	})

	t.Run("读取绝对路径文件", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		content := "测试文件内容"
		os.WriteFile(testFile, []byte(content), 0644)

		tool := &Tool{}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"path": "`+testFile+`"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result != content {
			t.Errorf("Run() = %q, 期望 %q", result, content)
		}
	})
}

// TestTool_InvokableRun 测试 InvokableRun 方法
func TestTool_InvokableRun(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "测试文件内容"
	os.WriteFile(testFile, []byte(content), 0644)

	tool := &Tool{AllowedDir: tmpDir}
	ctx := context.Background()

	result, err := tool.InvokableRun(ctx, `{"path": "`+testFile+`"}`)
	if err != nil {
		t.Errorf("InvokableRun() 返回错误: %v", err)
	}

	if result != content {
		t.Errorf("InvokableRun() = %q, 期望 %q", result, content)
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

// TestTool_EmptyAllowedDir 测试空允许目录
func TestTool_EmptyAllowedDir(t *testing.T) {
	tool := &Tool{}

	if tool.AllowedDir != "" {
		t.Errorf("AllowedDir = %q, 期望空字符串", tool.AllowedDir)
	}
}

// TestTool_ReadMultipleFiles 测试读取多个文件
func TestTool_ReadMultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	os.WriteFile(file1, []byte("文件1内容"), 0644)
	os.WriteFile(file2, []byte("文件2内容"), 0644)

	tool := &Tool{AllowedDir: tmpDir}
	ctx := context.Background()

	result1, err := tool.Run(ctx, `{"path": "`+file1+`"}`)
	if err != nil {
		t.Errorf("Run() 返回错误: %v", err)
	}

	result2, err := tool.Run(ctx, `{"path": "`+file2+`"}`)
	if err != nil {
		t.Errorf("Run() 返回错误: %v", err)
	}

	if result1 != "文件1内容" {
		t.Errorf("result1 = %q, 期望 文件1内容", result1)
	}

	if result2 != "文件2内容" {
		t.Errorf("result2 = %q, 期望 文件2内容", result2)
	}
}

// TestTool_ReadSubdirectoryFile 测试读取子目录文件
func TestTool_ReadSubdirectoryFile(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	os.MkdirAll(subDir, 0755)

	testFile := filepath.Join(subDir, "test.txt")
	content := "子目录文件内容"
	os.WriteFile(testFile, []byte(content), 0644)

	tool := &Tool{AllowedDir: tmpDir}
	ctx := context.Background()

	result, err := tool.Run(ctx, `{"path": "`+testFile+`"}`)
	if err != nil {
		t.Errorf("Run() 返回错误: %v", err)
	}

	if result != content {
		t.Errorf("Run() = %q, 期望 %q", result, content)
	}
}

// TestTool_ReadEmptyFile 测试读取空文件
func TestTool_ReadEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.txt")
	os.WriteFile(testFile, []byte{}, 0644)

	tool := &Tool{AllowedDir: tmpDir}
	ctx := context.Background()

	result, err := tool.Run(ctx, `{"path": "`+testFile+`"}`)
	if err != nil {
		t.Errorf("Run() 返回错误: %v", err)
	}

	if result != "" {
		t.Errorf("Run() = %q, 期望空字符串", result)
	}
}

// TestTool_ReadLargeFile 测试读取大文件
func TestTool_ReadLargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")

	largeContent := make([]byte, 1024)
	for i := range largeContent {
		largeContent[i] = 'a'
	}
	os.WriteFile(testFile, largeContent, 0644)

	tool := &Tool{AllowedDir: tmpDir}
	ctx := context.Background()

	result, err := tool.Run(ctx, `{"path": "`+testFile+`"}`)
	if err != nil {
		t.Errorf("Run() 返回错误: %v", err)
	}

	if len(result) != 1024 {
		t.Errorf("result 长度 = %d, 期望 1024", len(result))
	}
}
