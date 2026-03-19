package config

import (
	"os"
	"path/filepath"
)

// getWorkingDir 获取当前工作目录
func getWorkingDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

// GetConfigPath 获取默认配置文件路径
// 固定路径：当前工作目录/config.json
func GetConfigPath() string {
	return filepath.Join(getWorkingDir(), "config.json")
}

// GetDataDir 获取 nanobot 数据目录
// 固定路径：当前工作目录/data
func GetDataDir() string {
	dir := filepath.Join(getWorkingDir(), "data")
	os.MkdirAll(dir, 0755)
	return dir
}

// GetWorkspacePath 获取工作区路径
// 固定路径：当前工作目录/workspace
func GetWorkspacePath(workspace string) string {
	if workspace != "" {
		path := expandPath(workspace)
		os.MkdirAll(path, 0755)
		return path
	}
	path := filepath.Join(getWorkingDir(), "workspace")
	os.MkdirAll(path, 0755)
	return path
}

// GetSessionsPath 获取会话存储目录
// 固定路径：程序所在目录/data/sessions
func GetSessionsPath() string {
	dir := filepath.Join(GetDataDir(), "sessions")
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
