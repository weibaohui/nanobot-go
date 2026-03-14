package api

import (
	"github.com/weibaohui/nanobot-go/internal/repository"
	"github.com/weibaohui/nanobot-go/internal/service"
	"github.com/weibaohui/nanobot-go/internal/service/codelookup"
	"github.com/weibaohui/nanobot-go/internal/service/conversation"
	ms "github.com/weibaohui/nanobot-go/internal/service/memory"
	"gorm.io/gorm"
)

// SessionManager 会话管理器接口（内存中的 session.Manager）
type SessionManager interface {
	CancelSession(sessionKey string) bool
	IsSessionActive(sessionKey string) bool
}

// Providers 包含所有的服务和仓库
type Providers struct {
	DB                        *gorm.DB
	UserRepo                  repository.UserRepository
	AgentRepo                 repository.AgentRepository
	ChannelRepo               repository.ChannelRepository
	SessionRepo               repository.SessionRepository
	UserService               service.UserService
	AgentService              service.AgentService
	ChannelService            service.ChannelService
	SessionService            service.SessionService
	ProviderService           ProviderService
	CronJobService            CronJobService
	ConversationRecordService ConversationRecordService
	ConversationService       conversation.Service
	StreamMemoryService       ms.StreamMemoryService
	LongTermMemoryService     ms.LongTermMemoryService
	SessionManager            SessionManager
}

// NewProviders 创建所有服务和仓库
func NewProviders(db *gorm.DB) *Providers {
	// 创建仓库
	userRepo := repository.NewUserRepository(db)
	agentRepo := repository.NewAgentRepository(db)
	channelRepo := repository.NewChannelRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	convRepo := conversation.NewRepository(db)

	// 创建服务
	codeService := service.NewCodeService()
	userService := service.NewUserService(userRepo, agentRepo, codeService)
	agentService := service.NewAgentService(agentRepo, codeService)
	channelService := service.NewChannelService(channelRepo, agentRepo, codeService)
	codeLookupService := codelookup.NewService(userRepo, channelRepo, agentRepo)
	sessionService := service.NewSessionService(sessionRepo, codeLookupService)
	providerService := service.NewProviderService(db, codeLookupService)
	cronJobService := service.NewCronJobService(db, codeLookupService)
	conversationRecordService := conversation.NewRecordServiceAdapter(convRepo)
	streamMemoryService := ms.NewStreamMemoryService(db)
	longTermMemoryService := ms.NewLongTermMemoryService(db)

	// 创建新的对话服务（支持统计功能）
	convService := conversation.NewService(convRepo)

	return &Providers{
		DB:                        db,
		UserRepo:                  userRepo,
		AgentRepo:                 agentRepo,
		ChannelRepo:               channelRepo,
		SessionRepo:               sessionRepo,
		UserService:               userService,
		AgentService:              agentService,
		ChannelService:            channelService,
		SessionService:            sessionService,
		ProviderService:           providerService,
		CronJobService:            cronJobService,
		ConversationRecordService: conversationRecordService,
		ConversationService:       convService,
		StreamMemoryService:       streamMemoryService,
		LongTermMemoryService:     longTermMemoryService,
	}
}

// InitDefaultData 初始化默认数据
func (p *Providers) InitDefaultData() error {
	// 初始化默认用户
	_, err := p.UserService.InitDefaultUser()
	return err
}
