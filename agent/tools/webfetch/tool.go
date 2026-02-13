package webfetch

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/agent/tools/common"
)

type Tool struct {
	MaxChars int
}

func (t *Tool) Name() string {
	return "web_fetch"
}

func (t *Tool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.Name(),
		Desc: "获取网页内容",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"url": {
				Type:     schema.DataType("string"),
				Desc:     "URL",
				Required: true,
			},
		}),
	}, nil
}

func (t *Tool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		URL string `json:"url"`
	}
	if err := common.DecodeArgs(argumentsInJSON, &args); err != nil {
		return "", err
	}
	return "网页获取功能需要实现 HTTP 请求", nil
}

func (t *Tool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.Run(ctx, argumentsInJSON, opts...)
}
