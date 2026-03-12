package provider

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/agent/hooks/trace"
	"github.com/weibaohui/nanobot-go/session"
	"go.uber.org/zap"
)

// NewChatModelAdapter 创建 ChatModel 适配器
func NewChatModelAdapter(logger *zap.Logger, configLoader LLMConfigLoader, sessions *session.Manager) (*ChatModelAdapter, error) {
	ctx := context.Background()

	cfg, err := configLoader(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNilConfig, err)
	}

	if cfg == nil || cfg.APIKey == "" {
		logger.Warn("未找到有效的 API Key")
		return nil, ErrNilAPIKey
	}

	modelName := cfg.DefaultModel
	if modelName == "" {
		modelName = "gpt-4o-mini"
	}

	chatModel, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		APIKey:  cfg.APIKey,
		Model:   modelName,
		BaseURL: cfg.APIBase,
	})
	if err != nil {
		if logger != nil {
			logger.Error("创建 OpenAI ChatModel 失败", zap.Error(err))
		}
		return nil, fmt.Errorf("%w: %w", ErrCreateChatModel, err)
	}

	return &ChatModelAdapter{
		logger:        logger,
		chatModel:     chatModel,
		registeredMap: make(map[string]bool),
		sessions:      sessions,
	}, nil
}

// SetSkillLoader 设置技能加载器
func (a *ChatModelAdapter) SetSkillLoader(loader SkillLoader) {
	a.skillLoader = loader
}

// SetHookCallback 设置 Hook 回调函数
func (a *ChatModelAdapter) SetHookCallback(callback HookCallback) {
	a.hookCallback = callback
}

// SetRegisteredTools 设置已注册的工具名称列表
func (a *ChatModelAdapter) SetRegisteredTools(names []string) {
	a.registeredMap = make(map[string]bool)
	for _, name := range names {
		a.registeredMap[name] = true
	}
}

// isRegisteredTool 检查工具是否已注册
func (a *ChatModelAdapter) isRegisteredTool(name string) bool {
	if a.registeredMap == nil {
		return false
	}
	return a.registeredMap[name]
}

// isKnownSkill 检查是否是已知技能
func (a *ChatModelAdapter) isKnownSkill(name string) bool {
	if a.skillLoader == nil {
		return false
	}
	content := a.skillLoader(name)
	return content != ""
}

// Generate produces a complete model response
func (a *ChatModelAdapter) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	ctx, llmSpanID := trace.StartSpan(ctx)
	a.logger.Debug("LLM 调用开始",
		zap.String("span_id", llmSpanID),
		zap.Int("message_count", len(input)),
	)

	a.triggerLLMCallStart(ctx, input)

	if a.logger != nil && len(input) > 0 {
		for i, msg := range input {
			a.logger.Debug("[LLM] 发送消息",
				zap.Int("index", i),
				zap.String("role", string(msg.Role)),
				zap.String("content_preview", truncate(msg.Content, 200)),
			)
		}
	}

	response, err := a.chatModel.Generate(ctx, input, opts...)
	if err != nil {
		if a.logger != nil {
			a.logger.Error("调用 LLM 失败", zap.Error(err))
		}
		a.triggerLLMCallError(ctx, err)
		return nil, err
	}

	a.logger.Debug("LLM 调用完成",
		zap.String("span_id", llmSpanID),
		zap.Int("tool_calls", len(response.ToolCalls)),
	)

	a.triggerLLMCallEnd(ctx, response)
	a.interceptToolCalls(response)

	return response, nil
}

// Stream produces a response as a stream
func (a *ChatModelAdapter) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	msg, err := a.Generate(ctx, input, opts...)
	if err != nil {
		return nil, err
	}

	sr, sw := schema.Pipe[*schema.Message](1)
	go func() {
		defer sw.Close()
		sw.Send(msg, nil)
	}()

	return sr, nil
}

// WithTools returns a new adapter instance with the specified tools bound
func (a *ChatModelAdapter) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	boundModel, err := a.chatModel.WithTools(tools)
	if err != nil {
		return nil, err
	}

	return &ChatModelAdapter{
		logger:        a.logger,
		chatModel:     boundModel,
		registeredMap: a.registeredMap,
		skillLoader:   a.skillLoader,
		sessions:      a.sessions,
		hookCallback:  a.hookCallback,
	}, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
