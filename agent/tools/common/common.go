package common

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// DecodeArgs 解析 JSON 参数到结构体
func DecodeArgs(argumentsInJSON string, out any) error {
	if argumentsInJSON == "" || argumentsInJSON == "{}" {
		return nil
	}
	return json.Unmarshal([]byte(argumentsInJSON), out)
}

// ResolvePath 解析并校验路径
func ResolvePath(path, allowedDir string) string {
	if strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[1:])
	}
	absPath, _ := filepath.Abs(path)
	if allowedDir != "" {
		allowedAbs, _ := filepath.Abs(allowedDir)
		if !strings.HasPrefix(absPath, allowedAbs) {
			return path
		}
	}
	return absPath
}

// TruncateString 截断字符串
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
