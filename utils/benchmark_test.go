package utils

import (
	"strings"
	"testing"
)

// BenchmarkTruncateString 基准测试字符串截断
func BenchmarkTruncateString(b *testing.B) {
	testCases := []struct {
		name   string
		input  string
		maxLen int
	}{
		{"short", "hello", 10},
		{"long", strings.Repeat("hello world ", 100), 50},
		{"chinese", "你好世界测试字符串", 5},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = TruncateString(tc.input, tc.maxLen)
			}
		})
	}
}

// BenchmarkContainsInsensitive 基准测试不区分大小写包含检查
func BenchmarkContainsInsensitive(b *testing.B) {
	testCases := []struct {
		name   string
		s      string
		substr string
	}{
		{"short-match", "Hello World", "hello"},
		{"long-text", strings.Repeat("The quick brown fox jumps over the lazy dog. ", 100), "FOX"},
		{"no-match", "Hello World", "xyz"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = ContainsInsensitive(tc.s, tc.substr)
			}
		})
	}
}

// BenchmarkWrapError 基准测试错误包装
func BenchmarkWrapError(b *testing.B) {
	err := &testError{msg: "original error"}
	b.Run("wrap", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = WrapError("operation", err)
		}
	})
	b.Run("nil", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = WrapError("operation", nil)
		}
	})
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
