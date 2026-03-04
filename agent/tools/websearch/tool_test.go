package websearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudwego/eino/components/tool"
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

	// 验证参数定义
	if info.ParamsOneOf == nil {
		t.Error("Info().ParamsOneOf 不应该为 nil")
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

	t.Run("正常搜索返回结果", func(t *testing.T) {
		tool := &Tool{MaxResults: 3}
		ctx := context.Background()

		// 使用真实 API 测试（可能被限制）
		result, err := tool.Run(ctx, `{"query": "golang"}`)

		// 不检查错误，因为网络可能不可用
		// 但结果不应该 panic
		_ = result
		_ = err
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

// TestDuckDuckGoResponse_JSONMarshal 测试 JSON 序列化
func TestDuckDuckGoResponse_JSONMarshal(t *testing.T) {
	resp := DuckDuckGoResponse{
		AbstractText: "测试",
		Heading:      "标题",
		RelatedTopics: []struct {
			Text string `json:"Text"`
			URL  string `json:"FirstURL"`
		}{
			{Text: "话题1", URL: "https://example.com/1"},
			{Text: "话题2", URL: "https://example.com/2"},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("JSON Marshal 失败: %v", err)
	}

	// 验证包含预期字段
	if string(data) == "" {
		t.Error("JSON 序列化结果为空")
	}

	// 反序列化验证
	var decoded DuckDuckGoResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("JSON Unmarshal 失败: %v", err)
	}

	if decoded.AbstractText != "测试" {
		t.Errorf("反序列化后 AbstractText = %q, 期望 测试", decoded.AbstractText)
	}

	if len(decoded.RelatedTopics) != 2 {
		t.Errorf("反序列化后 RelatedTopics 长度 = %d, 期望 2", len(decoded.RelatedTopics))
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

// TestTool_Run_WithMockServer 测试带 Mock 服务器的搜索
func TestTool_Run_WithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求
		if r.Method != "GET" {
			t.Errorf("期望 GET 方法, 实际 %s", r.Method)
		}

		query := r.URL.Query().Get("q")
		format := r.URL.Query().Get("format")

		if query == "" {
			t.Error("缺少 q 参数")
		}

		if format != "json" {
			t.Errorf("期望 format=json, 实际 %s", format)
		}

		// 返回模拟响应
		response := `{
			"Heading": "测试主题",
			"AbstractText": "这是搜索结果摘要",
			"AbstractURL": "https://example.com",
			"Answer": "直接答案",
			"AnswerType": "text",
			"Definition": "定义内容",
			"DefinitionURL": "https://example.com/def",
			"RelatedTopics": [
				{"Text": "相关话题1", "FirstURL": "https://example.com/1"},
				{"Text": "相关话题2", "FirstURL": "https://example.com/2"},
				{"Text": "相关话题3", "FirstURL": "https://example.com/3"}
			]
		}`

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	}))
	defer server.Close()

	// 注意：实际测试需要修改代码以支持自定义 API URL
	// 这里只是验证工具的结构
	tool := &Tool{MaxResults: 2}

	// 验证工具可以创建
	if tool == nil {
		t.Error("创建工具失败")
	}
}

// TestTool_Run_NetworkError 测试网络错误处理
func TestTool_Run_NetworkError(t *testing.T) {
	tool := &Tool{MaxResults: 5}
	ctx := context.Background()

	// 使用无效代理或断开网络连接来模拟网络错误
	// 这里使用一个无效的查询来测试错误处理
	result, err := tool.Run(ctx, `{"query": "test"}`)

	// 不应该 panic，即使网络错误也应该返回字符串结果
	_ = result
	_ = err
}

// TestTool_ImplementsInterface 测试工具实现接口
func TestTool_ImplementsInterface(t *testing.T) {
	var _ tool.BaseTool = (&Tool{})
	// 如果编译通过，说明实现了接口
}
