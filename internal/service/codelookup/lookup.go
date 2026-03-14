// Package codelookup 提供实体 Code 查询服务
package codelookup

import (
	"github.com/weibaohui/nanobot-go/internal/models"
	"github.com/weibaohui/nanobot-go/internal/repository"
)

// Service Code 查询服务
type Service struct {
	userRepo    repository.UserRepository
	channelRepo repository.ChannelRepository
	agentRepo   repository.AgentRepository
}

// NewService 创建 Code 查询服务
func NewService(
	userRepo repository.UserRepository,
	channelRepo repository.ChannelRepository,
	agentRepo repository.AgentRepository,
) *Service {
	return &Service{
		userRepo:    userRepo,
		channelRepo: channelRepo,
		agentRepo:   agentRepo,
	}
}

// GetUserByID 根据 ID 获取用户
func (s *Service) GetUserByID(id uint) (*models.User, error) {
	return s.userRepo.GetByID(id)
}

// GetChannelByID 根据 ID 获取渠道
func (s *Service) GetChannelByID(id uint) (*models.Channel, error) {
	return s.channelRepo.GetByID(id)
}

// GetAgentByID 根据 ID 获取 Agent
func (s *Service) GetAgentByID(id uint) (*models.Agent, error) {
	return s.agentRepo.GetByID(id)
}

// GetUserByCode 根据 Code 获取用户
func (s *Service) GetUserByCode(code string) (*models.User, error) {
	return s.userRepo.GetByUserCode(code)
}

// GetChannelByCode 根据 Code 获取渠道
func (s *Service) GetChannelByCode(code string) (*models.Channel, error) {
	return s.channelRepo.GetByChannelCode(code)
}

// GetAgentByCode 根据 Code 获取 Agent
func (s *Service) GetAgentByCode(code string) (*models.Agent, error) {
	return s.agentRepo.GetByAgentCode(code)
}
