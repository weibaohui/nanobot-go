package common

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// DecodeArgs 解析 JSON 参数到结构体
func DecodeArgs(argumentsInJSON string, out any) error {
	trimmed := strings.TrimSpace(argumentsInJSON)
	if trimmed == "" || trimmed == "{}" {
		return nil
	}
	if err := json.Unmarshal([]byte(trimmed), out); err == nil {
		return nil
	}
	fixed := strings.Trim(trimmed, "\u0000")
	if fixed != trimmed {
		if err := json.Unmarshal([]byte(fixed), out); err == nil {
			return nil
		}
	}
	unquoted, ok := tryUnquoteJSON(fixed)
	if ok {
		if err := json.Unmarshal([]byte(unquoted), out); err == nil {
			return nil
		}
		fixed = unquoted
	}
	balanced := balanceJSON(fixed)
	if balanced != fixed {
		if err := json.Unmarshal([]byte(balanced), out); err == nil {
			return nil
		}
	}
	return json.Unmarshal([]byte(trimmed), out)
}

// tryUnquoteJSON 尝试解包被包裹的 JSON 字符串
func tryUnquoteJSON(input string) (string, bool) {
	if len(input) < 2 {
		return input, false
	}
	if (input[0] == '"' && input[len(input)-1] == '"') || (input[0] == '\'' && input[len(input)-1] == '\'') {
		unquoted, err := strconv.Unquote(input)
		if err != nil {
			return input, false
		}
		return unquoted, true
	}
	return input, false
}

// balanceJSON 通过补齐结尾括号修复不完整的 JSON
func balanceJSON(input string) string {
	var (
		inString   bool
		escapeNext bool
		openBrace  int
		closeBrace int
		openBrack  int
		closeBrack int
	)
	for i := 0; i < len(input); i++ {
		ch := input[i]
		if escapeNext {
			escapeNext = false
			continue
		}
		if ch == '\\' && inString {
			escapeNext = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch ch {
		case '{':
			openBrace++
		case '}':
			closeBrace++
		case '[':
			openBrack++
		case ']':
			closeBrack++
		}
	}
	if openBrace == closeBrace && openBrack == closeBrack {
		return input
	}
	var builder strings.Builder
	builder.WriteString(input)
	for i := 0; i < openBrack-closeBrack; i++ {
		builder.WriteByte(']')
	}
	for i := 0; i < openBrace-closeBrace; i++ {
		builder.WriteByte('}')
	}
	return builder.String()
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
