package agent

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
)

// BootstrapMode 引导文件加载模式
type BootstrapMode string

const (
	// BootstrapFull 完整模式：加载所有引导文件（用于 MasterAgent）
	BootstrapFull BootstrapMode = "full"
	// BootstrapLight 轻量模式：只加载 AGENTS.md 和 TOOLS.md（用于后台任务）
	BootstrapLight BootstrapMode = "light"
)

// AgentConfig Agent 配置内容（从数据库加载）
type AgentConfig struct {
	IdentityContent string // IDENTITY.md 内容
	SoulContent     string // SOUL.md 内容
	AgentsContent   string // AGENTS.md 内容
	ToolsContent    string // TOOLS.md 内容
	UserContent     string // USER.md 内容

	// 运行时配置
	HistoryMessages int // 携带的历史对话消息数量（默认10，范围0-50）
}

// MCPServerInfo MCP Server 基本信息
type MCPServerInfo struct {
	Code        string
	Name        string
	Description string
}

// ContextBuilder 上下文构建器
type ContextBuilder struct {
	workspace       string
	skills          *SkillsLoader
	bootstrapMode   BootstrapMode   // 引导文件加载模式
	agentConfig     *AgentConfig    // Agent 配置内容（从数据库加载，优先使用）
	mcpServers      []MCPServerInfo // MCP Server 列表
}

// NewContextBuilder 创建上下文构建器
func NewContextBuilder(workspace string) *ContextBuilder {
	return &ContextBuilder{
		workspace:     workspace,
		skills:        NewSkillsLoader(workspace),
		bootstrapMode: BootstrapFull, // 默认完整模式
	}
}

// SetBootstrapMode 设置引导文件加载模式
func (c *ContextBuilder) SetBootstrapMode(mode BootstrapMode) {
	c.bootstrapMode = mode
}

// SetAgentConfig 设置 Agent 配置（从数据库加载）
// 设置后，引导文件将从这些配置中加载，而不是从 workspace 文件
func (c *ContextBuilder) SetAgentConfig(config *AgentConfig) {
	c.agentConfig = config
}

// GetSkillsLoader 获取技能加载器
func (c *ContextBuilder) GetSkillsLoader() *SkillsLoader {
	return c.skills
}

// GetHistoryMessages 获取历史消息数量
// 如果未设置 AgentConfig 或 HistoryMessages 为 0，返回默认值 10
// 返回值范围限制在 0-50 之间
func (c *ContextBuilder) GetHistoryMessages() int {
	if c.agentConfig == nil {
		return 10
	}
	n := c.agentConfig.HistoryMessages
	if n <= 0 {
		return 10
	}
	if n > 50 {
		return 50
	}
	return n
}

// SetMCPServers 设置 MCP Server 列表
func (c *ContextBuilder) SetMCPServers(servers []MCPServerInfo) {
	c.mcpServers = servers
}

// BuildMCPServersSection 构建 MCP Servers 部分
func (c *ContextBuilder) BuildMCPServersSection() string {
	if len(c.mcpServers) == 0 {
		return ""
	}

	var parts []string
	parts = append(parts, "## 可用的 MCP Servers")
	parts = append(parts, "你可以使用 use_mcp 工具加载以下 MCP Server 的工具：")
	parts = append(parts, "")

	for _, server := range c.mcpServers {
		desc := server.Description
		if desc == "" {
			desc = "无描述"
		}
		parts = append(parts, fmt.Sprintf("- **%s** (%s): %s", server.Code, server.Name, desc))
	}

	parts = append(parts, "")
	parts = append(parts, "使用示例：use_mcp(server_code=\"abcd\", action=\"load\")")

	return strings.Join(parts, "\n")
}

// BuildSystemPrompt 构建系统提示
func (c *ContextBuilder) BuildSystemPrompt() string {
	return c.BuildSystemPromptWithMode(c.bootstrapMode)
}

// BuildSystemPromptWithMode 使用指定模式构建系统提示
func (c *ContextBuilder) BuildSystemPromptWithMode(mode BootstrapMode) string {
	var parts []string

	// 核心身份
	parts = append(parts, c.getIdentity())

	// 引导文件（使用指定模式）
	bootstrap := c.loadBootstrapFilesWithMode(mode)
	if bootstrap != "" {
		parts = append(parts, bootstrap)
	}

	// 始终加载的技能
	alwaysSkills := c.skills.GetAlwaysSkills()
	if len(alwaysSkills) > 0 {
		alwaysContent := c.skills.LoadSkillsForContext(alwaysSkills)
		if alwaysContent != "" {
			parts = append(parts, "# 活动技能\n\n"+alwaysContent)
		}
	}

	// 可用技能摘要
	skillsSummary := c.skills.BuildSkillsSummary()
	if skillsSummary != "" {
		parts = append(parts, `# 技能

以下技能扩展了你的能力。要使用技能，请调用 use_skill 工具并传入技能名称。
例如：use_skill(name="github", action="workflow", params={"repo": "owner/repo"})
available="false" 的技能需要先安装依赖 - 你可以尝试使用 apt/brew 安装。

`+skillsSummary)
	}

	// MCP Servers
	mcpSection := c.BuildMCPServersSection()
	if mcpSection != "" {
		parts = append(parts, mcpSection)
	}

	return strings.Join(parts, "\n\n---\n\n")
}

