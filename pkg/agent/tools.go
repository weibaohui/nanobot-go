package agent

import (
	"fmt"

	"github.com/weibaohui/nanobot-go/pkg/agent/tools/askuser"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/config"
	toolcron "github.com/weibaohui/nanobot-go/pkg/agent/tools/cron"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/editfile"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/exec"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/listdir"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/mcp"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/message"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/readfile"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/skill"
	tasktool "github.com/weibaohui/nanobot-go/pkg/agent/tools/task"
	"github.com/weibaohui/nanobot-go/pkg/agent/task"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/webfetch"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/websearch"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/writefile"
	"github.com/weibaohui/nanobot-go/pkg/bus"
	"go.uber.org/zap"
)

// registerDefaultTools 注册默认工具
func (l *Loop) registerDefaultTools() {
	allowedDir := ""
	if l.restrictToWorkspace {
		allowedDir = l.workspace
	}

	// 文件工具
	l.tools.Register(&readfile.Tool{AllowedDir: allowedDir})
	l.tools.Register(&writefile.Tool{AllowedDir: allowedDir})
	l.tools.Register(&editfile.Tool{AllowedDir: allowedDir})
	l.tools.Register(&listdir.Tool{AllowedDir: allowedDir})

	// Shell 工具
	l.tools.Register(&exec.Tool{Timeout: l.execTimeout, WorkingDir: l.workspace, RestrictToWorkspace: l.restrictToWorkspace})

	// Web 工具
	l.tools.Register(&websearch.Tool{MaxResults: 5})
	l.tools.Register(&webfetch.Tool{MaxChars: 50000})

	// 消息工具
	l.tools.Register(&message.Tool{SendCallback: func(msg any) error {
		if outMsg, ok := msg.(*bus.OutboundMessage); ok {
			l.bus.PublishOutbound(outMsg)
		}
		return nil
	}})

	// Cron 工具
	if l.cronService != nil {
		l.tools.Register(&toolcron.Tool{CronService: l.cronService})
	}

	// Ask User 工具（用于向用户提问并中断等待响应）
	l.tools.Register(askuser.NewTool(func(channel, chatID, question string, options []string) (string, error) {
		// 这个回调会在 InterruptManager 中处理
		// 实际的中断处理在 tool 的 InvokableRun 中通过 StatefulInterrupt 完成
		return "", nil
	}))

	// 注册通用技能工具（用于拦截后的技能调用）
	l.tools.Register(skill.NewGenericSkillTool(l.context.GetSkillsLoader().LoadSkill))

	// 注册 Agent 配置管理工具
	if l.agentService != nil {
		configTools := config.NewTools(l.agentService)
		l.tools.Register(configTools.ReadAgentConfigTool)
		l.tools.Register(configTools.UpdateAgentConfigTool)
	}

	// 注册 use_mcp 工具（用于按需加载 MCP Server）
	if l.mcpManager != nil {
		l.tools.Register(mcp.NewUseMCPTool(l.mcpManager))
		l.tools.Register(mcp.NewCallMCPTool(l.mcpManager))
		l.logger.Info("MCP 工具已注册", zap.Strings("tools", []string{"use_mcp", "call_mcp_tool"}))
	}
}

// registerTaskTools 注册后台任务工具
func (l *Loop) registerTaskTools(manager tasktool.Manager) {
	if manager == nil {
		return
	}
	l.tools.Register(&tasktool.StartTool{Manager: manager, Logger: l.logger})
	l.tools.Register(&tasktool.GetTool{Manager: manager, Logger: l.logger})
	l.tools.Register(&tasktool.StopTool{Manager: manager, Logger: l.logger})
	l.tools.Register(&tasktool.ListTool{Manager: manager, Logger: l.logger})
}

// createBackgroundAgentTaskManager 创建任务管理器
func (l *Loop) createBackgroundAgentTaskManager() *task.Manager {
	taskManager, err := task.NewManager(&task.ManagerConfig{
		ConfigLoader:    l.configLoader,
		Workspace:       l.workspace,
		Tools:           l.tools.GetTools(),
		Logger:          l.logger,
		Context:         l.context,
		CheckpointStore: l.interruptManager.GetCheckpointStore(),
		MaxIterations:   l.maxIterations,
		Sessions:        l.sessions,
		HookManager:     l.hookManager,
		EventBus:        l.bus, // 传递EventBus用于WebSocket推送
		OnTaskComplete: func(channel, chatID, taskID string, status task.Status, result string) {
			// 任务完成时发送通知消息
			statusText := map[task.Status]string{
				task.StatusFinished: "完成",
				task.StatusFailed:   "失败",
				task.StatusStopped:  "已停止",
			}[status]
			msg := fmt.Sprintf("后台任务 %s\n状态: %s\n任务ID: %s", statusText, statusText, taskID)
			if result != "" && status == task.StatusFinished {
				msg = fmt.Sprintf("后台任务完成\n任务ID: %s\n\n%s", taskID, result)
			}
			l.bus.PublishOutbound(bus.NewOutboundMessage(channel, chatID, msg))
		},
	})
	if err != nil {
		l.logger.Error("创建任务管理器失败", zap.Error(err))
		return nil
	}
	return taskManager
}
