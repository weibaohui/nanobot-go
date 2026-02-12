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
)

// ContextBuilder 上下文构建器
type ContextBuilder struct {
	workspace string
	memory    *MemoryStore
	skills    *SkillsLoader
}

// NewContextBuilder 创建上下文构建器
func NewContextBuilder(workspace string) *ContextBuilder {
	return &ContextBuilder{
		workspace: workspace,
		memory:    NewMemoryStore(workspace),
		skills:    NewSkillsLoader(workspace),
	}
}

// BuildSystemPrompt 构建系统提示
func (c *ContextBuilder) BuildSystemPrompt(skillNames []string) string {
	var parts []string

	// 核心身份
	parts = append(parts, c.getIdentity())

	// 引导文件
	bootstrap := c.loadBootstrapFiles()
	if bootstrap != "" {
		parts = append(parts, bootstrap)
	}

	// 内存上下文
	memory := c.memory.GetMemoryContext()
	if memory != "" {
		parts = append(parts, "# 内存\n\n"+memory)
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

以下技能扩展了你的能力。要使用技能，请使用 read_file 工具读取其 SKILL.md 文件。
available="false" 的技能需要先安装依赖 - 你可以尝试使用 apt/brew 安装。

`+skillsSummary)
	}

	return strings.Join(parts, "\n\n---\n\n")
}

// getIdentity 获取核心身份部分
func (c *ContextBuilder) getIdentity() string {
	now := time.Now().Format("2006-01-02 15:04 (Monday)")
	tz, _ := time.Now().Zone()
	workspacePath, _ := filepath.Abs(c.workspace)
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
- 创建子代理处理后台任务

## 当前时间
%s (%s)

## 运行环境
%s %s, Go %s

## 工作区
你的工作区位于: %s
- 内存文件: %s/memory/MEMORY.md
- 每日笔记: %s/memory/YYYY-MM-DD.md
- 自定义技能: %s/skills/{skill-name}/SKILL.md

重要: 当回答直接问题或对话时，直接回复文本。
只有当你需要向特定聊天渠道（如 WhatsApp）发送消息时才使用 'message' 工具。
对于普通对话，只需回复文本 - 不要调用 message 工具。

始终保持有帮助、准确和简洁。使用工具时，逐步思考：你知道什么、你需要什么、以及为什么选择这个工具。
当记住某些内容时，写入 %s/memory/MEMORY.md`, now, tz, system, runtime.GOARCH, goVersion, workspacePath, workspacePath, workspacePath, workspacePath, workspacePath)
}

// loadBootstrapFiles 加载引导文件
func (c *ContextBuilder) loadBootstrapFiles() string {
	bootstrapFiles := []string{"AGENTS.md", "SOUL.md", "USER.md", "TOOLS.md", "IDENTITY.md"}
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

// BuildMessages 构建消息列表
func (c *ContextBuilder) BuildMessages(history []map[string]any, currentMessage string, skillNames []string, media []string, channel, chatID string) []map[string]any {
	var messages []map[string]any

	// 系统提示
	systemPrompt := c.BuildSystemPrompt(skillNames)
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
