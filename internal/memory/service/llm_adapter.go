package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
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
	db     *gorm.DB
}

// NewSystemLLMClient 创建使用系统配置的 LLM 客户端
func NewSystemLLMClient(cfg *config.Config, logger *zap.Logger, db *gorm.DB) LLMClient {
	return &systemLLMClient{
		cfg:    cfg,
		logger: logger,
		db:     db,
	}
}

// Complete 调用 LLM 生成文本
func (c *systemLLMClient) Complete(ctx context.Context, prompt string, systemPrompt string) (string, error) {
	// 获取模型名称（使用配置的或系统默认的）
	modelName := c.cfg.Memory.Summarization.Model
	if modelName == "" {
		modelName = c.cfg.Agents.Defaults.Model
	}

	// 从数据库获取 API 配置
	apiKey, apiBase, err := c.getAPIConfigFromDB(modelName)
	if err != nil {
		return "", fmt.Errorf("从数据库获取 API 配置失败: %w", err)
	}

	if apiKey == "" {
		return "", fmt.Errorf("未找到模型 %s 的 API Key", modelName)
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

// getAPIConfigFromDB 从数据库获取 API 配置
func (c *systemLLMClient) getAPIConfigFromDB(modelName string) (string, string, error) {
	if c.db == nil {
		return "", "", fmt.Errorf("数据库未初始化")
	}

	// 查找支持该模型的默认 Provider
	var provider models.LLMProvider
	err := c.db.Where("is_default = ? AND is_active = ?", true, true).
		First(&provider).Error
	if err != nil {
		// 如果没有默认 Provider，尝试查找任意活跃的 Provider
		err = c.db.Where("is_active = ?", true).
			Order("priority DESC").
			First(&provider).Error
		if err != nil {
			return "", "", fmt.Errorf("未找到可用的 Provider: %w", err)
		}
	}

	// 检查 Provider 是否支持该模型
	if provider.SupportedModels != "" {
		var supportedModels []map[string]interface{}
		if err := json.Unmarshal([]byte(provider.SupportedModels), &supportedModels); err == nil {
			modelSupported := false
			for _, m := range supportedModels {
				if id, ok := m["id"].(string); ok && id == modelName {
					modelSupported = true
					break
				}
			}
			if !modelSupported && provider.DefaultModel != "" {
				// 使用 Provider 的默认模型
				c.logger.Debug("模型不支持，使用 Provider 默认模型",
					zap.String("requested", modelName),
					zap.String("using", provider.DefaultModel))
			}
		}
	}

	return provider.APIKey, provider.APIBase, nil
}

// EinoLLMClient 使用 Eino ChatModel 的客户端
type EinoLLMClient struct {
	chatModel model.ToolCallingChatModel
	logger    *zap.Logger
}

// NewEinoLLMClient 创建基于 Eino ChatModel 的客户端
func NewEinoLLMClient(chatModel model.ToolCallingChatModel, logger *zap.Logger) LLMClient {
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
