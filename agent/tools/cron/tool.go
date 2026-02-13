package cron

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/agent/tools/common"
	"github.com/weibaohui/nanobot-go/cron"
)

// Tool 定时任务工具
type Tool struct {
	CronService *cron.Service
	Channel     string
	ChatID      string
}

// Args 定时任务参数
type Args struct {
	Action       string  `json:"action"`
	Message      string  `json:"message"`
	EverySeconds float64 `json:"every_seconds"`
	CronExpr     string  `json:"cron_expr"`
	JobID        string  `json:"job_id"`
}

// Name 返回工具名称
func (t *Tool) Name() string {
	return "cron"
}

// Info 返回工具信息
func (t *Tool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.Name(),
		Desc: "调度提醒和周期性任务",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"action": {
				Type:     schema.DataType("string"),
				Desc:     "操作: add, list, remove",
				Required: true,
			},
			"message": {
				Type: schema.DataType("string"),
				Desc: "提醒消息",
			},
			"every_seconds": {
				Type: schema.DataType("integer"),
				Desc: "间隔秒数",
			},
			"cron_expr": {
				Type: schema.DataType("string"),
				Desc: "Cron表达式",
			},
			"job_id": {
				Type: schema.DataType("string"),
				Desc: "任务ID",
			},
		}),
	}, nil
}

// Run 执行工具逻辑
func (t *Tool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args Args
	if err := common.DecodeArgs(argumentsInJSON, &args); err != nil {
		return "", err
	}
	switch args.Action {
	case "add":
		return t.addJob(args)
	case "list":
		return t.listJobs()
	case "remove":
		return t.removeJob(args)
	}
	return fmt.Sprintf("未知操作: %s", args.Action), nil
}

// InvokableRun 可直接调用的执行入口
func (t *Tool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.Run(ctx, argumentsInJSON, opts...)
}

// SetContext 设置上下文
func (t *Tool) SetContext(channel, chatID string) {
	t.Channel = channel
	t.ChatID = chatID
}

// addJob 添加任务
func (t *Tool) addJob(args Args) (string, error) {
	if args.Message == "" {
		return "错误: 需要消息参数", nil
	}
	if t.Channel == "" || t.ChatID == "" {
		return "错误: 没有会话上下文", nil
	}
	var schedule *cron.Schedule
	if args.EverySeconds > 0 {
		schedule = &cron.Schedule{Kind: "every", EveryMs: int(args.EverySeconds * 1000)}
	} else if args.CronExpr != "" {
		schedule = &cron.Schedule{Kind: "cron", Expr: args.CronExpr}
	} else {
		return "错误: 需要 every_seconds 或 cron_expr 参数", nil
	}
	job := t.CronService.AddJob(common.TruncateString(args.Message, 30), schedule, args.Message, true, t.Channel, t.ChatID, false)
	return fmt.Sprintf("已创建任务 '%s' (id: %s)", job.Name, job.ID), nil
}

// listJobs 列出任务
func (t *Tool) listJobs() (string, error) {
	jobs := t.CronService.ListJobs()
	if len(jobs) == 0 {
		return "没有计划任务", nil
	}
	var lines []string
	for _, j := range jobs {
		lines = append(lines, fmt.Sprintf("- %s (id: %s, %s)", j.Name, j.ID, j.Schedule.Kind))
	}
	return "计划任务:\n" + strings.Join(lines, "\n"), nil
}

// removeJob 删除任务
func (t *Tool) removeJob(args Args) (string, error) {
	if args.JobID == "" {
		return "错误: 需要 job_id 参数", nil
	}
	if t.CronService.RemoveJob(args.JobID) {
		return fmt.Sprintf("已删除任务 %s", args.JobID), nil
	}
	return fmt.Sprintf("任务 %s 未找到", args.JobID), nil
}
