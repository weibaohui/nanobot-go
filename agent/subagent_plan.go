package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/session"
	"go.uber.org/zap"
)

// PlanSubAgent Plan-Execute-Replan 模式子 Agent
// 适用于复杂任务的规划、执行和动态调整
type PlanSubAgent struct {
	agent    adk.ResumableAgent
	runner   *adk.Runner
	adapter  *ChatModelAdapter
	sessions *session.Manager
	tools    []tool.BaseTool
	logger   *zap.Logger
}

// PlanConfig Plan Agent 配置
type PlanConfig struct {
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

// NewPlanSubAgent 创建 Plan 子 Agent
func NewPlanSubAgent(ctx context.Context, cfg *PlanConfig) (*PlanSubAgent, error) {
	if cfg == nil {
		return nil, ErrConfigNil
	}

	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	maxIter := cfg.MaxIterations
	if maxIter <= 0 {
		maxIter = 20
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

	planner, err := planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ToolCallingChatModel: adapter,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrPlannerCreate, err)
	}

	var toolDescriptions []string
	var toolInfos []*schema.ToolInfo
	for _, t := range cfg.Tools {
		info, err := t.Info(ctx)
		if err == nil && info != nil {
			toolDescriptions = append(toolDescriptions, fmt.Sprintf("- %s: %s", info.Name, info.Desc))
			toolInfos = append(toolInfos, info)
		}
	}

	var executorModel model.ToolCallingChatModel = adapter
	if len(toolInfos) > 0 {
		boundModel, err := adapter.WithTools(toolInfos)
		if err != nil {
			logger.Warn("绑定工具到执行器失败", zap.Error(err))
		} else {
			executorModel = boundModel
		}
	}

	toolsConfig := buildToolsConfig(cfg.Tools)

	executor, err := planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model:       executorModel,
		ToolsConfig: toolsConfig,
		GenInputFn:  buildExecutorInputFn(cfg.Workspace, toolDescriptions, logger),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrExecutorCreate, err)
	}

	replanner, err := planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel: adapter,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrReplannerCreate, err)
	}

	peAgent, err := planexecute.New(ctx, &planexecute.Config{
		Planner:       planner,
		Executor:      executor,
		Replanner:     replanner,
		MaxIterations: maxIter,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAgentCreate, err)
	}

	// 包装为有名称的 Agent
	namedAgent := &namedPlanAgent{
		ResumableAgent: peAgent,
		name:           "plan_agent",
		description:    "Plan-Execute-Replan 模式 Agent，用于复杂任务的规划与执行",
	}

	// 创建 Runner
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           namedAgent,
		EnableStreaming: true,
		CheckPointStore: cfg.CheckpointStore,
	})

	logger.Info("Plan Agent 创建成功",
		zap.Int("max_iterations", maxIter),
		zap.Int("tools_count", len(cfg.Tools)),
		zap.Bool("has_sessions", cfg.Sessions != nil),
	)

	return &PlanSubAgent{
		agent:    namedAgent,
		runner:   runner,
		adapter:  adapter,
		sessions: cfg.Sessions,
		tools:    cfg.Tools,
		logger:   logger,
	}, nil
}

