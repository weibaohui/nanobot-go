package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/weibaohui/nanobot-go/agent/tools"
	"github.com/weibaohui/nanobot-go/agent/tools/editfile"
	"github.com/weibaohui/nanobot-go/agent/tools/exec"
	"github.com/weibaohui/nanobot-go/agent/tools/listdir"
	"github.com/weibaohui/nanobot-go/agent/tools/readfile"
	"github.com/weibaohui/nanobot-go/agent/tools/webfetch"
	"github.com/weibaohui/nanobot-go/agent/tools/websearch"
	"github.com/weibaohui/nanobot-go/agent/tools/writefile"
	"github.com/weibaohui/nanobot-go/bus"
	"github.com/weibaohui/nanobot-go/providers"
	"go.uber.org/zap"
)

// SubagentManager 子代理管理器
type SubagentManager struct {
	provider            providers.LLMProvider
	workspace           string
	bus                 *bus.MessageBus
	model               string
	execTimeout         int
	restrictToWorkspace bool
	runningTasks        map[string]context.CancelFunc
	mu                  sync.Mutex
	logger              *zap.Logger
}

// NewSubagentManager 创建子代理管理器
func NewSubagentManager(provider providers.LLMProvider, workspace string, messageBus *bus.MessageBus, model string, execTimeout int, restrictToWorkspace bool, logger *zap.Logger) *SubagentManager {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SubagentManager{
		provider:            provider,
		workspace:           workspace,
		bus:                 messageBus,
		model:               model,
		execTimeout:         execTimeout,
		restrictToWorkspace: restrictToWorkspace,
		runningTasks:        make(map[string]context.CancelFunc),
		logger:              logger,
	}
}

// Spawn 创建子代理执行后台任务
func (m *SubagentManager) Spawn(ctx context.Context, task, label, originChannel, originChatID string) (string, error) {
	taskID := generateTaskID()
	displayLabel := label
	if displayLabel == "" {
		if len(task) > 30 {
			displayLabel = task[:30] + "..."
		} else {
			displayLabel = task
		}
	}

	bgCtx, cancel := context.WithCancel(context.Background())
	m.mu.Lock()
	m.runningTasks[taskID] = cancel
	m.mu.Unlock()

	cleanup := func() {
		m.mu.Lock()
		delete(m.runningTasks, taskID)
		m.mu.Unlock()
	}

	go func() {
		defer cleanup()
		m.runSubagent(bgCtx, taskID, task, displayLabel, originChannel, originChatID)
	}()

	m.logger.Info("创建子代理",
		zap.String("任务ID", taskID),
		zap.String("标签", displayLabel),
	)

	return fmt.Sprintf("子代理 [%s] 已启动 (id: %s)。完成后会通知你。", displayLabel, taskID), nil
}

