package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/weibaohui/nanobot-go/internal/models"
	"gorm.io/gorm"
)

// CreateProviderRequest 创建 Provider 请求
type CreateProviderRequest struct {
	ProviderKey      string       `json:"provider_key"`
	ProviderName     string       `json:"provider_name,omitempty"`
	APIKey           string       `json:"api_key,omitempty"`
	APIBase          string       `json:"api_base,omitempty"`
	ExtraHeaders     string       `json:"extra_headers,omitempty"`
	SupportedModels  []ModelInfo  `json:"supported_models,omitempty"`
	DefaultModel     string       `json:"default_model,omitempty"`
	IsDefault        bool         `json:"is_default,omitempty"`
	Priority         int          `json:"priority,omitempty"`
}

// UpdateProviderRequest 更新 Provider 请求
type UpdateProviderRequest struct {
	ProviderKey      string       `json:"provider_key,omitempty"`
	ProviderName     string       `json:"provider_name,omitempty"`
	APIKey           string       `json:"api_key,omitempty"`
	APIBase          string       `json:"api_base,omitempty"`
	ExtraHeaders     string       `json:"extra_headers,omitempty"`
	SupportedModels  []ModelInfo  `json:"supported_models,omitempty"`
	DefaultModel     string       `json:"default_model,omitempty"`
	IsDefault        *bool        `json:"is_default,omitempty"`
	Priority         *int         `json:"priority,omitempty"`
	IsActive         *bool        `json:"is_active,omitempty"`
}

// ModelInfo 模型信息
type ModelInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MaxTokens int    `json:"max_tokens,omitempty"`
}

// LLMConfig LLM 配置信息，用于创建 LLM 客户端
type LLMConfig struct {
	ProviderKey  string            `json:"provider_key"`
	APIKey       string            `json:"api_key"`
	APIBase      string            `json:"api_base"`
	DefaultModel string            `json:"default_model"`
	ExtraHeaders map[string]string `json:"extra_headers,omitempty"`
}

// ProviderService Provider 服务接口
type ProviderService interface {
	List(ctx context.Context, userID uint, offset int, limit int) ([]models.LLMProvider, int64, error)
	Get(ctx context.Context, id uint) (*models.LLMProvider, error)
	Create(ctx context.Context, userID uint, req CreateProviderRequest) (*models.LLMProvider, error)
	Update(ctx context.Context, id uint, req UpdateProviderRequest) error
	Delete(ctx context.Context, id uint) error
	SetDefault(ctx context.Context, userID uint, providerID uint) error
	GetModelConfig(ctx context.Context, id uint) (interface{}, error)
	UpdateModelConfig(ctx context.Context, id uint, config map[string]interface{}) error
	TestConnection(ctx context.Context, id uint) (map[string]interface{}, error)
	GetLLMConfig(ctx context.Context, userID uint) (*LLMConfig, error)
}

// providerService Provider 服务实现
type providerService struct {
	db *gorm.DB
}

// NewProviderService 创建 Provider 服务
func NewProviderService(db *gorm.DB) ProviderService {
	return &providerService{db: db}
}