// getIdentity 获取核心身份部分
func (c *ContextBuilder) getIdentity() string {
	now := time.Now().Format("2006-01-02 15:04 (Monday)")
	tz, _ := time.Now().Zone()
	system := runtime.GOOS
	if system == "darwin" {
		system = "macOS"
	}
	goVersion := runtime.Version()

	return fmt.Sprintf(`# nanobot 🐈

你是 nanobot，一个有帮助的 AI 助手。你可以使用以下工具：
- 读取、写入和编辑文件
- 执行 shell 命令
- 搜索网络和获取网页
- 向用户发送消息到聊天渠道
- 加载并使用技能（use_skill 工具）

## 当前时间
%s (%s)

## 运行环境
%s %s, Go %s

重要:当回答直接问题或对话时，直接回复文本。
只有当你需要向特定聊天渠道（如 WhatsApp）发送消息时才使用 'message' 工具。
对于普通对话，只需回复文本 - 不要调用 message 工具。

始终保持有帮助、准确和简洁。使用工具时，逐步思考：你知道什么、你需要什么、以及为什么选择这个工具。`, now, tz, system, runtime.GOARCH, goVersion)
}

// loadBootstrapFiles 加载引导文件
// 根据 bootstrapMode 决定加载哪些文件：
// - BootstrapFull: 加载所有引导文件（用于 MasterAgent）
// - BootstrapLight: 只加载 AGENTS.md 和 TOOLS.md（用于后台任务）
func (c *ContextBuilder) loadBootstrapFiles() string {
	return c.loadBootstrapFilesWithMode(c.bootstrapMode)
}

// loadBootstrapFilesWithMode 使用指定模式加载引导文件
// 优先从数据库 Agent 配置加载，如果没有则从 workspace 文件加载
func (c *ContextBuilder) loadBootstrapFilesWithMode(mode BootstrapMode) string {
	// 如果设置了 Agent 配置（从数据库加载），优先使用
	if c.agentConfig != nil {
		return c.loadBootstrapFromAgentConfig(mode)
	}

	// 否则从 workspace 文件加载
	var bootstrapFiles []string

	switch mode {
	case BootstrapLight:
		// 轻量模式：只加载代理定义和工具定义
		bootstrapFiles = []string{"AGENTS.md", "TOOLS.md"}
	case BootstrapFull:
		// 完整模式：加载所有引导文件
		bootstrapFiles = []string{"AGENTS.md", "SOUL.md", "USER.md", "TOOLS.md", "IDENTITY.md"}
	default:
		// 默认完整模式
		bootstrapFiles = []string{"AGENTS.md", "SOUL.md", "USER.md", "TOOLS.md", "IDENTITY.md"}
	}

	var parts []string

	for _, filename := range bootstrapFiles {
		filePath := filepath.Join(c.workspace, filename)
		if data, err := os.ReadFile(filePath); err == nil {
			content := string(data)
			parts = append(parts, "## "+filename+"\n\n"+content)
		}
	}

	return strings.Join(parts, "\n\n")
}

// loadBootstrapFromAgentConfig 从 Agent 配置（数据库）加载引导文件
func (c *ContextBuilder) loadBootstrapFromAgentConfig(mode BootstrapMode) string {
	var filesToLoad []struct {
		name    string
		content string
	}

	// 根据模式决定加载哪些文件
	switch mode {
	case BootstrapLight:
		// 轻量模式：只加载代理定义和工具定义
		filesToLoad = []struct {
			name    string
			content string
		}{
			{"AGENTS.md", c.agentConfig.AgentsContent},
			{"TOOLS.md", c.agentConfig.ToolsContent},
		}
	case BootstrapFull:
		// 完整模式：加载所有引导文件
		filesToLoad = []struct {
			name    string
			content string
		}{
			{"IDENTITY.md", c.agentConfig.IdentityContent},
			{"SOUL.md", c.agentConfig.SoulContent},
			{"AGENTS.md", c.agentConfig.AgentsContent},
			{"TOOLS.md", c.agentConfig.ToolsContent},
			{"USER.md", c.agentConfig.UserContent},
		}
	default:
		// 默认完整模式
		filesToLoad = []struct {
			name    string
			content string
		}{
			{"IDENTITY.md", c.agentConfig.IdentityContent},
			{"SOUL.md", c.agentConfig.SoulContent},
			{"AGENTS.md", c.agentConfig.AgentsContent},
			{"TOOLS.md", c.agentConfig.ToolsContent},
			{"USER.md", c.agentConfig.UserContent},
		}
	}

	var parts []string
	for _, file := range filesToLoad {
		if file.content != "" {
			parts = append(parts, "## "+file.name+"\n\n"+file.content)
		}
	}

	return strings.Join(parts, "\n\n")
}

