package listdir

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/agent/tools/common"
)

// Tool 列出目录工具
type Tool struct {
	AllowedDir string
}

// Name 返回工具名称
func (t *Tool) Name() string {
	return "list_dir"
}

// Info 返回工具信息
func (t *Tool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.Name(),
		Desc: "列出目录内容",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.DataType("string"),
				Desc:     "目录路径",
				Required: true,
			},
		}),
	}, nil
}

// Run 执行工具逻辑
func (t *Tool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		Path string `json:"path"`
	}
	if err := common.DecodeArgs(argumentsInJSON, &args); err != nil {
		return "", err
	}
	resolved := common.ResolvePath(args.Path, t.AllowedDir)
	entries, err := os.ReadDir(resolved)
	if err != nil {
		return fmt.Sprintf("错误: 读取目录失败: %s", err), nil
	}
	var lines []string
	for _, e := range entries {
		prefix := "📄 "
		if e.IsDir() {
			prefix = "📁 "
		}
		lines = append(lines, prefix+e.Name())
	}
	if len(lines) == 0 {
		return "目录为空", nil
	}
	return strings.Join(lines, "\n"), nil
}

// InvokableRun 可直接调用的执行入口
func (t *Tool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.Run(ctx, argumentsInJSON, opts...)
}