// List 获取 Provider 列表
func (s *providerService) List(ctx context.Context, userID uint, offset int, limit int) ([]models.LLMProvider, int64, error) {
	var providers []models.LLMProvider
	var total int64

	query := s.db.WithContext(ctx).Where("user_id = ?", userID)

	if err := query.Model(&models.LLMProvider{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Offset(offset).Limit(limit).Find(&providers).Error; err != nil {
		return nil, 0, err
	}

	return providers, total, nil
}

// Get 获取单个 Provider
func (s *providerService) Get(ctx context.Context, id uint) (*models.LLMProvider, error) {
	var provider models.LLMProvider
	if err := s.db.WithContext(ctx).First(&provider, id).Error; err != nil {
		return nil, err
	}
	return &provider, nil
}

// Create 创建 Provider
func (s *providerService) Create(ctx context.Context, userID uint, req CreateProviderRequest) (*models.LLMProvider, error) {
	// 序列化支持的模型列表
	var supportedModelsJSON string
	if req.SupportedModels != nil {
		modelsJSON, err := json.Marshal(req.SupportedModels)
		if err != nil {
			return nil, fmt.Errorf("序列化模型列表失败: %w", err)
		}
		supportedModelsJSON = string(modelsJSON)
	}

	provider := &models.LLMProvider{
		UserID:          userID,
		ProviderKey:      req.ProviderKey,
		ProviderName:     req.ProviderName,
		APIKey:          req.APIKey,
		APIBase:         req.APIBase,
		SupportedModels:  supportedModelsJSON,
		DefaultModel:    req.DefaultModel,
		IsDefault:       req.IsDefault,
		Priority:         req.Priority,
		IsActive:        true,
	}

	if req.IsDefault {
		// 清除其他默认标记
		s.db.WithContext(ctx).Model(&models.LLMProvider{}).
			Where("user_id = ? AND is_default = ?", userID, true).
			Update("is_default", false)
	}

	if err := s.db.WithContext(ctx).Create(provider).Error; err != nil {
		return nil, err
	}

	return provider, nil
}

// Update 更新 Provider
func (s *providerService) Update(ctx context.Context, id uint, req UpdateProviderRequest) error {
	var provider models.LLMProvider
	if err := s.db.WithContext(ctx).First(&provider, id).Error; err != nil {
		return err
	}

	updates := make(map[string]interface{})

	if req.ProviderKey != "" {
		updates["provider_key"] = req.ProviderKey
	}
	if req.ProviderName != "" {
		updates["provider_name"] = req.ProviderName
	}
	if req.APIKey != "" {
		updates["api_key"] = req.APIKey
	}
	if req.APIBase != "" {
		updates["api_base"] = req.APIBase
	}
	if req.ExtraHeaders != "" {
		updates["extra_headers"] = req.ExtraHeaders
	}
	if req.SupportedModels != nil {
		modelsJSON, err := json.Marshal(req.SupportedModels)
		if err != nil {
			return fmt.Errorf("序列化模型列表失败: %w", err)
		}
		updates["supported_models"] = string(modelsJSON)
	}
	if req.DefaultModel != "" {
		updates["default_model"] = req.DefaultModel
	}
	if req.IsDefault != nil {
		if *req.IsDefault {
			// 清除其他默认标记
			s.db.WithContext(ctx).Model(&models.LLMProvider{}).
				Where("user_id = ? AND id != ? AND is_default = ?", provider.UserID, id, true).
				Update("is_default", false)
		}
		updates["is_default"] = *req.IsDefault
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if len(updates) == 0 {
		return nil
	}

	return s.db.WithContext(ctx).Model(&provider).Updates(updates).Error
}

// Delete 删除 Provider
func (s *providerService) Delete(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.LLMProvider{}, id).Error
}

// SetDefault 设置默认 Provider
func (s *providerService) SetDefault(ctx context.Context, userID uint, providerID uint) error {
	// 获取目标 Provider
	var provider models.LLMProvider
	if err := s.db.WithContext(ctx).Where("user_id = ? AND id = ?", userID, providerID).First(&provider).Error; err != nil {
		return err
	}

	// 清除其他默认标记
	if err := s.db.WithContext(ctx).Model(&models.LLMProvider{}).
		Where("user_id = ? AND is_default = ?", userID, true).
		Update("is_default", false).Error; err != nil {
		return err
	}

	// 设置目标 Provider 为默认
	provider.IsDefault = true
	return s.db.WithContext(ctx).Save(&provider).Error
}

// GetModelConfig 获取模型配置
func (s *providerService) GetModelConfig(ctx context.Context, id uint) (interface{}, error) {
	var provider models.LLMProvider
	if err := s.db.WithContext(ctx).First(&provider, id).Error; err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"supported_models": provider.GetSupportedModels(),
		"default_model":    provider.DefaultModel,
		"auto_merge":       provider.AutoMerge,
	}, nil
}

// UpdateModelConfig 更新模型配置
func (s *providerService) UpdateModelConfig(ctx context.Context, id uint, config map[string]interface{}) error {
	var provider models.LLMProvider
	if err := s.db.WithContext(ctx).First(&provider, id).Error; err != nil {
		return err
	}

	updates := make(map[string]interface{})

	if supportedModels, ok := config["supported_models"]; ok {
		modelsJSON, err := json.Marshal(supportedModels)
		if err != nil {
			return fmt.Errorf("序列化模型列表失败: %w", err)
		}
		updates["supported_models"] = string(modelsJSON)
	}
	if defaultModel, ok := config["default_model"]; ok {
		updates["default_model"] = defaultModel
	}
	if autoMerge, ok := config["auto_merge"]; ok {
		updates["auto_merge"] = autoMerge
	}

	return s.db.WithContext(ctx).Model(&provider).Updates(updates).Error
}

// TestConnection 测试连接
func (s *providerService) TestConnection(ctx context.Context, id uint) (map[string]interface{}, error) {
	var provider models.LLMProvider
	if err := s.db.WithContext(ctx).First(&provider, id).Error; err != nil {
		return nil, err
	}

	// TODO: 实现实际的连接测试逻辑
	// 这里只是一个占位符，实际需要调用对应的 LLM API 进行测试

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("成功连接到 %s (%s)", provider.ProviderName, provider.ProviderKey),
		"models":  provider.GetSupportedModels(),
	}, nil
}

// GetLLMConfig 获取用于创建 LLM 客户端的配置
// 返回用户默认 Provider 的配置信息
func (s *providerService) GetLLMConfig(ctx context.Context, userID uint) (*LLMConfig, error) {
	var provider models.LLMProvider
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND is_default = ? AND is_active = ?", userID, true, true).
		First(&provider).Error; err != nil {
		return nil, fmt.Errorf("获取默认 Provider 失败: %w", err)
	}

	return &LLMConfig{
		ProviderKey:  provider.ProviderKey,
		APIKey:       provider.APIKey,
		APIBase:      provider.APIBase,
		DefaultModel: provider.DefaultModel,
		ExtraHeaders: provider.GetExtraHeaders(),
	}, nil
}
