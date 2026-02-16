package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/supervisor"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/agent/tools/askuser"
	"github.com/weibaohui/nanobot-go/bus"
	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/session"
	"go.uber.org/zap"
)

// SupervisorAgent 监督者 Agent
// 作为统一入口，根据用户输入自动路由到合适的子 Agent
type SupervisorAgent struct {
	cfg       *config.Config
	workspace string
	tools     []tool.BaseTool
	logger    *zap.Logger
	sessions  *session.Manager
	bus       *bus.MessageBus
	context   *ContextBuilder

	reactAgent SubAgent
	planAgent  SubAgent
	chatAgent  SubAgent

	adkSupervisor adk.Agent
	adkRunner     *adk.Runner

	interruptManager *InterruptManager
	checkpointStore  compose.CheckPointStore
	maxIterations    int
	registeredTools  []string
}

// SupervisorConfig Supervisor 配置
type SupervisorConfig struct {
	Cfg             *config.Config
	Workspace       string
	Tools           []tool.BaseTool
	Logger          *zap.Logger
	Sessions        *session.Manager
	Bus             *bus.MessageBus
	Context         *ContextBuilder // 上下文构建器
	InterruptMgr    *InterruptManager
	CheckpointStore compose.CheckPointStore
	MaxIterations   int
	// 已注册的工具名称列表
	RegisteredTools []string
}

// NewSupervisorAgent 创建 Supervisor Agent
func NewSupervisorAgent(ctx context.Context, cfg *SupervisorConfig) (*SupervisorAgent, error) {
	if cfg == nil {
		return nil, ErrConfigNil
	}

	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	maxIter := cfg.MaxIterations
	if maxIter <= 0 {
		maxIter = 10
	}

	sa := &SupervisorAgent{
		cfg:              cfg.Cfg,
		workspace:        cfg.Workspace,
		tools:            cfg.Tools,
		logger:           logger,
		sessions:         cfg.Sessions,
		bus:              cfg.Bus,
		context:          cfg.Context,
		interruptManager: cfg.InterruptMgr,
		checkpointStore:  cfg.CheckpointStore,
		maxIterations:    maxIter,
		registeredTools:  cfg.RegisteredTools,
	}

	if err := sa.initSubAgents(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSubAgentCreate, err)
	}

	if err := sa.initSupervisor(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSupervisorInit, err)
	}

	logger.Info("Supervisor Agent 创建成功",
		zap.String("model", cfg.Context.workspace),
		zap.Int("max_iterations", maxIter),
	)

	return sa, nil
}

// initSubAgents 创建子 Agent
func (sa *SupervisorAgent) initSubAgents(ctx context.Context) error {
	var err error

	var skillsLoader func(skillName string) string
	if sa.context != nil {
		skillsLoader = sa.context.GetSkillsLoader().LoadSkill
	}

	sa.reactAgent, err = NewReActSubAgent(ctx, &ReActConfig{
		Cfg:             sa.cfg,
		Workspace:       sa.workspace,
		Tools:           sa.tools,
		Logger:          sa.logger,
		Sessions:        sa.sessions,
		CheckpointStore: sa.checkpointStore,
		MaxIterations:   sa.maxIterations,
		SkillsLoader:    skillsLoader,
		RegisteredTools: sa.registeredTools,
	})
	if err != nil {
		return fmt.Errorf("创建 ReAct Agent 失败: %w", err)
	}

	sa.planAgent, err = NewPlanSubAgent(ctx, &PlanConfig{
		Cfg:             sa.cfg,
		Workspace:       sa.workspace,
		Tools:           sa.tools,
		Logger:          sa.logger,
		Sessions:        sa.sessions,
		CheckpointStore: sa.checkpointStore,
		MaxIterations:   sa.maxIterations,
		SkillsLoader:    skillsLoader,
		RegisteredTools: sa.registeredTools,
	})
	if err != nil {
		return fmt.Errorf("创建 Plan Agent 失败: %w", err)
	}

	// 创建 Chat Agent
	sa.chatAgent, err = NewChatSubAgent(ctx, &ChatConfig{
		Cfg:             sa.cfg,
		Tools:           sa.tools,
		Logger:          sa.logger,
		Sessions:        sa.sessions,
		CheckpointStore: sa.checkpointStore,
		SkillsLoader:    skillsLoader,
		RegisteredTools: sa.registeredTools,
	})
	if err != nil {
		return fmt.Errorf("创建 Chat Agent 失败: %w", err)
	}

	return nil
}

