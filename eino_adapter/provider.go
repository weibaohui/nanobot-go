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
	toolChoice    any              // 工具选择策略
	registeredMap map[string]bool  // 已注册的工具名称
	skillLoader   SkillLoader      // 技能加载器
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

// interceptToolCall 拦截工具调用，如果工具不存在则转换为技能调用
func (a *ProviderAdapter) interceptToolCall(toolName string, arguments map[string]any) (string, map[string]any, error) {
	// 如果工具已注册，不拦截
	if a.isRegisteredTool(toolName) {
		return toolName, arguments, nil
	}

	// 如果是已知技能，将工具调用转换为技能调用
	if a.isKnownSkill(toolName) {
		// 将原始参数包装成技能参数
		skillParams := map[string]any{
			"skill_name": toolName,
			"action":     arguments["action"],
		}

		// 移除 action 后的其他参数放入 params
		filteredParams := make(map[string]any)
		for k, v := range arguments {
			if k != "action" {
				filteredParams[k] = v
			}
		}
		if len(filteredParams) > 0 {
			skillParams["params"] = filteredParams
		}

		// 返回 use_skill 作为工具名
		return "use_skill", skillParams, nil
	}

	// 既不是工具也不是技能，保持原样（会在执行时报错）
	return toolName, arguments, nil
}

// Generate produces a complete model response
func (a *ProviderAdapter) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	options := model.GetCommonOptions(&model.Options{}, opts...)

	// Convert eino messages to nanobot-go format
	messages := convertToProviderMessages(input)

	// Convert bound tools to provider format
	var tools []map[string]any
	if len(a.tools) > 0 {
		tools = convertToolInfoToProviderFormat(a.tools)
		if a.logger != nil {
			a.logger.Info("Generate 使用绑定的工具",
				zap.Int("tools_count", len(tools)),
			)
		}
	} else if a.logger != nil {
		a.logger.Warn("Generate 没有绑定的工具")
	}

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

	// Call the provider
	response, err := a.provider.Chat(ctx, messages, tools, toolChoice, modelName, maxTokens, float64(temperature))
	if err != nil {
		if a.logger != nil {
			a.logger.Error("调用 LLM 失败", zap.Error(err))
		}
		return nil, err
	}

	if a.logger != nil {
		a.logger.Info("原始响应",
			zap.Any("内容", response),
		)
	}
	// 拦截并转换工具调用
	a.interceptToolCalls(response)

	// Convert response to eino format
	return convertToEinoMessage(response), nil
}

// interceptToolCalls 拦截并转换工具调用
func (a *ProviderAdapter) interceptToolCalls(response *providers.LLMResponse) {

	if len(response.ToolCalls) == 0 {
		return
	}

	for i, tc := range response.ToolCalls {
		if a.logger != nil {
			a.logger.Info("工具调用",
				zap.String("名称", tc.Name),
				zap.Any("参数", tc.Arguments),
			)
		}
		newName, newArgs, err := a.interceptToolCall(tc.Name, tc.Arguments)
		if err != nil {
			continue
		}
		if newName != tc.Name {
			if a.logger != nil {
				a.logger.Info("工具调用被拦截",
					zap.String("原始名称", tc.Name),
					zap.String("新名称", newName),
					zap.Any("新参数", newArgs),
				)
			}
			// 工具名被修改了，说明需要转换为技能调用
			response.ToolCalls[i].Name = newName
			response.ToolCalls[i].Arguments = newArgs
		}
	}
}

// Stream produces a response as a stream
// Note: The current provider interface doesn't support streaming, so we simulate it
func (a *ProviderAdapter) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	// Generate the full response
	msg, err := a.Generate(ctx, input, opts...)
	if err != nil {
		return nil, err
	}

	// Create a stream that yields the complete message
	sr, sw := schema.Pipe[*schema.Message](1)
	go func() {
		defer sw.Close()
		sw.Send(msg, nil)
	}()

	return sr, nil
}

// WithTools returns a new adapter instance with the specified tools bound
func (a *ProviderAdapter) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	if a.logger != nil {
		a.logger.Info("WithTools 被调用",
			zap.Int("tools_count", len(tools)),
		)
	}
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

// BindTools binds tools to the model (alias for WithTools for compatibility)
func (a *ProviderAdapter) BindTools(tools []*schema.ToolInfo) error {
	a.tools = tools
	return nil
}

// convertToProviderMessages converts eino messages to provider message format
func convertToProviderMessages(messages []*schema.Message) []map[string]any {
	result := make([]map[string]any, 0, len(messages))

	for _, msg := range messages {
		m := map[string]any{
			"role":    string(msg.Role),
			"content": msg.Content,
		}

		// Handle tool calls
		if len(msg.ToolCalls) > 0 {
			toolCalls := make([]map[string]any, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				toolCalls[i] = map[string]any{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]any{
						"name":      tc.Function.Name,
						"arguments": tc.Function.Arguments,
					},
				}
			}
			m["tool_calls"] = toolCalls
		}

		// Handle tool responses
		if msg.ToolCallID != "" {
			m["tool_call_id"] = msg.ToolCallID
		}

		// Handle name
		if msg.Name != "" {
			m["name"] = msg.Name
		}

		result = append(result, m)
	}

	return result
}

// convertToEinoMessage converts provider response to eino message format
func convertToEinoMessage(response *providers.LLMResponse) *schema.Message {
	msg := &schema.Message{
		Role:    schema.Assistant,
		Content: response.Content,
	}

	// Convert tool calls
	if len(response.ToolCalls) > 0 {
		msg.ToolCalls = make([]schema.ToolCall, len(response.ToolCalls))
		for i, tc := range response.ToolCalls {
			// Convert arguments to JSON string
			argsBytes, _ := json.Marshal(tc.Arguments)
			msg.ToolCalls[i] = schema.ToolCall{
				ID: tc.ID,
				Function: schema.FunctionCall{
					Name:      tc.Name,
					Arguments: string(argsBytes),
				},
			}
		}
	}

	// Add response metadata if available
	if len(response.Usage) > 0 {
		msg.ResponseMeta = &schema.ResponseMeta{
			Usage: &schema.TokenUsage{
				PromptTokens:     response.Usage["prompt_tokens"],
				CompletionTokens: response.Usage["completion_tokens"],
				TotalTokens:      response.Usage["total_tokens"],
			},
		}
	}

	return msg
}

// convertToolInfoToProviderFormat converts eino ToolInfo to provider format
func convertToolInfoToProviderFormat(tools []*schema.ToolInfo) []map[string]any {
	result := make([]map[string]any, len(tools))

	for i, tool := range tools {
		t := map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        tool.Name,
				"description": tool.Desc,
			},
		}

		// Convert parameters using ToJSONSchema
		if tool.ParamsOneOf != nil {
			jsonSchema, err := tool.ParamsOneOf.ToJSONSchema()
			if err == nil && jsonSchema != nil {
				t["function"].(map[string]any)["parameters"] = jsonSchema
			}
		}

		result[i] = t
	}

	return result
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
