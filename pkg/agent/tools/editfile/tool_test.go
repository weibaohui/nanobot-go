package editfile

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
	if tool.Name() != "edit_file" {
		t.Errorf("Name() = %q, 期望 edit_file", tool.Name())
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

	if info.Name != "edit_file" {
		t.Errorf("Info.Name = %q, 期望 edit_file", info.Name)
	}
}

// TestTool_Run 测试执行工具
func TestTool_Run(t *testing.T) {
	t.Run("成功替换文本", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		originalContent := "Hello World"
		os.WriteFile(testFile, []byte(originalContent), 0644)

		tool := &Tool{AllowedDir: tmpDir}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"path": "`+testFile+`", "old_text": "World", "new_text": "Go"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("Run() 不应该返回空结果")
		}

		data, _ := os.ReadFile(testFile)
		if string(data) != "Hello Go" {
			t.Errorf("文件内容 = %q, 期望 Hello Go", string(data))
		}
	})

	t.Run("文件不存在", func(t *testing.T) {
		tool := &Tool{AllowedDir: "/tmp"}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"path": "/tmp/nonexistent_file_12345.txt", "old_text": "old", "new_text": "new"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("Run() 不应该返回空结果")
		}
	})

	t.Run("old_text 未找到", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("Hello World"), 0644)

		tool := &Tool{AllowedDir: tmpDir}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"path": "`+testFile+`", "old_text": "NotFound", "new_text": "new"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result != "错误: old_text 在文件中未找到" {
			t.Errorf("Run() = %q, 期望 错误: old_text 在文件中未找到", result)
		}
	})

	t.Run("替换多行文本", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		originalContent := "line1\nline2\nline3"
		os.WriteFile(testFile, []byte(originalContent), 0644)

		tool := &Tool{AllowedDir: tmpDir}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"path": "`+testFile+`", "old_text": "line2", "new_text": "replaced"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("Run() 不应该返回空结果")
		}

		data, _ := os.ReadFile(testFile)
		expected := "line1\nreplaced\nline3"
		if string(data) != expected {
			t.Errorf("文件内容 = %q, 期望 %q", string(data), expected)
		}
	})

	t.Run("只替换第一个匹配", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("aaa aaa aaa"), 0644)

		tool := &Tool{AllowedDir: tmpDir}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"path": "`+testFile+`", "old_text": "aaa", "new_text": "bbb"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("Run() 不应该返回空结果")
		}

		data, _ := os.ReadFile(testFile)
		if string(data) != "bbb aaa aaa" {
			t.Errorf("文件内容 = %q, 期望 bbb aaa aaa", string(data))
		}
	})
}

// TestTool_InvokableRun 测试 InvokableRun 方法
func TestTool_InvokableRun(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("Hello World"), 0644)

	tool := &Tool{AllowedDir: tmpDir}
	ctx := context.Background()

	result, err := tool.InvokableRun(ctx, `{"path": "`+testFile+`", "old_text": "World", "new_text": "Go"}`)
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

// TestTool_ReplaceWithEmpty 测试替换为空字符串
func TestTool_ReplaceWithEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("Hello World"), 0644)

	tool := &Tool{AllowedDir: tmpDir}
	ctx := context.Background()

	result, err := tool.Run(ctx, `{"path": "`+testFile+`", "old_text": "World", "new_text": ""}`)
	if err != nil {
		t.Errorf("Run() 返回错误: %v", err)
	}

	if result == "" {
		t.Error("Run() 不应该返回空结果")
	}

	data, _ := os.ReadFile(testFile)
	if string(data) != "Hello " {
		t.Errorf("文件内容 = %q, 期望 Hello ", string(data))
	}
}
