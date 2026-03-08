package service

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/config"
	"go.uber.org/zap"
)

// LLMClient LLM 客户端接口
type LLMClient interface {
	// Complete 调用 LLM 生成文本
	Complete(ctx context.Context, prompt string, systemPrompt string) (string, error)
}

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

	if apiBase == "" {
		apiBase = "https://api.openai.com/v1"
	}

	c.logger.Debug("调用 LLM 进行总结",
		zap.String("model", modelName),
		zap.String("api_base", apiBase),
		zap.Int("prompt_length", len(prompt)),
	)

	// 创建 ChatModel
	temp := float32(c.cfg.Memory.Summarization.Temperature)
	maxTokens := c.cfg.Memory.Summarization.MaxTokens
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:      apiKey,
		Model:       modelName,
		BaseURL:     apiBase,
		Temperature: &temp,
		MaxTokens:   &maxTokens,
	})
	if err != nil {
		return "", fmt.Errorf("创建 ChatModel 失败: %w", err)
	}

	// 构建消息
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

	// 调用 LLM
	response, err := chatModel.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("LLM 生成失败: %w", err)
	}

	return response.Content, nil
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
