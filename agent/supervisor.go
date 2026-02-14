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
	"github.com/weibaohui/nanobot-go/bus"
	"github.com/weibaohui/nanobot-go/eino_adapter"
	"github.com/weibaohui/nanobot-go/providers"
	"github.com/weibaohui/nanobot-go/session"
	"go.uber.org/zap"
)

// SubAgent 子 Agent 接口
type SubAgent interface {
	// Name 返回 Agent 名称
	Name() string
	// Description 返回 Agent 描述
	Description() string
	// Type 返回 Agent 类型
	Type() AgentType
	// GetADKAgent 返回底层的 ADK Agent
	GetADKAgent() adk.Agent
}

// SupervisorAgent 监督者 Agent
// 作为统一入口，根据用户输入自动路由到合适的子 Agent
type SupervisorAgent struct {
	provider  providers.LLMProvider
	model     string
	workspace string
	tools     []tool.BaseTool
	logger    *zap.Logger
	router    *Router
	sessions  *session.Manager
	bus       *bus.MessageBus
	context   *ContextBuilder // 上下文构建器，用于复用基础系统提示词

	// 子 Agent
	reactAgent SubAgent
	planAgent  SubAgent
	chatAgent  SubAgent

	// ADK Supervisor
	adkSupervisor adk.Agent
	adkRunner     *adk.Runner

	// 中断管理
	interruptManager *InterruptManager
	checkpointStore  compose.CheckPointStore

	// 配置
	maxIterations   int
	enableStream    bool
	registeredTools []string // 已注册的工具名称列表
}

// SupervisorConfig Supervisor 配置
type SupervisorConfig struct {
	Provider        providers.LLMProvider
	Model           string
	Workspace       string
	Tools           []tool.BaseTool
	Logger          *zap.Logger
	Sessions        *session.Manager
	Bus             *bus.MessageBus
	Context         *ContextBuilder // 上下文构建器
	InterruptMgr    *InterruptManager
	CheckpointStore compose.CheckPointStore
	MaxIterations   int
	EnableStream    bool
	// 已注册的工具名称列表
	RegisteredTools []string
}

// NewSupervisorAgent 创建 Supervisor Agent
func NewSupervisorAgent(ctx context.Context, cfg *SupervisorConfig) (*SupervisorAgent, error) {
	if cfg == nil {
		return nil, fmt.Errorf("配置不能为空")
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
		provider:         cfg.Provider,
		model:            cfg.Model,
		workspace:        cfg.Workspace,
		tools:            cfg.Tools,
		logger:           logger,
		router:           NewRouter(&RouterConfig{Logger: logger}),
		sessions:         cfg.Sessions,
		bus:              cfg.Bus,
		context:          cfg.Context,
		interruptManager: cfg.InterruptMgr,
		checkpointStore:  cfg.CheckpointStore,
		maxIterations:    maxIter,
		enableStream:     cfg.EnableStream,
		registeredTools:  cfg.RegisteredTools,
	}

	// 创建子 Agent
	if err := sa.createSubAgents(ctx); err != nil {
		return nil, fmt.Errorf("创建子 Agent 失败: %w", err)
	}

	// 创建 ADK Supervisor
	if err := sa.createADKSupervisor(ctx); err != nil {
		return nil, fmt.Errorf("创建 ADK Supervisor 失败: %w", err)
	}

	logger.Info("Supervisor Agent 创建成功",
		zap.String("model", cfg.Model),
		zap.Int("max_iterations", maxIter),
	)

	return sa, nil
}