// initSupervisor 创建 ADK Supervisor
func (sa *SupervisorAgent) initSupervisor(ctx context.Context) error {
	adapter, err := NewChatModelAdapter(sa.logger, sa.cfg, sa.sessions)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrChatModelAdapter, err)
	}
	if sa.context != nil {
		adapter.SetSkillLoader(sa.context.GetSkillsLoader().LoadSkill)
	}
	if len(sa.registeredTools) > 0 {
		adapter.SetRegisteredTools(sa.registeredTools)
	}

	var toolsConfig adk.ToolsConfig
	supervisorTools := filterToolsByNames(sa.tools, map[string]bool{"ask_user": true})
	if len(supervisorTools) > 0 {
		toolsConfig = adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: supervisorTools,
			},
		}
	}

	svAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "supervisor",
		Description: "统一入口 Agent，负责路由用户请求到合适的子 Agent",
		Instruction: sa.buildSupervisorInstruction(),
		Model:       adapter,
		ToolsConfig: toolsConfig,
		Exit:        &adk.ExitTool{},
	})
	if err != nil {
		return fmt.Errorf("%w: %w", ErrAgentCreate, err)
	}

	sv, err := supervisor.New(ctx, &supervisor.Config{
		Supervisor: svAgent,
		SubAgents: []adk.Agent{
			sa.reactAgent.GetADKAgent(),
			sa.planAgent.GetADKAgent(),
			sa.chatAgent.GetADKAgent(),
		},
	})
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSupervisorCreate, err)
	}

	sa.adkSupervisor = sv

	sa.adkRunner = adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           sv,
		CheckPointStore: sa.checkpointStore,
	})

	return nil
}

func filterToolsByNames(tools []tool.BaseTool, allowed map[string]bool) []tool.BaseTool {
	if len(allowed) == 0 {
		return nil
	}
	ctx := context.Background()
	result := make([]tool.BaseTool, 0, len(allowed))
	for _, t := range tools {
		if t == nil {
			continue
		}
		info, err := t.Info(ctx)
		if err != nil || info == nil || info.Name == "" {
			continue
		}
		if allowed[info.Name] {
			result = append(result, t)
		}
	}
	return result
}

// buildSupervisorInstruction 构建 Supervisor 指令
func (sa *SupervisorAgent) buildSupervisorInstruction() string {
	return `你是 nanobot 的路由器 Agent，负责将用户请求路由到最合适的子 Agent。

## 你的唯一职责
分析用户请求，并调用 transfer_to_agent 工具将任务委托给最合适的子 Agent。

## 可用的子 Agent

### react_agent
- 适合：需要工具调用、文件操作、网络搜索、执行命令等任务
- 能力：读取/写入文件、执行 shell 命令、搜索网络、使用技能

### plan_agent
- 适合：需要规划的任务（如旅行规划、项目规划、多步骤复杂任务）
- 能力：规划 → 执行 → 重规划的闭环

### chat_agent
- 适合：简单闲聊、快速问答、信息查询
- 能力：轻量级对话，不使用工具

## 路由决策规则

1. **优先检查是否需要 plan_agent**：
   - 包含"规划"、"计划"、"帮我完成"、"如何做"等关键词
   - 多步骤复杂任务
   - 需要目标分解的任务

2. **检查是否需要 react_agent**：
   - 包含"读取"、"写入"、"执行"、"搜索"、"文件"等关键词
   - 需要操作文件或系统
   - 需要使用技能

3. **默认使用 chat_agent**：
   - 简单问候、闲聊
   - 快速问答
   - 不需要工具调用的请求

## 重要说明
- 你必须调用 transfer_to_agent 工具
- 调用时指定 agent_name 参数（react_agent、plan_agent 或 chat_agent）
- 不要输出其他文本，只输出工具调用`
}

