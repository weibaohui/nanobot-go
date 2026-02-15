package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/eino_adapter"
	"go.uber.org/zap"
)

// ReActSubAgent ReAct 模式子 Agent
// 适用于工具调用、推理、长对话场景
type ReActSubAgent struct {
	cfg    *config.Config
	agent  *adk.ChatModelAgent
	runner *adk.Runner
	tools  []tool.BaseTool
	logger *zap.Logger
}

// ReActConfig ReAct Agent 配置
type ReActConfig struct {
	Cfg             *config.Config
	Workspace       string
	Tools           []tool.BaseTool
	Logger          *zap.Logger
	CheckpointStore compose.CheckPointStore
	MaxIterations   int
	// 技能加载器
	SkillsLoader func(skillName string) string
	// 已注册的工具名称列表
	RegisteredTools []string
}

// NewReActSubAgent 创建 ReAct 子 Agent
func NewReActSubAgent(ctx context.Context, cfg *ReActConfig) (*ReActSubAgent, error) {
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

	adapter, err := eino_adapter.NewProviderAdapter(logger, cfg.Cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrProviderAdapter, err)
	}

	if cfg.SkillsLoader != nil {
		adapter.SetSkillLoader(cfg.SkillsLoader)
	}
	if len(cfg.RegisteredTools) > 0 {
		adapter.SetRegisteredTools(cfg.RegisteredTools)
	}

	toolsConfig := buildToolsConfig(cfg.Tools)

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "react_agent",
		Description:   "ReAct 模式 Agent，用于工具调用、推理和长对话",
		Instruction:   buildReActInstruction(cfg.Workspace),
		Model:         adapter,
		ToolsConfig:   toolsConfig,
		MaxIterations: maxIter,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAgentCreate, err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
		CheckPointStore: cfg.CheckpointStore,
	})

	logger.Info("ReAct Agent 创建成功",
		zap.Int("max_iterations", maxIter),
		zap.Int("tools_count", len(cfg.Tools)),
	)

	return &ReActSubAgent{
		agent:  agent,
		runner: runner,
		cfg:    cfg.Cfg,
		tools:  cfg.Tools,
		logger: logger,
	}, nil
}


// buildReActInstruction 构建 ReAct Agent 指令
func buildReActInstruction(workspace string) string {
	now := time.Now().Format("2006-01-02 15:04 (Monday)")
	tz, _ := time.Now().Zone()

	return fmt.Sprintf(`# ReAct Agent

## 当前时间
%s (%s)

你是一个 ReAct 模式的 AI Agent，专门处理需要工具调用、推理和长对话的任务。

## ReAct 模式说明
你遵循 ReAct 模式工作：
1. **思考 (Reasoning)**：分析当前情况，决定下一步行动
2. **行动 (Acting)**：调用工具执行操作
3. **观察 (Observing)**：获取工具执行结果
4. **循环**：根据观察结果继续思考，直到完成任务

## 你的能力
- 读取、写入、编辑文件
- 执行 shell 命令
- 搜索网络和获取网页内容
- 使用技能扩展功能
- 向用户提问（使用 ask_user 工具）

## 工作区
你的工作区位于: %s

## 工作规则
1. 逐步思考，每一步都要明确你的目标
2. 选择最合适的工具完成任务
3. 如果信息不足，使用 ask_user 工具向用户提问
4. 保持专注，只完成分配给你的任务
5. 完成后提供清晰的结果摘要

## 重要提示
- 不要编造信息，如果不确定就提问
- 使用工具时要仔细检查参数
- 遇到错误时分析原因并尝试解决
- 完成任务后简洁地报告结果`, now, tz, workspace)
}

// Name 返回 Agent 名称
func (a *ReActSubAgent) Name() string {
	return string(AgentTypeReAct)
}

// Description 返回 Agent 描述
func (a *ReActSubAgent) Description() string {
	return "ReAct 模式 Agent，用于工具调用、推理和长对话"
}

// Type 返回 Agent 类型
func (a *ReActSubAgent) Type() AgentType {
	return AgentTypeReAct
}

// GetADKAgent 返回底层的 ADK Agent
func (a *ReActSubAgent) GetADKAgent() adk.Agent {
	return a.agent
}

// Execute 执行任务
func (a *ReActSubAgent) Execute(ctx context.Context, input string, history []*schema.Message) (string, error) {
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
			return "", fmt.Errorf("ReAct Agent 执行失败: %w", event.Err)
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
func (a *ReActSubAgent) Stream(ctx context.Context, input string, history []*schema.Message) (*adk.AsyncIterator[*adk.AgentEvent], error) {
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

// ExecuteWithCheckpoint 带检查点执行（支持中断恢复）
func (a *ReActSubAgent) ExecuteWithCheckpoint(ctx context.Context, input string, history []*schema.Message, checkpointID string) (string, *adk.AgentEvent, error) {
	// 构建消息
	messages := make([]*schema.Message, 0, len(history)+2)

	if len(history) > 0 {
		messages = append(messages, history...)
	}

	messages = append(messages, &schema.Message{
		Role:    schema.User,
		Content: input,
	})

	// 执行
	iter := a.runner.Run(ctx, messages, adk.WithCheckPointID(checkpointID))

	var response string
	var lastEvent *adk.AgentEvent

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return "", nil, fmt.Errorf("ReAct Agent 执行失败: %w", event.Err)
		}

		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				continue
			}
			response = msg.Content
		}

		lastEvent = event
	}

	return response, lastEvent, nil
}

// Resume 恢复执行
func (a *ReActSubAgent) Resume(ctx context.Context, checkpointID string, resumeParams *adk.ResumeParams) (string, error) {
	iter, err := a.runner.ResumeWithParams(ctx, checkpointID, resumeParams)
	if err != nil {
		return "", fmt.Errorf("恢复执行失败: %w", err)
	}

	var response string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return "", fmt.Errorf("恢复后执行失败: %w", event.Err)
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

// GetRunner 获取 Runner
func (a *ReActSubAgent) GetRunner() *adk.Runner {
	return a.runner
}
