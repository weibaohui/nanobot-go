package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/nanobot-go/internal/models"
	"github.com/weibaohui/nanobot-go/internal/service"
)

// ProviderService Provider 服务接口
type ProviderService interface {
	List(ctx context.Context, userCode string, offset int, limit int) ([]models.LLMProvider, int64, error)
	Get(ctx context.Context, id uint) (*models.LLMProvider, error)
	Create(ctx context.Context, userCode string, req service.CreateProviderRequest) (*models.LLMProvider, error)
	Update(ctx context.Context, id uint, req service.UpdateProviderRequest) error
	Delete(ctx context.Context, id uint) error
	SetDefault(ctx context.Context, userCode string, providerID uint) error
	GetModelConfig(ctx context.Context, id uint) (interface{}, error)
	UpdateModelConfig(ctx context.Context, id uint, config map[string]interface{}) error
	TestConnection(ctx context.Context, id uint) (map[string]interface{}, error)
}

// === Provider Handlers ===

func (h *Handler) handleProviders(c *gin.Context) {
	userCode := c.Query("user_code")
	offset, _ := strconv.Atoi(c.Query("offset"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit == 0 {
		limit = 20
	}

	providers, total, err := h.providerService.List(c.Request.Context(), userCode, offset, limit)
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
	userCode := c.Query("user_code")
	var req service.CreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	provider, err := h.providerService.Create(c.Request.Context(), userCode, req)
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

// === Embedding Model Handlers ===

// UpdateEmbeddingModelsRequest 更新嵌入模型请求
type UpdateEmbeddingModelsRequest struct {
	EmbeddingModels       []models.EmbeddingModelInfo `json:"embedding_models"`
	DefaultEmbeddingModel string                      `json:"default_embedding_model"`
}

// updateProviderEmbeddingModels 更新 Provider 的嵌入模型配置
func (h *Handler) updateProviderEmbeddingModels(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	var req UpdateEmbeddingModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// 获取 Provider
	provider, err := h.providerService.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "provider not found"})
		return
	}

	// 设置嵌入模型
	if err := provider.SetEmbeddingModels(req.EmbeddingModels); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	provider.DefaultEmbeddingModel = req.DefaultEmbeddingModel

	// 保存
	updateReq := service.UpdateProviderRequest{
		EmbeddingModels:       provider.EmbeddingModels,
		DefaultEmbeddingModel: provider.DefaultEmbeddingModel,
	}
	if err := h.providerService.Update(c.Request.Context(), id, updateReq); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "嵌入模型配置更新成功"})
}

// getProviderEmbeddingModels 获取 Provider 的嵌入模型配置
func (h *Handler) getProviderEmbeddingModels(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	// 获取 Provider
	provider, err := h.providerService.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "provider not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"embedding_models":        provider.GetEmbeddingModels(),
		"default_embedding_model": provider.DefaultEmbeddingModel,
		"has_embedding_models":    provider.HasEmbeddingModels(),
	})
}
