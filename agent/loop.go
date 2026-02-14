package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/agent/tools"
	toolcron "github.com/weibaohui/nanobot-go/agent/tools/cron"
	"github.com/weibaohui/nanobot-go/agent/tools/editfile"
	"github.com/weibaohui/nanobot-go/agent/tools/exec"
	"github.com/weibaohui/nanobot-go/agent/tools/listdir"
	"github.com/weibaohui/nanobot-go/agent/tools/message"
	"github.com/weibaohui/nanobot-go/agent/tools/readfile"
	"github.com/weibaohui/nanobot-go/agent/tools/skill"
	"github.com/weibaohui/nanobot-go/agent/tools/spawn"
	"github.com/weibaohui/nanobot-go/agent/tools/webfetch"
	"github.com/weibaohui/nanobot-go/agent/tools/websearch"
	"github.com/weibaohui/nanobot-go/agent/tools/writefile"
	"github.com/weibaohui/nanobot-go/bus"
	"github.com/weibaohui/nanobot-go/cron"
	"github.com/weibaohui/nanobot-go/eino_adapter"
	"github.com/weibaohui/nanobot-go/providers"
	"github.com/weibaohui/nanobot-go/session"
	"go.uber.org/zap"
)

// Loop 代理循环核心
type Loop struct {
	bus                 *bus.MessageBus
	provider            providers.LLMProvider
	workspace           string
	model               string
	maxIterations       int
	execTimeout         int
	restrictToWorkspace bool
	cronService         *cron.Service
	context             *ContextBuilder
	sessions            *session.Manager
	tools               *tools.Registry
	subagents           *SubagentManager
	running             bool
	logger              *zap.Logger

	// ADK Agent
	adkAgent  *adk.ChatModelAgent
	adkRunner *adk.Runner

	// Plan-Execute mode support
	planAgent *eino_adapter.PlanExecuteAgent
	selector  *eino_adapter.ModeSelector
}

// NewLoop 创建代理循环
func NewLoop(messageBus *bus.MessageBus, provider providers.LLMProvider, workspace string, model string, maxIterations int, execTimeout int, restrictToWorkspace bool, cronService *cron.Service, sessionManager *session.Manager, logger *zap.Logger) *Loop {
	if logger == nil {
		logger = zap.NewNop()
	}

	loop := &Loop{
		bus:                 messageBus,
		provider:            provider,
		workspace:           workspace,
		model:               model,
		maxIterations:       maxIterations,
		execTimeout:         execTimeout,
		restrictToWorkspace: restrictToWorkspace,
		cronService:         cronService,
		context:             NewContextBuilder(workspace),
		sessions:            sessionManager,
		tools:               tools.NewRegistry(),
		logger:              logger,
	}

	// 创建子代理管理器
	loop.subagents = NewSubagentManager(provider, workspace, messageBus, model, execTimeout, restrictToWorkspace, logger)

	// 注册默认工具
	loop.registerDefaultTools()

	registeredTools := loop.GetTools()
	toolNames := make([]string, 0, len(registeredTools))
	for _, t := range registeredTools {
		if t != nil {
			info, err := t.Info(context.Background())
			if err == nil && info != nil && info.Name != "" {
				toolNames = append(toolNames, info.Name)
			}
		}
	}
	logger.Info("已注册工具",
		zap.Int("数量", len(registeredTools)),
		zap.Strings("工具列表", toolNames),
	)

	// 创建 ADK Agent
	ctx := context.Background()
	adapter := eino_adapter.NewProviderAdapter(provider, model)

	// 配置适配器：设置技能加载器和已注册工具列表
	adapter.SetSkillLoader(loop.context.GetSkillsLoader().LoadSkill)
	adapter.SetRegisteredTools(toolNames)

	adkTools := loop.tools.GetADKTools()
	var toolsConfig adk.ToolsConfig
	if len(adkTools) > 0 {
		toolsConfig = adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: adkTools,
			},
		}
	}
	adkAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "nanobot",
		Description:   "nanobot AI assistant",
		Model:         adapter,
		ToolsConfig:   toolsConfig,
		MaxIterations: maxIterations,
	})
	if err != nil {
		logger.Error("创建 ADK Agent 失败，将使用回退模式", zap.Error(err))
	} else {
		loop.adkAgent = adkAgent
		loop.adkRunner = adk.NewRunner(ctx, adk.RunnerConfig{
			Agent:           adkAgent,
			EnableStreaming: true,
		})
		logger.Info("ADK Agent 创建成功")
	}

	return loop
}

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

	// Spawn 工具
	l.tools.Register(&spawn.Tool{Manager: l.subagents})

	// Cron 工具
	if l.cronService != nil {
		l.tools.Register(&toolcron.Tool{CronService: l.cronService})
	}

	// 注册通用技能工具（用于拦截后的技能调用）
	l.tools.Register(skill.NewGenericSkillTool(l.context.GetSkillsLoader().LoadSkill))

	// 动态注册所有技能为工具
	// l.registerSkillTools()
}

