package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/bus"
	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/session"
	"go.uber.org/zap"
)

const interruptErrorPrefix = "INTERRUPT:"

// MasterAgent 监督者 Agent
// 作为统一入口，根据用户输入自动路由到合适的子 Agent
type MasterAgent struct {
	*interruptible
	cfg       *config.Config
	workspace string
	tools     []tool.BaseTool
	logger    *zap.Logger
	sessions  *session.Manager
	context   *ContextBuilder

	adkRunner *adk.Runner
}

// MasterAgentConfig Master 配置
type MasterAgentConfig struct {
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

// NewMasterAgent 创建 Master Agent
func NewMasterAgent(ctx context.Context, cfg *MasterAgentConfig) (*MasterAgent, error) {
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
		AgentType:       "master",
	})
	if err != nil {
		return nil, err
	}

	// 初始化 MasterAgent
	sa := &MasterAgent{
		interruptible: interruptible,
		cfg:           cfg.Cfg,
		workspace:     cfg.Workspace,
		tools:         cfg.Tools,
		logger:        logger,
		sessions:      cfg.Sessions,
		context:       cfg.Context,
	}

	// 创建 ADK Runner
	adapter, err := NewChatModelAdapter(sa.logger, sa.cfg, sa.sessions)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrChatModelAdapter, err)
	}
	if sa.context != nil {
		adapter.SetSkillLoader(sa.context.GetSkillsLoader().LoadSkill)
	}
	if len(cfg.RegisteredTools) > 0 {
		adapter.SetRegisteredTools(cfg.RegisteredTools)
	}

	var toolsConfig adk.ToolsConfig
	if len(cfg.Tools) > 0 {
		toolsConfig = adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: cfg.Tools,
			},
		}
	}

	masterAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "Master",
		Description: "主智能体",
		Instruction: sa.context.BuildSystemPrompt(),
		Model:       adapter,
		ToolsConfig: toolsConfig,
		Exit:        &adk.ExitTool{},
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAgentCreate, err)
	}

	sa.adkRunner = adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           masterAgent,
		CheckPointStore: cfg.CheckpointStore,
	})

	// 设置 ADK Runner 到 interruptible
	interruptible.adkRunner = sa.adkRunner

	logger.Info("Master Agent 创建成功",
		zap.String("model", cfg.Workspace),
	)

	return sa, nil
}

// Process 处理用户消息
func (sa *MasterAgent) Process(ctx context.Context, msg *bus.InboundMessage) (string, error) {
	return sa.interruptible.Process(ctx, msg, sa.buildMessages)
}

// buildMessages 构建消息列表
func (sa *MasterAgent) buildMessages(history []*schema.Message, userInput, channel, chatID string) []*schema.Message {
	// 复用公共方法构建消息列表
	return BuildMessageList("", history, userInput, channel, chatID)
}

// convertHistory 转换会话历史
func (sa *MasterAgent) convertHistory(history []map[string]any) []*schema.Message {
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

// GetADKRunner 获取 ADK Runner
func (sa *MasterAgent) GetADKRunner() *adk.Runner {
	return sa.adkRunner
}

// Resume 恢复被中断的执行
// 用于处理 Master 模式下的中断恢复
func (sa *MasterAgent) Resume(ctx context.Context, checkpointID string, resumeParams *adk.ResumeParams, msg *bus.InboundMessage) (string, error) {
	if sa.adkRunner == nil {
		return "", fmt.Errorf("ADK Runner 未初始化")
	}

	// 将 session key 放入 context，用于记录 token 用量
	sessionKey := msg.SenderID
	ctx = context.WithValue(ctx, SessionKeyContextKey, sessionKey)

	// 使用 Master 的 Runner 恢复执行
	iter, err := sa.adkRunner.ResumeWithParams(ctx, checkpointID, resumeParams)
	if err != nil {
		return "", fmt.Errorf("Master 恢复执行失败: %w", err)
	}

	var response string
	var lastEvent *adk.AgentEvent

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return "", fmt.Errorf("Master 恢复后执行失败: %w", event.Err)
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
		return "", sa.handleInterrupt(msg, newCheckpointID, checkpointID, lastEvent)
	}

	return response, nil
}
