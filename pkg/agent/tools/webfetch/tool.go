package webfetch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	readability "github.com/go-shiori/go-readability"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/common"
)

const (
	userAgent    = "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_7_2) AppleWebKit/537.36"
	maxRedirects = 5
)

// Tool 网页获取工具
type Tool struct {
	MaxChars int
}

// Name 返回工具名称
func (t *Tool) Name() string {
	return "web_fetch"
}

// ToSchema 返回工具 schema
func (t *Tool) ToSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        t.Name(),
			"description": "获取网页内容并转换为 Markdown 格式",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "要获取的 URL",
					},
					"extractMode": map[string]any{
						"type":        "string",
						"enum":        []string{"markdown", "text"},
						"default":     "markdown",
						"description": "提取模式",
					},
				},
				"required": []string{"url"},
			},
		},
	}
}

// Info 返回工具信息
func (t *Tool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.Name(),
		Desc: "获取网页内容并转换为 Markdown 格式",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"url": {
				Type:     schema.DataType("string"),
				Desc:     "要获取的 URL",
				Required: true,
			},
			"extractMode": {
				Type:     schema.DataType("string"),
				Desc:     "提取模式: markdown 或 text",
				Required: false,
			},
		}),
	}, nil
}

// Run 执行工具逻辑
func (t *Tool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		URL         string `json:"url"`
		ExtractMode string `json:"extractMode"`
	}
	if err := common.DecodeArgs(argumentsInJSON, &args); err != nil {
		return "", err
	}

	if args.ExtractMode == "" {
		args.ExtractMode = "markdown"
	}

	maxChars := t.MaxChars
	if maxChars <= 0 {
		maxChars = 50000
	}

	result, err := t.fetchURL(ctx, args.URL, args.ExtractMode, maxChars)
	if err != nil {
		return "", err
	}

	return result, nil
}

// InvokableRun 可直接调用的执行入口
func (t *Tool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.Run(ctx, argumentsInJSON, opts...)
}

// fetchResult 获取结果
type fetchResult struct {
	URL       string `json:"url"`
	FinalURL  string `json:"finalUrl,omitempty"`
	Status    int    `json:"status"`
	Extractor string `json:"extractor"`
	Truncated bool   `json:"truncated"`
	Length    int    `json:"length"`
	Text      string `json:"text"`
	Error     string `json:"error,omitempty"`
}

// fetchURL 获取 URL 内容
func (t *Tool) fetchURL(ctx context.Context, rawURL, extractMode string, maxChars int) (string, error) {
	// 验证 URL
	if err := validateURL(rawURL); err != nil {
		result := fetchResult{
			URL:   rawURL,
			Error: fmt.Sprintf("URL 验证失败: %s", err.Error()),
		}
		data, _ := json.Marshal(result)
		return string(data), nil
	}

	// 创建 HTTP 客户端
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("stopped after %d redirects", maxRedirects)
			}
			return nil
		},
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		result := fetchResult{
			URL:   rawURL,
			Error: fmt.Sprintf("请求失败: %s", err.Error()),
		}
		data, _ := json.Marshal(result)
		return string(data), nil
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result := fetchResult{
			URL:   rawURL,
			Error: fmt.Sprintf("读取响应失败: %s", err.Error()),
		}
		data, _ := json.Marshal(result)
		return string(data), nil
	}

	contentType := resp.Header.Get("Content-Type")
	var text string
	var extractor string

	// 根据内容类型处理
	if strings.Contains(contentType, "application/json") {
		// JSON 响应
		var prettyJSON map[string]any
		if err := json.Unmarshal(body, &prettyJSON); err == nil {
			formatted, _ := json.MarshalIndent(prettyJSON, "", "  ")
			text = string(formatted)
		} else {
			text = string(body)
		}
		extractor = "json"
	} else if strings.Contains(contentType, "text/html") || isHTML(body) {
		// HTML 响应 - 使用 readability 提取
		parsedURL, _ := url.Parse(rawURL)
		article, err := readability.FromReader(strings.NewReader(string(body)), parsedURL)
		if err != nil {
			// readability 失败，直接提取文本
			text = stripTags(string(body))
			extractor = "raw"
		} else {
			// 成功提取
			if extractMode == "markdown" {
				content := toMarkdown(article.Content)
				if article.Title != "" {
					text = fmt.Sprintf("# %s\n\n%s", article.Title, content)
				} else {
					text = content
				}
			} else {
				content := stripTags(article.Content)
				if article.Title != "" {
					text = fmt.Sprintf("# %s\n\n%s", article.Title, content)
				} else {
					text = content
				}
			}
			extractor = "readability"
		}
	} else {
		// 其他类型，直接返回文本
		text = string(body)
		extractor = "raw"
	}

	// 截断内容
	truncated := len(text) > maxChars
	if truncated {
		text = text[:maxChars]
	}

	// 构建结果
	result := fetchResult{
		URL:       rawURL,
		FinalURL:  resp.Request.URL.String(),
		Status:    resp.StatusCode,
		Extractor: extractor,
		Truncated: truncated,
		Length:    len(text),
		Text:      text,
	}

	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}

	return string(data), nil
}
