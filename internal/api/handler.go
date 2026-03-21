package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/nanobot-go/internal/models"
	"github.com/weibaohui/nanobot-go/internal/service"
	"github.com/weibaohui/nanobot-go/internal/service/conversation"
	mcpsvc "github.com/weibaohui/nanobot-go/internal/service/mcp"
	skillsvc "github.com/weibaohui/nanobot-go/internal/service/skill"
)

// CodeLookupService Code 查询服务接口
type CodeLookupService interface {
	GetUserByCode(code string) (*models.User, error)
	GetChannelByCode(code string) (*models.Channel, error)
	GetAgentByCode(code string) (*models.Agent, error)
}

// Handler API 处理器
type Handler struct {
	userService               service.UserService
	agentService              service.AgentService
	channelService            service.ChannelService
	sessionService            service.SessionService
	providerService           ProviderService
	cronJobService            CronJobService
	conversationRecordService ConversationRecordService
	conversationService       conversation.Service
	sessionManager            SessionManager
	mcpService                MCPService
	skillService              skillsvc.Service
	codeLookupService         CodeLookupService
}

// NewHandler 创建 API 处理器
func NewHandler(
	userService service.UserService,
	agentService service.AgentService,
	channelService service.ChannelService,
	sessionService service.SessionService,
	providerService ProviderService,
	cronJobService CronJobService,
	conversationRecordService ConversationRecordService,
	conversationService conversation.Service,
	sessionManager SessionManager,
	mcpService MCPService,
	skillService skillsvc.Service,
	codeLookupService CodeLookupService,
) *Handler {
	return &Handler{
		userService:               userService,
		agentService:              agentService,
		channelService:            channelService,
		sessionService:            sessionService,
		providerService:           providerService,
		cronJobService:            cronJobService,
		conversationRecordService: conversationRecordService,
		conversationService:       conversationService,
		sessionManager:            sessionManager,
		mcpService:                mcpService,
		skillService:              skillService,
		codeLookupService:         codeLookupService,
	}
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(router *gin.Engine) {
	// 公开 API（不需要认证）
	router.POST("/api/v1/auth/login", h.login)

	// 需要认证的 API
	authorized := router.Group("/api/v1")
	authorized.Use(AuthMiddleware())
	{
		// 当前用户信息
		authorized.GET("/auth/me", h.getCurrentUser)

		// User API
		users := authorized.Group("/users")
		{
			users.GET("", h.listUsers)
			users.POST("", h.createUser)
			users.GET("/:id", h.getUserByID)
			users.PUT("/:id", h.updateUserByID)
			users.DELETE("/:id", h.deleteUserByID)
			users.POST("/:id/change-password", h.changePasswordByID)
			users.GET("/code/:code", h.getUserByCode)
		}

		// Agent API
		agents := authorized.Group("/agents")
		{
			agents.GET("", h.listAgents)
			agents.POST("", h.createAgent)
			agents.GET("/:id", h.getAgentByID)
			agents.PUT("/:id", h.updateAgentByID)
			agents.DELETE("/:id", h.deleteAgentByID)
			agents.GET("/code/:code", h.getAgentByCode)
		}

		// Channel API
		channels := authorized.Group("/channels")
		{
			channels.GET("", h.listChannels)
			channels.POST("", h.createChannel)
			channels.GET("/:id", h.getChannelByID)
			channels.PUT("/:id", h.updateChannel)
			channels.DELETE("/:id", h.deleteChannel)
			channels.GET("/code/:code", h.getChannelByCode)
		}

		// Session API
		sessions := authorized.Group("/sessions")
		{
			sessions.GET("", h.handleSessions)
			sessions.POST("", h.createSession)
			sessions.GET("/:id", h.handleSessionByKey)
			sessions.DELETE("/:id", h.handleSessionByKey)
			sessions.POST("/:id/touch", func(c *gin.Context) {
				h.handleSessionByKey(c)
			})
			sessions.GET("/:id/metadata", func(c *gin.Context) {
				h.handleSessionByKey(c)
			})
			sessions.PUT("/:id/metadata", func(c *gin.Context) {
				h.handleSessionByKey(c)
			})
			sessions.POST("/:id/cancel", h.cancelSession)
			sessions.GET("/:id/active", h.checkSessionActive)
		}

		// Provider API
		providers := authorized.Group("/providers")
		{
			providers.GET("", h.handleProviders)
			providers.POST("", h.createProvider)
			providers.GET("/:id", h.handleProviderByID)
			providers.PUT("/:id", h.updateProvider)
			providers.DELETE("/:id", h.deleteProvider)
			providers.POST("/:id/test", h.testProviderConnection)
			providers.GET("/:id/embedding", h.getProviderEmbeddingModels)
			providers.PUT("/:id/embedding", h.updateProviderEmbeddingModels)
		}

		// Cron Job API
		cronJobs := authorized.Group("/cron-jobs")
		{
			cronJobs.GET("", h.handleCronJobs)
			cronJobs.POST("", h.createCronJob)
			cronJobs.GET("/pending", h.handlePendingCronJobs)
			cronJobs.GET("/:id", h.handleCronJobByID)
			cronJobs.PUT("/:id", h.updateCronJob)
			cronJobs.DELETE("/:id", h.deleteCronJob)
			cronJobs.POST("/:id/enable", h.enableCronJob)
			cronJobs.POST("/:id/disable", h.disableCronJob)
			cronJobs.POST("/:id/execute", h.executeCronJob)
		}

		// Conversation Record API
		conversations := authorized.Group("/conversations")
		{
			conversations.GET("", h.handleConversationRecords)
			conversations.GET("/:id", h.handleConversationRecordByID)
			conversations.POST("", h.createConversationRecord)
			conversations.PUT("/:id", h.updateConversationRecord)
			conversations.DELETE("/:id", h.deleteConversationRecord)
			conversations.GET("/session/:sessionKey", h.handleConversationBySession)
			conversations.GET("/trace/:traceID", h.handleConversationByTrace)
			conversations.GET("/user/:userCode/date/:date", h.handleConversationByUserAndDate)
			conversations.GET("/stats", h.handleConversationStats)
		}

		// MCP Server API
		mcpServers := authorized.Group("/mcp-servers")
		{
			mcpServers.GET("", h.listMCPServers)
			mcpServers.POST("", h.createMCPServer)
			mcpServers.GET("/:id", h.getMCPServer)
			mcpServers.PUT("/:id", h.updateMCPServer)
			mcpServers.DELETE("/:id", h.deleteMCPServer)
			mcpServers.POST("/:id/test", h.testMCPServer)
			mcpServers.POST("/:id/refresh", h.refreshMCPServerCapabilities)
			mcpServers.GET("/:id/tools", h.listMCPTools)
		}

		// Agent MCP Binding API - 使用 :id 保持与现有路由一致
		agentMCPBindings := authorized.Group("/agents/:id/mcp-bindings")
		{
			agentMCPBindings.GET("", h.listAgentMCPBindings)
			agentMCPBindings.POST("", h.createAgentMCPBinding)
			agentMCPBindings.GET("/:binding_id", h.getAgentMCPBinding)
			agentMCPBindings.PUT("/:binding_id", h.updateAgentMCPBinding)
			agentMCPBindings.DELETE("/:binding_id", h.deleteAgentMCPBinding)
			agentMCPBindings.GET("/tools", h.getAgentMCPTools)
		}

		// Skills API
		skills := authorized.Group("/skills")
		{
			skills.GET("", h.listSkills)
			skills.GET("/:name", h.getSkill)
		}
	}
}

// MCPService MCP 服务接口
type MCPService interface {
	ListServers() ([]models.MCPServer, error)
	CreateServer(req mcpsvc.CreateMCPServerRequest) (*models.MCPServer, error)
	GetServer(id uint) (*models.MCPServer, error)
	UpdateServer(id uint, req mcpsvc.UpdateMCPServerRequest) (*models.MCPServer, error)
	DeleteServer(id uint) error
	TestServer(id uint) error
	RefreshCapabilities(id uint) error
	ListTools(serverID uint) ([]models.MCPToolModel, error)

	GetAgentBindings(agentID uint) ([]models.AgentMCPBinding, error)
	CreateAgentBinding(agentID uint, req mcpsvc.CreateAgentMCPBindingRequest) (*models.AgentMCPBinding, error)
	GetAgentBindingByID(bindingID uint) (*models.AgentMCPBinding, error)
	UpdateAgentBinding(bindingID uint, req mcpsvc.UpdateAgentMCPBindingRequest) (*models.AgentMCPBinding, error)
	DeleteAgentBinding(bindingID uint) error
	GetAgentMCPTools(agentCode string) ([]mcpsvc.AgentMCPToolInfo, error)
}

// === Helper Functions ===

// parseID 从 Gin Context 中解析 ID 参数
func parseID(c *gin.Context, param string) (uint, bool) {
	idStr := c.Param(param)
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, false
	}
	return uint(id), true
}

// === Response Types ===

// ListResponse 列表响应
type ListResponse struct {
	Items    interface{} `json:"items"`
	Total    int64       `json:"total"`
	Page     int         `json:"page,omitempty"`
	PageSize int         `json:"page_size,omitempty"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error string `json:"error"`
}

// SuccessResponse 成功响应
type SuccessResponse struct {
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

