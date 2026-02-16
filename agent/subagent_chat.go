package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/session"
	"go.uber.org/zap"
)

// ChatSubAgent Chat 模式子 Agent
// 适用于简单对话和问答场景
type ChatSubAgent struct {
	Cfg      *config.Config
	sessions *session.Manager
	agent    *adk.ChatModelAgent
	runner   *adk.Runner
	tools    []tool.BaseTool
	logger   *zap.Logger
}

// ChatConfig Chat Agent 配置
type ChatConfig struct {
	Cfg             *config.Config
	Workspace       string
	Tools           []tool.BaseTool
	Logger          *zap.Logger
	Sessions        *session.Manager
	CheckpointStore compose.CheckPointStore
	MaxIterations   int
	// 技能加载器
	SkillsLoader func(skillName string) string
	// 已注册的工具名称列表
	RegisteredTools []string
}

// NewChatSubAgent 创建 Chat 子 Agent
func NewChatSubAgent(ctx context.Context, cfg *ChatConfig) (*ChatSubAgent, error) {
	if cfg == nil {
		return nil, ErrConfigNil
	}

	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	maxIter := cfg.MaxIterations
	if maxIter <= 0 {
		maxIter = 15
	}

	adapter, err := NewChatModelAdapter(logger, cfg.Cfg, cfg.Sessions)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrChatModelAdapter, err)
	}

	if cfg.SkillsLoader != nil {
		adapter.SetSkillLoader(cfg.SkillsLoader)
	}
	if len(cfg.RegisteredTools) > 0 {
		adapter.SetRegisteredTools(cfg.RegisteredTools)
	}

	toolsConfig := buildToolsConfig(cfg.Tools)

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "chat_agent",
		Description:   "Chat 模式 Agent，用于简单对话和问答",
		Instruction:   buildChatInstruction(),
		Model:         adapter,
		ToolsConfig:   toolsConfig,
		MaxIterations: 5,
		Exit:          &adk.ExitTool{},
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAgentCreate, err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
		CheckPointStore: cfg.CheckpointStore,
	})

	logger.Info("Chat Agent 创建成功",
		zap.Int("max_iterations", maxIter),
		zap.Int("tools_count", len(cfg.Tools)),
		zap.Bool("has_sessions", cfg.Sessions != nil),
	)
	return &ChatSubAgent{
		Cfg:      cfg.Cfg,
		sessions: cfg.Sessions,
		agent:    agent,
		runner:   runner,

		tools:  cfg.Tools,
		logger: logger,
	}, nil
}

// buildChatInstruction 构建 Chat Agent 指令
func buildChatInstruction() string {
	return `# Chat Agent

你是一个友好的对话助手，专门处理简单对话和问答。

## 你的职责
- 进行友好的日常对话
- 回答简单问题
- 提供信息和解释
- 保持对话流畅自然

## 你的特点
- 轻量级，快速响应
- 不使用复杂工具
- 专注于对话本身
- 保持简洁但有帮助

## 对话风格
- 友好、礼貌、专业
- 回答直接，不啰嗦
- 适当使用表情符号增加亲和力
- 如果问题超出你的能力范围，诚实告知用户

## 重要约束
- 不要调用任何工具，包括 transfer_to_agent
- 直接回答用户的问题
- 保持上下文理解
- 对话完成后，调用 exit 工具结束

## 注意事项
- 对于需要文件操作、网络搜索等复杂任务，建议用户重新描述需求
- 保持对话的连续性和上下文理解
- 不要编造信息，如果不确定就诚实说明`
}

// Name 返回 Agent 名称
func (a *ChatSubAgent) Name() string {
	return string(AgentTypeChat)
}

// Description 返回 Agent 描述
func (a *ChatSubAgent) Description() string {
	return "Chat 模式 Agent，用于简单对话和问答"
}

// Type 返回 Agent 类型
func (a *ChatSubAgent) Type() AgentType {
	return AgentTypeChat
}

// GetADKAgent 返回底层的 ADK Agent
func (a *ChatSubAgent) GetADKAgent() adk.Agent {
	return a.agent
}

// Execute 执行任务
func (a *ChatSubAgent) Execute(ctx context.Context, input string, history []*schema.Message) (string, error) {
	// 构建消息
	messages := make([]*schema.Message, 0, len(history)+2)

	// 添加历史消息
	if len(history) > 0 {
		messages = append(messages, history...)
	}

	// 添加当前用户消息
	messages = append(messages, &schema.Message{
		Role:    schema.User,
		Content: input,
	})

	// 执行
	iter := a.runner.Run(ctx, messages)

	var response string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return "", fmt.Errorf("Chat Agent 执行失败: %w", event.Err)
		}

		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				continue
			}
			response = msg.Content
		}
	}

	return response, nil
}

// Stream 流式执行
func (a *ChatSubAgent) Stream(ctx context.Context, input string, history []*schema.Message) (*adk.AsyncIterator[*adk.AgentEvent], error) {
	// 构建消息
	messages := make([]*schema.Message, 0, len(history)+2)

	if len(history) > 0 {
		messages = append(messages, history...)
	}

	messages = append(messages, &schema.Message{
		Role:    schema.User,
		Content: input,
	})

	return a.runner.Run(ctx, messages), nil
}

// GetRunner 获取 Runner
func (a *ChatSubAgent) GetRunner() *adk.Runner {
	return a.runner
}
