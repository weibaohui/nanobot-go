package common

import (
	"testing"
)

// TestDecodeArgs 测试解析 JSON 参数
func TestDecodeArgs(t *testing.T) {
	t.Run("解析有效JSON", func(t *testing.T) {
		type TestArgs struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		jsonStr := `{"name": "test", "value": 42}`
		var args TestArgs

		err := DecodeArgs(jsonStr, &args)
		if err != nil {
			t.Fatalf("DecodeArgs 返回错误: %v", err)
		}

		if args.Name != "test" {
			t.Errorf("Name = %q, 期望 test", args.Name)
		}

		if args.Value != 42 {
			t.Errorf("Value = %d, 期望 42", args.Value)
		}
	})

	t.Run("解析空字符串", func(t *testing.T) {
		type TestArgs struct {
			Name string `json:"name"`
		}

		var args TestArgs
		err := DecodeArgs("", &args)
		if err != nil {
			t.Errorf("DecodeArgs 对空字符串应该返回 nil, 但返回: %v", err)
		}
	})

	t.Run("解析空对象", func(t *testing.T) {
		type TestArgs struct {
			Name string `json:"name"`
		}

		var args TestArgs
		err := DecodeArgs("{}", &args)
		if err != nil {
			t.Errorf("DecodeArgs 对空对象应该返回 nil, 但返回: %v", err)
		}
	})

	t.Run("解析带空格的JSON", func(t *testing.T) {
		type TestArgs struct {
			Name string `json:"name"`
		}

		var args TestArgs
		err := DecodeArgs("  {\"name\": \"test\"}  ", &args)
		if err != nil {
			t.Fatalf("DecodeArgs 返回错误: %v", err)
		}

		if args.Name != "test" {
			t.Errorf("Name = %q, 期望 test", args.Name)
		}
	})

	t.Run("解析被引号包裹的JSON", func(t *testing.T) {
		type TestArgs struct {
			Name string `json:"name"`
		}

		var args TestArgs
		// 被双引号包裹的 JSON 字符串
		err := DecodeArgs(`"{\"name\": \"test\"}"`, &args)
		if err != nil {
			t.Fatalf("DecodeArgs 返回错误: %v", err)
		}

		if args.Name != "test" {
			t.Errorf("Name = %q, 期望 test", args.Name)
		}
	})

	t.Run("解析不完整的JSON并自动修复", func(t *testing.T) {
		type TestArgs struct {
			Name string `json:"name"`
		}

		var args TestArgs
		// 缺少闭合括号的 JSON
		err := DecodeArgs(`{"name": "test"`, &args)
		if err != nil {
			t.Fatalf("DecodeArgs 返回错误: %v", err)
		}

		if args.Name != "test" {
			t.Errorf("Name = %q, 期望 test", args.Name)
		}
	})
}

// TestTryUnquoteJSON 测试解包被包裹的 JSON 字符串
func TestTryUnquoteJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectUnzip bool
	}{
		{"双引号包裹", `"hello"`, "hello", true},
		{"无引号", `hello`, "hello", false},
		{"空字符串", `""`, "", true},
		{"带转义", `"hello\"world"`, "hello\"world", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := tryUnquoteJSON(tt.input)
			if ok != tt.expectUnzip {
				t.Errorf("返回的 ok = %v, 期望 %v", ok, tt.expectUnzip)
			}

			if result != tt.expected {
				t.Errorf("结果 = %q, 期望 %q", result, tt.expected)
			}
		})
	}
}

// TestBalanceJSON 测试修复不完整的 JSON
func TestBalanceJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"完整JSON", `{"key": "value"}`, `{"key": "value"}`},
		{"缺少闭合大括号", `{"key": "value"`, `{"key": "value"}`},
		{"缺少多个闭合括号", `{"arr": [1, 2`, `{"arr": [1, 2]}`},
		{"缺少数组和大括号", `{"arr": [1`, `{"arr": [1]}`},
		{"嵌套结构", `{"outer": {"inner": 1`, `{"outer": {"inner": 1}}`},
		{"完整数组", `[1, 2, 3]`, `[1, 2, 3]`},
		{"字符串中的括号", `{"text": "a{b}c"}`, `{"text": "a{b}c"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := balanceJSON(tt.input)
			if result != tt.expected {
				t.Errorf("balanceJSON() = %q, 期望 %q", result, tt.expected)
			}
		})
	}
}

// TestResolvePath 测试路径解析
func TestResolvePath(t *testing.T) {
	t.Run("绝对路径", func(t *testing.T) {
		result := ResolvePath("/tmp/test", "")
		if result != "/tmp/test" {
			t.Errorf("ResolvePath() = %q, 期望 /tmp/test", result)
		}
	})

	t.Run("相对路径转绝对路径", func(t *testing.T) {
		result := ResolvePath("test.txt", "")
		if result == "test.txt" {
			t.Error("相对路径应该被转换为绝对路径")
		}
	})

	t.Run("波浪号展开", func(t *testing.T) {
		result := ResolvePath("~/test", "")
		if len(result) < 2 || result[:2] == "~/" {
			t.Error("波浪号路径应该被展开")
		}
	})
}

// TestTruncateString 测试字符串截断
func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"短字符串不截断", "hello", 10, "hello"},
		{"长字符串截断", "hello world", 5, "hello..."},
		{"刚好等于最大长度", "hello", 5, "hello"},
		{"空字符串", "", 10, ""},
		{"最大长度为0", "hello", 0, "..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("TruncateString() = %q, 期望 %q", result, tt.expected)
			}
		})
	}
}

// TestDecodeArgs_ComplexStruct 测试解析复杂结构
func TestDecodeArgs_ComplexStruct(t *testing.T) {
	type Nested struct {
		Value string `json:"value"`
	}

	type ComplexArgs struct {
		Name    string   `json:"name"`
		Numbers []int    `json:"numbers"`
		Nested  *Nested  `json:"nested,omitempty"`
		Enabled bool     `json:"enabled"`
	}

	jsonStr := `{
		"name": "complex",
		"numbers": [1, 2, 3],
		"nested": {"value": "inner"},
		"enabled": true
	}`

	var args ComplexArgs
	err := DecodeArgs(jsonStr, &args)
	if err != nil {
		t.Fatalf("DecodeArgs 返回错误: %v", err)
	}

	if args.Name != "complex" {
		t.Errorf("Name = %q, 期望 complex", args.Name)
	}

	if len(args.Numbers) != 3 {
		t.Errorf("Numbers 长度 = %d, 期望 3", len(args.Numbers))
	}

	if args.Nested == nil || args.Nested.Value != "inner" {
		t.Errorf("Nested.Value 不正确")
	}

	if !args.Enabled {
		t.Error("Enabled 应该为 true")
	}
}

// TestDecodeArgs_WithNullBytes 测试包含空字节的 JSON
func TestDecodeArgs_WithNullBytes(t *testing.T) {
	type TestArgs struct {
		Name string `json:"name"`
	}

	var args TestArgs
	// 包含空字节的 JSON
	err := DecodeArgs("\x00{\"name\": \"test\"}\x00", &args)
	if err != nil {
		t.Fatalf("DecodeArgs 返回错误: %v", err)
	}

	if args.Name != "test" {
		t.Errorf("Name = %q, 期望 test", args.Name)
	}
}
