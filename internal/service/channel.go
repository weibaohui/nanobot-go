package service

import (
	"encoding/json"
	"fmt"

	"github.com/weibaohui/nanobot-go/internal/models"
	"github.com/weibaohui/nanobot-go/internal/repository"
)

// CreateChannelRequest 创建 Channel 请求
type CreateChannelRequest struct {
	Name      string                 `json:"name"`
	Type      models.ChannelType     `json:"type"`
	Config    map[string]interface{} `json:"config"`
	AllowFrom []string               `json:"allow_from"`
	AgentCode string                 `json:"agent_code,omitempty"`
}

// UpdateChannelRequest 更新 Channel 请求
type UpdateChannelRequest struct {
	Name      string                 `json:"name,omitempty"`
	Config    map[string]interface{} `json:"config,omitempty"`
	AllowFrom []string               `json:"allow_from,omitempty"`
	IsActive  *bool                  `json:"is_active,omitempty"`
	AgentCode string                 `json:"agent_code,omitempty"`
}

// ChannelService Channel 服务接口
type ChannelService interface {
	// CRUD
	CreateChannel(userCode string, req CreateChannelRequest) (*models.Channel, error)
	GetChannel(id uint) (*models.Channel, error)
	GetChannelByCode(code string) (*models.Channel, error)
	GetUserChannels(userCode string) ([]models.Channel, error)
	GetUserActiveChannels(userCode string) ([]models.Channel, error)
	UpdateChannel(id uint, req UpdateChannelRequest) (*models.Channel, error)
	DeleteChannel(id uint) error

	// Agent 绑定
	BindAgent(channelCode, agentCode string) error
	UnbindAgent(channelCode string) error
	GetAgentChannels(agentCode string) ([]models.Channel, error)

	// 配置管理
	GetChannelConfig(channelID uint) (map[string]interface{}, error)
	UpdateChannelConfig(channelID uint, config map[string]interface{}) error

	// 白名单管理
	GetAllowList(channelID uint) ([]string, error)
	SetAllowList(channelID uint, allowList []string) error
}

// channelService Channel 服务实现
type channelService struct {
	channelRepo repository.ChannelRepository
	agentRepo   repository.AgentRepository
	codeService CodeService
}

// NewChannelService 创建 Channel 服务
func NewChannelService(channelRepo repository.ChannelRepository, agentRepo repository.AgentRepository, codeService CodeService) ChannelService {
	return &channelService{
		channelRepo: channelRepo,
		agentRepo:   agentRepo,
		codeService: codeService,
	}
}

// CreateChannel 创建 Channel
func (s *channelService) CreateChannel(userCode string, req CreateChannelRequest) (*models.Channel, error) {
	// 验证渠道类型
	if !isValidChannelType(req.Type) {
		return nil, fmt.Errorf("invalid channel type: %s", req.Type)
	}

	// 生成唯一 ChannelCode
	channelCode, err := GenerateUniqueCodeWithRetry(
		s.codeService.GenerateChannelCode,
		s.channelRepo.CheckChannelCodeExists,
		3,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate channel code: %w", err)
	}

	// 序列化配置
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		return nil, fmt.Errorf("序列化配置失败: %w", err)
	}

	// 序列化白名单
	allowFromJSON, err := json.Marshal(req.AllowFrom)
	if err != nil {
		return nil, fmt.Errorf("序列化白名单失败: %w", err)
	}

	channel := &models.Channel{
		UserCode:    userCode,
		ChannelCode: channelCode,
		Name:        req.Name,
		Type:        req.Type,
		IsActive:    true,
		AllowFrom:   string(allowFromJSON),
		Config:      string(configJSON),
		AgentCode:   req.AgentCode,
	}

	if err := s.channelRepo.Create(channel); err != nil {
		return nil, err
	}

	return channel, nil
}

// GetChannel 获取 Channel
func (s *channelService) GetChannel(id uint) (*models.Channel, error) {
	return s.channelRepo.GetByID(id)
}

// GetChannelByCode 根据 Code 获取 Channel
func (s *channelService) GetChannelByCode(code string) (*models.Channel, error) {
	return s.channelRepo.GetByChannelCode(code)
}

// GetUserChannels 获取用户的所有 Channel
func (s *channelService) GetUserChannels(userCode string) ([]models.Channel, error) {
	return s.channelRepo.GetByUserCode(userCode)
}

