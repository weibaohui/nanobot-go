package api

import (
	"github.com/weibaohui/nanobot-go/internal/repository"
	"github.com/weibaohui/nanobot-go/internal/service"
	"gorm.io/gorm"
)

// Providers 包含所有的服务和仓库
type Providers struct {
	DB             *gorm.DB
	UserRepo       repository.UserRepository
	AgentRepo      repository.AgentRepository
	ChannelRepo    repository.ChannelRepository
	SessionRepo    repository.SessionRepository
	UserService    service.UserService
	AgentService   service.AgentService
	ChannelService service.ChannelService
	SessionService service.SessionService
}

// NewProviders 创建所有服务和仓库
func NewProviders(db *gorm.DB) *Providers {
	// 创建仓库
	userRepo := repository.NewUserRepository(db)
	agentRepo := repository.NewAgentRepository(db)
	channelRepo := repository.NewChannelRepository(db)
	sessionRepo := repository.NewSessionRepository(db)

	// 创建服务
	userService := service.NewUserService(userRepo, agentRepo)
	agentService := service.NewAgentService(agentRepo)
	channelService := service.NewChannelService(channelRepo, agentRepo)
	sessionService := service.NewSessionService(sessionRepo)

	return &Providers{
		DB:             db,
		UserRepo:       userRepo,
		AgentRepo:      agentRepo,
		ChannelRepo:    channelRepo,
		SessionRepo:    sessionRepo,
		UserService:    userService,
		AgentService:   agentService,
		ChannelService: channelService,
		SessionService: sessionService,
	}
}

// InitDefaultData 初始化默认数据
func (p *Providers) InitDefaultData() error {
	// 初始化默认用户
	_, err := p.UserService.InitDefaultUser()
	return err
}
