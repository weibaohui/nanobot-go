package webfetch

import (
	"context"
	"encoding/json"
	"testing"
)

// TestTool_Name 测试工具名称
func TestTool_Name(t *testing.T) {
	tool := &Tool{}
	if tool.Name() != "web_fetch" {
		t.Errorf("Name() = %q, 期望 web_fetch", tool.Name())
	}
}

// TestTool_ToSchema 测试工具 Schema
func TestTool_ToSchema(t *testing.T) {
	tool := &Tool{}
	schema := tool.ToSchema()

	if schema["type"] != "function" {
		t.Errorf("ToSchema() type = %v, 期望 function", schema["type"])
	}

	fn, ok := schema["function"].(map[string]any)
	if !ok {
		t.Fatal("ToSchema() function 不是 map[string]any")
	}

	if fn["name"] != "web_fetch" {
		t.Errorf("ToSchema() function.name = %v, 期望 web_fetch", fn["name"])
	}
}

// TestTool_Info 测试工具信息
func TestTool_Info(t *testing.T) {
	tool := &Tool{}
	ctx := context.Background()

	info, err := tool.Info(ctx)
	if err != nil {
		t.Errorf("Info() 返回错误: %v", err)
	}

	if info.Name != "web_fetch" {
		t.Errorf("Info().Name = %q, 期望 web_fetch", info.Name)
	}

	if info.Desc == "" {
		t.Error("Info().Desc 不应该为空")
	}
}

// TestTool_Run 测试执行工具
func TestTool_Run(t *testing.T) {
	t.Run("无效JSON参数", func(t *testing.T) {
		tool := &Tool{}
		ctx := context.Background()

		_, err := tool.Run(ctx, "invalid json")
		if err == nil {
			t.Error("Run() 无效 JSON 应该返回错误")
		}
	})

	t.Run("无效URL", func(t *testing.T) {
		tool := &Tool{}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"url": "invalid-url"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		var fetchResult fetchResult
		if err := json.Unmarshal([]byte(result), &fetchResult); err != nil {
			t.Errorf("Run() 返回的结果不是有效 JSON: %v", err)
		}

		if fetchResult.Error == "" {
			t.Error("无效 URL 应该返回错误信息")
		}
	})

	t.Run("不支持协议", func(t *testing.T) {
		tool := &Tool{}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"url": "ftp://example.com/file"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		var fetchResult fetchResult
		if err := json.Unmarshal([]byte(result), &fetchResult); err != nil {
			t.Errorf("Run() 返回的结果不是有效 JSON: %v", err)
		}

		if fetchResult.Error == "" {
			t.Error("不支持协议应该返回错误信息")
		}
	})
}

// TestTool_InvokableRun 测试可直接调用的执行入口
func TestTool_InvokableRun(t *testing.T) {
	tool := &Tool{}
	ctx := context.Background()

	result, err := tool.InvokableRun(ctx, `{"url": "invalid"}`)
	if err != nil {
		t.Errorf("InvokableRun() 返回错误: %v", err)
	}

	if result == "" {
		t.Error("InvokableRun() 不应该返回空结果")
	}
}

// TestValidateURL 测试 URL 验证
func TestValidateURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		hasError bool
	}{
		{"有效HTTP URL", "http://example.com", false},
		{"有效HTTPS URL", "https://example.com/path", false},
		{"无效URL格式", "not-a-url", true},
		{"不支持协议", "ftp://example.com", true},
		{"空URL", "", true},
		{"缺少协议", "example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateURL(tt.url)
			if tt.hasError && err == nil {
				t.Errorf("validateURL(%q) 应该返回错误", tt.url)
			}
			if !tt.hasError && err != nil {
				t.Errorf("validateURL(%q) 不应该返回错误: %v", tt.url, err)
			}
		})
	}
}

