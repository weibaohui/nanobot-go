package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/supervisor"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/bus"
	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/session"
	"go.uber.org/zap"
)

// SupervisorAgent 监督者 Agent
// 作为统一入口，根据用户输入自动路由到合适的子 Agent
type SupervisorAgent struct {
	*interruptible

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

	// 先创建 interruptible
	interruptible, err := newInterruptible(ctx, &interruptibleConfig{
		Cfg:             cfg.Cfg,
		Workspace:       cfg.Workspace,
		Tools:           cfg.Tools,
		Logger:          logger,
		Sessions:        cfg.Sessions,
		Bus:             cfg.Bus,
		Context:         cfg.Context,
		InterruptMgr:    cfg.InterruptMgr,
		CheckpointStore: cfg.CheckpointStore,
		MaxIterations:   cfg.MaxIterations,
		RegisteredTools: cfg.RegisteredTools,
		AgentType:       "supervisor",
	})
	if err != nil {
		return nil, err
	}

	// 初始化 SupervisorAgent
	sa := &SupervisorAgent{
		interruptible: interruptible,
		cfg:           cfg.Cfg,
		workspace:     cfg.Workspace,
		tools:         cfg.Tools,
		logger:        logger,
		sessions:      cfg.Sessions,
		bus:           cfg.Bus,
		context:       cfg.Context,
	}

	// 初始化子 Agent
	if err := sa.initSubAgents(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSubAgentCreate, err)
	}

	if err := sa.initSupervisor(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSupervisorInit, err)
	}

	// 设置 ADK Runner 到 interruptible
	interruptible.adkRunner = sa.adkRunner

	logger.Info("Supervisor Agent 创建成功",
		zap.String("model", cfg.Context.workspace),
		zap.Int("max_iterations", cfg.MaxIterations),
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
	llm, err := sa.interruptible.BuildChatModelAdapter()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrChatModelAdapter, err)
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
		Model:       llm,
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
	return sa.interruptible.Process(ctx, msg, sa.buildMessages)
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
