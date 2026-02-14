package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
)

// LiteLLMProvider 使用 OpenAI 兼容 API 的 LLM 提供商
type LiteLLMProvider struct {
	logger       *zap.Logger
	apiKey       string
	apiBase      string
	defaultModel string
	extraHeaders map[string]string
	httpClient   *http.Client
}

// NewLiteLLMProvider 创建 LiteLLM 提供商
func NewLiteLLMProvider(logger *zap.Logger, apiKey, apiBase, defaultModel string, extraHeaders map[string]string) *LiteLLMProvider {
	if apiBase == "" {
		apiBase = "https://api.openai.com/v1"
	}
	if defaultModel == "" {
		defaultModel = "gpt-4o"
	}

	return &LiteLLMProvider{
		logger:       logger,
		apiKey:       apiKey,
		apiBase:      strings.TrimSuffix(apiBase, "/"),
		defaultModel: defaultModel,
		extraHeaders: extraHeaders,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Chat 发送聊天完成请求 - 返回 eino 原生 Message
func (p *LiteLLMProvider) Chat(ctx context.Context, messages []*schema.Message, tools []*schema.ToolInfo, toolChoice any, model string, maxTokens int, temperature float64) (*schema.Message, error) {
	if model == "" {
		model = p.defaultModel
	}
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	if temperature <= 0 {
		temperature = 0.7
	}

	// 构建请求体
	reqBody := map[string]any{
		"model":      p.resolveModel(model),
		"messages":   convertMessagesToAPIFormat(messages),
		"max_tokens": maxTokens,
	}

	// 只有在 temperature > 0 时才设置（某些模型不支持）
	if temperature > 0 {
		reqBody["temperature"] = temperature
	}

	// 添加工具
	if len(tools) > 0 {
		reqBody["tools"] = convertToolsToAPIFormat(tools)
		// 使用传入的 toolChoice，如果为 nil 则使用 "auto"
		if toolChoice != nil {
			reqBody["tool_choice"] = toolChoice
		} else {
			reqBody["tool_choice"] = "auto"
		}
	}

	// 序列化请求
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建请求
	reqURL := p.apiBase + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	// 添加额外头部
	for k, v := range p.extraHeaders {
		req.Header.Set(k, v)
	}

	// 发送请求
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return &schema.Message{
			Role:    schema.Assistant,
			Content: fmt.Sprintf("API 错误 (状态码 %d): %s", resp.StatusCode, string(body)),
		}, nil
	}

	// 解析响应并转换为 eino Message
	return p.parseResponseToMessage(body)
}

// toolCallAccumulator 用于累积流式工具调用
type toolCallAccumulator struct {
	id        string
	name      string
	arguments strings.Builder
}

// ChatStream 发送流式聊天请求 - 返回 eino 原生 StreamReader
func (p *LiteLLMProvider) ChatStream(ctx context.Context, messages []*schema.Message, tools []*schema.ToolInfo, toolChoice any, model string, maxTokens int, temperature float64) (*schema.StreamReader[*schema.Message], error) {
	if model == "" {
		model = p.defaultModel
	}
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	if temperature <= 0 {
		temperature = 0.7
	}

	// 构建请求体
	reqBody := map[string]any{
		"model":      p.resolveModel(model),
		"messages":   convertMessagesToAPIFormat(messages),
		"max_tokens": maxTokens,
		"stream":     true, // 启用流式
	}

	if temperature > 0 {
		reqBody["temperature"] = temperature
	}

	if len(tools) > 0 {
		reqBody["tools"] = convertToolsToAPIFormat(tools)
		// 使用传入的 toolChoice，如果为 nil 则使用 "auto"
		if toolChoice != nil {
			reqBody["tool_choice"] = toolChoice
		} else {
			reqBody["tool_choice"] = "auto"
		}
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	reqURL := p.apiBase + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	for k, v := range p.extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}

	// 创建 eino StreamReader/StreamWriter
	sr, sw := schema.Pipe[*schema.Message](100)

	go func() {
		defer sw.Close()
		defer resp.Body.Close()

		// 工具调用累积器 - 按 ID 索引
		toolCallAccumulators := make(map[int]*toolCallAccumulator)
		var toolCallIndex int

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				sw.Send(&schema.Message{
					Role:    schema.Assistant,
					Content: fmt.Sprintf("流式读取错误: %v", err),
				}, nil)
				return
			}

			line = strings.TrimSpace(line)
			if line == "" || !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				// 流结束，发送累积的完整工具调用
				if len(toolCallAccumulators) > 0 {
					msg := &schema.Message{
						Role:      schema.Assistant,
						ToolCalls: make([]schema.ToolCall, len(toolCallAccumulators)),
					}
					for idx, acc := range toolCallAccumulators {
						msg.ToolCalls[idx] = schema.ToolCall{
							ID: acc.id,
							Function: schema.FunctionCall{
								Name:      acc.name,
								Arguments: acc.arguments.String(),
							},
						}
					}
					sw.Send(msg, nil)
				}
				return
			}

			var streamResp struct {
				Choices []struct {
					Delta struct {
						Content   string `json:"content"`
						ToolCalls []struct {
							Index    int    `json:"index"`
							ID       string `json:"id"`
							Type     string `json:"type"`
							Function struct {
								Name      string `json:"name"`
								Arguments string `json:"arguments"`
							} `json:"function"`
						} `json:"tool_calls"`
					} `json:"delta"`
					FinishReason string `json:"finish_reason"`
				} `json:"choices"`
			}

			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				continue
			}

			if len(streamResp.Choices) == 0 {
				continue
			}

			choice := streamResp.Choices[0]

			// 处理工具调用 - 累积参数
			if len(choice.Delta.ToolCalls) > 0 {
				for _, tc := range choice.Delta.ToolCalls {
					idx := tc.Index
					if idx == 0 && tc.ID != "" {
						// 新工具调用开始
						toolCallAccumulators[idx] = &toolCallAccumulator{
							id:   tc.ID,
							name: tc.Function.Name,
						}
						toolCallIndex = idx
					}

					// 累积参数
					if acc, ok := toolCallAccumulators[idx]; ok {
						if tc.Function.Name != "" && acc.name == "" {
							acc.name = tc.Function.Name
						}
						if tc.Function.Arguments != "" {
							acc.arguments.WriteString(tc.Function.Arguments)
						}
					} else if tc.Function.Arguments != "" {
						// 可能是继续累积
						if acc, ok := toolCallAccumulators[toolCallIndex]; ok {
							acc.arguments.WriteString(tc.Function.Arguments)
						}
					}
				}
			}

			// 发送内容增量
			if choice.Delta.Content != "" {
				sw.Send(&schema.Message{
					Role:    schema.Assistant,
					Content: choice.Delta.Content,
				}, nil)
			}

			// 如果有 finish_reason，发送完整的工具调用
			if choice.FinishReason != "" && len(toolCallAccumulators) > 0 {
				msg := &schema.Message{
					Role:      schema.Assistant,
					ToolCalls: make([]schema.ToolCall, len(toolCallAccumulators)),
				}
				for idx, acc := range toolCallAccumulators {
					msg.ToolCalls[idx] = schema.ToolCall{
						ID: acc.id,
						Function: schema.FunctionCall{
							Name:      acc.name,
							Arguments: acc.arguments.String(),
						},
					}
				}
				sw.Send(msg, nil)
			}
		}
	}()

	return sr, nil
}

