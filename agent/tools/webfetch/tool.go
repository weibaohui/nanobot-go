package webfetch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	readability "github.com/go-shiori/go-readability"
	"github.com/weibaohui/nanobot-go/agent/tools/common"
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

// validateURL 验证 URL
func validateURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("无效的 URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("只支持 http/https 协议，当前: %s", parsed.Scheme)
	}

	if parsed.Host == "" {
		return fmt.Errorf("缺少域名")
	}

	return nil
}

// isHTML 检查内容是否为 HTML
func isHTML(body []byte) bool {
	preview := strings.ToLower(string(body[:min(256, len(body))]))
	return strings.HasPrefix(preview, "<!doctype") || strings.HasPrefix(preview, "<html")
}

// stripTags 移除 HTML 标签
func stripTags(html string) string {
	// 移除 script 标签
	re := regexp.MustCompile(`(?i)<script[\s\S]*?</script>`)
	html = re.ReplaceAllString(html, "")

	// 移除 style 标签
	re = regexp.MustCompile(`(?i)<style[\s\S]*?</style>`)
	html = re.ReplaceAllString(html, "")

	// 移除所有 HTML 标签
	re = regexp.MustCompile(`<[^>]+>`)
	text := re.ReplaceAllString(html, "")

	// 解码 HTML 实体
	text = decodeHTMLEntities(text)

	// 规范化空白
	return normalize(text)
}

// decodeHTMLEntities 解码 HTML 实体
func decodeHTMLEntities(text string) string {
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&apos;", "'")
	return text
}

// normalize 规范化空白
func normalize(text string) string {
	// 替换多个空格/制表符为单个空格
	re := regexp.MustCompile(`[ \t]+`)
	text = re.ReplaceAllString(text, " ")

	// 替换多个换行为两个换行
	re = regexp.MustCompile(`\n{3,}`)
	text = re.ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}

// toMarkdown 将 HTML 转换为 Markdown
func toMarkdown(html string) string {
	// 转换链接: <a href="url">text</a> -> [text](url)
	re := regexp.MustCompile(`(?i)<a\s+[^>]*href=["']([^"']+)["'][^>]*>([\s\S]*?)</a>`)
	html = re.ReplaceAllStringFunc(html, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) >= 3 {
			return fmt.Sprintf("[%s](%s)", stripTags(submatches[2]), submatches[1])
		}
		return match
	})

	// 转换标题: <h1>text</h1> -> # text
	re = regexp.MustCompile(`(?i)<h([1-6])[^>]*>([\s\S]*?)</h[1-6]>`)
	html = re.ReplaceAllStringFunc(html, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) >= 3 {
			level := submatches[1]
			text := stripTags(submatches[2])
			return fmt.Sprintf("\n%s %s\n", strings.Repeat("#", parseInt(level)), text)
		}
		return match
	})

	// 转换列表项: <li>text</li> -> - text
	re = regexp.MustCompile(`(?i)<li[^>]*>([\s\S]*?)</li>`)
	html = re.ReplaceAllStringFunc(html, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) >= 2 {
			return fmt.Sprintf("\n- %s", stripTags(submatches[1]))
		}
		return match
	})

	// 转换段落/块元素
	re = regexp.MustCompile(`(?i)</(p|div|section|article)>`)
	html = re.ReplaceAllString(html, "\n\n")

	// 转换换行
	re = regexp.MustCompile(`(?i)<(br|hr)\s*/?>`)
	html = re.ReplaceAllString(html, "\n")

	// 转换粗体
	re = regexp.MustCompile(`(?i)<(?:strong|b)[^>]*>([\s\S]*?)</(?:strong|b)>`)
	html = re.ReplaceAllStringFunc(html, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) >= 2 {
			return fmt.Sprintf("**%s**", stripTags(submatches[1]))
		}
		return match
	})

	// 转换斜体
	re = regexp.MustCompile(`(?i)<(?:em|i)[^>]*>([\s\S]*?)</(?:em|i)>`)
	html = re.ReplaceAllStringFunc(html, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) >= 2 {
			return fmt.Sprintf("*%s*", stripTags(submatches[1]))
		}
		return match
	})

	// 转换代码块
	re = regexp.MustCompile(`(?i)<code[^>]*>([\s\S]*?)</code>`)
	html = re.ReplaceAllStringFunc(html, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) >= 2 {
			return fmt.Sprintf("`%s`", submatches[1])
		}
		return match
	})

	// 转换预格式化块
	re = regexp.MustCompile(`(?i)<pre[^>]*>([\s\S]*?)</pre>`)
	html = re.ReplaceAllStringFunc(html, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) >= 2 {
			return fmt.Sprintf("\n```\n%s\n```\n", stripTags(submatches[1]))
		}
		return match
	})

	// 移除剩余标签并规范化
	return normalize(stripTags(html))
}

// parseInt 解析整数
func parseInt(s string) int {
	var result int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		}
	}
	return result
}
