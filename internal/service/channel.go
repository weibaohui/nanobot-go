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
	AgentID   *uint                  `json:"agent_id,omitempty"`
}

// UpdateChannelRequest 更新 Channel 请求
type UpdateChannelRequest struct {
	Name      string                 `json:"name,omitempty"`
	Config    map[string]interface{} `json:"config,omitempty"`
	AllowFrom []string               `json:"allow_from,omitempty"`
	IsActive  *bool                  `json:"is_active,omitempty"`
	AgentID   *uint                  `json:"agent_id,omitempty"`
}

// ChannelService Channel 服务接口
type ChannelService interface {
	// CRUD
	CreateChannel(userID uint, req CreateChannelRequest) (*models.Channel, error)
	GetChannel(id uint) (*models.Channel, error)
	GetUserChannels(userID uint) ([]models.Channel, error)
	GetUserActiveChannels(userID uint) ([]models.Channel, error)
	UpdateChannel(id uint, req UpdateChannelRequest) (*models.Channel, error)
	DeleteChannel(id uint) error

	// Agent 绑定
	BindAgent(channelID, agentID uint) error
	UnbindAgent(channelID uint) error
	GetAgentChannels(agentID uint) ([]models.Channel, error)

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
}

// NewChannelService 创建 Channel 服务
func NewChannelService(channelRepo repository.ChannelRepository, agentRepo repository.AgentRepository) ChannelService {
	return &channelService{
		channelRepo: channelRepo,
		agentRepo:   agentRepo,
	}
}

// CreateChannel 创建 Channel
func (s *channelService) CreateChannel(userID uint, req CreateChannelRequest) (*models.Channel, error) {
	// 验证渠道类型
	if !isValidChannelType(req.Type) {
		return nil, fmt.Errorf("invalid channel type: %s", req.Type)
	}

	// 如果指定了 AgentID，验证该 Agent 存在且属于该用户
	if req.AgentID != nil {
		agent, err := s.agentRepo.GetByID(*req.AgentID)
		if err != nil {
			return nil, err
		}
		if agent == nil {
			return nil, fmt.Errorf("agent not found")
		}
		if agent.UserID != userID {
			return nil, fmt.Errorf("agent does not belong to user")
		}
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
		UserID:    userID,
		AgentID:   req.AgentID,
		Name:      req.Name,
		Type:      req.Type,
		IsActive:  true,
		AllowFrom: string(allowFromJSON),
		Config:    string(configJSON),
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

// GetUserChannels 获取用户的所有 Channel
func (s *channelService) GetUserChannels(userID uint) ([]models.Channel, error) {
	return s.channelRepo.GetByUserID(userID)
}

// GetUserActiveChannels 获取用户的所有活跃 Channel
func (s *channelService) GetUserActiveChannels(userID uint) ([]models.Channel, error) {
	return s.channelRepo.GetActiveByUserID(userID)
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
	if req.AgentID != nil {
		// 验证新的 AgentID
		agent, err := s.agentRepo.GetByID(*req.AgentID)
		if err != nil {
			return nil, err
		}
		if agent == nil {
			return nil, fmt.Errorf("agent not found")
		}
		if agent.UserID != channel.UserID {
			return nil, fmt.Errorf("agent does not belong to user")
		}
		channel.AgentID = req.AgentID
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
func (s *channelService) BindAgent(channelID, agentID uint) error {
	channel, err := s.channelRepo.GetByID(channelID)
	if err != nil {
		return err
	}
	if channel == nil {
		return fmt.Errorf("channel not found")
	}

	// 验证 Agent 存在且属于同一用户
	agent, err := s.agentRepo.GetByID(agentID)
	if err != nil {
		return err
	}
	if agent == nil {
		return fmt.Errorf("agent not found")
	}
	if agent.UserID != channel.UserID {
		return fmt.Errorf("agent does not belong to user")
	}

	return s.channelRepo.BindAgent(channelID, agentID)
}

// UnbindAgent 解除 Channel 的 Agent 绑定
func (s *channelService) UnbindAgent(channelID uint) error {
	return s.channelRepo.UnbindAgent(channelID)
}

// GetAgentChannels 获取绑定到指定 Agent 的所有 Channel
func (s *channelService) GetAgentChannels(agentID uint) ([]models.Channel, error) {
	return s.channelRepo.GetByAgentID(agentID)
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
	case models.ChannelTypeFeishu:
		return true
	}
	return false
}