// BuildMessages 构建消息列表
func (c *ContextBuilder) BuildMessages(history []map[string]any, currentMessage string, skillNames []string, media []string, channel, chatID string) []map[string]any {
	var messages []map[string]any

	// 系统提示
	systemPrompt := c.BuildSystemPrompt()
	if channel != "" && chatID != "" {
		systemPrompt += fmt.Sprintf("\n\n## 当前会话\n渠道: %s\n聊天 ID: %s", channel, chatID)
	}
	messages = append(messages, map[string]any{
		"role":    "system",
		"content": systemPrompt,
	})

	// 历史消息
	messages = append(messages, history...)

	// 当前消息（带可选图片附件）
	userContent := c.buildUserContent(currentMessage, media)
	messages = append(messages, map[string]any{
		"role":    "user",
		"content": userContent,
	})

	return messages
}

// buildUserContent 构建用户消息内容
func (c *ContextBuilder) buildUserContent(text string, media []string) any {
	if len(media) == 0 {
		return text
	}

	var images []map[string]any
	for _, path := range media {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		// 检测 MIME 类型
		mime := "image/jpeg"
		if strings.HasSuffix(strings.ToLower(path), ".png") {
			mime = "image/png"
		} else if strings.HasSuffix(strings.ToLower(path), ".gif") {
			mime = "image/gif"
		} else if strings.HasSuffix(strings.ToLower(path), ".webp") {
			mime = "image/webp"
		}

		b64 := base64.StdEncoding.EncodeToString(data)
		images = append(images, map[string]any{
			"type": "image_url",
			"image_url": map[string]string{
				"url": fmt.Sprintf("data:%s;base64,%s", mime, b64),
			},
		})
	}

	if len(images) == 0 {
		return text
	}

	// 返回多部分内容
	var content []map[string]any
	content = append(content, images...)
	content = append(content, map[string]any{
		"type": "text",
		"text": text,
	})
	return content
}

// AddToolResult 添加工具结果到消息列表
func (c *ContextBuilder) AddToolResult(messages []map[string]any, toolCallID, toolName, result string) []map[string]any {
	return append(messages, map[string]any{
		"role":         "tool",
		"tool_call_id": toolCallID,
		"name":         toolName,
		"content":      result,
	})
}

// AddAssistantMessage 添加助手消息到消息列表
func (c *ContextBuilder) AddAssistantMessage(messages []map[string]any, content string, toolCalls []map[string]any, reasoningContent string) []map[string]any {
	msg := map[string]any{
		"role":    "assistant",
		"content": content,
	}

	if len(toolCalls) > 0 {
		msg["tool_calls"] = toolCalls
	}

	if reasoningContent != "" {
		msg["reasoning_content"] = reasoningContent
	}

	return append(messages, msg)
}

// HasBinary 检查二进制文件是否存在
func HasBinary(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// BuildMessageList 构建消息列表（使用 schema.Message 类型）
// 这是一个公共方法，供 Loop 和 SupervisorAgent 复用
func BuildMessageList(systemPrompt string, history []*schema.Message, userInput, channel, chatID string) []*schema.Message {
	// 添加会话信息
	if channel != "" && chatID != "" {
		systemPrompt += fmt.Sprintf("\n\n## 当前会话\n渠道: %s\n聊天 ID: %s", channel, chatID)
	}

	// 构建消息列表
	messages := make([]*schema.Message, 0, len(history)+2)

	// 添加系统消息
	messages = append(messages, &schema.Message{
		Role:    schema.System,
		Content: systemPrompt,
	})

	// 添加历史消息
	if len(history) > 0 {
		messages = append(messages, history...)
	}

	// 添加当前用户消息
	messages = append(messages, &schema.Message{
		Role:    schema.User,
		Content: userInput,
	})

	return messages
}
