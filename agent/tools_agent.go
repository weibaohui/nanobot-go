package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/weibaohui/nanobot-go/bus"
	"github.com/weibaohui/nanobot-go/cron"
)

// SpawnTool 子代理工具
type SpawnTool struct {
	Manager       *SubagentManager
	OriginChannel string
	OriginChatID  string
}

func (t *SpawnTool) Name() string        { return "spawn" }
func (t *SpawnTool) Description() string { return "创建子代理执行后台任务" }
func (t *SpawnTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task":  map[string]any{"type": "string", "description": "任务描述"},
			"label": map[string]any{"type": "string", "description": "任务标签"},
		},
		"required": []string{"task"},
	}
}
func (t *SpawnTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	task, _ := params["task"].(string)
	label, _ := params["label"].(string)
	if t.Manager == nil {
		return "错误: 子代理管理器未配置", nil
	}
	return t.Manager.Spawn(ctx, task, label, t.OriginChannel, t.OriginChatID)
}

// SetContext 设置上下文
func (t *SpawnTool) SetContext(channel, chatID string) {
	t.OriginChannel = channel
	t.OriginChatID = chatID
}

func (t *SpawnTool) ToSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        t.Name(),
			"description": t.Description(),
			"parameters":  t.Parameters(),
		},
	}
}

// CronTool 定时任务工具
type CronTool struct {
	CronService *cron.Service
	Channel     string
	ChatID      string
}

func (t *CronTool) Name() string        { return "cron" }
func (t *CronTool) Description() string { return "调度提醒和周期性任务" }
func (t *CronTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action":        map[string]any{"type": "string", "description": "操作: add, list, remove"},
			"message":       map[string]any{"type": "string", "description": "提醒消息"},
			"every_seconds": map[string]any{"type": "integer", "description": "间隔秒数"},
			"cron_expr":     map[string]any{"type": "string", "description": "Cron表达式"},
			"job_id":        map[string]any{"type": "string", "description": "任务ID"},
		},
		"required": []string{"action"},
	}
}
func (t *CronTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	action, _ := params["action"].(string)
	switch action {
	case "add":
		return t.addJob(params)
	case "list":
		return t.listJobs()
	case "remove":
		return t.removeJob(params)
	}
	return fmt.Sprintf("未知操作: %s", action), nil
}

func (t *CronTool) addJob(params map[string]any) (string, error) {
	message, _ := params["message"].(string)
	if message == "" {
		return "错误: 需要消息参数", nil
	}
	if t.Channel == "" || t.ChatID == "" {
		return "错误: 没有会话上下文", nil
	}
	var schedule *cron.Schedule
	if everySeconds, ok := params["every_seconds"].(float64); ok {
		schedule = &cron.Schedule{Kind: "every", EveryMs: int(everySeconds * 1000)}
	} else if cronExpr, ok := params["cron_expr"].(string); ok {
		schedule = &cron.Schedule{Kind: "cron", Expr: cronExpr}
	} else {
		return "错误: 需要 every_seconds 或 cron_expr 参数", nil
	}
	job := t.CronService.AddJob(truncateString(message, 30), schedule, message, true, t.Channel, t.ChatID, false)
	return fmt.Sprintf("已创建任务 '%s' (id: %s)", job.Name, job.ID), nil
}

func (t *CronTool) listJobs() (string, error) {
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

func (t *CronTool) removeJob(params map[string]any) (string, error) {
	jobID, _ := params["job_id"].(string)
	if jobID == "" {
		return "错误: 需要 job_id 参数", nil
	}
	if t.CronService.RemoveJob(jobID) {
		return fmt.Sprintf("已删除任务 %s", jobID), nil
	}
	return fmt.Sprintf("任务 %s 未找到", jobID), nil
}

// SetContext 设置上下文
func (t *CronTool) SetContext(channel, chatID string) {
	t.Channel = channel
	t.ChatID = chatID
}

func (t *CronTool) ToSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        t.Name(),
			"description": t.Description(),
			"parameters":  t.Parameters(),
		},
	}
}

// MessageTool 消息工具
type MessageTool struct {
	SendCallback   func(msg any) error
	DefaultChannel string
	DefaultChatID  string
}

func (t *MessageTool) Name() string        { return "message" }
func (t *MessageTool) Description() string { return "发送消息给用户" }
func (t *MessageTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string", "description": "消息内容"},
			"channel": map[string]any{"type": "string", "description": "目标渠道"},
			"chat_id": map[string]any{"type": "string", "description": "目标聊天ID"},
		},
		"required": []string{"content"},
	}
}

// Execute 执行发送消息逻辑（含渠道与聊天上下文回退）
func (t *MessageTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	content, _ := params["content"].(string)
	channel, _ := params["channel"].(string)
	chatID, _ := params["chat_id"].(string)

	if channel == "" {
		channel = t.DefaultChannel
	}
	if channel == "user" {
		// 模型可能用 user 作为占位渠道，回退到当前会话的默认渠道
		channel = t.DefaultChannel
	}
	if chatID == "" {
		chatID = t.DefaultChatID
	}
	if channel == "" || chatID == "" {
		return "错误: 没有目标渠道或聊天ID", nil
	}
	if t.SendCallback == nil {
		return "错误: 消息发送未配置", nil
	}

	msg := bus.NewOutboundMessage(channel, chatID, content)
	if err := t.SendCallback(msg); err != nil {
		return fmt.Sprintf("错误: 发送消息失败: %s", err), nil
	}
	return fmt.Sprintf("消息已发送到 %s:%s", channel, chatID), nil
}

// SetContext 设置上下文
func (t *MessageTool) SetContext(channel, chatID string) {
	t.DefaultChannel = channel
	t.DefaultChatID = chatID
}

func (t *MessageTool) ToSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        t.Name(),
			"description": t.Description(),
			"parameters":  t.Parameters(),
		},
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
