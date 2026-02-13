package eino_adapter

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/providers"
)

// ProviderAdapter adapts providers.LLMProvider to eino's ToolCallingChatModel interface
type ProviderAdapter struct {
	provider providers.LLMProvider
	model    string
	tools    []*schema.ToolInfo
}

// NewProviderAdapter creates a new adapter that wraps nanobot-go's LLMProvider
func NewProviderAdapter(provider providers.LLMProvider, modelName string) *ProviderAdapter {
	if modelName == "" {
		modelName = provider.GetDefaultModel()
	}
	return &ProviderAdapter{
		provider: provider,
		model:    modelName,
	}
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

	// Call the provider
	response, err := a.provider.Chat(ctx, messages, tools, modelName, maxTokens, float64(temperature))
	if err != nil {
		return nil, err
	}

	// Convert response to eino format
	return convertToEinoMessage(response), nil
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
	return &ProviderAdapter{
		provider: a.provider,
		model:    a.model,
		tools:    tools,
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
