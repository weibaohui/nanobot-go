package embedding

import (
	"context"
	"fmt"

	"github.com/weibaohui/nanobot-go/internal/models"
	"gorm.io/gorm"
)

// Config 嵌入模型完整配置
type Config struct {
	ProviderID uint   // Provider ID
	APIKey     string // API密钥
	BaseURL    string // API基础地址
	Model      string // 模型ID
	ModelName  string // 模型名称
	Dimensions int    // 向量维度
}

// Service 嵌入配置服务接口
type Service interface {
	// GetConfig 获取嵌入模型配置
	// 优先使用指定模型，否则使用默认模型
	GetConfig(ctx context.Context, modelID string) (*Config, error)

	// GetDefaultConfig 获取默认嵌入模型配置
	GetDefaultConfig(ctx context.Context) (*Config, error)

	// ListProviders 列出支持嵌入的 Provider 列表
	ListProviders(ctx context.Context) ([]*models.LLMProvider, error)

	// GetProviderByModel 根据模型ID获取 Provider
	GetProviderByModel(ctx context.Context, modelID string) (*models.LLMProvider, error)
}

// service 实现
type service struct {
	db *gorm.DB
}

// NewService 创建服务
func NewService(db *gorm.DB) Service {
	return &service{db: db}
}

// GetConfig 获取嵌入模型配置
func (s *service) GetConfig(ctx context.Context, modelID string) (*Config, error) {
	// 查询支持嵌入模型的 Provider
	var providers []models.LLMProvider
	err := s.db.WithContext(ctx).
		Where("is_active = ? AND embedding_models IS NOT NULL AND embedding_models != '' AND embedding_models != '[]'", true).
		Order("priority DESC").
		Find(&providers).Error
	if err != nil {
		return nil, err
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("no embedding providers configured")
	}

	// 如果指定了模型ID，查找对应 Provider
	if modelID != "" {
		for _, p := range providers {
			if cfg, err := p.GetEmbeddingConfig(modelID); err == nil {
				// 获取模型信息以填充 ModelName
				models := p.GetEmbeddingModels()
				modelName := ""
				for _, m := range models {
					if m.ID == modelID {
						modelName = m.Name
						break
					}
				}
				return &Config{
					ProviderID: p.ID,
					APIKey:     cfg.APIKey,
					BaseURL:    cfg.BaseURL,
					Model:      cfg.Model,
					ModelName:  modelName,
					Dimensions: cfg.Dimensions,
				}, nil
			}
		}
		return nil, fmt.Errorf("embedding model %s not found", modelID)
	}

	// 使用默认模型（按优先级第一个 Provider 的默认模型）
	for _, p := range providers {
		if cfg, err := p.GetDefaultEmbeddingConfig(); err == nil {
			// 获取模型信息以填充 ModelName
			models := p.GetEmbeddingModels()
			modelName := ""
			for _, m := range models {
				if m.ID == cfg.Model {
					modelName = m.Name
					break
				}
			}
			return &Config{
				ProviderID: p.ID,
				APIKey:     cfg.APIKey,
				BaseURL:    cfg.BaseURL,
				Model:      cfg.Model,
				ModelName:  modelName,
				Dimensions: cfg.Dimensions,
			}, nil
		}
	}

	return nil, fmt.Errorf("no embedding model configured")
}

// GetDefaultConfig 获取默认嵌入模型配置
func (s *service) GetDefaultConfig(ctx context.Context) (*Config, error) {
	return s.GetConfig(ctx, "")
}

// ListProviders 列出支持嵌入的 Provider 列表
func (s *service) ListProviders(ctx context.Context) ([]*models.LLMProvider, error) {
	var providers []*models.LLMProvider
	err := s.db.WithContext(ctx).
		Where("is_active = ? AND embedding_models IS NOT NULL AND embedding_models != '' AND embedding_models != '[]'", true).
		Order("priority DESC").
		Find(&providers).Error
	return providers, err
}

// GetProviderByModel 根据模型ID获取 Provider
func (s *service) GetProviderByModel(ctx context.Context, modelID string) (*models.LLMProvider, error) {
	if modelID == "" {
		return nil, fmt.Errorf("modelID is required")
	}

	// 查询所有支持嵌入的 Provider
	var providers []models.LLMProvider
	err := s.db.WithContext(ctx).
		Where("is_active = ? AND embedding_models IS NOT NULL AND embedding_models != ''", true).
		Find(&providers).Error
	if err != nil {
		return nil, err
	}

	// 查找包含指定模型的 Provider
	for _, p := range providers {
		models := p.GetEmbeddingModels()
		for _, m := range models {
			if m.ID == modelID {
				// 创建副本返回
				providerCopy := p
				return &providerCopy, nil
			}
		}
	}

	return nil, fmt.Errorf("provider with embedding model %s not found", modelID)
}
