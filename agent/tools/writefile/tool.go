package writefile

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/agent/tools/common"
)

type Tool struct {
	AllowedDir string
}

func (t *Tool) Name() string {
	return "write_file"
}

func (t *Tool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.Name(),
		Desc: "将内容写入文件，必要时创建父目录",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.DataType("string"),
				Desc:     "文件路径",
				Required: true,
			},
			"content": {
				Type:     schema.DataType("string"),
				Desc:     "要写入的内容",
				Required: true,
			},
		}),
	}, nil
}

func (t *Tool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := common.DecodeArgs(argumentsInJSON, &args); err != nil {
		return "", err
	}
	resolved := common.ResolvePath(args.Path, t.AllowedDir)
	if err := os.MkdirAll(filepath.Dir(resolved), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(resolved, []byte(args.Content), 0644); err != nil {
		return "", err
	}
	return fmt.Sprintf("成功写入 %d 字节到 %s", len(args.Content), args.Path), nil
}

func (t *Tool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.Run(ctx, argumentsInJSON, opts...)
}