// registerSkillTools 扫描并注册所有技能为工具
func (l *Loop) registerSkillTools() {
	skillsLoader := l.context.GetSkillsLoader()
	skills := skillsLoader.ListSkills(false)

	loadSkillFunc := skillsLoader.LoadSkill

	for _, sk := range skills {
		// 获取技能描述
		description := sk.Name
		if meta := skillsLoader.GetSkillMetadata(sk.Name); meta != nil {
			if desc, ok := meta["description"]; ok && desc != "" {
				description = desc
			}
		}

		// 为每个技能创建并注册工具
		skillTool := skill.NewDynamicTool(sk.Name, description, loadSkillFunc)
		l.tools.Register(skillTool)
		l.logger.Debug("注册技能工具",
			zap.String("name", sk.Name),
			zap.String("source", sk.Source),
		)
	}

	l.logger.Info("已注册技能工具",
		zap.Int("数量", len(skills)),
	)
}

// Run 运行代理循环
func (l *Loop) Run(ctx context.Context) error {
	l.running = true
	l.logger.Info("代理循环已启动")

	for l.running {
		select {
		case <-ctx.Done():
			l.running = false
			return ctx.Err()
		default:
		}

		// 等待消息
		msg, err := l.bus.ConsumeInbound(ctx)
		if err != nil {
			if err == context.DeadlineExceeded {
				continue
			}
			return err
		}

		// 处理消息
		if err := l.processMessage(ctx, msg); err != nil {
			l.logger.Error("处理消息失败", zap.Error(err))
			l.bus.PublishOutbound(bus.NewOutboundMessage(msg.Channel, msg.ChatID, fmt.Sprintf("抱歉，我遇到了错误: %s", err)))
		}
	}

	return nil
}

// Stop 停止代理循环
func (l *Loop) Stop() {
	l.running = false
	l.logger.Info("代理循环正在停止")
}

// processMessage 处理单条消息
func (l *Loop) processMessage(ctx context.Context, msg *bus.InboundMessage) error {
	preview := msg.Content
	if len(preview) > 80 {
		preview = preview[:80] + "..."
	}
	l.logger.Info("处理消息",
		zap.String("渠道", msg.Channel),
		zap.String("发送者", msg.SenderID),
		zap.String("内容", preview),
	)

	// 更新工具上下文
	l.updateToolContext(msg.Channel, msg.ChatID)

	// 检查是否使用计划模式
	if l.planAgent != nil && l.selector != nil && l.selector.ShouldUsePlanMode(msg.Content) {
		l.logger.Info("使用计划执行模式")
		return l.processWithPlan(ctx, msg)
	}

	// 检查是否使用流式处理
	if l.ShouldUseStream(msg.Channel) {
		return l.processWithStream(ctx, msg)
	}

	// 使用普通模式
	return l.processWithADK(ctx, msg)
}

// updateToolContext 更新工具上下文
func (l *Loop) updateToolContext(channel, chatID string) {
	if mt, ok := l.tools.Get("message").(tools.ContextSetter); ok {
		mt.SetContext(channel, chatID)
	}
	if st, ok := l.tools.Get("spawn").(tools.ContextSetter); ok {
		st.SetContext(channel, chatID)
	}
	if ct, ok := l.tools.Get("cron").(tools.ContextSetter); ok {
		ct.SetContext(channel, chatID)
	}
}

// buildMessagesWithSystem 构建包含系统提示词的消息列表
func (l *Loop) buildMessagesWithSystem(history []*schema.Message, userInput, channel, chatID string) []*schema.Message {
	// 构建系统提示词
	systemPrompt := l.context.BuildSystemPrompt(nil)
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

// processWithADK 使用 ADK Agent 处理消息
func (l *Loop) processWithADK(ctx context.Context, msg *bus.InboundMessage) error {
	if l.adkAgent == nil || l.adkRunner == nil {
		return fmt.Errorf("ADK Agent 未初始化")
	}

	// 获取会话历史
	sessionKey := msg.SessionKey()
	sess := l.sessions.GetOrCreate(sessionKey)
	history := l.convertHistory(sess.GetHistory(50))

	// 构建消息（包含系统提示词）
	messages := l.buildMessagesWithSystem(history, msg.Content, msg.Channel, msg.ChatID)

	iter := l.adkRunner.Run(ctx, messages)
	var response string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return fmt.Errorf("ADK Agent 执行失败: %w", event.Err)
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			msgOutput, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				continue
			}
			response = msgOutput.Content
		}
	}

	// 发布响应
	l.bus.PublishOutbound(bus.NewOutboundMessage(msg.Channel, msg.ChatID, response))

	// 保存会话
	sess.AddMessage("user", msg.Content)
	sess.AddMessage("assistant", response)
	l.sessions.Save(sess)

	l.logger.Info("响应完成",
		zap.String("渠道", msg.Channel),
		zap.Int("内容长度", len(response)),
	)

	return nil
}

