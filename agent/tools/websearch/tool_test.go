package websearch

import (
	"context"
	"testing"
)

// TestTool_Name 测试工具名称
func TestTool_Name(t *testing.T) {
	tool := &Tool{}
	if tool.Name() != "web_search" {
		t.Errorf("Name() = %q, 期望 web_search", tool.Name())
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

	if info.Name != "web_search" {
		t.Errorf("Info().Name = %q, 期望 web_search", info.Name)
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

	t.Run("空查询", func(t *testing.T) {
		tool := &Tool{}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"query": ""}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result != "错误: 搜索查询不能为空" {
			t.Errorf("Run() = %q, 期望 错误: 搜索查询不能为空", result)
		}
	})

	t.Run("无查询字段", func(t *testing.T) {
		tool := &Tool{}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result != "错误: 搜索查询不能为空" {
			t.Errorf("Run() = %q, 期望 错误: 搜索查询不能为空", result)
		}
	})
}

// TestTool_InvokableRun 测试可直接调用的执行入口
func TestTool_InvokableRun(t *testing.T) {
	tool := &Tool{}
	ctx := context.Background()

	result, err := tool.InvokableRun(ctx, `{"query": ""}`)
	if err != nil {
		t.Errorf("InvokableRun() 返回错误: %v", err)
	}

	if result == "" {
		t.Error("InvokableRun() 不应该返回空结果")
	}
}

// TestDuckDuckGoResponse 测试 DuckDuckGo 响应结构
func TestDuckDuckGoResponse(t *testing.T) {
	response := DuckDuckGoResponse{
		AbstractText:   "Test abstract",
		AbstractSource: "Wikipedia",
		AbstractURL:    "https://example.com",
		Heading:        "Test Heading",
		Answer:         "Test Answer",
		AnswerType:     "calc",
		Definition:     "Test Definition",
		DefinitionURL:  "https://definition.com",
	}

	if response.AbstractText != "Test abstract" {
		t.Errorf("DuckDuckGoResponse.AbstractText = %q, 期望 Test abstract", response.AbstractText)
	}

	if response.Heading != "Test Heading" {
		t.Errorf("DuckDuckGoResponse.Heading = %q, 期望 Test Heading", response.Heading)
	}

	if response.Answer != "Test Answer" {
		t.Errorf("DuckDuckGoResponse.Answer = %q, 期望 Test Answer", response.Answer)
	}
}

// TestTool_MaxResults 测试最大结果数设置
func TestTool_MaxResults(t *testing.T) {
	t.Run("默认最大结果数", func(t *testing.T) {
		tool := &Tool{}
		if tool.MaxResults != 0 {
			t.Errorf("Tool.MaxResults = %d, 期望 0 (默认)", tool.MaxResults)
		}
	})

	t.Run("自定义最大结果数", func(t *testing.T) {
		tool := &Tool{MaxResults: 10}
		if tool.MaxResults != 10 {
			t.Errorf("Tool.MaxResults = %d, 期望 10", tool.MaxResults)
		}
	})
}