// createSubAgents 创建子 Agent
func (sa *SupervisorAgent) createSubAgents(ctx context.Context) error {
	var err error

	// 获取技能加载器
	var skillsLoader func(skillName string) string
	if sa.context != nil {
		skillsLoader = sa.context.GetSkillsLoader().LoadSkill
	}

	// 创建 ReAct Agent
	sa.reactAgent, err = NewReActSubAgent(ctx, &ReActConfig{
		Provider:        sa.provider,
		Model:           sa.model,
		Workspace:       sa.workspace,
		Tools:           sa.tools,
		Logger:          sa.logger,
		CheckpointStore: sa.checkpointStore,
		MaxIterations:   sa.maxIterations,
		SkillsLoader:    skillsLoader,
		RegisteredTools: sa.registeredTools,
	})
	if err != nil {
		return fmt.Errorf("创建 ReAct Agent 失败: %w", err)
	}
	// sa.logger.Info("SupervisorAgent createSubAgents tools数量", zap.Int("tools_count", len(sa.tools)))
	// 创建 Plan Agent
	sa.planAgent, err = NewPlanSubAgent(ctx, &PlanConfig{
		Provider:        sa.provider,
		Model:           sa.model,
		Workspace:       sa.workspace,
		Tools:           sa.tools,
		Logger:          sa.logger,
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
		Provider:        sa.provider,
		Model:           sa.model,
		Tools:           sa.tools,
		Logger:          sa.logger,
		CheckpointStore: sa.checkpointStore,
		SkillsLoader:    skillsLoader,
		RegisteredTools: sa.registeredTools,
	})
	if err != nil {
		return fmt.Errorf("创建 Chat Agent 失败: %w", err)
	}

	return nil
}

// createADKSupervisor 创建 ADK Supervisor
func (sa *SupervisorAgent) createADKSupervisor(ctx context.Context) error {
	// 创建 Supervisor 的 ChatModel
	adapter := eino_adapter.NewProviderAdapter(sa.logger, sa.provider, sa.model)

	// 创建 Supervisor Agent
	svAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "supervisor",
		Description: "统一入口 Agent，负责路由用户请求到合适的子 Agent",
		Instruction: sa.buildSupervisorInstruction(),
		Model:       adapter,
		Exit:        &adk.ExitTool{},
	})
	if err != nil {
		return fmt.Errorf("创建 Supervisor Agent 失败: %w", err)
	}

	// 使用 supervisor prebuilt 创建编排
	sv, err := supervisor.New(ctx, &supervisor.Config{
		Supervisor: svAgent,
		SubAgents: []adk.Agent{
			sa.reactAgent.GetADKAgent(),
			sa.planAgent.GetADKAgent(),
			sa.chatAgent.GetADKAgent(),
		},
	})
	if err != nil {
		return fmt.Errorf("创建 Supervisor 编排失败: %w", err)
	}

	sa.adkSupervisor = sv

	// 创建 Runner
	sa.adkRunner = adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           sv,
		EnableStreaming: sa.enableStream,
		CheckPointStore: sa.checkpointStore,
	})

	return nil
}

// buildSupervisorInstruction 构建 Supervisor 指令
func (sa *SupervisorAgent) buildSupervisorInstruction() string {
	return `你是 nanobot 的统一入口 Agent，负责分析用户请求并路由到最合适的子 Agent。

## 可用的子 Agent（仅限以下三个）

### 1. react_agent (ReAct Agent)
- 用途：工具调用、推理、长对话
- 适用场景：
  - 需要读取、写入、编辑文件
  - 需要执行 shell 命令
  - 需要搜索网络或获取网页
  - 需要使用技能（如 weather、translate 等）
  - 需要多步推理的复杂问题
- 特点：ReAct 模式（推理 → 行动 → 观察 → 再推理）

### 2. plan_execute_replan (Plan-Execute-Replan Agent)
- 用途：复杂任务的规划与执行
- 适用场景：
  - 需要规划的任务（如旅行规划、项目规划）
  - 多步骤复杂任务
  - 需要分步骤执行的任务
  - 执行过程中可能需要调整计划
- 特点：规划 → 执行 → 重规划的闭环

### 3. chat_agent (Chat Agent)
- 用途：简单对话和问答
- 适用场景：
  - 简单闲聊
  - 快速问答
  - 信息查询
  - 不需要工具调用的简单请求
- 特点：轻量级，快速响应

## 重要说明

1. **只有上述三个子 Agent**：不要尝试转移任务到其他名称的 agent
2. **技能（Skills）不是 Agent**：weather、translate 等是 react_agent 可以使用的技能，不是独立的 agent
3. **当用户请求涉及技能时**：应该转移给 react_agent，而不是尝试转移给不存在的 agent

## 路由决策规则

1. **优先检查是否需要 Plan Agent**：
   - 包含"规划"、"计划"、"帮我完成"等关键词
   - 多步骤复杂任务
   - 需要目标分解的任务

2. **检查是否需要 ReAct Agent**：
   - 包含文件操作关键词（读取、写入、编辑等）
   - 包含网络操作关键词（搜索、获取网页等）
   - 包含系统操作关键词（执行、运行命令等）
   - 需要使用技能（weather、translate 等）

3. **默认使用 Chat Agent**：
   - 简单问候
   - 快速问答
   - 不需要工具调用的请求

## 转移规则

- 一次只调用一个子 Agent
- 只能转移到：react_agent、plan_execute_replan、chat_agent
- 不要自己执行任务，总是委托给子 Agent
- 子 Agent 完成后，汇总结果返回给用户
- 如果任务需要用户确认，子 Agent 会处理中断
`
}

