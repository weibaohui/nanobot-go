package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// LLMProvider LLM 提供商模型
// 存储用户的 LLM API 密钥和配置
type LLMProvider struct {
	ID           uint   `gorm:"primarykey" json:"id"`
	UserCode     string `gorm:"type:varchar(16);index" json:"user_code"`
	ProviderKey  string `gorm:"type:text;not null" json:"provider_key"`  // 如: 'anthropic', 'openai'
	ProviderName string `gorm:"type:text" json:"provider_name"`          // 如: 'Anthropic', 'OpenAI'

	// API 配置
	APIKey       string `gorm:"type:text" json:"-"`                      // API密钥，不序列化到JSON
	APIBase      string `gorm:"type:text" json:"api_base,omitempty"`     // API基础地址
	ExtraHeaders string `gorm:"type:text" json:"extra_headers,omitempty"` // JSON格式额外请求头

	// 支持的模型列表
	SupportedModels string `gorm:"type:text" json:"supported_models,omitempty"` // JSON数组 [{"id": "xxx", "name": "xxx"}]

	// 默认配置
	DefaultModel string `gorm:"type:text" json:"default_model,omitempty"` // 默认模型ID
	IsDefault    bool   `gorm:"default:false" json:"is_default"`          // 是否为默认提供商
	Priority     int    `gorm:"default:0" json:"priority"`                // 优先级 (数值越大优先级越高，auto模式下使用)
	AutoMerge    bool   `gorm:"default:true" json:"auto_merge"`           // 是否自动合并从API获取的模型列表到supported_models

	// 嵌入模型配置（新增）
	EmbeddingModels       string `gorm:"type:text" json:"embedding_models,omitempty"`       // JSON数组，包含模型ID、名称、维度
	DefaultEmbeddingModel string `gorm:"type:text" json:"default_embedding_model,omitempty"` // 默认嵌入模型ID

	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName 指定表名
func (LLMProvider) TableName() string {
	return "llm_providers"
}

// ModelInfo 模型信息结构
type ModelInfo struct {
	ID        string `json:"id"`                   // 模型ID
	Name      string `json:"name"`                 // 模型名称
	MaxTokens int    `json:"max_tokens,omitempty"` // 最大Token数
}

// EmbeddingModelInfo 嵌入模型信息
type EmbeddingModelInfo struct {
	ID         string `json:"id"`         // 模型ID
	Name       string `json:"name"`       // 模型名称
	Dimensions int    `json:"dimensions"` // 向量维度
}

// EmbeddingModelConfig 嵌入模型完整配置
type EmbeddingModelConfig struct {
	APIKey     string
	BaseURL    string
	Model      string
	Dimensions int
}

// GetSupportedModels 获取支持的模型列表
func (p *LLMProvider) GetSupportedModels() []ModelInfo {
	if p.SupportedModels == "" || p.SupportedModels == "null" {
		return nil
	}
	var models []ModelInfo
	if err := json.Unmarshal([]byte(p.SupportedModels), &models); err != nil {
		return nil
	}
	return models
}

// SetSupportedModels 设置支持的模型列表
func (p *LLMProvider) SetSupportedModels(models []ModelInfo) error {
	data, err := json.Marshal(models)
	if err != nil {
		return err
	}
	p.SupportedModels = string(data)
	return nil
}

// GetExtraHeaders 获取额外请求头
func (p *LLMProvider) GetExtraHeaders() map[string]string {
	if p.ExtraHeaders == "" || p.ExtraHeaders == "null" {
		return nil
	}
	var headers map[string]string
	if err := json.Unmarshal([]byte(p.ExtraHeaders), &headers); err != nil {
		return nil
	}
	return headers
}

// SetExtraHeaders 设置额外请求头
func (p *LLMProvider) SetExtraHeaders(headers map[string]string) error {
	data, err := json.Marshal(headers)
	if err != nil {
		return err
	}
	p.ExtraHeaders = string(data)
	return nil
}

// GetEmbeddingModels 获取嵌入模型列表
func (p *LLMProvider) GetEmbeddingModels() []EmbeddingModelInfo {
	if p.EmbeddingModels == "" || p.EmbeddingModels == "null" {
		return nil
	}
	var models []EmbeddingModelInfo
	if err := json.Unmarshal([]byte(p.EmbeddingModels), &models); err != nil {
		return nil
	}
	return models
}

// SetEmbeddingModels 设置嵌入模型列表
func (p *LLMProvider) SetEmbeddingModels(models []EmbeddingModelInfo) error {
	data, err := json.Marshal(models)
	if err != nil {
		return err
	}
	p.EmbeddingModels = string(data)
	return nil
}

// GetEmbeddingConfig 获取指定嵌入模型的完整配置
func (p *LLMProvider) GetEmbeddingConfig(modelID string) (*EmbeddingModelConfig, error) {
	models := p.GetEmbeddingModels()
	if len(models) == 0 {
		return nil, fmt.Errorf("no embedding models configured")
	}

	// 查找指定模型
	for _, m := range models {
		if m.ID == modelID {
			return &EmbeddingModelConfig{
				APIKey:     p.APIKey,
				BaseURL:    p.APIBase,
				Model:      m.ID,
				Dimensions: m.Dimensions,
			}, nil
		}
	}

	return nil, fmt.Errorf("embedding model %s not found", modelID)
}

// GetDefaultEmbeddingConfig 获取默认嵌入模型配置
func (p *LLMProvider) GetDefaultEmbeddingConfig() (*EmbeddingModelConfig, error) {
	modelID := p.DefaultEmbeddingModel
	if modelID == "" {
		// 使用第一个嵌入模型作为默认
		models := p.GetEmbeddingModels()
		if len(models) == 0 {
			return nil, fmt.Errorf("no embedding models configured")
		}
		modelID = models[0].ID
	}
	return p.GetEmbeddingConfig(modelID)
}

// HasEmbeddingModels 检查是否配置了嵌入模型
func (p *LLMProvider) HasEmbeddingModels() bool {
	return p.EmbeddingModels != "" && p.EmbeddingModels != "null" && p.EmbeddingModels != "[]"
}