// buildExecutorInputFn 构建执行器输入生成函数
func buildExecutorInputFn(workspace string, toolDescriptions []string, logger *zap.Logger) func(ctx context.Context, in *planexecute.ExecutionContext) ([]*schema.Message, error) {
	return func(ctx context.Context, in *planexecute.ExecutionContext) ([]*schema.Message, error) {
		logger.Debug("Executor GenInputFn 被调用",
			zap.String("step", in.Plan.FirstStep()),
			zap.Int("executed_steps", len(in.ExecutedSteps)),
		)

		planContent, err := in.Plan.MarshalJSON()
		if err != nil {
			return nil, err
		}

		firstStep := in.Plan.FirstStep()

		// 构建已执行步骤摘要
		var executedSteps strings.Builder
		for idx, m := range in.ExecutedSteps {
			executedSteps.WriteString(fmt.Sprintf("## %d. 步骤: %s\n  结果: %s\n\n", idx+1, m.Step, m.Result))
		}

		// 构建系统提示
		systemPrompt := fmt.Sprintf(`你是一个勤奋的 AI 助手，按照计划逐步执行任务。

## 当前时间
%s

## 工作区
%s

## 可用工具
%s

## 关键规则
1. 必须使用工具完成任务，不要只是回复文本
2. 当步骤需要用户输入时，必须调用 "ask_user" 工具
3. 不要用纯文本提问 - 始终使用 ask_user 工具
4. 如果无法在没有用户输入的情况下完成步骤，立即调用 ask_user

## 何时使用 ask_user 工具
- 需要用户偏好（目的地、日期、预算等）
- 需要用户确认
- 缺少完成任务所需的信息

## 示例
错误: "请告诉我您想去哪里旅行？"
正确: 调用 ask_user 工具，question="您想去哪里旅行？"

记住：始终使用工具，当需要信息时不要只是用文本回复。`, time.Now().Format("2006-01-02 15:04 (Monday)"), workspace, strings.Join(toolDescriptions, "\n"))

		userMessage := fmt.Sprintf(`## 目标
%s

## 计划
%s

## 已完成的步骤和结果
%s

## 你的任务是执行第一个步骤:
%s`, in.UserInput[0].Content, string(planContent), executedSteps.String(), firstStep)

		return []*schema.Message{
			{Role: schema.System, Content: systemPrompt},
			{Role: schema.User, Content: userMessage},
		}, nil
	}
}

// Name 返回 Agent 名称
func (a *PlanSubAgent) Name() string {
	return string(AgentTypePlan)
}

// Description 返回 Agent 描述
func (a *PlanSubAgent) Description() string {
	return "Plan-Execute-Replan 模式 Agent，用于复杂任务的规划与执行"
}

// Type 返回 Agent 类型
func (a *PlanSubAgent) Type() AgentType {
	return AgentTypePlan
}

// GetADKAgent 返回底层的 ADK Agent
func (a *PlanSubAgent) GetADKAgent() adk.Agent {
	return a.agent
}

// Execute 执行任务
func (a *PlanSubAgent) Execute(ctx context.Context, input string, history []*schema.Message) (string, error) {
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
	iter := a.runner.Run(ctx, messages)

	var response string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return "", fmt.Errorf("Plan Agent 执行失败: %w", event.Err)
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
func (a *PlanSubAgent) Stream(ctx context.Context, input string, history []*schema.Message) (*adk.AsyncIterator[*adk.AgentEvent], error) {
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

// ExecuteWithCheckpoint 带检查点执行
func (a *PlanSubAgent) ExecuteWithCheckpoint(ctx context.Context, input string, history []*schema.Message, checkpointID string) (string, *adk.AgentEvent, error) {
	messages := make([]*schema.Message, 0, len(history)+2)

	if len(history) > 0 {
		messages = append(messages, history...)
	}

	messages = append(messages, &schema.Message{
		Role:    schema.User,
		Content: input,
	})

	iter := a.runner.Run(ctx, messages, adk.WithCheckPointID(checkpointID))

	var response string
	var lastEvent *adk.AgentEvent

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return "", nil, fmt.Errorf("Plan Agent 执行失败: %w", event.Err)
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
func (a *PlanSubAgent) Resume(ctx context.Context, checkpointID string, resumeParams *adk.ResumeParams) (string, error) {
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
func (a *PlanSubAgent) GetRunner() *adk.Runner {
	return a.runner
}

// namedPlanAgent 为 PlanExecuteAgent 提供名称包装
type namedPlanAgent struct {
	adk.ResumableAgent
	name        string
	description string
}

// Name 返回 Agent 名称
func (n *namedPlanAgent) Name(_ context.Context) string {
	return n.name
}

// Description 返回 Agent 描述
func (n *namedPlanAgent) Description(_ context.Context) string {
	return n.description
}