// TestIsHTML 测试 HTML 检测
func TestIsHTML(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected bool
	}{
		{"HTML doctype", "<!DOCTYPE html><html>", true},
		{"HTML tag", "<html><body>", true},
		{"JSON内容", `{"key": "value"}`, false},
		{"纯文本", "Hello World", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isHTML([]byte(tt.body))
			if result != tt.expected {
				t.Errorf("isHTML(%q) = %v, 期望 %v", tt.body, result, tt.expected)
			}
		})
	}
}

// TestStripTags 测试移除 HTML 标签
func TestStripTags(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		contains string
	}{
		{"简单标签", "<p>Hello</p>", "Hello"},
		{"嵌套标签", "<div><p>World</p></div>", "World"},
		{"带属性", `<a href="url">Link</a>`, "Link"},
		{"脚本标签", "<script>alert('x')</script>Hello", "Hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripTags(tt.html)
			if !containsSubstring(result, tt.contains) {
				t.Errorf("stripTags(%q) = %q, 应包含 %q", tt.html, result, tt.contains)
			}
		})
	}
}

// TestDecodeHTMLEntities 测试 HTML 实体解码
func TestDecodeHTMLEntities(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"&nbsp;", "Hello&nbsp;World", "Hello World"},
		{"&amp;", "A &amp; B", "A & B"},
		{"&lt;", "x &lt; y", "x < y"},
		{"&gt;", "x &gt; y", "x > y"},
		{"&quot;", `Say &quot;Hi&quot;`, `Say "Hi"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := decodeHTMLEntities(tt.input)
			if result != tt.expected {
				t.Errorf("decodeHTMLEntities(%q) = %q, 期望 %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestNormalize 测试规范化空白
func TestNormalize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"多个空格", "Hello   World", "Hello World"},
		{"多个换行", "Hello\n\n\n\nWorld", "Hello\n\nWorld"},
		{"前后空白", "  Hello World  ", "Hello World"},
		{"制表符", "Hello\t\tWorld", "Hello World"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalize(tt.input)
			if result != tt.expected {
				t.Errorf("normalize(%q) = %q, 期望 %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestToMarkdown 测试 HTML 转 Markdown
func TestToMarkdown(t *testing.T) {
	t.Run("标题转换", func(t *testing.T) {
		html := "<h1>Title</h1>"
		result := toMarkdown(html)
		if !containsSubstring(result, "# Title") {
			t.Errorf("toMarkdown(%q) = %q, 应包含 # Title", html, result)
		}
	})

	t.Run("链接转换", func(t *testing.T) {
		html := `<a href="https://example.com">Link</a>`
		result := toMarkdown(html)
		if !containsSubstring(result, "[Link](https://example.com)") {
			t.Errorf("toMarkdown(%q) = %q, 应包含 [Link](https://example.com)", html, result)
		}
	})

	t.Run("粗体转换", func(t *testing.T) {
		html := "<strong>Bold</strong>"
		result := toMarkdown(html)
		if !containsSubstring(result, "**Bold**") {
			t.Errorf("toMarkdown(%q) = %q, 应包含 **Bold**", html, result)
		}
	})

	t.Run("斜体转换", func(t *testing.T) {
		html := "<em>Italic</em>"
		result := toMarkdown(html)
		if !containsSubstring(result, "*Italic*") {
			t.Errorf("toMarkdown(%q) = %q, 应包含 *Italic*", html, result)
		}
	})

	t.Run("代码转换", func(t *testing.T) {
		html := "<code>code</code>"
		result := toMarkdown(html)
		if !containsSubstring(result, "`code`") {
			t.Errorf("toMarkdown(%q) = %q, 应包含 `code`", html, result)
		}
	})
}

// TestParseInt 测试解析整数
func TestParseInt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"普通数字", "123", 123},
		{"单个数字", "5", 5},
		{"空字符串", "", 0},
		{"非数字", "abc", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseInt(tt.input)
			if result != tt.expected {
				t.Errorf("parseInt(%q) = %d, 期望 %d", tt.input, result, tt.expected)
			}
		})
	}
}

// containsSubstring 检查字符串是否包含子串
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
