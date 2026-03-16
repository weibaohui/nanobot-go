package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks"
	"github.com/weibaohui/nanobot-go/pkg/agent/interrupt"
	"github.com/weibaohui/nanobot-go/pkg/bus"
	"github.com/weibaohui/nanobot-go/pkg/session"
	"go.uber.org/zap"
)

// MasterAgent 监督者 Agent
// 作为统一入口，根据用户输入自动路由到合适的子 Agent
type MasterAgent struct {
	*interruptible
	configLoader LLMConfigLoader
	workspace    string
	tools        []tool.BaseTool
	logger       *zap.Logger
	sessions     *session.Manager
	context      *ContextBuilder

	adkRunner *adk.Runner
}

// MasterAgentConfig Master 配置
type MasterAgentConfig struct {
	ConfigLoader   LLMConfigLoader
	Workspace      string
	Tools          []tool.BaseTool
	Logger         *zap.Logger
	Sessions       *session.Manager
	Bus            *bus.MessageBus
	Context        *ContextBuilder
	InterruptMgr   *interrupt.Manager
	CheckpointStore compose.CheckPointStore
	MaxIterations  int
	RegisteredTools []string
	HookManager    *hooks.HookManager
}

// NewMasterAgent 创建 Master Agent
func NewMasterAgent(ctx context.Context, cfg *MasterAgentConfig) (*MasterAgent, error) {
	if cfg == nil {
		return nil, ErrConfigNil
	}

	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	// 检查必要配置
	if cfg.Context == nil {
		return nil, fmt.Errorf("Context 不能为空")
	}

	// 创建 ChatModelAdapter
	var skillLoader func(string) string
	if cfg.Context != nil && cfg.Context.GetSkillsLoader() != nil {
		skillLoader = cfg.Context.GetSkillsLoader().LoadSkill
	}
	llm, err := buildChatModelAdapter(logger, cfg.ConfigLoader, cfg.Sessions, skillLoader, cfg.RegisteredTools, CreateHookCallback(cfg.HookManager, logger))
	if err != nil {
		return nil, fmt.Errorf("创建 LLM 适配器失败: %w", err)
	}

	// 配置工具
	var toolsConfig adk.ToolsConfig
	if len(cfg.Tools) > 0 {
		toolsConfig = adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: cfg.Tools,
			},
		}
	}

	// 创建 ADK Agent
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "master_agent",
		Description:   "主 Agent，负责处理用户请求并协调工具调用",
		Instruction:   "", // 系统提示词在 Process 中动态构建
		Model:         llm,
		ToolsConfig:   toolsConfig,
		MaxIterations: cfg.MaxIterations,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 ADK Agent 失败: %w", err)
	}

	// 创建 Runner
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		CheckPointStore: cfg.CheckpointStore,
	})

	// 创建 interruptible 能力
	interruptCfg := &interruptibleConfig{
		ConfigLoader:    cfg.ConfigLoader,
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
		AgentType:       "master",
		ADKAgent:        agent,
		ADKRunner:       runner,
		HookManager:     cfg.HookManager,
	}

	ic, err := newInterruptible(ctx, interruptCfg)
	if err != nil {
		return nil, fmt.Errorf("创建中断能力失败: %w", err)
	}

	master := &MasterAgent{
		interruptible: ic,
		configLoader:  cfg.ConfigLoader,
		workspace:     cfg.Workspace,
		tools:         cfg.Tools,
		logger:        logger,
		sessions:      cfg.Sessions,
		context:       cfg.Context,
		adkRunner:     runner,
	}

	logger.Info("Master Agent 初始化成功",
		zap.String("workspace", cfg.Workspace),
		zap.Int("max_iterations", cfg.MaxIterations),
	)

	return master, nil
}

// Process 处理用户消息
func (m *MasterAgent) Process(ctx context.Context, msg *bus.InboundMessage) (string, error) {
	if m == nil {
		return "", fmt.Errorf("MasterAgent 未初始化")
	}
	if m.interruptible == nil {
		return "", fmt.Errorf("MasterAgent 中断能力未初始化")
	}

	// 构建消息构建函数
	buildMessagesFunc := func(history []*schema.Message, userInput, channel, chatID string) []*schema.Message {
		systemPrompt := ""
		if m.context != nil {
			systemPrompt = m.context.BuildSystemPrompt()
		}
		return BuildMessageList(systemPrompt, history, userInput, channel, chatID)
	}

	return m.interruptible.Process(ctx, msg, buildMessagesFunc)
}
