package eino_adapter

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"

	"github.com/weibaohui/nanobot-go/providers"
)

// ChatModelAgent 封装 eino ADK 的 ChatModelAgent
type ChatModelAgent struct {
	agent   *adk.ChatModelAgent
	runner  *adk.Runner
	adapter *ProviderAdapter
	tools   []tool.BaseTool
	logger  *zap.Logger
}

// ChatModelAgentConfig 配置
type ChatModelAgentConfig struct {
	Provider      providers.LLMProvider
	Model         string
	Tools         []tool.BaseTool
	Logger        *zap.Logger
	MaxIterations int
	EnableStream  bool
}

// NewChatModelAgent 创建基于 eino ADK 的 Agent
func NewChatModelAgent(ctx context.Context, cfg *ChatModelAgentConfig) (*ChatModelAgent, error) {
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
		maxIterations = 15
	}

	// Create the provider adapter
	adapter := NewProviderAdapter(cfg.Provider, cfg.Model)

	// Convert tools to eino format
	var toolsConfig adk.ToolsConfig
	if len(cfg.Tools) > 0 {
		toolsConfig = adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: cfg.Tools,
			},
		}
	}

	// Create the ChatModelAgent
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "nanobot",
		Description:   "nanobot AI assistant",
		Model:         adapter,
		ToolsConfig:   toolsConfig,
		MaxIterations: maxIterations,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create chat model agent: %w", err)
	}

	// Create the runner
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: cfg.EnableStream,
	})

	logger.Info("ChatModelAgent 创建成功",
		zap.Int("max_iterations", maxIterations),
		zap.Int("tools_count", len(cfg.Tools)),
	)

	return &ChatModelAgent{
		agent:   agent,
		runner:  runner,
		adapter: adapter,
		tools:   cfg.Tools,
		logger:  logger,
	}, nil
}

// Execute 执行单次对话
func (a *ChatModelAgent) Execute(ctx context.Context, input string) (string, error) {
	return a.ExecuteWithHistory(ctx, input, nil)
}

// ExecuteWithHistory 带历史记录执行
func (a *ChatModelAgent) ExecuteWithHistory(ctx context.Context, input string, history []*schema.Message) (string, error) {
	a.logger.Debug("开始执行", zap.String("输入", input))

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
			a.logger.Error("Agent 执行出错", zap.Error(event.Err))
			return "", event.Err
		}

		// Handle output
		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				a.logger.Error("获取消息失败", zap.Error(err))
				continue
			}
			result = msg.Content
			a.logger.Debug("Agent 输出",
				zap.String("内容预览", truncate(msg.Content, 100)),
				zap.Int("工具调用次数", len(msg.ToolCalls)),
			)
		}

		// Log actions
		if event.Action != nil {
			a.logger.Debug("Agent 动作",
				zap.Bool("退出", event.Action.Exit),
			)
		}
	}

	a.logger.Debug("执行完成",
		zap.Int("事件数", eventCount),
		zap.Int("结果长度", len(result)),
	)

	return result, nil
}

// ExecuteStream 流式执行，通过 callback 返回每个增量
func (a *ChatModelAgent) ExecuteStream(ctx context.Context, input string, history []*schema.Message, onDelta func(delta, fullContent string) error) (string, error) {
	a.logger.Debug("开始流式执行", zap.String("输入", input))

	// Prepare messages
	messages := make([]*schema.Message, 0)
	if len(history) > 0 {
		messages = append(messages, history...)
	}
	messages = append(messages, &schema.Message{
		Role:    schema.User,
		Content: input,
	})

	// Run the agent with streaming enabled
	iter := a.runner.Run(ctx, messages)

	var fullContent string
	var eventCount int

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		eventCount++

		if event.Err != nil {
			a.logger.Error("Agent 流式执行出错", zap.Error(event.Err))
			return fullContent, event.Err
		}

		// Handle streaming output
		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				continue
			}

			// Calculate delta
			delta := msg.Content[len(fullContent):]
			if delta != "" && onDelta != nil {
				if err := onDelta(delta, msg.Content); err != nil {
					a.logger.Error("增量回调出错", zap.Error(err))
				}
			}
			fullContent = msg.Content

			a.logger.Debug("流式片段",
				zap.Int("delta_len", len(delta)),
				zap.Int("total_len", len(fullContent)),
			)
		}

		// Check for completion
		if event.Action != nil && event.Action.Exit {
			a.logger.Debug("Agent exited")
			break
		}
	}

	a.logger.Debug("Stream execution completed",
		zap.Int("events", eventCount),
		zap.Int("result_length", len(fullContent)),
	)

	return fullContent, nil
}

// Query 便捷方法
func (a *ChatModelAgent) Query(ctx context.Context, query string) (string, error) {
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

// GetAdapter 获取底层 adapter
func (a *ChatModelAgent) GetAdapter() *ProviderAdapter {
	return a.adapter
}

// GetTools 获取配置的工具
func (a *ChatModelAgent) GetTools() []tool.BaseTool {
	return a.tools
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
