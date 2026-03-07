package utils

import (
	"strings"
)

// ContainsInsensitive 检查字符串 s 是否包含子串 substr（不区分大小写）
func ContainsInsensitive(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// HasPrefixInsensitive 检查字符串 s 是否以 substr 开头（不区分大小写）
func HasPrefixInsensitive(s, prefix string) bool {
	return strings.HasPrefix(strings.ToLower(s), strings.ToLower(prefix))
}

// HasSuffixInsensitive 检查字符串 s 是否以 substr 结尾（不区分大小写）
func HasSuffixInsensitive(s, suffix string) bool {
	return strings.HasSuffix(strings.ToLower(s), strings.ToLower(suffix))
}

// TruncateString 截断字符串到指定长度（按 Unicode 字符计算），超出部分用 "..." 省略
func TruncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
