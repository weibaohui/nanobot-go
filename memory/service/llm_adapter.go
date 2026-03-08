package service

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/config"
	"go.uber.org/zap"
)

// systemLLMClient 使用系统整体 LLM 配置的客户端
type systemLLMClient struct {
	cfg    *config.Config
	logger *zap.Logger
}

// NewSystemLLMClient 创建使用系统配置的 LLM 客户端
func NewSystemLLMClient(cfg *config.Config, logger *zap.Logger) LLMClient {
	return &systemLLMClient{
		cfg:    cfg,
		logger: logger,
	}
}

// Complete 调用 LLM 生成文本
func (c *systemLLMClient) Complete(ctx context.Context, prompt string, systemPrompt string) (string, error) {
	// 获取模型名称（使用配置的或系统默认的）
	modelName := c.cfg.Memory.Summarization.Model
	if modelName == "" {
		modelName = c.cfg.Agents.Defaults.Model
	}

	// 获取 API 配置
	apiKey := c.cfg.GetAPIKey(modelName)
	apiBase := c.cfg.GetAPIBase(modelName)

	if apiKey == "" {
		return "", fmt.Errorf("未找到模型 %s 的 API Key", modelName)
	}

	c.logger.Debug("调用 LLM 进行总结",
		zap.String("model", modelName),
		zap.String("api_base", apiBase),
		zap.Int("prompt_length", len(prompt)),
	)

	// 这里使用系统的 ChatModel 创建方式
	// 由于 ChatModel 初始化在 agent 包，这里简化处理
	// 实际项目中可能需要更好的解耦

	// 简化实现：直接返回提示词的摘要
	// TODO: 接入实际的 LLM 调用
	return c.mockComplete(ctx, prompt, systemPrompt)
}

// mockComplete 模拟 LLM 调用（实际项目中替换为真实调用）
func (c *systemLLMClient) mockComplete(ctx context.Context, prompt string, systemPrompt string) (string, error) {
	// 这是一个简化实现
	// 实际应该使用 agent.NewChatModel 创建客户端并调用

	c.logger.Warn("使用模拟 LLM 完成，请实现真实的 LLM 调用",
		zap.String("model", c.cfg.Memory.Summarization.Model),
	)

	// 返回一个简单的 JSON 格式的总结
	return `{
		"summary": "这是一个模拟的总结，请实现真实的 LLM 调用",
		"key_points": "要点1\n要点2\n要点3"
	}`, nil
}

// EinoLLMClient 使用 Eino ChatModel 的客户端
type EinoLLMClient struct {
	chatModel model.ChatModel
	logger    *zap.Logger
}

// NewEinoLLMClient 创建基于 Eino ChatModel 的客户端
func NewEinoLLMClient(chatModel model.ChatModel, logger *zap.Logger) LLMClient {
	return &EinoLLMClient{
		chatModel: chatModel,
		logger:    logger,
	}
}

// Complete 调用 LLM 生成文本
func (c *EinoLLMClient) Complete(ctx context.Context, prompt string, systemPrompt string) (string, error) {
	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: systemPrompt,
		},
		{
			Role:    schema.User,
			Content: prompt,
		},
	}

	response, err := c.chatModel.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("llm generate failed: %w", err)
	}

	return response.Content, nil
}
