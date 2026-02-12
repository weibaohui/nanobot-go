package config

import (
	"os"
	"path/filepath"
)

// GetConfigPath 获取默认配置文件路径
func GetConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nanobot", "config.json")
}

// GetDataDir 获取 nanobot 数据目录
func GetDataDir() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".nanobot")
	os.MkdirAll(dir, 0755)
	return dir
}

// GetWorkspacePath 获取工作区路径
func GetWorkspacePath(workspace string) string {
	if workspace != "" {
		path := expandPath(workspace)
		os.MkdirAll(path, 0755)
		return path
	}
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".nanobot", "workspace")
	os.MkdirAll(path, 0755)
	return path
}

// GetSessionsPath 获取会话存储目录
func GetSessionsPath() string {
	dir := filepath.Join(GetDataDir(), "sessions")
	os.MkdirAll(dir, 0755)
	return dir
}

// GetMemoryPath 获取内存目录
func GetMemoryPath(workspace string) string {
	ws := GetWorkspacePath(workspace)
	dir := filepath.Join(ws, "memory")
	os.MkdirAll(dir, 0755)
	return dir
}

// GetSkillsPath 获取技能目录
func GetSkillsPath(workspace string) string {
	ws := GetWorkspacePath(workspace)
	dir := filepath.Join(ws, "skills")
	os.MkdirAll(dir, 0755)
	return dir
}

// expandPath 展开路径中的 ~ 为用户主目录
func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[1:])
	}
	return path
}
