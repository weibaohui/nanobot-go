package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/weibaohui/nanobot-go/bus"
	"github.com/weibaohui/nanobot-go/cron"
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
	braveAPIKey         string
	execTimeout         int
	restrictToWorkspace bool
	cronService         *cron.Service
	context             *ContextBuilder
	sessions            *session.Manager
	tools               *ToolRegistry
	subagents           *SubagentManager
	running             bool
	logger              *zap.Logger
}

// ToolRegistry 工具注册表（本地定义避免循环导入）
type ToolRegistry struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

// Tool 工具接口（本地定义）
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	Execute(ctx context.Context, params map[string]any) (string, error)
	ToSchema() map[string]any
}

// NewToolRegistry 创建工具注册表
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// Register 注册工具
func (r *ToolRegistry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
}

// Get 获取工具
func (r *ToolRegistry) Get(name string) Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tools[name]
}

// GetDefinitions 获取所有工具定义
func (r *ToolRegistry) GetDefinitions() []map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var defs []map[string]any
	for _, tool := range r.tools {
		defs = append(defs, tool.ToSchema())
	}
	return defs
}

// Execute 执行工具
func (r *ToolRegistry) Execute(ctx context.Context, name string, params map[string]any) (string, error) {
	r.mu.RLock()
	tool, ok := r.tools[name]
	r.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("工具 '%s' 不存在", name)
	}
	return tool.Execute(ctx, params)
}

// NewLoop 创建代理循环
func NewLoop(messageBus *bus.MessageBus, provider providers.LLMProvider, workspace string, model string, maxIterations int, braveAPIKey string, execTimeout int, restrictToWorkspace bool, cronService *cron.Service, sessionManager *session.Manager, logger *zap.Logger) *Loop {
	if logger == nil {
		logger = zap.NewNop()
	}

	loop := &Loop{
		bus:                 messageBus,
		provider:            provider,
		workspace:           workspace,
		model:               model,
		maxIterations:       maxIterations,
		braveAPIKey:         braveAPIKey,
		execTimeout:         execTimeout,
		restrictToWorkspace: restrictToWorkspace,
		cronService:         cronService,
		context:             NewContextBuilder(workspace),
		sessions:            sessionManager,
		tools:               NewToolRegistry(),
		logger:              logger,
	}

	// 创建子代理管理器
	loop.subagents = NewSubagentManager(provider, workspace, messageBus, model, braveAPIKey, execTimeout, restrictToWorkspace, logger)

	// 注册默认工具
	loop.registerDefaultTools()

	return loop
}

