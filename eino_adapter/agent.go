package eino_adapter

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"

	"github.com/weibaohui/nanobot-go/providers"
)

// PlanExecuteAgent wraps the eino plan-execute-replan agent
type PlanExecuteAgent struct {
	agent   adk.ResumableAgent
	runner  *adk.Runner
	adapter *ProviderAdapter
	tools   []tool.BaseTool
	logger  *zap.Logger
}

// Config holds configuration for the PlanExecuteAgent
type Config struct {
	Provider      providers.LLMProvider
	Model         string
	Tools         []tool.BaseTool
	Logger        *zap.Logger
	EnableStream  bool
	MaxIterations int
}

// NewPlanExecuteAgent creates a new Plan-Execute-Replan agent
func NewPlanExecuteAgent(ctx context.Context, cfg *Config) (*PlanExecuteAgent, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	if cfg.Provider == nil {
		return nil, fmt.Errorf("provider is required")
	}

	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	maxIterations := cfg.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 10
	}

	// Create the provider adapter
	adapter := NewProviderAdapter(logger, cfg.Provider, cfg.Model)

	// Create planner agent using ToolCallingChatModel (not ChatModelWithFormattedOutput)
	// This uses tool calling to generate structured Plan output
	planner, err := planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ToolCallingChatModel: adapter,
		// ToolInfo is optional, will use default PlanToolInfo if not provided
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create planner: %w", err)
	}

	// Convert tools to eino format and create tools config
	var toolsConfig adk.ToolsConfig
	if len(cfg.Tools) > 0 {
		toolsConfig = adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: cfg.Tools,
			},
		}
	}

	// Create executor agent
	executor, err := planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model:       adapter,
		ToolsConfig: toolsConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	// Create replanner agent
	replanner, err := planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel: adapter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create replanner: %w", err)
	}

	// Create the plan-execute-replan agent
	peAgent, err := planexecute.New(ctx, &planexecute.Config{
		Planner:       planner,
		Executor:      executor,
		Replanner:     replanner,
		MaxIterations: maxIterations,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create plan-execute agent: %w", err)
	}

	// Create the runner
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           peAgent,
		EnableStreaming: cfg.EnableStream,
	})

	logger.Info("Plan-Execute-Replan agent created successfully",
		zap.Int("max_iterations", maxIterations),
		zap.Int("tools_count", len(cfg.Tools)),
	)

	return &PlanExecuteAgent{
		agent:   peAgent,
		runner:  runner,
		adapter: adapter,
		tools:   cfg.Tools,
		logger:  logger,
	}, nil
}

// Execute runs the plan-execute-replan cycle for the given messages
func (a *PlanExecuteAgent) Execute(ctx context.Context, messages []*schema.Message) (string, error) {
	return a.ExecuteWithHistory(ctx, messages[len(messages)-1].Content, messages[:len(messages)-1])
}

// ExecuteWithHistory runs the agent with conversation history
func (a *PlanExecuteAgent) ExecuteWithHistory(ctx context.Context, input string, history []*schema.Message) (string, error) {
	a.logger.Debug("Starting plan-execute cycle", zap.String("input", input))

	// Prepare messages
	messages := make([]*schema.Message, 0)
	if len(history) > 0 {
		messages = append(messages, history...)
	}
	messages = append(messages, &schema.Message{
		Role:    schema.User,
		Content: input,
	})

	// Run the agent
	iter := a.runner.Run(ctx, messages)

	var result string
	var eventCount int

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		eventCount++

		if event.Err != nil {
			a.logger.Error("Agent error", zap.Error(event.Err))
			return "", event.Err
		}

		// Handle different event types
		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				a.logger.Error("Failed to get message from output", zap.Error(err))
				continue
			}
			result = msg.Content
			a.logger.Debug("Agent output",
				zap.String("content", msg.Content),
				zap.Int("tool_calls", len(msg.ToolCalls)),
			)
		}

		// Log agent actions
		if event.Action != nil {
			a.logger.Debug("Agent action",
				zap.Bool("exit", event.Action.Exit),
				zap.Any("transfer", event.Action.TransferToAgent),
			)
		}
	}

	a.logger.Debug("Plan-execute cycle completed",
		zap.Int("events", eventCount),
		zap.Int("result_length", len(result)),
	)

	return result, nil
}

// Query is a convenience method that starts a new execution with a single user query
func (a *PlanExecuteAgent) Query(ctx context.Context, query string) (string, error) {
	a.logger.Debug("Query received", zap.String("query", query))

	iter := a.runner.Query(ctx, query)

	var result string

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return "", event.Err
		}

		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				continue
			}
			result = msg.Content
		}
	}

	return result, nil
}

// Stream executes and returns an iterator for streaming results
func (a *PlanExecuteAgent) Stream(ctx context.Context, input string) *adk.AsyncIterator[*adk.AgentEvent] {
	messages := []*schema.Message{
		{Role: schema.User, Content: input},
	}
	return a.runner.Run(ctx, messages)
}

// GetChatModel returns the underlying chat model adapter
func (a *PlanExecuteAgent) GetChatModel() model.ToolCallingChatModel {
	return a.adapter
}

// GetTools returns the configured tools
func (a *PlanExecuteAgent) GetTools() []tool.BaseTool {
	return a.tools
}
