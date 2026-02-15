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
