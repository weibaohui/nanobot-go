package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/nanobot-go/internal/models"
)

// ConversationRecordService 对话记录服务接口
type ConversationRecordService interface {
	List(ctx context.Context, userID uint, agentID uint, channelID uint, sessionKey string, offset int, limit int) ([]models.ConversationRecord, int64, error)
	Get(ctx context.Context, id uint) (*models.ConversationRecord, error)
	Create(ctx context.Context, record *models.ConversationRecord) error
	Update(ctx context.Context, id uint, record *models.ConversationRecord) error
	Delete(ctx context.Context, id uint) error
	GetBySessionKey(ctx context.Context, sessionKey string, offset int, limit int) ([]models.ConversationRecord, int64, error)
	GetByTraceID(ctx context.Context, traceID string) ([]models.ConversationRecord, error)
}

// === Conversation Record Handlers ===

func (h *Handler) handleConversationRecords(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Query("user_id"), 10, 32)
	agentID, _ := strconv.ParseUint(c.Query("agent_id"), 10, 32)
	channelID, _ := strconv.ParseUint(c.Query("channel_id"), 10, 32)
	sessionKey := c.Query("session_key")
	offset, _ := strconv.Atoi(c.Query("offset"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit == 0 {
		limit = 50
	}

	records, total, err := h.conversationRecordService.List(c.Request.Context(), uint(userID), uint(agentID), uint(channelID), sessionKey, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ListResponse{
		Items:    records,
		Total:    total,
		Page:     offset/limit + 1,
		PageSize: limit,
	})
}

func (h *Handler) handleConversationRecordByID(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	record, err := h.conversationRecordService.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *Handler) createConversationRecord(c *gin.Context) {
	var record models.ConversationRecord
	if err := c.ShouldBindJSON(&record); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.conversationRecordService.Create(c.Request.Context(), &record); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *Handler) updateConversationRecord(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	var record models.ConversationRecord
	if err := c.ShouldBindJSON(&record); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.conversationRecordService.Update(c.Request.Context(), id, &record); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "更新成功"})
}

func (h *Handler) deleteConversationRecord(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	if err := h.conversationRecordService.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "删除成功"})
}

func (h *Handler) handleConversationBySession(c *gin.Context) {
	sessionKey := c.Param("sessionKey")
	offset, _ := strconv.Atoi(c.Query("offset"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit == 0 {
		limit = 50
	}

	records, total, err := h.conversationRecordService.GetBySessionKey(c.Request.Context(), sessionKey, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ListResponse{
		Items:    records,
		Total:    total,
		Page:     offset/limit + 1,
		PageSize: limit,
	})
}

func (h *Handler) handleConversationByTrace(c *gin.Context) {
	traceID := c.Param("traceID")

	records, err := h.conversationRecordService.GetByTraceID(c.Request.Context(), traceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, records)
}
