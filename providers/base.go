package providers

import "context"

// ToolCallRequest LLM 的工具调用请求
type ToolCallRequest struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// LLMResponse LLM 响应
type LLMResponse struct {
	Content          string             `json:"content"`
	ToolCalls        []ToolCallRequest  `json:"toolCalls"`
	FinishReason     string             `json:"finishReason"`
	Usage            map[string]int     `json:"usage"`
	ReasoningContent string             `json:"reasoningContent,omitempty"`
}

// HasToolCalls 检查响应是否包含工具调用
func (r *LLMResponse) HasToolCalls() bool {
	return len(r.ToolCalls) > 0
}

// LLMProvider LLM 提供商接口
type LLMProvider interface {
	// Chat 发送聊天完成请求
	Chat(ctx context.Context, messages []map[string]any, tools []map[string]any, model string, maxTokens int, temperature float64) (*LLMResponse, error)

	// GetDefaultModel 获取默认模型
	GetDefaultModel() string
}

// ChatOptions 聊天选项
type ChatOptions struct {
	Messages    []map[string]any
	Tools       []map[string]any
	Model       string
	MaxTokens   int
	Temperature float64
}

// DefaultChatOptions 返回默认聊天选项
func DefaultChatOptions() *ChatOptions {
	return &ChatOptions{
		MaxTokens:   4096,
		Temperature: 0.7,
	}
}