// GetUserActiveChannels 获取用户的所有活跃 Channel
func (s *channelService) GetUserActiveChannels(userCode string) ([]models.Channel, error) {
	return s.channelRepo.GetActiveByUserCode(userCode)
}

// UpdateChannel 更新 Channel
func (s *channelService) UpdateChannel(id uint, req UpdateChannelRequest) (*models.Channel, error) {
	channel, err := s.channelRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, fmt.Errorf("channel not found")
	}

	// 更新字段
	if req.Name != "" {
		channel.Name = req.Name
	}
	if req.IsActive != nil {
		channel.IsActive = *req.IsActive
	}
	if req.Config != nil {
		configJSON, err := json.Marshal(req.Config)
		if err != nil {
			return nil, fmt.Errorf("序列化配置失败: %w", err)
		}
		channel.Config = string(configJSON)
	}
	if req.AllowFrom != nil {
		allowFromJSON, err := json.Marshal(req.AllowFrom)
		if err != nil {
			return nil, fmt.Errorf("序列化白名单失败: %w", err)
		}
		channel.AllowFrom = string(allowFromJSON)
	}
	// 更新 AgentCode（允许为空字符串来解绑）
	channel.AgentCode = req.AgentCode

	if err := s.channelRepo.Update(channel); err != nil {
		return nil, err
	}

	return channel, nil
}

// DeleteChannel 删除 Channel
func (s *channelService) DeleteChannel(id uint) error {
	return s.channelRepo.Delete(id)
}

// BindAgent 绑定 Agent 到 Channel
func (s *channelService) BindAgent(channelCode, agentCode string) error {
	// 验证 Agent 存在
	agent, err := s.agentRepo.GetByAgentCode(agentCode)
	if err != nil {
		return err
	}
	if agent == nil {
		return fmt.Errorf("agent not found")
	}

	return s.channelRepo.BindAgent(channelCode, agentCode)
}

// UnbindAgent 解除 Channel 的 Agent 绑定
func (s *channelService) UnbindAgent(channelCode string) error {
	return s.channelRepo.UnbindAgent(channelCode)
}

// GetAgentChannels 获取绑定到指定 Agent 的所有 Channel
func (s *channelService) GetAgentChannels(agentCode string) ([]models.Channel, error) {
	return s.channelRepo.GetByAgentCode(agentCode)
}

// GetChannelConfig 获取 Channel 配置
func (s *channelService) GetChannelConfig(channelID uint) (map[string]interface{}, error) {
	channel, err := s.channelRepo.GetByID(channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, fmt.Errorf("channel not found")
	}

	if channel.Config == "" || channel.Config == "null" {
		return map[string]interface{}{}, nil
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(channel.Config), &config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	return config, nil
}

// UpdateChannelConfig 更新 Channel 配置
func (s *channelService) UpdateChannelConfig(channelID uint, config map[string]interface{}) error {
	channel, err := s.channelRepo.GetByID(channelID)
	if err != nil {
		return err
	}
	if channel == nil {
		return fmt.Errorf("channel not found")
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	channel.Config = string(configJSON)
	return s.channelRepo.Update(channel)
}

// GetAllowList 获取白名单
func (s *channelService) GetAllowList(channelID uint) ([]string, error) {
	channel, err := s.channelRepo.GetByID(channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, fmt.Errorf("channel not found")
	}

	if channel.AllowFrom == "" || channel.AllowFrom == "null" {
		return []string{}, nil
	}

	var allowList []string
	if err := json.Unmarshal([]byte(channel.AllowFrom), &allowList); err != nil {
		return nil, fmt.Errorf("解析白名单失败: %w", err)
	}

	return allowList, nil
}

// SetAllowList 设置白名单
func (s *channelService) SetAllowList(channelID uint, allowList []string) error {
	channel, err := s.channelRepo.GetByID(channelID)
	if err != nil {
		return err
	}
	if channel == nil {
		return fmt.Errorf("channel not found")
	}

	allowFromJSON, err := json.Marshal(allowList)
	if err != nil {
		return fmt.Errorf("序列化白名单失败: %w", err)
	}

	channel.AllowFrom = string(allowFromJSON)
	return s.channelRepo.Update(channel)
}

// isValidChannelType 验证渠道类型是否有效
func isValidChannelType(t models.ChannelType) bool {
	switch t {
	case models.ChannelTypeFeishu, models.ChannelTypeWebSocket:
		return true
	}
	return false
}
