package editfile

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/agent/tools/common"
)

type Tool struct {
	AllowedDir string
}

func (t *Tool) Name() string {
	return "edit_file"
}

func (t *Tool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.Name(),
		Desc: "通过替换文本编辑文件",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.DataType("string"),
				Desc:     "文件路径",
				Required: true,
			},
			"old_text": {
				Type:     schema.DataType("string"),
				Desc:     "要替换的文本",
				Required: true,
			},
			"new_text": {
				Type:     schema.DataType("string"),
				Desc:     "替换成的文本",
				Required: true,
			},
		}),
	}, nil
}

func (t *Tool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		Path    string `json:"path"`
		OldText string `json:"old_text"`
		NewText string `json:"new_text"`
	}
	if err := common.DecodeArgs(argumentsInJSON, &args); err != nil {
		return "", err
	}
	resolved := common.ResolvePath(args.Path, t.AllowedDir)
	data, err := os.ReadFile(resolved)
	if err != nil {
		return fmt.Sprintf("错误: 文件不存在: %s", args.Path), nil
	}
	content := string(data)
	if !strings.Contains(content, args.OldText) {
		return "错误: old_text 在文件中未找到", nil
	}
	newContent := strings.Replace(content, args.OldText, args.NewText, 1)
	if err := os.WriteFile(resolved, []byte(newContent), 0644); err != nil {
		return "", err
	}
	return fmt.Sprintf("成功编辑 %s", args.Path), nil
}

func (t *Tool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.Run(ctx, argumentsInJSON, opts...)
}
