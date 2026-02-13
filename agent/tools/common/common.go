package common

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func DecodeArgs(argumentsInJSON string, out any) error {
	if argumentsInJSON == "" || argumentsInJSON == "{}" {
		return nil
	}
	return json.Unmarshal([]byte(argumentsInJSON), out)
}

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

func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
