package config

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/internal/models"
	agentsvc "github.com/weibaohui/nanobot-go/internal/service/agent"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/common"
)

// ManageAgentMemoryTool 管理 Agent 记忆工具
type ManageAgentMemoryTool struct {
	agentService agentsvc.Service
}

// NewManageAgentMemoryTool 创建记忆管理工具实例
func NewManageAgentMemoryTool(agentService agentsvc.Service) *ManageAgentMemoryTool {
	return &ManageAgentMemoryTool{
		agentService: agentService,
	}
}

// Name 返回工具名称
func (t *ManageAgentMemoryTool) Name() string {
	return "manage_agent_memory"
}

// Info 返回工具信息
func (t *ManageAgentMemoryTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.Name(),
		Desc: "管理 Agent 的长期记忆（read/append/clear）",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"action": {
				Type:     schema.DataType("string"),
				Desc:     "操作类型，可选: read, append, clear",
				Required: true,
			},
			"content": {
				Type:     schema.DataType("string"),
				Desc:     "要追加的内容（action=append 时必填）",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 可直接调用的执行入口
func (t *ManageAgentMemoryTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// 1. 强制提取并验证上下文
	cfgCtx, err := GetAgentConfigContext(ctx)
	if err != nil {
		return "", fmt.Errorf("security check failed: %w", err)
	}

	// 2. 解析参数
	var args struct {
		Action  string `json:"action"`
		Content string `json:"content"`
	}
	if err := common.DecodeArgs(argumentsInJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// 3. 验证 action
	action := strings.ToLower(args.Action)
	if action != "read" && action != "append" && action != "clear" {
		return "", fmt.Errorf("invalid action: %s, must be one of: read, append, clear", args.Action)
	}

	// 4. 权限检查：Agent 存在且属于当前用户
	agent, err := t.agentService.GetAgentByCode(cfgCtx.AgentCode)
	if err != nil {
		return "", fmt.Errorf("failed to get agent: %w", err)
	}
	if agent == nil {
		return "", fmt.Errorf("agent not found: %s", cfgCtx.AgentCode)
	}
	if agent.UserCode != cfgCtx.UserCode {
		return "", fmt.Errorf("access denied: agent %s does not belong to user %s", cfgCtx.AgentCode, cfgCtx.UserCode)
	}

	// 5. 执行具体操作
	switch action {
	case "read":
		return t.handleRead(agent)
	case "append":
		return t.handleAppend(agent, args.Content)
	case "clear":
		return t.handleClear(agent)
	}

	return "", fmt.Errorf("unknown action: %s", action)
}

// handleRead 读取记忆
func (t *ManageAgentMemoryTool) handleRead(agent *models.Agent) (string, error) {
	result := map[string]interface{}{
		"success":    true,
		"action":     "read",
		"content":    agent.MemoryContent,
		"size_bytes": len(agent.MemoryContent),
		"updated_at": agent.UpdatedAt.Format(time.RFC3339),
	}

	out, _ := json.Marshal(result)
	return string(out), nil
}

// handleAppend 追加记忆
func (t *ManageAgentMemoryTool) handleAppend(agent *models.Agent, content string) (string, error) {
	if content == "" {
		return "", fmt.Errorf("content is required for append action")
	}

	// 验证内容大小（1MB限制）
	const maxMemorySize = 1024 * 1024 // 1MB
	newSize := len(agent.MemoryContent) + len(content) + 100 // 预留格式开销
	if newSize > maxMemorySize {
		return "", fmt.Errorf("memory size would exceed limit: current %d + new %d > max %d bytes",
			len(agent.MemoryContent), len(content), maxMemorySize)
	}

	// 追加内容，带时间戳
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	newEntry := fmt.Sprintf("\n## %s\n%s\n", timestamp, content)

	// 如果当前记忆为空，添加标题
	var newContent string
	if agent.MemoryContent == "" {
		newContent = "# Agent 长期记忆\n" + newEntry
	} else {
		newContent = agent.MemoryContent + newEntry
	}

	// 保存到数据库
	if err := t.agentService.UpdateMemoryByCode(agent.AgentCode, newContent); err != nil {
		return "", fmt.Errorf("failed to append memory: %w", err)
	}

	result := map[string]interface{}{
		"success":        true,
		"action":         "append",
		"message":        "记忆已追加",
		"bytes_appended": len(content),
		"total_size":     len(newContent),
		"updated_at":     time.Now().Format(time.RFC3339),
	}

	out, _ := json.Marshal(result)
	return string(out), nil
}

// handleClear 清空记忆
func (t *ManageAgentMemoryTool) handleClear(agent *models.Agent) (string, error) {
	oldSize := len(agent.MemoryContent)

	// 保存到数据库
	if err := t.agentService.UpdateMemoryByCode(agent.AgentCode, ""); err != nil {
		return "", fmt.Errorf("failed to clear memory: %w", err)
	}

	result := map[string]interface{}{
		"success":      true,
		"action":       "clear",
		"message":      "记忆已清空",
		"cleared_size": oldSize,
		"updated_at":   time.Now().Format(time.RFC3339),
	}

	out, _ := json.Marshal(result)
	return string(out), nil
}

// Run 执行工具逻辑（兼容接口）
func (t *ManageAgentMemoryTool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.InvokableRun(ctx, argumentsInJSON, opts...)
}