// Process 处理用户消息
func (sa *SupervisorAgent) Process(ctx context.Context, msg *bus.InboundMessage) (string, error) {
	sessionKey := msg.SessionKey()
	sess := sa.sessions.GetOrCreate(sessionKey)

	// 将 session key 放入 context，用于记录 token 用量
	ctx = context.WithValue(ctx, SessionKeyContextKey, sessionKey)

	// 构建消息
	history := sa.convertHistory(sess.GetHistory(0))
	messages := sa.buildMessages(history, msg.Content, msg.Channel, msg.ChatID)

	// 生成 checkpoint ID
	checkpointID := fmt.Sprintf("%s_%d", sessionKey, time.Now().UnixNano())

	// 执行
	var response string
	var err error

	response, err = sa.processNormal(ctx, messages, checkpointID, msg)

	if err != nil {
		// 检查是否是中断
		if strings.HasPrefix(err.Error(), "INTERRUPT:") {
			return "", err
		}
		return "", err
	}

	// 保存会话
	sess.AddMessage("user", msg.Content)
	sess.AddMessage("assistant", response)
	sa.sessions.Save(sess)

	return response, nil
}

// processNormal 普通模式处理
func (sa *SupervisorAgent) processNormal(ctx context.Context, messages []*schema.Message, checkpointID string, msg *bus.InboundMessage) (string, error) {
	iter := sa.adkRunner.Run(ctx, messages, adk.WithCheckPointID(checkpointID))

	var response string
	var lastEvent *adk.AgentEvent

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return "", fmt.Errorf("Supervisor 执行失败: %w", event.Err)
		}

		if event.Output != nil && event.Output.MessageOutput != nil {
			msgOutput, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				continue
			}
			response = msgOutput.Content
		}

		lastEvent = event
	}

	// 检查中断
	if lastEvent != nil && lastEvent.Action != nil && lastEvent.Action.Interrupted != nil {
		return "", sa.handleInterrupt(msg, checkpointID, lastEvent)
	}

	return response, nil
}

// handleInterrupt 处理中断
func (sa *SupervisorAgent) handleInterrupt(msg *bus.InboundMessage, checkpointID string, event *adk.AgentEvent) error {
	if event.Action == nil || event.Action.Interrupted == nil {
		return nil
	}

	interruptCtx := event.Action.Interrupted.InterruptContexts[0]
	interruptID := interruptCtx.ID

	// 解析中断信息
	var question string
	var options []string
	isAskUser := false

	if info, ok := interruptCtx.Info.(*askuser.AskUserInfo); ok {
		question = info.Question
		options = append(options, info.Options...)
		isAskUser = true
	} else if info, ok := interruptCtx.Info.(map[string]any); ok {
		if q, ok := info["question"].(string); ok {
			question = q
		}
		if opts, ok := info["options"].([]any); ok {
			for _, opt := range opts {
				if s, ok := opt.(string); ok {
					options = append(options, s)
				}
			}
		}
		if question != "" {
			isAskUser = true
		}
	} else {
		question = fmt.Sprintf("%v", interruptCtx.Info)
	}

	// 发送中断请求
	sa.interruptManager.HandleInterrupt(&InterruptInfo{
		CheckpointID: checkpointID,
		InterruptID:  interruptID,
		Channel:      msg.Channel,
		ChatID:       msg.ChatID,
		Question:     question,
		Options:      options,
		SessionKey:   msg.SessionKey(),
		IsAskUser:    isAskUser,
		IsSupervisor: true, // 标记来自 Supervisor 模式的中断
	})

	sa.logger.Info("等待用户输入以恢复执行",
		zap.String("checkpoint_id", checkpointID),
		zap.String("question", question),
	)

	return fmt.Errorf("INTERRUPT:%s:%s", checkpointID, interruptID)
}

