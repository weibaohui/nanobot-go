package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// DuckDuckGoResponse DuckDuckGo Instant Answer API 响应结构
type DuckDuckGoResponse struct {
	AbstractText   string `json:"AbstractText"`
	AbstractSource string `json:"AbstractSource"`
	AbstractURL    string `json:"AbstractURL"`
	Image          string `json:"Image"`
	Heading        string `json:"Heading"`
	Answer         string `json:"Answer"`
	AnswerType     string `json:"AnswerType"`
	Definition     string `json:"Definition"`
	DefinitionURL  string `json:"DefinitionURL"`
	RelatedTopics  []struct {
		Text string `json:"Text"`
		URL  string `json:"FirstURL"`
	} `json:"RelatedTopics"`
	Results []struct {
		Text string `json:"Text"`
		URL  string `json:"FirstURL"`
	} `json:"Results"`
}

// WebSearchTool 网络搜索工具（使用 DuckDuckGo）
type WebSearchTool struct {
	MaxResults int
}

// Name 返回工具名称
func (t *WebSearchTool) Name() string { return "web_search" }

// Description 返回工具描述
func (t *WebSearchTool) Description() string { return "使用 DuckDuckGo 搜索网络信息" }

// Parameters 返回工具参数定义
func (t *WebSearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string", "description": "搜索查询关键词"},
		},
		"required": []string{"query"},
	}
}

// Execute 执行 DuckDuckGo 搜索
func (t *WebSearchTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return "错误: 搜索查询不能为空", nil
	}

	maxResults := t.MaxResults
	if maxResults <= 0 {
		maxResults = 5
	}

	// 构建 DuckDuckGo Instant Answer API 请求
	apiURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1&skip_disambig=1", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return fmt.Sprintf("创建请求失败: %v", err), nil
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Nanobot/1.0)")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("请求失败: %v", err), nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("读取响应失败: %v", err), nil
	}

	var ddgResp DuckDuckGoResponse
	if err := json.Unmarshal(body, &ddgResp); err != nil {
		return fmt.Sprintf("解析响应失败: %v", err), nil
	}

	// 构建搜索结果
	var results []string

	// 添加标题
	if ddgResp.Heading != "" {
		results = append(results, fmt.Sprintf("【主题】%s", ddgResp.Heading))
	}

	// 添加摘要
	if ddgResp.AbstractText != "" {
		results = append(results, fmt.Sprintf("【摘要】%s", ddgResp.AbstractText))
		if ddgResp.AbstractURL != "" {
			results = append(results, fmt.Sprintf("【来源】%s", ddgResp.AbstractURL))
		}
	}

	// 添加直接答案
	if ddgResp.Answer != "" && ddgResp.AnswerType != "" {
		results = append(results, fmt.Sprintf("【答案】%s", ddgResp.Answer))
	}

	// 添加定义
	if ddgResp.Definition != "" {
		results = append(results, fmt.Sprintf("【定义】%s", ddgResp.Definition))
		if ddgResp.DefinitionURL != "" {
			results = append(results, fmt.Sprintf("【定义来源】%s", ddgResp.DefinitionURL))
		}
	}

	// 添加相关主题
	if len(ddgResp.RelatedTopics) > 0 {
		count := 0
		for _, topic := range ddgResp.RelatedTopics {
			if count >= maxResults {
				break
			}
			if topic.Text != "" {
				result := topic.Text
				if topic.URL != "" {
					result = fmt.Sprintf("%s\n  链接: %s", topic.Text, topic.URL)
				}
				results = append(results, fmt.Sprintf("【相关】%s", result))
				count++
			}
		}
	}

	if len(results) == 0 {
		return fmt.Sprintf("未找到关于 \"%s\" 的相关信息", query), nil
	}

	return strings.Join(results, "\n\n"), nil
}

// ToSchema 返回工具的 JSON Schema
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
