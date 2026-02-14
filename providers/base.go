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

// StreamChunk 流式响应片段
type StreamChunk struct {
	Content      string            `json:"content"`
	Delta        string            `json:"delta"`        // 增量内容
	ToolCalls    []ToolCallRequest `json:"toolCalls"`    // 工具调用（可能有）
	FinishReason string            `json:"finishReason"` // 完成原因
	Done         bool              `json:"done"`         // 是否完成
}

// LLMProvider LLM 提供商接口
type LLMProvider interface {
	// Chat 发送聊天完成请求
	// toolChoice: "auto" | "none" | "required" | {"type": "function", "function": {"name": "..."}}
	Chat(ctx context.Context, messages []map[string]any, tools []map[string]any, toolChoice any, model string, maxTokens int, temperature float64) (*LLMResponse, error)

	// ChatStream 发送流式聊天请求
	ChatStream(ctx context.Context, messages []map[string]any, tools []map[string]any, toolChoice any, model string, maxTokens int, temperature float64) (<-chan StreamChunk, error)

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