// buildMessages 构建消息列表
func (sa *SupervisorAgent) buildMessages(history []*schema.Message, userInput, channel, chatID string) []*schema.Message {
	// 构建系统提示
	systemPrompt := sa.buildSystemPrompt()
	// 复用公共方法构建消息列表
	return BuildMessageList(systemPrompt, history, userInput, channel, chatID)
}

// buildSystemPrompt 构建系统提示
// 不再添加额外的 Supervisor 角色说明，避免与 nanobot 身义冲突
// Supervisor 的路由逻辑已经在 buildSupervisorInstruction 中清晰定义
func (sa *SupervisorAgent) buildSystemPrompt() string {
	// Supervisor 不需要基础系统提示词
	// 它的职责纯粹是路由，不需要 nanobot 的身份、技能等信息
	return ""
}

// convertHistory 转换会话历史
func (sa *SupervisorAgent) convertHistory(history []map[string]any) []*schema.Message {
	result := make([]*schema.Message, 0, len(history))
	for _, h := range history {
		role := schema.User
		if r, ok := h["role"].(string); ok && r == "assistant" {
			role = schema.Assistant
		}

		content, _ := h["content"].(string)
		result = append(result, &schema.Message{
			Role:    role,
			Content: content,
		})
	}
	return result
}

// GetSubAgents 获取子 Agent
func (sa *SupervisorAgent) GetSubAgents() map[AgentType]SubAgent {
	return map[AgentType]SubAgent{
		AgentTypeReAct: sa.reactAgent,
		AgentTypePlan:  sa.planAgent,
		AgentTypeChat:  sa.chatAgent,
	}
}

// GetADKRunner 获取 ADK Runner
func (sa *SupervisorAgent) GetADKRunner() *adk.Runner {
	return sa.adkRunner
}

// Resume 恢复被中断的执行
// 用于处理 Supervisor 模式下的中断恢复
func (sa *SupervisorAgent) Resume(ctx context.Context, checkpointID string, resumeParams *adk.ResumeParams, msg *bus.InboundMessage) (string, error) {
	if sa.adkRunner == nil {
		return "", fmt.Errorf("ADK Runner 未初始化")
	}

	// 使用 Supervisor 的 Runner 恢复执行
	iter, err := sa.adkRunner.ResumeWithParams(ctx, checkpointID, resumeParams)
	if err != nil {
		return "", fmt.Errorf("Supervisor 恢复执行失败: %w", err)
	}

	var response string
	var lastEvent *adk.AgentEvent

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return "", fmt.Errorf("Supervisor 恢复后执行失败: %w", event.Err)
		}

		if event.Output != nil && event.Output.MessageOutput != nil {
			msgOutput, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				continue
			}
			response = msgOutput.Content
		}

		lastEvent = event
	}

	// 检查是否再次被中断
	if lastEvent != nil && lastEvent.Action != nil && lastEvent.Action.Interrupted != nil {
		// 生成新的 checkpoint ID
		newCheckpointID := fmt.Sprintf("%s_resume_%d", checkpointID, time.Now().UnixNano())
		return "", sa.handleInterrupt(msg, newCheckpointID, lastEvent)
	}

	return response, nil
}

func buildToolsConfig(tools []tool.BaseTool) adk.ToolsConfig {
	if len(tools) == 0 {
		return adk.ToolsConfig{}
	}
	return adk.ToolsConfig{
		ToolsNodeConfig: compose.ToolsNodeConfig{
			Tools: tools,
		},
	}
}