// processWithStream 使用流式处理
func (l *Loop) processWithStream(ctx context.Context, msg *bus.InboundMessage) error {
	if l.adkAgent == nil || l.adkRunner == nil {
		return fmt.Errorf("ADK Agent 未初始化")
	}

	// 获取会话历史
	sessionKey := msg.SessionKey()
	sess := l.sessions.GetOrCreate(sessionKey)
	history := l.convertHistory(sess.GetHistory(50))

	// 构建消息（包含系统提示词）
	messages := l.buildMessagesWithSystem(history, msg.Content, msg.Channel, msg.ChatID)

	iter := l.adkRunner.Run(ctx, messages)
	var response string
	var fullContent string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return fmt.Errorf("流式执行失败: %w", event.Err)
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			msgOutput, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				continue
			}
			delta := ""
			if fullContent != "" && strings.HasPrefix(msgOutput.Content, fullContent) {
				delta = msgOutput.Content[len(fullContent):]
			} else {
				delta = msgOutput.Content
			}
			if delta != "" {
				l.bus.PublishStream(bus.NewStreamChunk(msg.Channel, msg.ChatID, delta, msgOutput.Content, false))
			}
			fullContent = msgOutput.Content
			response = msgOutput.Content
		}
	}

	// 发送完成标记
	l.bus.PublishStream(bus.NewStreamChunk(msg.Channel, msg.ChatID, "", response, true))

	// 保存会话
	sess.AddMessage("user", msg.Content)
	sess.AddMessage("assistant", response)
	l.sessions.Save(sess)

	l.logger.Info("流式响应完成",
		zap.String("渠道", msg.Channel),
		zap.Int("内容长度", len(response)),
	)

	return nil
}

// processWithPlan 使用计划执行模式
func (l *Loop) processWithPlan(ctx context.Context, msg *bus.InboundMessage) error {
	response, err := l.planAgent.Execute(ctx, msg.Content)
	if err != nil {
		return fmt.Errorf("计划执行失败: %w", err)
	}

	// 发布响应
	l.bus.PublishOutbound(bus.NewOutboundMessage(msg.Channel, msg.ChatID, response))

	// 保存会话
	sessionKey := msg.SessionKey()
	sess := l.sessions.GetOrCreate(sessionKey)
	sess.AddMessage("user", msg.Content)
	sess.AddMessage("assistant", response)
	l.sessions.Save(sess)

	return nil
}

// convertHistory 转换会话历史为 eino Message 格式
func (l *Loop) convertHistory(history []map[string]any) []*schema.Message {
	result := make([]*schema.Message, 0, len(history))
	for _, h := range history {
		role := schema.User
		if r, ok := h["role"].(string); ok && r == "assistant" {
			role = schema.Assistant
		}
		content := ""
		if c, ok := h["content"].(string); ok {
			content = c
		}
		result = append(result, &schema.Message{
			Role:    role,
			Content: content,
		})
	}
	return result
}

// ShouldUseStream 判断是否应该使用流式处理
func (l *Loop) ShouldUseStream(channel string) bool {
	return channel == "websocket"
}

// ProcessDirect 直接处理消息（用于 CLI 或 cron）
func (l *Loop) ProcessDirect(ctx context.Context, content, sessionKey, channel, chatID string) (string, error) {
	// 更新工具上下文
	l.updateToolContext(channel, chatID)

	if l.adkAgent == nil || l.adkRunner == nil {
		return "", fmt.Errorf("ADK Agent 未初始化")
	}

	// 获取会话历史
	sess := l.sessions.GetOrCreate(sessionKey)
	history := l.convertHistory(sess.GetHistory(50))

	// 构建消息（包含系统提示词）
	messages := l.buildMessagesWithSystem(history, content, channel, chatID)

	iter := l.adkRunner.Run(ctx, messages)
	var response string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", event.Err
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			msgOutput, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				continue
			}
			response = msgOutput.Content
		}
	}

	// 保存会话
	sess.AddMessage("user", content)
	sess.AddMessage("assistant", response)
	l.sessions.Save(sess)

	return response, nil
}

// GetTools returns all registered tools as a slice
func (l *Loop) GetTools() []tool.BaseTool {
	return l.tools.GetADKTools()
}

// SetPlanAgent sets the plan-execute agent for complex task handling
func (l *Loop) SetPlanAgent(planAgent *eino_adapter.PlanExecuteAgent) {
	l.planAgent = planAgent
}

// SetModeSelector sets the mode selector for automatic mode switching
func (l *Loop) SetModeSelector(selector *eino_adapter.ModeSelector) {
	l.selector = selector
}

// GetPlanAgent returns the plan-execute agent
func (l *Loop) GetPlanAgent() *eino_adapter.PlanExecuteAgent {
	return l.planAgent
}

// GetModeSelector returns the mode selector
func (l *Loop) GetModeSelector() *eino_adapter.ModeSelector {
	return l.selector
}

// GetADKAgent returns the ADK agent
func (l *Loop) GetADKAgent() *adk.ChatModelAgent {
	return l.adkAgent
}
