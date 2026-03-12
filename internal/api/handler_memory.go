package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	memorymodels "github.com/weibaohui/nanobot-go/memory/models"
)

// StreamMemoryService 短期记忆服务接口
type StreamMemoryService interface {
	List(ctx context.Context, sessionKey string, eventType string, offset int, limit int) ([]memorymodels.StreamMemory, int64, error)
	Get(ctx context.Context, id uint64) (*memorymodels.StreamMemory, error)
	Create(ctx context.Context, memory *memorymodels.StreamMemory) error
	Update(ctx context.Context, id uint64, memory *memorymodels.StreamMemory) error
	Delete(ctx context.Context, id uint64) error
	MarkProcessed(ctx context.Context, id uint64) error
	GetUnprocessed(ctx context.Context) ([]memorymodels.StreamMemory, error)
}

// LongTermMemoryService 长期记忆服务接口
type LongTermMemoryService interface {
	List(ctx context.Context, offset int, limit int) ([]memorymodels.LongTermMemory, int64, error)
	Get(ctx context.Context, id uint64) (*memorymodels.LongTermMemory, error)
	GetByDate(ctx context.Context, date string) (*memorymodels.LongTermMemory, error)
	Create(ctx context.Context, memory *memorymodels.LongTermMemory) error
	Update(ctx context.Context, id uint64, memory *memorymodels.LongTermMemory) error
	Delete(ctx context.Context, id uint64) error
	Search(ctx context.Context, query string, offset int, limit int) ([]memorymodels.LongTermMemory, int64, error)
	GetRecent(ctx context.Context, days int, offset int, limit int) ([]memorymodels.LongTermMemory, int64, error)
}

// === Stream Memory Handlers ===

func (h *Handler) handleStreamMemories(c *gin.Context) {
	sessionKey := c.Query("session_key")
	eventType := c.Query("event_type")
	offset, _ := strconv.Atoi(c.Query("offset"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit == 0 {
		limit = 50
	}

	memories, total, err := h.streamMemoryService.List(c.Request.Context(), sessionKey, eventType, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ListResponse{
		Items:    memories,
		Total:    total,
		Page:     offset/limit + 1,
		PageSize: limit,
	})
}

func (h *Handler) handleStreamMemoryByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	memory, err := h.streamMemoryService.Get(c.Request.Context(), uint64(id))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, memory)
}

func (h *Handler) createStreamMemory(c *gin.Context) {
	var memory memorymodels.StreamMemory
	if err := c.ShouldBindJSON(&memory); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.streamMemoryService.Create(c.Request.Context(), &memory); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, memory)
}

func (h *Handler) updateStreamMemory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	var memory memorymodels.StreamMemory
	if err := c.ShouldBindJSON(&memory); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.streamMemoryService.Update(c.Request.Context(), uint64(id), &memory); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "更新成功"})
}

func (h *Handler) deleteStreamMemory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	if err := h.streamMemoryService.Delete(c.Request.Context(), uint64(id)); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "删除成功"})
}

func (h *Handler) handleUnprocessedMemories(c *gin.Context) {
	memories, err := h.streamMemoryService.GetUnprocessed(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, memories)
}

// === Long-term Memory Handlers ===

func (h *Handler) handleLongTermMemories(c *gin.Context) {
	offset, _ := strconv.Atoi(c.Query("offset"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit == 0 {
		limit = 50
	}

	memories, total, err := h.longTermMemoryService.List(c.Request.Context(), offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ListResponse{
		Items:    memories,
		Total:    total,
		Page:     offset/limit + 1,
		PageSize: limit,
	})
}

func (h *Handler) handleLongTermMemoryByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	memory, err := h.longTermMemoryService.Get(c.Request.Context(), uint64(id))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, memory)
}

func (h *Handler) handleLongTermMemoryByDate(c *gin.Context) {
	date := c.Param("date")

	memory, err := h.longTermMemoryService.GetByDate(c.Request.Context(), date)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, memory)
}

func (h *Handler) createLongTermMemory(c *gin.Context) {
	var memory memorymodels.LongTermMemory
	if err := c.ShouldBindJSON(&memory); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.longTermMemoryService.Create(c.Request.Context(), &memory); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, memory)
}

func (h *Handler) updateLongTermMemory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	var memory memorymodels.LongTermMemory
	if err := c.ShouldBindJSON(&memory); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.longTermMemoryService.Update(c.Request.Context(), uint64(id), &memory); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "更新成功"})
}

func (h *Handler) deleteLongTermMemory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	if err := h.longTermMemoryService.Delete(c.Request.Context(), uint64(id)); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "删除成功"})
}

func (h *Handler) searchLongTermMemories(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "query parameter is required"})
		return
	}

	offset, _ := strconv.Atoi(c.Query("offset"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit == 0 {
		limit = 50
	}

	memories, total, err := h.longTermMemoryService.Search(c.Request.Context(), query, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ListResponse{
		Items:    memories,
		Total:    total,
		Page:     offset/limit + 1,
		PageSize: limit,
	})
}

func (h *Handler) getRecentLongTermMemories(c *gin.Context) {
	days, _ := strconv.Atoi(c.Query("days"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit == 0 {
		limit = 50
	}

	memories, total, err := h.longTermMemoryService.GetRecent(c.Request.Context(), days, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ListResponse{
		Items:    memories,
		Total:    total,
		Page:     offset/limit + 1,
		PageSize: limit,
	})
}