// runSubagent 执行子代理任务
func (m *SubagentManager) runSubagent(ctx context.Context, taskID, task, label, originChannel, originChatID string) {
	m.logger.Info("子代理开始执行任务",
		zap.String("任务ID", taskID),
		zap.String("标签", label),
	)

	registry := tools.NewRegistry()
	allowedDir := ""
	if m.restrictToWorkspace {
		allowedDir = m.workspace
	}

	registry.Register(&readfile.Tool{AllowedDir: allowedDir})
	registry.Register(&writefile.Tool{AllowedDir: allowedDir})
	registry.Register(&editfile.Tool{AllowedDir: allowedDir})
	registry.Register(&listdir.Tool{AllowedDir: allowedDir})
	registry.Register(&exec.Tool{Timeout: m.execTimeout, WorkingDir: m.workspace, RestrictToWorkspace: m.restrictToWorkspace})
	registry.Register(&websearch.Tool{MaxResults: 5})
	registry.Register(&webfetch.Tool{MaxChars: 50000})

	systemPrompt := m.buildSubagentPrompt(task)
	messages := []map[string]any{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": task},
	}

	maxIterations := 15
	var finalResult string

	for i := 0; i < maxIterations; i++ {
		select {
		case <-ctx.Done():
			m.logger.Info("子代理任务被取消", zap.String("任务ID", taskID))
			return
		default:
		}

		response, err := m.provider.Chat(ctx, messages, registry.GetDefinitions(ctx), nil, m.model, 4096, 0.7)
		if err != nil {
			m.logger.Error("子代理 LLM 调用失败", zap.Error(err))
			m.announceResult(taskID, label, task, fmt.Sprintf("错误: %s", err), originChannel, originChatID, "error")
			return
		}

		if response.HasToolCalls() {
			toolCallDicts := m.buildToolCallDicts(response.ToolCalls)
			messages = append(messages, map[string]any{
				"role":       "assistant",
				"content":    response.Content,
				"tool_calls": toolCallDicts,
			})

			for _, tc := range response.ToolCalls {
				argsJSON, _ := json.Marshal(tc.Arguments)
				m.logger.Debug("子代理执行工具",
					zap.String("任务ID", taskID),
					zap.String("工具", tc.Name),
					zap.String("参数", string(argsJSON)),
				)

				result, err := registry.Execute(ctx, tc.Name, tc.Arguments)
				if err != nil {
					result = fmt.Sprintf("错误: %s", err)
				}
				messages = append(messages, map[string]any{
					"role":         "tool",
					"tool_call_id": tc.ID,
					"name":         tc.Name,
					"content":      result,
				})
			}

			messages = append(messages, map[string]any{
				"role":    "user",
				"content": "反思结果并决定下一步。",
			})
		} else {
			finalResult = response.Content
			break
		}
	}

	if finalResult == "" {
		finalResult = "任务完成但没有生成最终响应。"
	}

	m.logger.Info("子代理任务完成", zap.String("任务ID", taskID))
	m.announceResult(taskID, label, task, finalResult, originChannel, originChatID, "ok")
}

// buildSubagentPrompt 构建子代理系统提示
func (m *SubagentManager) buildSubagentPrompt(task string) string {
	now := time.Now().Format("2006-01-02 15:04 (Monday)")
	tz, _ := time.Now().Zone()

	return fmt.Sprintf(`# 子代理

## 当前时间
%s (%s)

你是主代理创建的子代理，用于完成特定任务。

## 规则
1. 保持专注 - 只完成分配的任务，不做其他事情
2. 你的最终响应会报告回主代理
3. 不要发起对话或接受额外任务
4. 在发现中要简洁但信息丰富

## 你能做什么
- 在工作区读取和写入文件
- 执行 shell 命令
- 搜索网络和获取网页
- 彻底完成任务

## 你不能做什么
- 直接向用户发送消息（没有 message 工具）
- 创建其他子代理
- 访问主代理的对话历史

## 工作区
你的工作区位于: %s
技能位于: %s/skills/（按需读取 SKILL.md 文件）

完成任务后，提供清晰的发现或操作摘要。`, now, tz, m.workspace, m.workspace)
}

// announceResult 公告子代理结果
func (m *SubagentManager) announceResult(taskID, label, task, result, originChannel, originChatID, status string) {
	statusText := "成功完成"
	if status == "error" {
		statusText = "失败"
	}

	content := fmt.Sprintf(`[子代理 '%s' %s]

任务: %s

结果:
%s

自然地为用户总结这个结果。保持简洁（1-2句话）。不要提及"子代理"或任务 ID 等技术细节。`, label, statusText, task, result)

	msg := bus.NewInboundMessage("system", "subagent", originChannel+":"+originChatID, content)
	m.bus.PublishInbound(msg)

	m.logger.Debug("子代理公告结果",
		zap.String("任务ID", taskID),
		zap.String("渠道", originChannel),
		zap.String("聊天ID", originChatID),
	)
}

// buildToolCallDicts 构建工具调用字典
func (m *SubagentManager) buildToolCallDicts(toolCalls []providers.ToolCallRequest) []map[string]any {
	var dicts []map[string]any
	for _, tc := range toolCalls {
		argsJSON, _ := json.Marshal(tc.Arguments)
		dicts = append(dicts, map[string]any{
			"id":   tc.ID,
			"type": "function",
			"function": map[string]any{
				"name":      tc.Name,
				"arguments": string(argsJSON),
			},
		})
	}
	return dicts
}

// GetRunningCount 获取正在运行的任务数量
func (m *SubagentManager) GetRunningCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.runningTasks)
}

// generateTaskID 生成任务 ID
func generateTaskID() string {
	return fmt.Sprintf("%08x", time.Now().UnixNano()&0xFFFFFFFF)
}