// resolveModel 解析模型名称
func (p *LiteLLMProvider) resolveModel(model string) string {
	// SiliconFlow、OpenRouter 等网关需要完整的模型名称（包含前缀）
	// 直接返回原始模型名称
	return model
}

// parseResponseToMessage 解析 API 响应为 eino Message
func (p *LiteLLMProvider) parseResponseToMessage(body []byte) (*schema.Message, error) {
	var resp struct {
		Choices []struct {
			Message struct {
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
				ToolCalls        []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if len(resp.Choices) == 0 {
		return &schema.Message{
			Role:    schema.Assistant,
			Content: "API 返回空响应",
		}, nil
	}

	choice := resp.Choices[0]
	msg := &schema.Message{
		Role:    schema.Assistant,
		Content: choice.Message.Content,
	}

	// 解析工具调用 - Arguments 保持原始 JSON 字符串
	if len(choice.Message.ToolCalls) > 0 {
		msg.ToolCalls = make([]schema.ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			msg.ToolCalls[i] = schema.ToolCall{
				ID: tc.ID,
				Function: schema.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments, // 保持原始字符串，不做解析
				},
			}
		}
	}

	// 设置 Token 使用情况
	msg.ResponseMeta = &schema.ResponseMeta{
		Usage: &schema.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	return msg, nil
}

// convertMessagesToAPIFormat 将 eino Message 转换为 API 请求格式
func convertMessagesToAPIFormat(messages []*schema.Message) []map[string]any {
	result := make([]map[string]any, 0, len(messages))

	for _, msg := range messages {
		m := map[string]any{
			"role":    string(msg.Role),
			"content": msg.Content,
		}

		// 处理工具调用
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

		// 处理工具响应
		if msg.ToolCallID != "" {
			m["tool_call_id"] = msg.ToolCallID
		}

		// 处理名称
		if msg.Name != "" {
			m["name"] = msg.Name
		}

		result = append(result, m)
	}

	return result
}

// convertToolsToAPIFormat 将 eino ToolInfo 转换为 API 请求格式
func convertToolsToAPIFormat(tools []*schema.ToolInfo) []map[string]any {
	result := make([]map[string]any, len(tools))

	for i, tool := range tools {
		t := map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        tool.Name,
				"description": tool.Desc,
			},
		}

		// 使用 ToJSONSchema 转换参数
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

// GetDefaultModel 获取默认模型
func (p *LiteLLMProvider) GetDefaultModel() string {
	return p.defaultModel
}
