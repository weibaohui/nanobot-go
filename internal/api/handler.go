package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/nanobot-go/internal/service"
)

// Handler API 处理器
type Handler struct {
	userService    service.UserService
	agentService   service.AgentService
	channelService service.ChannelService
	sessionService service.SessionService
}

// NewHandler 创建 API 处理器
func NewHandler(
	userService service.UserService,
	agentService service.AgentService,
	channelService service.ChannelService,
	sessionService service.SessionService,
) *Handler {
	return &Handler{
		userService:    userService,
		agentService:   agentService,
		channelService: channelService,
		sessionService: sessionService,
	}
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(router *gin.Engine) {
	// User API
	users := router.Group("/api/v1/users")
	{
		users.GET("", h.handleUsers)
		users.POST("", h.handleUsers)
		users.GET("/:id", h.handleUserByID)
		users.PUT("/:id", h.handleUserByID)
		users.DELETE("/:id", h.handleUserByID)
	}

	// Agent API
	agents := router.Group("/api/v1/agents")
	{
		agents.GET("", h.handleAgents)
		agents.POST("", h.handleAgents)
		agents.GET("/:id", h.handleAgentByID)
		agents.PUT("/:id", h.handleAgentByID)
		agents.DELETE("/:id", h.handleAgentByID)
	}

	// Channel API
	channels := router.Group("/api/v1/channels")
	{
		channels.GET("", h.handleChannels)
		channels.POST("", h.createChannel)
		channels.GET("/:id", h.handleChannelByID)
		channels.PUT("/:id", h.updateChannel)
		channels.DELETE("/:id", h.deleteChannel)
	}

	// Session API
	sessions := router.Group("/api/v1/sessions")
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
	}
}

// === Helper Functions ===

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// parseID 从 Gin Context 中解析 ID 参数
func parseID(c *gin.Context, param string) (uint, bool) {
	idStr := c.Param(param)
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, false
	}
	return uint(id), true
}

// parseIDFromPath 从路径解析 ID（兼容旧版本）
func parseIDFromPath(path string, prefix string) (uint, bool) {
	idStr := strings.TrimPrefix(path, prefix)
	idStr = strings.Split(idStr, "/")[0]
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, false
	}
	return uint(id), true
}

// === Response Types ===

// ListResponse 列表响应
type ListResponse struct {
	Items interface{} `json:"items"`
	Total int64       `json:"total,omitempty"`
	Page  int          `json:"page,omitempty"`
	PageSize int      `json:"page_size,omitempty"`
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
