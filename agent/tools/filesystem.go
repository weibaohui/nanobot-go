package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ========== 文件工具 ==========

// ReadFileTool 读取文件工具
type ReadFileTool struct {
	AllowedDir string
}

func (t *ReadFileTool) Name() string        { return "read_file" }
func (t *ReadFileTool) Description() string { return "读取指定路径的文件内容" }
func (t *ReadFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string", "description": "要读取的文件路径"},
		},
		"required": []string{"path"},
	}
}
func (t *ReadFileTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	path, _ := params["path"].(string)
	resolved := resolvePath(path, t.AllowedDir)
	data, err := os.ReadFile(resolved)
	if err != nil {
		return fmt.Sprintf("错误: 读取文件失败: %s", err), nil
	}
	return string(data), nil
}
func (t *ReadFileTool) ToSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        t.Name(),
			"description": t.Description(),
			"parameters":  t.Parameters(),
		},
	}
}

// WriteFileTool 写入文件工具
type WriteFileTool struct {
	AllowedDir string
}

func (t *WriteFileTool) Name() string { return "write_file" }
func (t *WriteFileTool) Description() string {
	return "将内容写入文件，必要时创建父目录"
}
func (t *WriteFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":    map[string]any{"type": "string", "description": "文件路径"},
			"content": map[string]any{"type": "string", "description": "要写入的内容"},
		},
		"required": []string{"path", "content"},
	}
}
func (t *WriteFileTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	path, _ := params["path"].(string)
	content, _ := params["content"].(string)
	resolved := resolvePath(path, t.AllowedDir)
	os.MkdirAll(filepath.Dir(resolved), 0755)
	if err := os.WriteFile(resolved, []byte(content), 0644); err != nil {
		return "", err
	}
	return fmt.Sprintf("成功写入 %d 字节到 %s", len(content), path), nil
}
func (t *WriteFileTool) ToSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        t.Name(),
			"description": t.Description(),
			"parameters":  t.Parameters(),
		},
	}
}

// EditFileTool 编辑文件工具
type EditFileTool struct {
	AllowedDir string
}

func (t *EditFileTool) Name() string        { return "edit_file" }
func (t *EditFileTool) Description() string { return "通过替换文本编辑文件" }
func (t *EditFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":     map[string]any{"type": "string", "description": "文件路径"},
			"old_text": map[string]any{"type": "string", "description": "要替换的文本"},
			"new_text": map[string]any{"type": "string", "description": "替换成的文本"},
		},
		"required": []string{"path", "old_text", "new_text"},
	}
}
func (t *EditFileTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	path, _ := params["path"].(string)
	oldText, _ := params["old_text"].(string)
	newText, _ := params["new_text"].(string)
	resolved := resolvePath(path, t.AllowedDir)
	data, err := os.ReadFile(resolved)
	if err != nil {
		return fmt.Sprintf("错误: 文件不存在: %s", path), nil
	}
	content := string(data)
	if !strings.Contains(content, oldText) {
		return "错误: old_text 在文件中未找到", nil
	}
	newContent := strings.Replace(content, oldText, newText, 1)
	os.WriteFile(resolved, []byte(newContent), 0644)
	return fmt.Sprintf("成功编辑 %s", path), nil
}
func (t *EditFileTool) ToSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        t.Name(),
			"description": t.Description(),
			"parameters":  t.Parameters(),
		},
	}
}

// ListDirTool 列出目录工具
type ListDirTool struct {
	AllowedDir string
}

func (t *ListDirTool) Name() string        { return "list_dir" }
func (t *ListDirTool) Description() string { return "列出目录内容" }
func (t *ListDirTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string", "description": "目录路径"},
		},
		"required": []string{"path"},
	}
}
func (t *ListDirTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	path, _ := params["path"].(string)
	resolved := resolvePath(path, t.AllowedDir)
	entries, err := os.ReadDir(resolved)
	if err != nil {
		return fmt.Sprintf("错误: 读取目录失败: %s", err), nil
	}
	var lines []string
	for _, e := range entries {
		prefix := "📄 "
		if e.IsDir() {
			prefix = "📁 "
		}
		lines = append(lines, prefix+e.Name())
	}
	if len(lines) == 0 {
		return "目录为空", nil
	}
	return strings.Join(lines, "\n"), nil
}
func (t *ListDirTool) ToSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        t.Name(),
			"description": t.Description(),
			"parameters":  t.Parameters(),
		},
	}
}
