package websearch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/agent/tools/common"
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

// Tool 网络搜索工具
type Tool struct {
	MaxResults int
}

// Name 返回工具名称
func (t *Tool) Name() string {
	return "web_search"
}

// Info 返回工具信息
func (t *Tool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.Name(),
		Desc: "使用 DuckDuckGo 搜索网络信息",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type:     schema.DataType("string"),
				Desc:     "搜索查询关键词",
				Required: true,
			},
		}),
	}, nil
}

// Run 执行工具逻辑
func (t *Tool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		Query string `json:"query"`
	}
	if err := common.DecodeArgs(argumentsInJSON, &args); err != nil {
		return "", err
	}
	if args.Query == "" {
		return "错误: 搜索查询不能为空", nil
	}
	maxResults := t.MaxResults
	if maxResults <= 0 {
		maxResults = 5
	}
	apiURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1&skip_disambig=1", url.QueryEscape(args.Query))
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
	var results []string
	if ddgResp.Heading != "" {
		results = append(results, fmt.Sprintf("【主题】%s", ddgResp.Heading))
	}
	if ddgResp.AbstractText != "" {
		results = append(results, fmt.Sprintf("【摘要】%s", ddgResp.AbstractText))
		if ddgResp.AbstractURL != "" {
			results = append(results, fmt.Sprintf("【来源】%s", ddgResp.AbstractURL))
		}
	}
	if ddgResp.Answer != "" && ddgResp.AnswerType != "" {
		results = append(results, fmt.Sprintf("【答案】%s", ddgResp.Answer))
	}
	if ddgResp.Definition != "" {
		results = append(results, fmt.Sprintf("【定义】%s", ddgResp.Definition))
		if ddgResp.DefinitionURL != "" {
			results = append(results, fmt.Sprintf("【定义来源】%s", ddgResp.DefinitionURL))
		}
	}
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
		return fmt.Sprintf("未找到关于 \"%s\" 的相关信息", args.Query), nil
	}
	return strings.Join(results, "\n\n"), nil
}

// InvokableRun 可直接调用的执行入口
func (t *Tool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.Run(ctx, argumentsInJSON, opts...)
}
