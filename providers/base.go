package providers

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

// LLMProvider LLM 提供商接口 - 使用 eino 原生数据结构
type LLMProvider interface {
	// Chat 发送聊天完成请求
	// toolChoice: "auto" | "none" | "required" | {"type": "function", "function": {"name": "..."}}
	// 返回 eino 原生的 Message 结构
	Chat(ctx context.Context, messages []*schema.Message, tools []*schema.ToolInfo, toolChoice any, model string, maxTokens int, temperature float64) (*schema.Message, error)

	// ChatStream 发送流式聊天请求
	// 返回 eino 原生的 StreamReader
	ChatStream(ctx context.Context, messages []*schema.Message, tools []*schema.ToolInfo, toolChoice any, model string, maxTokens int, temperature float64) (*schema.StreamReader[*schema.Message], error)

	// GetDefaultModel 获取默认模型
	GetDefaultModel() string
}

// ChatOptions 聊天选项
type ChatOptions struct {
	Messages    []*schema.Message
	Tools       []*schema.ToolInfo
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
