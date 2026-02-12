package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/weibaohui/nanobot-go/bus"
	"github.com/weibaohui/nanobot-go/cron"
)

// ========== 文件工具 ==========

// ReadFileTool 读取文件工具
type ReadFileTool struct {
	AllowedDir string
}

func (t *ReadFileTool) Name() string        { return "read_file" }
func (t *ReadFileTool) Description() string { return "读取指定路径的文件内容" }
func (t *ReadFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string", "description": "要读取的文件路径"},
		},
		"required": []string{"path"},
	}
}
func (t *ReadFileTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	path, _ := params["path"].(string)
	resolved := resolvePath(path, t.AllowedDir)
	data, err := os.ReadFile(resolved)
	if err != nil {
		return fmt.Sprintf("错误: 读取文件失败: %s", err), nil
	}
	return string(data), nil
}
func (t *ReadFileTool) ToSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        t.Name(),
			"description": t.Description(),
			"parameters":  t.Parameters(),
		},
	}
}

// WriteFileTool 写入文件工具
type WriteFileTool struct {
	AllowedDir string
}

func (t *WriteFileTool) Name() string { return "write_file" }
func (t *WriteFileTool) Description() string {
	return "将内容写入文件，必要时创建父目录"
}
func (t *WriteFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":    map[string]any{"type": "string", "description": "文件路径"},
			"content": map[string]any{"type": "string", "description": "要写入的内容"},
		},
		"required": []string{"path", "content"},
	}
}
func (t *WriteFileTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	path, _ := params["path"].(string)
	content, _ := params["content"].(string)
	resolved := resolvePath(path, t.AllowedDir)
	os.MkdirAll(filepath.Dir(resolved), 0755)
	if err := os.WriteFile(resolved, []byte(content), 0644); err != nil {
		return "", err
	}
	return fmt.Sprintf("成功写入 %d 字节到 %s", len(content), path), nil
}
func (t *WriteFileTool) ToSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        t.Name(),
			"description": t.Description(),
			"parameters":  t.Parameters(),
		},
	}
}

// EditFileTool 编辑文件工具
type EditFileTool struct {
	AllowedDir string
}

func (t *EditFileTool) Name() string        { return "edit_file" }
func (t *EditFileTool) Description() string { return "通过替换文本编辑文件" }
func (t *EditFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":     map[string]any{"type": "string", "description": "文件路径"},
			"old_text": map[string]any{"type": "string", "description": "要替换的文本"},
			"new_text": map[string]any{"type": "string", "description": "替换成的文本"},
		},
		"required": []string{"path", "old_text", "new_text"},
	}
}
func (t *EditFileTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	path, _ := params["path"].(string)
	oldText, _ := params["old_text"].(string)
	newText, _ := params["new_text"].(string)
	resolved := resolvePath(path, t.AllowedDir)
	data, err := os.ReadFile(resolved)
	if err != nil {
		return fmt.Sprintf("错误: 文件不存在: %s", path), nil
	}
	content := string(data)
	if !strings.Contains(content, oldText) {
		return "错误: old_text 在文件中未找到", nil
	}
	newContent := strings.Replace(content, oldText, newText, 1)
	os.WriteFile(resolved, []byte(newContent), 0644)
	return fmt.Sprintf("成功编辑 %s", path), nil
}
func (t *EditFileTool) ToSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        t.Name(),
			"description": t.Description(),
			"parameters":  t.Parameters(),
		},
	}
}

// ListDirTool 列出目录工具
type ListDirTool struct {
	AllowedDir string
}

func (t *ListDirTool) Name() string        { return "list_dir" }
func (t *ListDirTool) Description() string { return "列出目录内容" }
func (t *ListDirTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string", "description": "目录路径"},
		},
		"required": []string{"path"},
	}
}
func (t *ListDirTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	path, _ := params["path"].(string)
	resolved := resolvePath(path, t.AllowedDir)
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
func (t *ListDirTool) ToSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        t.Name(),
			"description": t.Description(),
			"parameters":  t.Parameters(),
		},
	}
}

// ========== Shell 工具 ==========

// ExecTool 执行命令工具
type ExecTool struct {
	Timeout             int
	WorkingDir          string
	RestrictToWorkspace bool
}

func (t *ExecTool) Name() string        { return "exec" }
func (t *ExecTool) Description() string { return "执行 shell 命令" }
func (t *ExecTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{"type": "string", "description": "要执行的命令"},
		},
		"required": []string{"command"},
	}
}
func (t *ExecTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	command, _ := params["command"].(string)
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
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
func (t *ExecTool) ToSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        t.Name(),
			"description": t.Description(),
			"parameters":  t.Parameters(),
		},
	}
}

// ========== Web 工具 ==========

// WebSearchTool 网络搜索工具
type WebSearchTool struct {
	APIKey     string
	MaxResults int
}

func (t *WebSearchTool) Name() string        { return "web_search" }
func (t *WebSearchTool) Description() string { return "搜索网络" }
func (t *WebSearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string", "description": "搜索查询"},
		},
		"required": []string{"query"},
	}
}
func (t *WebSearchTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	if t.APIKey == "" {
		return "错误: BRAVE_API_KEY 未配置", nil
	}
	// 简化实现，实际应调用 Brave Search API
	return "网络搜索功能需要实现 Brave API 调用", nil
}
func (t *WebSearchTool) ToSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        t.Name(),
			"description": t.Description(),
			"parameters":  t.Parameters(),
		},
	}
}

// WebFetchTool 网页获取工具
type WebFetchTool struct {
	MaxChars int
}

func (t *WebFetchTool) Name() string        { return "web_fetch" }
func (t *WebFetchTool) Description() string { return "获取网页内容" }
func (t *WebFetchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{"type": "string", "description": "URL"},
		},
		"required": []string{"url"},
	}
}
func (t *WebFetchTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	// 简化实现
	return "网页获取功能需要实现 HTTP 请求", nil
}
func (t *WebFetchTool) ToSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        t.Name(),
			"description": t.Description(),
			"parameters":  t.Parameters(),
		},
	}
}

// ========== 消息工具 ==========

// MessageTool 消息工具
type MessageTool struct {
	SendCallback   func(msg *bus.OutboundMessage) error
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
func (t *MessageTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	content, _ := params["content"].(string)
	channel, _ := params["channel"].(string)
	if channel == "" {
		channel = t.DefaultChannel
	}
	chatID, _ := params["chat_id"].(string)
	if chatID == "" {
		chatID = t.DefaultChatID
	}
	if channel == "" || chatID == "" {
		return "错误: 未指定目标渠道/聊天", nil
	}
	if t.SendCallback == nil {
		return "错误: 消息发送未配置", nil
	}
	msg := bus.NewOutboundMessage(channel, chatID, content)
	if err := t.SendCallback(msg); err != nil {
		return fmt.Sprintf("发送失败: %s", err), nil
	}
	return fmt.Sprintf("消息已发送到 %s:%s", channel, chatID), nil
}
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

// ========== Spawn 工具 ==========

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

// ========== Cron 工具 ==========

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

// ========== 辅助函数 ==========

func resolvePath(path, allowedDir string) string {
	if strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[1:])
	}
	absPath, _ := filepath.Abs(path)
	if allowedDir != "" {
		allowedAbs, _ := filepath.Abs(allowedDir)
		if !strings.HasPrefix(absPath, allowedAbs) {
			return path // 允许检查在调用方处理
		}
	}
	return absPath
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
