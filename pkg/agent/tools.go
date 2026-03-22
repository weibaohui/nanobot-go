package agent

import (
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/askuser"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/config"
	toolcron "github.com/weibaohui/nanobot-go/pkg/agent/tools/cron"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/editfile"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/exec"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/listdir"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/mcp"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/readfile"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/skill"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/webfetch"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/websearch"
	"github.com/weibaohui/nanobot-go/pkg/agent/tools/writefile"
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
