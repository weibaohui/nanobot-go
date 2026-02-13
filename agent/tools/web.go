package tools

import (
	"context"
)

// WebSearchTool 网络搜索工具
type WebSearchTool struct {
	APIKey     string
	MaxResults int
}

func (t *WebSearchTool) Name() string        { return "web_search" }
func (t *WebSearchTool) Description() string { return "搜索网络" }
func (t *WebSearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string", "description": "搜索查询"},
		},
		"required": []string{"query"},
	}
}
func (t *WebSearchTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	if t.APIKey == "" {
		return "错误: BRAVE_API_KEY 未配置", nil
	}
	return "网络搜索功能需要实现 Brave API 调用", nil
}
func (t *WebSearchTool) ToSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        t.Name(),
			"description": t.Description(),
			"parameters":  t.Parameters(),
		},
	}
}

// WebFetchTool 网页获取工具
type WebFetchTool struct {
	MaxChars int
}

func (t *WebFetchTool) Name() string        { return "web_fetch" }
func (t *WebFetchTool) Description() string { return "获取网页内容" }
func (t *WebFetchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{"type": "string", "description": "URL"},
		},
		"required": []string{"url"},
	}
}
func (t *WebFetchTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	return "网页获取功能需要实现 HTTP 请求", nil
}
func (t *WebFetchTool) ToSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        t.Name(),
			"description": t.Description(),
			"parameters":  t.Parameters(),
		},
	}
}
