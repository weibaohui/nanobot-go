package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/session"
	"go.uber.org/zap"
)

// ContextKey session key 的 context key
type ContextKey string

const SessionKeyContextKey ContextKey = "session_key"

// SkillLoader 技能加载函数类型
type SkillLoader func(name string) string

// ChatModelAdapter 包装 eino 的 ChatModel，添加工具调用拦截功能
// 主要功能：
//   - 实现 model.ToolCallingChatModel 接口
//   - 拦截未注册的工具调用，转换为技能调用
type ChatModelAdapter struct {
	logger        *zap.Logger
	chatModel     model.ToolCallingChatModel
	registeredMap map[string]bool  // 已注册的工具名称
	skillLoader   SkillLoader      // 技能加载器
	sessions      *session.Manager // 会话管理器，用于记录 token 用量
}

// Sentinel errors 定义包级别的错误常量
var (
	ErrNilConfig       = fmt.Errorf("配置不能为空")
	ErrCreateChatModel = fmt.Errorf("创建 ChatModel 失败")
	ErrNilAPIKey       = fmt.Errorf("API Key 不能为空")
)

func createChatModelConfig(logger *zap.Logger, cfg *config.Config) (apiKey, apiBase, modelName string, err error) {
	if cfg == nil {
		return "", "", "", ErrNilConfig
	}

	providerCfg := cfg.GetProvider(cfg.Agents.Defaults.Model)
	if providerCfg == nil || providerCfg.APIKey == "" {
		logger.Warn("未找到有效的 API Key，请设置环境变量")
		return "", "", "gpt-4o-mini", ErrNilAPIKey
	}

	apiBase = providerCfg.APIBase
	if apiBase == "" {
		apiBase = "https://api.openai.com/v1"
	}

	return providerCfg.APIKey, apiBase, cfg.Agents.Defaults.Model, nil
}

// NewChatModelAdapter 创建 ChatModel 适配器
func NewChatModelAdapter(logger *zap.Logger, cfg *config.Config, sessions *session.Manager) (*ChatModelAdapter, error) {
	apiKey, apiBase, modelName, err := createChatModelConfig(logger, cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNilConfig, err)
	}

	chatModel, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		APIKey:  apiKey,
		Model:   modelName,
		BaseURL: apiBase,
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
	// 调用底层 ChatModel
	response, err := a.chatModel.Generate(ctx, input, opts...)
	if err != nil {
		if a.logger != nil {
			a.logger.Error("调用 LLM 失败", zap.Error(err))
		}
		return nil, err
	}

	if a.logger != nil {
		a.logger.Info("原始响应",
			zap.String("内容", response.Content),
			zap.Int("工具调用数", len(response.ToolCalls)),
		)
	}

	// 拦截并转换工具调用
	a.interceptToolCalls(response)

	// 记录 token 用量到 session
	a.recordTokenUsage(ctx, response)

	return response, nil
}

// recordTokenUsage 记录 token 用量到 session
func (a *ChatModelAdapter) recordTokenUsage(ctx context.Context, response *schema.Message) {
	if a.sessions == nil {
		a.logger.Info("Session 为Nil ，跳过记录TokenUsage")
		return
	}

	sessionKey := ctx.Value(SessionKeyContextKey)
	if sessionKey == nil {
		return
	}

	key, ok := sessionKey.(string)
	if !ok || key == "" {
		return
	}

	usage := response.ResponseMeta.Usage
	if usage == nil {
		return
	}

	tokenUsage := session.TokenUsage{
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
	}

	sess := a.sessions.GetOrCreate(key)
	sess.AddMessageWithTokenUsage("assistant", response.Content, tokenUsage)
	a.logger.Info("记录Session TokenUsage", zap.Any("Usage", tokenUsage))
	if err := a.sessions.Save(sess); err != nil {
		if a.logger != nil {
			a.logger.Error("保存 token 用量失败", zap.Error(err))
		}
	}
}

// interceptToolCall 拦截工具调用，如果工具不存在则转换为技能调用
func (a *ChatModelAdapter) interceptToolCall(toolName string, argumentsJSON string) (string, string, error) {

	// 如果工具已注册，不拦截
	if a.isRegisteredTool(toolName) {
		a.logger.Info("工具已注册，不拦截", zap.String("名称", toolName))
		return toolName, argumentsJSON, nil
	}

	// 如果是已知技能，将工具调用转换为技能调用
	if a.isKnownSkill(toolName) {
		a.logger.Info("isKnownSkill 工具转换为技能调用", zap.String("名称", toolName))
		// 解析原始参数
		var originalArgs map[string]any
		if err := json.Unmarshal([]byte(argumentsJSON), &originalArgs); err != nil {
			originalArgs = make(map[string]any)
		}

		// 将原始参数包装成技能参数
		skillParams := map[string]any{
			"skill_name": toolName,
			"action":     originalArgs["action"],
		}

		// 移除 action 后的其他参数放入 params
		filteredParams := make(map[string]any)
		for k, v := range originalArgs {
			if k != "action" {
				filteredParams[k] = v
			}
		}
		if len(filteredParams) > 0 {
			skillParams["params"] = filteredParams
		}

		// 序列化新参数
		newArgsJSON, err := json.Marshal(skillParams)
		if err != nil {
			return toolName, argumentsJSON, err
		}

		// 返回 use_skill 作为工具名
		return "use_skill", string(newArgsJSON), nil
	}

	a.logger.Info("既不是工具也不是技能，保持原样", zap.String("名称", toolName))
	// 既不是工具也不是技能，保持原样（会在执行时报错）
	return toolName, argumentsJSON, nil
}

// interceptToolCalls 拦截并转换工具调用
func (a *ChatModelAdapter) interceptToolCalls(msg *schema.Message) {
	if len(msg.ToolCalls) == 0 {
		return
	}

	for i, tc := range msg.ToolCalls {
		if a.logger != nil {
			a.logger.Info("工具调用",
				zap.String("名称", tc.Function.Name),
				zap.String("参数", tc.Function.Arguments),
			)
		}
		newName, newArgs, err := a.interceptToolCall(tc.Function.Name, tc.Function.Arguments)

		if err != nil {
			continue
		}
		if newName != tc.Function.Name {
			if a.logger != nil {
				a.logger.Info("工具调用被拦截",
					zap.String("原始名称", tc.Function.Name),
					zap.String("新名称", newName),
					zap.String("新参数", newArgs),
				)
			}
			// 工具名被修改了，说明需要转换为技能调用
			msg.ToolCalls[i].Function.Name = newName
			msg.ToolCalls[i].Function.Arguments = newArgs
		}
	}
}

// Stream produces a response as a stream
// 注意：为避免流式响应解析问题，这里使用 Generate 并模拟流式输出
func (a *ChatModelAdapter) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	// 直接调用 Generate 获取完整响应
	msg, err := a.Generate(ctx, input, opts...)
	if err != nil {
		return nil, err
	}

	// 创建 StreamReader 并发送完整消息
	sr, sw := schema.Pipe[*schema.Message](1)
	go func() {
		defer sw.Close()
		sw.Send(msg, nil)
	}()

	return sr, nil
}

// WithTools returns a new adapter instance with the specified tools bound
func (a *ChatModelAdapter) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	// 绑定工具到底层 ChatModel
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
	}, nil
}
