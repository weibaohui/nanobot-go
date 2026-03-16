package api

import (
	"github.com/weibaohui/nanobot-go/pkg/agent/provider"
	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/internal/repository"
	"github.com/weibaohui/nanobot-go/internal/service"
	"github.com/weibaohui/nanobot-go/internal/service/codelookup"
	"github.com/weibaohui/nanobot-go/internal/service/conversation"
	ms "github.com/weibaohui/nanobot-go/internal/service/memory"
	mcpsvc "github.com/weibaohui/nanobot-go/internal/service/mcp"
	skillsvc "github.com/weibaohui/nanobot-go/internal/service/skill"
	memservice "github.com/weibaohui/nanobot-go/internal/memory/service"
	"go.uber.org/zap"
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
	MCPServerRepo             repository.MCPServerRepository
	AgentMCPBindingRepo       repository.AgentMCPBindingRepository
	MCPService                mcpsvc.Service
	SkillService              skillsvc.Service
}

// NewProviders 创建所有服务和仓库
func NewProviders(db *gorm.DB, cfg *config.Config, logger *zap.Logger) *Providers {
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

	// 创建 LLM 客户端和 summarizer（用于短期记忆生成摘要）
	// 使用公共函数从数据库直接创建 ChatModelAdapter
	var summarizer memservice.MemorySummarizer
	if cfg != nil && logger != nil {
		// 使用公共函数直接创建 ChatModelAdapter
		adapter, err := provider.NewChatModelAdapterFromDB(db, logger, nil)
		if err != nil {
			logger.Warn("创建 ChatModelAdapter 失败，记忆总结功能将不可用", zap.Error(err))
		} else {
			// 获取 adapter 内部的 ChatModel
			llmClient := memservice.NewEinoLLMClient(adapter.GetChatModel(), logger)
			summarizer = memservice.NewMemorySummarizer(
				llmClient,
				cfg.Memory.Summarization.ConversationPrompt,
				cfg.Memory.Summarization.LongTermPrompt,
			)
		}
	}

	streamMemoryService := ms.NewStreamMemoryService(db, summarizer)
	longTermMemoryService := ms.NewLongTermMemoryService(db)

	// 创建新的对话服务（支持统计功能）
	convService := conversation.NewService(convRepo)

	// 创建 MCP 相关 repository 和 service
	mcpServerRepo := repository.NewMCPServerRepository(db)
	agentMCPBindingRepo := repository.NewAgentMCPBindingRepository(db)
	mcpService := mcpsvc.NewService(mcpServerRepo, agentMCPBindingRepo, agentRepo)

	// 创建 Skill service
	skillService := skillsvc.NewService(cfg.Agents.Defaults.Workspace, agentRepo)

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
		MCPServerRepo:             mcpServerRepo,
		AgentMCPBindingRepo:       agentMCPBindingRepo,
		MCPService:                mcpService,
		SkillService:              skillService,
	}
}

// InitDefaultData 初始化默认数据
func (p *Providers) InitDefaultData() error {
	// 初始化默认用户
	_, err := p.UserService.InitDefaultUser()
	return err
}
