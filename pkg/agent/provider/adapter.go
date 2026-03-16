package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/trace"
	"github.com/weibaohui/nanobot-go/pkg/session"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// NewChatModelAdapter 创建 ChatModel 适配器（使用 ConfigLoader）
// 注意：这是兼容旧代码的包装函数，建议使用 NewChatModelAdapterV2 配合 WithConfigLoader 选项
func NewChatModelAdapter(logger *zap.Logger, configLoader LLMConfigLoader, sessions *session.Manager) (*ChatModelAdapter, error) {
	return NewChatModelAdapterV2(
		WithLogger(logger),
		WithConfigLoader(configLoader),
		WithSessions(sessions),
	)
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

// GetChatModel 获取内部的 ChatModel
func (a *ChatModelAdapter) GetChatModel() model.ToolCallingChatModel {
	return a.chatModel
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

// NewChatModelAdapterV2 使用选项模式创建 ChatModelAdapter（新的统一入口）
func NewChatModelAdapterV2(opts ...AdapterOption) (*ChatModelAdapter, error) {
	// 默认配置
	cfg := &adapterConfig{
		logger:   zap.NewNop(),
		sessions: nil,
	}

	// 应用选项
	for _, opt := range opts {
		opt(cfg)
	}

	// 验证必要参数
	if cfg.db == nil && cfg.configLoader == nil {
		return nil, fmt.Errorf("必须提供数据库连接 (WithDB) 或配置加载器 (WithConfigLoader)")
	}

	// 如果没有提供 configLoader，但有 db，则创建一个
	if cfg.configLoader == nil && cfg.db != nil {
		cfg.configLoader = func(ctx context.Context) (*LLMConfig, error) {
			provider, err := loadDefaultProvider(cfg.db)
			if err != nil {
				return nil, err
			}
			return &LLMConfig{
				APIKey:       provider.APIKey,
				APIBase:      provider.APIBase,
				DefaultModel: provider.DefaultModel,
				ExtraHeaders: parseExtraHeaders(provider.ExtraHeaders, cfg.logger),
			}, nil
		}
	}

	// 使用 configLoader 获取配置
	llmConfig, err := cfg.configLoader(context.Background())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNilConfig, err)
	}

	if llmConfig == nil || llmConfig.APIKey == "" {
		cfg.logger.Warn("未找到有效的 API Key")
		return nil, ErrNilAPIKey
	}

	if llmConfig.DefaultModel == "" {
		return nil, fmt.Errorf("LLM 配置未设置默认模型")
	}

	chatModel, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		APIKey:  llmConfig.APIKey,
		Model:   llmConfig.DefaultModel,
		BaseURL: llmConfig.APIBase,
	})
	if err != nil {
		cfg.logger.Error("创建 OpenAI ChatModel 失败", zap.Error(err))
		return nil, fmt.Errorf("%w: %w", ErrCreateChatModel, err)
	}

	return &ChatModelAdapter{
		logger:        cfg.logger,
		chatModel:     chatModel,
		registeredMap: make(map[string]bool),
		sessions:      cfg.sessions,
	}, nil
}

// dbLLMProvider 数据库模型（简化版，避免循环依赖）
type dbLLMProvider struct {
	APIKey       string
	APIBase      string
	DefaultModel string
	ExtraHeaders string
}

// loadDefaultProvider 从数据库加载默认 Provider（内部函数，消除代码重复）
func loadDefaultProvider(db *gorm.DB) (*dbLLMProvider, error) {
	if db == nil {
		return nil, fmt.Errorf("数据库连接不能为空")
	}

	var provider dbLLMProvider
	err := db.Model(&dbLLMProvider{}).
		Table("llm_providers").
		Where("is_default = ? AND is_active = ?", true, true).
		Select("api_key, api_base, default_model, extra_headers").
		First(&provider).Error
	if err != nil {
		return nil, fmt.Errorf("获取默认 Provider 失败: %w", err)
	}

	return &provider, nil
}

// parseExtraHeaders 解析额外请求头
func parseExtraHeaders(extraHeadersJSON string, logger *zap.Logger) map[string]string {
	var extraHeaders map[string]string
	if extraHeadersJSON != "" && extraHeadersJSON != "null" {
		if err := json.Unmarshal([]byte(extraHeadersJSON), &extraHeaders); err != nil {
			if logger != nil {
				logger.Warn("解析 extra_headers 失败，将忽略该字段", zap.Error(err))
			}
		}
	}
	return extraHeaders
}

// NewChatModelAdapterFromDB 从数据库直接创建 ChatModelAdapter
// 注意：这是兼容旧代码的包装函数，建议使用 NewChatModelAdapterV2 配合 WithDB 选项
func NewChatModelAdapterFromDB(db *gorm.DB, logger *zap.Logger, sessions *session.Manager) (*ChatModelAdapter, error) {
	return NewChatModelAdapterV2(
		WithDB(db),
		WithLogger(logger),
		WithSessions(sessions),
	)
}

// CreateConfigLoaderFromDB 从数据库创建 LLMConfigLoader 函数
// 用于需要动态获取配置的场景（如 Main Agent）
func CreateConfigLoaderFromDB(db *gorm.DB, logger *zap.Logger) LLMConfigLoader {
	return func(ctx context.Context) (*LLMConfig, error) {
		provider, err := loadDefaultProvider(db)
		if err != nil {
			return nil, err
		}

		return &LLMConfig{
			APIKey:       provider.APIKey,
			APIBase:      provider.APIBase,
			DefaultModel: provider.DefaultModel,
			ExtraHeaders: parseExtraHeaders(provider.ExtraHeaders, logger),
		}, nil
	}
}
