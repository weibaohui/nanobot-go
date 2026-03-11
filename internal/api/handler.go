package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/nanobot-go/internal/models"
	"github.com/weibaohui/nanobot-go/internal/service"
)

// Handler API 处理器
type Handler struct {
	userService     service.UserService
	agentService    service.AgentService
	channelService  service.ChannelService
	sessionService  service.SessionService
	providerService ProviderService
	cronJobService  CronJobService
}

// NewHandler 创建 API 处理器
func NewHandler(
	userService service.UserService,
	agentService service.AgentService,
	channelService service.ChannelService,
	sessionService service.SessionService,
	providerService ProviderService,
	cronJobService CronJobService,
) *Handler {
	return &Handler{
		userService:     userService,
		agentService:   agentService,
		channelService: channelService,
		sessionService: sessionService,
		providerService: providerService,
		cronJobService: cronJobService,
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

	// Provider API
	providers := router.Group("/api/v1/providers")
	{
		providers.GET("", h.handleProviders)
		providers.POST("", h.createProvider)
		providers.GET("/:id", h.handleProviderByID)
		providers.PUT("/:id", h.updateProvider)
		providers.DELETE("/:id", h.deleteProvider)
		providers.POST("/:id/test", h.testProviderConnection)
	}

	// Cron Job API
	cronJobs := router.Group("/api/v1/cron-jobs")
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

// ProviderService Provider 服务接口
type ProviderService interface {
	List(ctx context.Context, userID uint, offset int, limit int) ([]models.LLMProvider, int64, error)
	Get(ctx context.Context, id uint) (*models.LLMProvider, error)
	Create(ctx context.Context, userID uint, req service.CreateProviderRequest) (*models.LLMProvider, error)
	Update(ctx context.Context, id uint, req service.UpdateProviderRequest) error
	Delete(ctx context.Context, id uint) error
	SetDefault(ctx context.Context, userID uint, providerID uint) error
	GetModelConfig(ctx context.Context, id uint) (interface{}, error)
	UpdateModelConfig(ctx context.Context, id uint, config map[string]interface{}) error
	TestConnection(ctx context.Context, id uint) (map[string]interface{}, error)
}

// CronJobService 定时任务服务接口
type CronJobService interface {
	List(ctx context.Context, userID uint, offset int, limit int) ([]models.CronJob, int64, error)
	Get(ctx context.Context, id uint) (*models.CronJob, error)
	Create(ctx context.Context, userID uint, req service.CreateCronJobRequest) (*models.CronJob, error)
	Update(ctx context.Context, id uint, req service.UpdateCronJobRequest) error
	Delete(ctx context.Context, id uint) error
	Enable(ctx context.Context, id uint) error
	Disable(ctx context.Context, id uint) error
	Execute(ctx context.Context, id uint) error
	GetPending(ctx context.Context) ([]models.CronJob, error)
}

// === Provider Handlers ===

func (h *Handler) handleProviders(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Query("user_id"), 10, 32)
	offset, _ := strconv.Atoi(c.Query("offset"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit == 0 {
		limit = 20
	}

	providers, total, err := h.providerService.List(c.Request.Context(), uint(userID), offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ListResponse{
		Items:    providers,
		Total:    total,
		Page:     offset/limit + 1,
		PageSize: limit,
	})
}

func (h *Handler) handleProviderByID(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	if c.Request.Method == "GET" {
		provider, err := h.providerService.Get(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusOK, provider)
	}
}

func (h *Handler) createProvider(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Query("user_id"), 10, 32)
	var req service.CreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	provider, err := h.providerService.Create(c.Request.Context(), uint(userID), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, provider)
}

func (h *Handler) updateProvider(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	var req service.UpdateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.providerService.Update(c.Request.Context(), id, req); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "更新成功"})
}

func (h *Handler) deleteProvider(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	if err := h.providerService.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "删除成功"})
}

func (h *Handler) testProviderConnection(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	result, err := h.providerService.TestConnection(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// === Cron Job Handlers ===

func (h *Handler) handleCronJobs(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Query("user_id"), 10, 32)
	offset, _ := strconv.Atoi(c.Query("offset"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit == 0 {
		limit = 20
	}

	jobs, total, err := h.cronJobService.List(c.Request.Context(), uint(userID), offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ListResponse{
		Items:    jobs,
		Total:    total,
		Page:     offset/limit + 1,
		PageSize: limit,
	})
}

func (h *Handler) handlePendingCronJobs(c *gin.Context) {
	jobs, err := h.cronJobService.GetPending(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, jobs)
}

func (h *Handler) handleCronJobByID(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	job, err := h.cronJobService.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, job)
}

func (h *Handler) createCronJob(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Query("user_id"), 10, 32)
	var req service.CreateCronJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	job, err := h.cronJobService.Create(c.Request.Context(), uint(userID), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, job)
}

func (h *Handler) updateCronJob(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	var req service.UpdateCronJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.cronJobService.Update(c.Request.Context(), id, req); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "更新成功"})
}

func (h *Handler) deleteCronJob(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	if err := h.cronJobService.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "删除成功"})
}

func (h *Handler) enableCronJob(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	if err := h.cronJobService.Enable(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "已启用"})
}

func (h *Handler) disableCronJob(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	if err := h.cronJobService.Disable(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "已禁用"})
}

func (h *Handler) executeCronJob(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	if err := h.cronJobService.Execute(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "已开始执行"})
}