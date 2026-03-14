package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/nanobot-go/internal/service"
)

// Handler API 处理器
type Handler struct {
	userService               service.UserService
	agentService              service.AgentService
	channelService            service.ChannelService
	sessionService            service.SessionService
	providerService           ProviderService
	cronJobService            CronJobService
	conversationRecordService ConversationRecordService
	streamMemoryService       StreamMemoryService
	longTermMemoryService     LongTermMemoryService
	sessionManager            SessionManager
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
	streamMemoryService StreamMemoryService,
	longTermMemoryService LongTermMemoryService,
	sessionManager SessionManager,
) *Handler {
	return &Handler{
		userService:               userService,
		agentService:              agentService,
		channelService:            channelService,
		sessionService:            sessionService,
		providerService:           providerService,
		cronJobService:            cronJobService,
		conversationRecordService: conversationRecordService,
		streamMemoryService:       streamMemoryService,
		longTermMemoryService:     longTermMemoryService,
		sessionManager:            sessionManager,
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
		}

		// Short-term Memory API
		streamMemories := authorized.Group("/stream-memories")
		{
			streamMemories.GET("", h.handleStreamMemories)
			streamMemories.GET("/:id", h.handleStreamMemoryByID)
			streamMemories.POST("", h.createStreamMemory)
			streamMemories.PUT("/:id", h.updateStreamMemory)
			streamMemories.DELETE("/:id", h.deleteStreamMemory)
			streamMemories.GET("/unprocessed", h.handleUnprocessedMemories)
		}

		// Long-term Memory API
		longTermMemories := authorized.Group("/long-term-memories")
		{
			longTermMemories.GET("", h.handleLongTermMemories)
			longTermMemories.GET("/:id", h.handleLongTermMemoryByID)
			longTermMemories.GET("/date/:date", h.handleLongTermMemoryByDate)
			longTermMemories.POST("", h.createLongTermMemory)
			longTermMemories.PUT("/:id", h.updateLongTermMemory)
			longTermMemories.DELETE("/:id", h.deleteLongTermMemory)
			longTermMemories.GET("/search", h.searchLongTermMemories)
			longTermMemories.GET("/recent", h.getRecentLongTermMemories)
		}
	}
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

// writeJSON writes JSON response (kept for backward compatibility)
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
}

// writeError writes error response (kept for backward compatibility)
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}
