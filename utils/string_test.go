package utils

import "testing"

// TestContainsInsensitive 测试不区分大小写的包含检查
func TestContainsInsensitive(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"完全匹配", "Hello World", "Hello", true},
		{"大小写不同", "Hello World", "HELLO", true},
		{"混合大小写", "HeLLo WoRLD", "hello", true},
		{"不包含", "Hello World", "xyz", false},
		{"空字符串包含空字符串", "", "", true},
		{"非空字符串包含空字符串", "Hello", "", true},
		{"空字符串不包含非空字符串", "", "Hello", false},
		{"中文匹配", "你好世界", "你好", true},
		{"中文不匹配", "你好世界", "再见", false},
		{"特殊字符", "Hello@World", "@", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsInsensitive(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("ContainsInsensitive(%q, %q) = %v, 期望 %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

// TestHasPrefixInsensitive 测试不区分大小写的前缀检查
func TestHasPrefixInsensitive(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		prefix   string
		expected bool
	}{
		{"完全匹配前缀", "Hello World", "Hello", true},
		{"大小写不同", "Hello World", "HELLO", true},
		{"混合大小写", "HeLLo WoRLD", "hello", true},
		{"不是前缀", "Hello World", "World", false},
		{"空前缀", "Hello", "", true},
		{"空字符串非空前缀", "", "Hello", false},
		{"空字符串空前缀", "", "", true},
		{"中文前缀", "你好世界", "你好", true},
		{"中文非前缀", "你好世界", "世界", false},
		{"前缀比字符串长", "Hi", "Hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasPrefixInsensitive(tt.s, tt.prefix)
			if result != tt.expected {
				t.Errorf("HasPrefixInsensitive(%q, %q) = %v, 期望 %v", tt.s, tt.prefix, result, tt.expected)
			}
		})
	}
}

// TestHasSuffixInsensitive 测试不区分大小写的后缀检查
func TestHasSuffixInsensitive(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		suffix   string
		expected bool
	}{
		{"完全匹配后缀", "Hello World", "World", true},
		{"大小写不同", "Hello World", "WORLD", true},
		{"混合大小写", "HeLLo WoRLD", "world", true},
		{"不是后缀", "Hello World", "Hello", false},
		{"空后缀", "Hello", "", true},
		{"空字符串非空后缀", "", "Hello", false},
		{"空字符串空后缀", "", "", true},
		{"中文后缀", "你好世界", "世界", true},
		{"中文非后缀", "你好世界", "你好", false},
		{"后缀比字符串长", "Hi", "Hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasSuffixInsensitive(tt.s, tt.suffix)
			if result != tt.expected {
				t.Errorf("HasSuffixInsensitive(%q, %q) = %v, 期望 %v", tt.s, tt.suffix, result, tt.expected)
			}
		})
	}
}
