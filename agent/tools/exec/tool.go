package exec

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/agent/tools/common"
)

// Tool 执行命令工具
type Tool struct {
	Timeout             int
	WorkingDir          string
	RestrictToWorkspace bool
}

// Name 返回工具名称
func (t *Tool) Name() string {
	return "exec"
}

// Info 返回工具信息
func (t *Tool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.Name(),
		Desc: "执行 shell 命令",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"command": {
				Type:     schema.DataType("string"),
				Desc:     "要执行的命令",
				Required: true,
			},
		}),
	}, nil
}

// Run 执行工具逻辑
func (t *Tool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		Command string `json:"command"`
	}
	if err := common.DecodeArgs(argumentsInJSON, &args); err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, "sh", "-c", args.Command)
	cmd.Dir = t.WorkingDir
	output, err := cmd.CombinedOutput()
	result := string(output)
	if err != nil {
		result += fmt.Sprintf("\n错误: %s", err)
	}
	if len(result) > 10000 {
		result = result[:10000] + "...(已截断)"
	}
	return result, nil
}

// InvokableRun 可直接调用的执行入口
func (t *Tool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.Run(ctx, argumentsInJSON, opts...)
}
