package eino_adapter

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/providers"
	"go.uber.org/zap"
)

// SkillLoader 技能加载函数类型
type SkillLoader func(name string) string

// ProviderAdapter adapts providers.LLMProvider to eino's ToolCallingChatModel interface
type ProviderAdapter struct {
	logger        *zap.Logger
	provider      providers.LLMProvider
	model         string
	tools         []*schema.ToolInfo
	toolChoice    any             // 工具选择策略
	registeredMap map[string]bool // 已注册的工具名称
	skillLoader   SkillLoader     // 技能加载器
}

// NewProviderAdapter creates a new adapter that wraps nanobot-go's LLMProvider
func NewProviderAdapter(logger *zap.Logger, provider providers.LLMProvider, modelName string) *ProviderAdapter {
	if modelName == "" {
		modelName = provider.GetDefaultModel()
	}
	return &ProviderAdapter{
		logger:        logger,
		provider:      provider,
		model:         modelName,
		registeredMap: make(map[string]bool),
	}
}

// SetSkillLoader 设置技能加载器
func (a *ProviderAdapter) SetSkillLoader(loader SkillLoader) {
	a.skillLoader = loader
}

// SetRegisteredTools 设置已注册的工具名称列表
func (a *ProviderAdapter) SetRegisteredTools(names []string) {
	a.registeredMap = make(map[string]bool)
	for _, name := range names {
		a.registeredMap[name] = true
	}
}

// isRegisteredTool 检查工具是否已注册
func (a *ProviderAdapter) isRegisteredTool(name string) bool {
	if a.registeredMap == nil {
		return false
	}
	return a.registeredMap[name]
}

// isKnownSkill 检查是否是已知技能
func (a *ProviderAdapter) isKnownSkill(name string) bool {
	if a.skillLoader == nil {
		return false
	}
	content := a.skillLoader(name)
	return content != ""
}

// Generate produces a complete model response
func (a *ProviderAdapter) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	options := model.GetCommonOptions(&model.Options{}, opts...)

	// Get options
	modelName := a.model
	if options.Model != nil && *options.Model != "" {
		modelName = *options.Model
	}

	maxTokens := 4096
	if options.MaxTokens != nil {
		maxTokens = *options.MaxTokens
	}

	temperature := float32(0.7)
	if options.Temperature != nil {
		temperature = *options.Temperature
	}

	// 处理 toolChoice - 转换 eino 格式到 OpenAI 格式
	toolChoice := a.toolChoice
	if options.ToolChoice != nil {
		toolChoice = convertToolChoiceToOpenAIFormat(*options.ToolChoice)
	}

	// 调用 provider - 直接传递 eino 原生类型
	response, err := a.provider.Chat(ctx, input, a.tools, toolChoice, modelName, maxTokens, float64(temperature))
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

	return response, nil
}

// interceptToolCall 拦截工具调用，如果工具不存在则转换为技能调用
func (a *ProviderAdapter) interceptToolCall(toolName string, argumentsJSON string) (string, string, error) {
	// 如果工具已注册，不拦截
	if a.isRegisteredTool(toolName) {
		return toolName, argumentsJSON, nil
	}

	// 如果是已知技能，将工具调用转换为技能调用
	if a.isKnownSkill(toolName) {
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

	// 既不是工具也不是技能，保持原样（会在执行时报错）
	return toolName, argumentsJSON, nil
}

// interceptToolCalls 拦截并转换工具调用
func (a *ProviderAdapter) interceptToolCalls(msg *schema.Message) {
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
func (a *ProviderAdapter) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	options := model.GetCommonOptions(&model.Options{}, opts...)

	// Get options
	modelName := a.model
	if options.Model != nil && *options.Model != "" {
		modelName = *options.Model
	}

	maxTokens := 4096
	if options.MaxTokens != nil {
		maxTokens = *options.MaxTokens
	}

	temperature := float32(0.7)
	if options.Temperature != nil {
		temperature = *options.Temperature
	}

	// 处理 toolChoice
	toolChoice := a.toolChoice
	if options.ToolChoice != nil {
		toolChoice = convertToolChoiceToOpenAIFormat(*options.ToolChoice)
	}

	// 调用 provider 的流式接口
	sr, err := a.provider.ChatStream(ctx, input, a.tools, toolChoice, modelName, maxTokens, float64(temperature))
	if err != nil {
		return nil, err
	}

	// 创建新的 StreamReader 用于拦截处理
	interceptedSR, interceptedSW := schema.Pipe[*schema.Message](100)

	go func() {
		defer interceptedSW.Close()

		for {
			msg, err := sr.Recv()
			if err != nil {
				return
			}

			// 对包含工具调用的消息进行拦截处理
			if len(msg.ToolCalls) > 0 {
				a.interceptToolCalls(msg)
			}

			interceptedSW.Send(msg, nil)
		}
	}()

	return interceptedSR, nil
}

// WithTools returns a new adapter instance with the specified tools bound
func (a *ProviderAdapter) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return &ProviderAdapter{
		logger:        a.logger,
		provider:      a.provider,
		model:         a.model,
		tools:         tools,
		toolChoice:    a.toolChoice,
		registeredMap: a.registeredMap,
		skillLoader:   a.skillLoader,
	}, nil
}

// BindTools binds tools to the model
func (a *ProviderAdapter) BindTools(tools []*schema.ToolInfo) error {
	a.tools = tools
	return nil
}

// convertToolChoiceToOpenAIFormat converts eino ToolChoice to OpenAI format
func convertToolChoiceToOpenAIFormat(toolChoice schema.ToolChoice) string {
	switch toolChoice {
	case schema.ToolChoiceForbidden:
		return "none"
	case schema.ToolChoiceAllowed:
		return "auto"
	case schema.ToolChoiceForced:
		return "required"
	default:
		return string(toolChoice)
	}
}
