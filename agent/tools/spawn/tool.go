package spawn

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/agent/tools/common"
)

type Manager interface {
	Spawn(ctx context.Context, task, label, originChannel, originChatID string) (string, error)
}

type Tool struct {
	Manager       Manager
	OriginChannel string
	OriginChatID  string
}

// Name 返回工具名称
func (t *Tool) Name() string {
	return "spawn"
}

// Info 返回工具信息
func (t *Tool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.Name(),
		Desc: "创建子代理执行后台任务",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"task": {
				Type:     schema.DataType("string"),
				Desc:     "任务描述",
				Required: true,
			},
			"label": {
				Type: schema.DataType("string"),
				Desc: "任务标签",
			},
		}),
	}, nil
}

// Run 执行工具逻辑
func (t *Tool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		Task  string `json:"task"`
		Label string `json:"label"`
	}
	if err := common.DecodeArgs(argumentsInJSON, &args); err != nil {
		return "", err
	}
	if t.Manager == nil {
		return "错误: 子代理管理器未配置", nil
	}
	return t.Manager.Spawn(ctx, args.Task, args.Label, t.OriginChannel, t.OriginChatID)
}

// InvokableRun 可直接调用的执行入口
func (t *Tool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.Run(ctx, argumentsInJSON, opts...)
}

// SetContext 设置上下文
func (t *Tool) SetContext(channel, chatID string) {
	t.OriginChannel = channel
	t.OriginChatID = chatID
}
