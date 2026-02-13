package tools

import (
	"os"
	"path/filepath"
	"strings"
)

// resolvePath 解析路径
func resolvePath(path, allowedDir string) string {
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

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