// registerDefaultTools 注册默认工具
func (l *Loop) registerDefaultTools() {
	allowedDir := ""
	if l.restrictToWorkspace {
		allowedDir = l.workspace
	}

	// 文件工具
	l.tools.Register(&ReadFileTool{AllowedDir: allowedDir})
	l.tools.Register(&WriteFileTool{AllowedDir: allowedDir})
	l.tools.Register(&EditFileTool{AllowedDir: allowedDir})
	l.tools.Register(&ListDirTool{AllowedDir: allowedDir})

	// Shell 工具
	l.tools.Register(&ExecTool{Timeout: l.execTimeout, WorkingDir: l.workspace, RestrictToWorkspace: l.restrictToWorkspace})

	// Web 工具
	l.tools.Register(&WebSearchTool{APIKey: l.braveAPIKey, MaxResults: 5})
	l.tools.Register(&WebFetchTool{MaxChars: 50000})

	// 消息工具
	l.tools.Register(&MessageTool{SendCallback: func(msg *bus.OutboundMessage) error {
		l.bus.PublishOutbound(msg)
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
		response, err := l.processMessage(ctx, msg)
		if err != nil {
			l.logger.Error("处理消息失败", zap.Error(err))
			// 发送错误响应
			l.bus.PublishOutbound(bus.NewOutboundMessage(msg.Channel, msg.ChatID, fmt.Sprintf("抱歉，我遇到了错误: %s", err)))
			continue
		}

		if response != nil {
			l.bus.PublishOutbound(response)
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
func (l *Loop) processMessage(ctx context.Context, msg *bus.InboundMessage) (*bus.OutboundMessage, error) {
	// 处理系统消息（子代理公告）
	if msg.Channel == "system" {
		return l.processSystemMessage(ctx, msg)
	}

	preview := msg.Content
	if len(preview) > 80 {
		preview = preview[:80] + "..."
	}
	l.logger.Info("处理消息",
		zap.String("渠道", msg.Channel),
		zap.String("发送者", msg.SenderID),
		zap.String("内容", preview),
	)

	// 获取或创建会话
	sessionKey := msg.SessionKey()
	sess := l.sessions.GetOrCreate(sessionKey)

	// 更新工具上下文
	if mt, ok := l.tools.Get("message").(*MessageTool); ok {
		mt.SetContext(msg.Channel, msg.ChatID)
	}
	if st, ok := l.tools.Get("spawn").(*SpawnTool); ok {
		st.SetContext(msg.Channel, msg.ChatID)
	}
	if ct, ok := l.tools.Get("cron").(*CronTool); ok {
		ct.SetContext(msg.Channel, msg.ChatID)
	}

	// 构建消息
	messages := l.context.BuildMessages(sess.GetHistory(50), msg.Content, nil, msg.Media, msg.Channel, msg.ChatID)

	// 代理循环
	var finalContent string

	for i := 0; i < l.maxIterations; i++ {
		response, err := l.provider.Chat(ctx, messages, l.tools.GetDefinitions(), l.model, 4096, 0.7)
		if err != nil {
			return nil, fmt.Errorf("LLM 调用失败: %w", err)
		}

		if response.HasToolCalls() {
			// 添加助手消息
			toolCallDicts := l.buildToolCallDicts(response.ToolCalls)
			messages = l.context.AddAssistantMessage(messages, response.Content, toolCallDicts, response.ReasoningContent)

			// 执行工具
			for _, tc := range response.ToolCalls {
				argsJSON, _ := json.Marshal(tc.Arguments)
				l.logger.Info("工具调用",
					zap.String("工具", tc.Name),
					zap.String("参数", string(argsJSON)),
				)

				result, err := l.tools.Execute(ctx, tc.Name, tc.Arguments)
				if err != nil {
					result = fmt.Sprintf("错误: %s", err)
				}
				messages = l.context.AddToolResult(messages, tc.ID, tc.Name, result)
			}

			// 添加反思提示
			messages = append(messages, map[string]any{
				"role":    "user",
				"content": "反思结果并决定下一步。",
			})
		} else {
			finalContent = response.Content
			break
		}
	}

	if finalContent == "" {
		finalContent = "我已完成处理但没有响应内容。"
	}

	// 记录响应预览
	preview = finalContent
	if len(preview) > 120 {
		preview = preview[:120] + "..."
	}
	l.logger.Info("响应",
		zap.String("渠道", msg.Channel),
		zap.String("发送者", msg.SenderID),
		zap.String("内容", preview),
	)

	// 保存会话
	sess.AddMessage("user", msg.Content)
	sess.AddMessage("assistant", finalContent)
	l.sessions.Save(sess)

	return bus.NewOutboundMessage(msg.Channel, msg.ChatID, finalContent), nil
}

// processSystemMessage 处理系统消息
func (l *Loop) processSystemMessage(ctx context.Context, msg *bus.InboundMessage) (*bus.OutboundMessage, error) {
	l.logger.Info("处理系统消息", zap.String("发送者", msg.SenderID))

	// 解析来源
	var originChannel, originChatID string
	if idx := len(msg.ChatID) - len(msg.ChatID) - 1; idx > 0 {
		// 查找冒号位置
		for i, c := range msg.ChatID {
			if c == ':' {
				originChannel = msg.ChatID[:i]
				originChatID = msg.ChatID[i+1:]
				break
			}
		}
	}
	if originChannel == "" {
		originChannel = "cli"
		originChatID = msg.ChatID
	}

	// 使用来源会话
	sessionKey := originChannel + ":" + originChatID
	sess := l.sessions.GetOrCreate(sessionKey)

	// 更新工具上下文
	if mt, ok := l.tools.Get("message").(*MessageTool); ok {
		mt.SetContext(originChannel, originChatID)
	}
	if st, ok := l.tools.Get("spawn").(*SpawnTool); ok {
		st.SetContext(originChannel, originChatID)
	}
	if ct, ok := l.tools.Get("cron").(*CronTool); ok {
		ct.SetContext(originChannel, originChatID)
	}

	// 构建消息
	messages := l.context.BuildMessages(sess.GetHistory(50), msg.Content, nil, nil, originChannel, originChatID)

	// 代理循环
	var finalContent string

	for i := 0; i < l.maxIterations; i++ {
		response, err := l.provider.Chat(ctx, messages, l.tools.GetDefinitions(), l.model, 4096, 0.7)
		if err != nil {
			return nil, fmt.Errorf("LLM 调用失败: %w", err)
		}

		if response.HasToolCalls() {
			toolCallDicts := l.buildToolCallDicts(response.ToolCalls)
			messages = l.context.AddAssistantMessage(messages, response.Content, toolCallDicts, response.ReasoningContent)

			for _, tc := range response.ToolCalls {
				result, err := l.tools.Execute(ctx, tc.Name, tc.Arguments)
				if err != nil {
					result = fmt.Sprintf("错误: %s", err)
				}
				messages = l.context.AddToolResult(messages, tc.ID, tc.Name, result)
			}

			messages = append(messages, map[string]any{
				"role":    "user",
				"content": "反思结果并决定下一步。",
			})
		} else {
			finalContent = response.Content
			break
		}
	}

	if finalContent == "" {
		finalContent = "后台任务已完成。"
	}

	// 保存会话
	sess.AddMessage("user", fmt.Sprintf("[系统: %s] %s", msg.SenderID, msg.Content))
	sess.AddMessage("assistant", finalContent)
	l.sessions.Save(sess)

	return bus.NewOutboundMessage(originChannel, originChatID, finalContent), nil
}

// ProcessDirect 直接处理消息（用于 CLI 或 cron）
func (l *Loop) ProcessDirect(ctx context.Context, content, sessionKey, channel, chatID string) (string, error) {
	msg := bus.NewInboundMessage(channel, "user", chatID, content)
	response, err := l.processMessage(ctx, msg)
	if err != nil {
		return "", err
	}
	if response == nil {
		return "", nil
	}
	return response.Content, nil
}

// buildToolCallDicts 构建工具调用字典
func (l *Loop) buildToolCallDicts(toolCalls []providers.ToolCallRequest) []map[string]any {
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
