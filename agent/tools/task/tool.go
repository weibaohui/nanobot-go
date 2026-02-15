package task

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/agent/tools/common"
	"go.uber.org/zap"
)

// TaskInfo 任务查询结果
type TaskInfo struct {
	ID            string
	Status        string
	ResultSummary string
}

// Manager 任务管理器接口
type Manager interface {
	StartTask(ctx context.Context, work, channel, chatID string) (string, string, error)
	GetTask(ctx context.Context, taskID string) (*TaskInfo, error)
	StopTask(ctx context.Context, taskID string) (bool, string, error)
	ListTasks() ([]*TaskInfo, error)
}

// StartTool 后台任务创建工具
type StartTool struct {
	Manager Manager
	Channel string
	ChatID  string
	Logger  *zap.Logger
}

// Name 返回工具名称
func (t *StartTool) Name() string {
	return "start_task"
}

// Info 返回工具信息
func (t *StartTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.Name(),
		Desc: "创建后台任务并返回任务ID",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"work": {
				Type:     schema.DataType("string"),
				Desc:     "任务内容或目标",
				Required: true,
			},
		}),
	}, nil
}

// Run 执行工具逻辑
func (t *StartTool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		Work string `json:"work"`
	}
	if err := common.DecodeArgs(argumentsInJSON, &args); err != nil {
		return "", err
	}
	if args.Work == "" {
		return "错误: 任务内容不能为空", nil
	}
	if t.Manager == nil {
		return "错误: 任务管理器未配置", nil
	}
	taskID, status, err := t.Manager.StartTask(ctx, args.Work, t.Channel, t.ChatID)
	if err != nil {
		return fmt.Sprintf("错误: 创建任务失败: %s", err), nil
	}
	if t.Logger != nil {
		t.Logger.Info("创建后台任务成功", zap.String("任务ID", taskID), zap.String("状态", status))
	}
	return fmt.Sprintf("任务已创建，ID: %s，状态: %s", taskID, status), nil
}

// InvokableRun 可直接调用的执行入口
func (t *StartTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.Run(ctx, argumentsInJSON, opts...)
}

// SetContext 设置上下文
func (t *StartTool) SetContext(channel, chatID string) {
	t.Channel = channel
	t.ChatID = chatID
}

// GetTool 后台任务查询工具
type GetTool struct {
	Manager Manager
	Logger  *zap.Logger
}

// Name 返回工具名称
func (t *GetTool) Name() string {
	return "get_task"
}

// Info 返回工具信息
func (t *GetTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.Name(),
		Desc: "查询后台任务状态与最近日志",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"task_id": {
				Type:     schema.DataType("string"),
				Desc:     "任务ID",
				Required: true,
			},
		}),
	}, nil
}

// Run 执行工具逻辑
func (t *GetTool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		TaskID string `json:"task_id"`
	}
	if err := common.DecodeArgs(argumentsInJSON, &args); err != nil {
		return "", err
	}
	if args.TaskID == "" {
		return "错误: 任务ID不能为空", nil
	}
	if t.Manager == nil {
		return "错误: 任务管理器未配置", nil
	}
	info, err := t.Manager.GetTask(ctx, args.TaskID)
	if err != nil {
		return fmt.Sprintf("错误: 查询任务失败: %s", err), nil
	}
	if t.Logger != nil {
		t.Logger.Info("查询后台任务成功", zap.String("任务ID", info.ID), zap.String("状态", info.Status))
	}
	return fmt.Sprintf("任务ID: %s\n状态: %s\n结果摘要: %s", info.ID, info.Status, info.ResultSummary), nil
}

// InvokableRun 可直接调用的执行入口
func (t *GetTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.Run(ctx, argumentsInJSON, opts...)
}

// StopTool 后台任务停止工具
type StopTool struct {
	Manager Manager
	Logger  *zap.Logger
}

// Name 返回工具名称
func (t *StopTool) Name() string {
	return "stop_task"
}

// Info 返回工具信息
func (t *StopTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.Name(),
		Desc: "停止后台任务并返回结果",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"task_id": {
				Type:     schema.DataType("string"),
				Desc:     "任务ID",
				Required: true,
			},
		}),
	}, nil
}

// Run 执行工具逻辑
func (t *StopTool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		TaskID string `json:"task_id"`
	}
	if err := common.DecodeArgs(argumentsInJSON, &args); err != nil {
		return "", err
	}
	if args.TaskID == "" {
		return "错误: 任务ID不能为空", nil
	}
	if t.Manager == nil {
		return "错误: 任务管理器未配置", nil
	}
	stopped, status, err := t.Manager.StopTask(ctx, args.TaskID)
	if err != nil {
		return fmt.Sprintf("错误: 停止任务失败: %s", err), nil
	}
	if t.Logger != nil {
		t.Logger.Info("停止后台任务完成", zap.String("任务ID", args.TaskID), zap.Bool("是否停止", stopped), zap.String("状态", status))
	}
	return fmt.Sprintf("任务ID: %s\n是否停止: %v\n状态: %s", args.TaskID, stopped, status), nil
}

// InvokableRun 可直接调用的执行入口
func (t *StopTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.Run(ctx, argumentsInJSON, opts...)
}

// ListTool 后台任务列表工具
type ListTool struct {
	Manager Manager
	Logger  *zap.Logger
}

// Name 返回工具名称
func (t *ListTool) Name() string {
	return "list_task"
}

// Info 返回工具信息
func (t *ListTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        t.Name(),
		Desc:        "列出所有后台任务",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

// Run 执行工具逻辑
func (t *ListTool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	if t.Manager == nil {
		return "错误: 任务管理器未配置", nil
	}
	items, err := t.Manager.ListTasks()
	if err != nil {
		return fmt.Sprintf("错误: 获取任务列表失败: %s", err), nil
	}
	if t.Logger != nil {
		t.Logger.Info("获取任务列表完成", zap.Int("数量", len(items)))
	}
	if len(items) == 0 {
		return "任务列表为空", nil
	}
	var result string
	for i, item := range items {
		line := fmt.Sprintf("任务ID: %s | 状态: %s | 摘要: %s", item.ID, item.Status, item.ResultSummary)
		if i == 0 {
			result = line
		} else {
			result += "\n" + line
		}
	}
	return result, nil
}

// InvokableRun 可直接调用的执行入口
func (t *ListTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.Run(ctx, argumentsInJSON, opts...)
}
