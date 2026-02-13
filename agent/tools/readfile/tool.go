package readfile

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/agent/tools/common"
)

type Tool struct {
	AllowedDir string
}

func (t *Tool) Name() string {
	return "read_file"
}

func (t *Tool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.Name(),
		Desc: "读取指定路径的文件内容",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.DataType("string"),
				Desc:     "要读取的文件路径",
				Required: true,
			},
		}),
	}, nil
}

func (t *Tool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		Path string `json:"path"`
	}
	if err := common.DecodeArgs(argumentsInJSON, &args); err != nil {
		return "", err
	}
	resolved := common.ResolvePath(args.Path, t.AllowedDir)
	data, err := os.ReadFile(resolved)
	if err != nil {
		return fmt.Sprintf("错误: 读取文件失败: %s", err), nil
	}
	return string(data), nil
}

func (t *Tool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.Run(ctx, argumentsInJSON, opts...)
}
