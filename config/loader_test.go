package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGetConfigPath 测试获取默认配置文件路径
func TestGetConfigPath(t *testing.T) {
	path := GetConfigPath()
	if path == "" {
		t.Error("GetConfigPath() 返回空路径")
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".nanobot", "config.json")
	if path != expected {
		t.Errorf("GetConfigPath() = %q, 期望 %q", path, expected)
	}
}

// TestGetDataDir 测试获取数据目录
func TestGetDataDir(t *testing.T) {
	dir := GetDataDir()
	if dir == "" {
		t.Error("GetDataDir() 返回空路径")
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".nanobot")
	if dir != expected {
		t.Errorf("GetDataDir() = %q, 期望 %q", dir, expected)
	}

	// 验证目录存在
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("数据目录 %q 不存在", dir)
	}
}

// TestGetWorkspacePath 测试获取工作区路径
func TestGetWorkspacePath(t *testing.T) {
	tests := []struct {
		name      string
		workspace string
		checkFunc func(t *testing.T, path string)
	}{
		{
			name:      "空工作区使用默认路径",
			workspace: "",
			checkFunc: func(t *testing.T, path string) {
				home, _ := os.UserHomeDir()
				expected := filepath.Join(home, ".nanobot", "workspace")
				if path != expected {
					t.Errorf("路径 = %q, 期望 %q", path, expected)
				}
			},
		},
		{
			name:      "使用~开头的路径",
			workspace: "~/test_workspace_unit",
			checkFunc: func(t *testing.T, path string) {
				home, _ := os.UserHomeDir()
				expected := filepath.Join(home, "test_workspace_unit")
				if path != expected {
					t.Errorf("路径 = %q, 期望 %q", path, expected)
				}
			},
		},
		{
			name:      "使用绝对路径",
			workspace: "/tmp/test_workspace_abs_unit",
			checkFunc: func(t *testing.T, path string) {
				if path != "/tmp/test_workspace_abs_unit" {
					t.Errorf("路径 = %q, 期望 %q", path, "/tmp/test_workspace_abs_unit")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := GetWorkspacePath(tt.workspace)
			tt.checkFunc(t, path)

			// 清理测试目录
			if tt.workspace != "" {
				os.RemoveAll(path)
			}
		})
	}
}

// TestGetSessionsPath 测试获取会话存储目录
func TestGetSessionsPath(t *testing.T) {
	path := GetSessionsPath()
	if path == "" {
		t.Error("GetSessionsPath() 返回空路径")
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".nanobot", "sessions")
	if path != expected {
		t.Errorf("GetSessionsPath() = %q, 期望 %q", path, expected)
	}
}

// TestGetMemoryPath 测试获取内存目录
func TestGetMemoryPath(t *testing.T) {
	workspace := "/tmp/test_memory_workspace"
	defer os.RemoveAll(workspace)

	path := GetMemoryPath(workspace)
	expected := filepath.Join(workspace, "memory")
	if path != expected {
		t.Errorf("GetMemoryPath() = %q, 期望 %q", path, expected)
	}

	// 验证目录存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("内存目录 %q 不存在", path)
	}
}

// TestGetSkillsPath 测试获取技能目录
func TestGetSkillsPath(t *testing.T) {
	workspace := "/tmp/test_skills_workspace"
	defer os.RemoveAll(workspace)

	path := GetSkillsPath(workspace)
	expected := filepath.Join(workspace, "skills")
	if path != expected {
		t.Errorf("GetSkillsPath() = %q, 期望 %q", path, expected)
	}

	// 验证目录存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("技能目录 %q 不存在", path)
	}
}

// TestExpandPath 测试路径展开
func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"波浪号展开", "~/test", filepath.Join(home, "test")},
		{"无波浪号", "/absolute/path", "/absolute/path"},
		{"空路径", "", ""},
		{"只有波浪号", "~", home},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPath(tt.path)
			if result != tt.expected {
				t.Errorf("expandPath(%q) = %q, 期望 %q", tt.path, result, tt.expected)
			}
		})
	}
}