// Process 处理用户消息
func (sa *SupervisorAgent) Process(ctx context.Context, msg *bus.InboundMessage) (string, error) {
	sessionKey := msg.SessionKey()
	sess := sa.sessions.GetOrCreate(sessionKey)

	// 路由决策
	agentType := sa.router.Route(ctx, msg.Content)
	sa.logger.Info("路由决策完成",
		zap.String("agent_type", string(agentType)),
		zap.String("session_key", sessionKey),
	)

	// 构建消息
	history := sa.convertHistory(sess.GetHistory(10))
	messages := sa.buildMessages(history, msg.Content, msg.Channel, msg.ChatID)

	// 生成 checkpoint ID
	checkpointID := fmt.Sprintf("%s_%d", sessionKey, time.Now().UnixNano())

	// 执行
	var response string
	var err error

	if sa.enableStream {
		response, err = sa.processWithStream(ctx, messages, checkpointID, msg)
	} else {
		response, err = sa.processNormal(ctx, messages, checkpointID, msg)
	}

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

// processWithStream 流式模式处理
func (sa *SupervisorAgent) processWithStream(ctx context.Context, messages []*schema.Message, checkpointID string, msg *bus.InboundMessage) (string, error) {
	iter := sa.adkRunner.Run(ctx, messages, adk.WithCheckPointID(checkpointID))

	var response string
	var fullContent string
	var lastEvent *adk.AgentEvent

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return "", fmt.Errorf("流式执行失败: %w", event.Err)
		}

		if event.Output != nil && event.Output.MessageOutput != nil {
			msgOutput, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				continue
			}

			// 计算增量内容
			delta := ""
			if fullContent != "" && strings.HasPrefix(msgOutput.Content, fullContent) {
				delta = msgOutput.Content[len(fullContent):]
			} else {
				delta = msgOutput.Content
			}

			if delta != "" {
				sa.bus.PublishStream(bus.NewStreamChunk(msg.Channel, msg.ChatID, delta, msgOutput.Content, false))
			}

			fullContent = msgOutput.Content
			response = msgOutput.Content
		}

		lastEvent = event
	}

	// 检查中断
	if lastEvent != nil && lastEvent.Action != nil && lastEvent.Action.Interrupted != nil {
		return "", sa.handleInterrupt(msg, checkpointID, lastEvent)
	}

	// 发送完成标记
	sa.bus.PublishStream(bus.NewStreamChunk(msg.Channel, msg.ChatID, "", response, true))

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

	if info, ok := interruptCtx.Info.(map[string]any); ok {
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
// 复用基础系统提示词，并添加 Supervisor 特有的角色说明
func (sa *SupervisorAgent) buildSystemPrompt() string {
	// 获取基础系统提示词（包含身份、时间、环境、工作区、内存、技能等）
	var basePrompt string
	if sa.context != nil {
		basePrompt = sa.context.BuildSystemPrompt(nil)
	}

	// Supervisor 特有的角色说明
	supervisorPrompt := `# Supervisor 角色

你是 nanobot 的统一入口 Agent。你的职责是分析用户请求，并将其路由到最合适的子 Agent。

## 工作流程
1. 分析用户请求的类型和复杂度
2. 选择最合适的子 Agent
3. 委托任务给子 Agent
4. 汇总结果返回给用户

记住：不要自己执行具体任务，总是委托给专业的子 Agent。`

	// 组合提示词
	if basePrompt != "" {
		return basePrompt + "\n\n---\n\n" + supervisorPrompt
	}
	return supervisorPrompt
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

// GetRouter 获取路由器
func (sa *SupervisorAgent) GetRouter() *Router {
	return sa.router
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
