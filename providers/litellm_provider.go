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
)

// LiteLLMProvider 使用 OpenAI 兼容 API 的 LLM 提供商
type LiteLLMProvider struct {
	apiKey       string
	apiBase      string
	defaultModel string
	extraHeaders map[string]string
	httpClient   *http.Client
}

// NewLiteLLMProvider 创建 LiteLLM 提供商
func NewLiteLLMProvider(apiKey, apiBase, defaultModel string, extraHeaders map[string]string) *LiteLLMProvider {
	if apiBase == "" {
		apiBase = "https://api.openai.com/v1"
	}
	if defaultModel == "" {
		defaultModel = "gpt-4o"
	}

	return &LiteLLMProvider{
		apiKey:       apiKey,
		apiBase:      strings.TrimSuffix(apiBase, "/"),
		defaultModel: defaultModel,
		extraHeaders: extraHeaders,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// ChatStream 发送流式聊天请求
func (p *LiteLLMProvider) ChatStream(ctx context.Context, messages []map[string]any, tools []map[string]any, model string, maxTokens int, temperature float64) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 100)

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
		"messages":   messages,
		"max_tokens": maxTokens,
		"stream":     true, // 启用流式
	}

	if temperature > 0 {
		reqBody["temperature"] = temperature
	}

	if len(tools) > 0 {
		reqBody["tools"] = tools
		reqBody["tool_choice"] = "auto"
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		close(ch)
		return ch, fmt.Errorf("序列化请求失败: %w", err)
	}

	reqURL := p.apiBase + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(jsonData))
	if err != nil {
		close(ch)
		return ch, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	for k, v := range p.extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		close(ch)
		return ch, fmt.Errorf("请求失败: %w", err)
	}

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				ch <- StreamChunk{Done: true, FinishReason: "error"}
				return
			}

			line = strings.TrimSpace(line)
			if line == "" || !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				ch <- StreamChunk{Done: true, FinishReason: "stop"}
				return
			}

			var streamResp struct {
				Choices []struct {
					Delta struct {
						Content   string `json:"content"`
						ToolCalls []struct {
							ID   string `json:"id"`
							Type string `json:"type"`
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
			chunk := StreamChunk{
				Delta:        choice.Delta.Content,
				Content:      choice.Delta.Content,
				FinishReason: choice.FinishReason,
				Done:         choice.FinishReason != "",
			}

			// 处理工具调用
			if len(choice.Delta.ToolCalls) > 0 {
				for _, tc := range choice.Delta.ToolCalls {
					args := make(map[string]any)
					if tc.Function.Arguments != "" {
						json.Unmarshal([]byte(tc.Function.Arguments), &args)
					}
					chunk.ToolCalls = append(chunk.ToolCalls, ToolCallRequest{
						ID:        tc.ID,
						Name:      tc.Function.Name,
						Arguments: args,
					})
				}
			}

			if chunk.Delta != "" || chunk.Done {
				ch <- chunk
			}
		}
	}()

	return ch, nil
}

// Chat 发送聊天完成请求
func (p *LiteLLMProvider) Chat(ctx context.Context, messages []map[string]any, tools []map[string]any, model string, maxTokens int, temperature float64) (*LLMResponse, error) {
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
		"messages":   messages,
		"max_tokens": maxTokens,
	}

	// 只有在 temperature > 0 时才设置（某些模型不支持）
	if temperature > 0 {
		reqBody["temperature"] = temperature
	}

	// 添加工具
	if len(tools) > 0 {
		reqBody["tools"] = tools
		reqBody["tool_choice"] = "auto"
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
		return &LLMResponse{
			Content:      fmt.Sprintf("API 错误 (状态码 %d): %s", resp.StatusCode, string(body)),
			FinishReason: "error",
		}, nil
	}

	// 解析响应
	return p.parseResponse(body)
}

// resolveModel 解析模型名称
func (p *LiteLLMProvider) resolveModel(model string) string {
	// SiliconFlow、OpenRouter 等网关需要完整的模型名称（包含前缀）
	// 直接返回原始模型名称
	return model
}

// parseResponse 解析 API 响应
func (p *LiteLLMProvider) parseResponse(body []byte) (*LLMResponse, error) {
	var resp struct {
		Choices []struct {
			Message struct {
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
				ToolCalls        []struct {
					ID   string `json:"id"`
					Type string `json:"type"`
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
		return &LLMResponse{
			Content:      "API 返回空响应",
			FinishReason: "empty",
		}, nil
	}

	choice := resp.Choices[0]
	result := &LLMResponse{
		Content:      choice.Message.Content,
		FinishReason: choice.FinishReason,
		Usage: map[string]int{
			"prompt_tokens":     resp.Usage.PromptTokens,
			"completion_tokens": resp.Usage.CompletionTokens,
			"total_tokens":      resp.Usage.TotalTokens,
		},
		ReasoningContent: choice.Message.ReasoningContent,
	}

	// 解析工具调用
	for _, tc := range choice.Message.ToolCalls {
		args := make(map[string]any)
		if tc.Function.Arguments != "" {
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				args = map[string]any{"raw": tc.Function.Arguments}
			}
		}
		result.ToolCalls = append(result.ToolCalls, ToolCallRequest{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: args,
		})
	}

	return result, nil
}

// GetDefaultModel 获取默认模型
func (p *LiteLLMProvider) GetDefaultModel() string {
	return p.defaultModel
}
