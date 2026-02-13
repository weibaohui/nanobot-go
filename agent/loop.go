package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/agent/tools"
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
	adkAgent *eino_adapter.ChatModelAgent

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
			toolNames = append(toolNames, t.Name())
		}
	}
	logger.Info("已注册工具",
		zap.Int("数量", len(registeredTools)),
		zap.Strings("工具列表", toolNames),
	)

	// 创建 ADK Agent
	ctx := context.Background()
	adkAgent, err := eino_adapter.NewChatModelAgent(ctx, &eino_adapter.ChatModelAgentConfig{
		Provider:      provider,
		Model:         model,
		Tools:         registeredTools,
		Logger:        logger,
		MaxIterations: maxIterations,
		EnableStream:  true,
	})
	if err != nil {
		logger.Error("创建 ADK Agent 失败，将使用回退模式", zap.Error(err))
	} else {
		loop.adkAgent = adkAgent
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
	l.tools.Register(&tools.ReadFileTool{AllowedDir: allowedDir})
	l.tools.Register(&tools.WriteFileTool{AllowedDir: allowedDir})
	l.tools.Register(&tools.EditFileTool{AllowedDir: allowedDir})
	l.tools.Register(&tools.ListDirTool{AllowedDir: allowedDir})

	// Shell 工具
	l.tools.Register(&tools.ExecTool{Timeout: l.execTimeout, WorkingDir: l.workspace, RestrictToWorkspace: l.restrictToWorkspace})

	// Web 工具
	l.tools.Register(&tools.WebSearchTool{MaxResults: 5})
	l.tools.Register(&tools.WebFetchTool{MaxChars: 50000})

	// 消息工具
	l.tools.Register(&MessageTool{SendCallback: func(msg any) error {
		if outMsg, ok := msg.(*bus.OutboundMessage); ok {
			l.bus.PublishOutbound(outMsg)
		}
		return nil
	}})

	// Spawn 工具
	l.tools.Register(&SpawnTool{Manager: l.subagents})

	// Cron 工具
	if l.cronService != nil {
		l.tools.Register(&CronTool{CronService: l.cronService})
	}
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
	if mt, ok := l.tools.Get("message").(*MessageTool); ok {
		mt.SetContext(channel, chatID)
	}
	if st, ok := l.tools.Get("spawn").(*SpawnTool); ok {
		st.SetContext(channel, chatID)
	}
	if ct, ok := l.tools.Get("cron").(*CronTool); ok {
		ct.SetContext(channel, chatID)
	}
}

// processWithADK 使用 ADK Agent 处理消息
func (l *Loop) processWithADK(ctx context.Context, msg *bus.InboundMessage) error {
	if l.adkAgent == nil {
		return fmt.Errorf("ADK Agent 未初始化")
	}

	// 获取会话历史
	sessionKey := msg.SessionKey()
	sess := l.sessions.GetOrCreate(sessionKey)
	history := l.convertHistory(sess.GetHistory(50))

	// 执行
	response, err := l.adkAgent.ExecuteWithHistory(ctx, msg.Content, history)
	if err != nil {
		return fmt.Errorf("ADK Agent 执行失败: %w", err)
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
	if l.adkAgent == nil {
		return fmt.Errorf("ADK Agent 未初始化")
	}

	// 获取会话历史
	sessionKey := msg.SessionKey()
	sess := l.sessions.GetOrCreate(sessionKey)
	history := l.convertHistory(sess.GetHistory(50))

	// 流式执行
	response, err := l.adkAgent.ExecuteStream(ctx, msg.Content, history, func(delta, fullContent string) error {
		// 发布流式片段
		l.bus.PublishStream(bus.NewStreamChunk(msg.Channel, msg.ChatID, delta, fullContent, false))
		return nil
	})
	if err != nil {
		return fmt.Errorf("流式执行失败: %w", err)
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

	if l.adkAgent == nil {
		return "", fmt.Errorf("ADK Agent 未初始化")
	}

	// 获取会话历史
	sess := l.sessions.GetOrCreate(sessionKey)
	history := l.convertHistory(sess.GetHistory(50))

	// 执行
	response, err := l.adkAgent.ExecuteWithHistory(ctx, content, history)
	if err != nil {
		return "", err
	}

	// 保存会话
	sess.AddMessage("user", content)
	sess.AddMessage("assistant", response)
	l.sessions.Save(sess)

	return response, nil
}

// GetTools returns all registered tools as a slice
func (l *Loop) GetTools() []tools.Tool {
	defs := l.tools.GetDefinitions()
	result := make([]tools.Tool, 0, len(defs))
	for _, def := range defs {
		name := ""
		if n, ok := def["name"].(string); ok {
			name = n
		} else if fn, ok := def["function"].(map[string]any); ok {
			if n, ok := fn["name"].(string); ok {
				name = n
			}
		}
		if name == "" {
			continue
		}
		if t := l.tools.Get(name); t != nil {
			result = append(result, t)
		}
	}
	return result
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
func (l *Loop) GetADKAgent() *eino_adapter.ChatModelAgent {
	return l.adkAgent
}
