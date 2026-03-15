package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	memorymodels "github.com/weibaohui/nanobot-go/internal/memory/models"
)

// StreamMemoryService 短期记忆服务接口
type StreamMemoryService interface {
	List(ctx context.Context, userCode string, agentCode string, offset int, limit int) ([]memorymodels.StreamMemory, int64, error)
	Get(ctx context.Context, id uint64) (*memorymodels.StreamMemory, error)
	GetByUserAgentAndDate(ctx context.Context, userCode string, agentCode string, date string) (*memorymodels.StreamMemory, error)
	Create(ctx context.Context, memory *memorymodels.StreamMemory) error
	Update(ctx context.Context, id uint64, memory *memorymodels.StreamMemory) error
	Delete(ctx context.Context, id uint64) error
	MarkProcessed(ctx context.Context, id uint64) error
	GetUnprocessed(ctx context.Context) ([]memorymodels.StreamMemory, error)
	BuildFromConversations(ctx context.Context, userCode string, agentCode string, date string, conversationIDs []string, contents []string) error
}

// LongTermMemoryService 长期记忆服务接口
type LongTermMemoryService interface {
	List(ctx context.Context, userCode string, agentCode string, offset int, limit int) ([]memorymodels.LongTermMemory, int64, error)
	Get(ctx context.Context, id uint64) (*memorymodels.LongTermMemory, error)
	GetByDate(ctx context.Context, userCode string, agentCode string, date string) (*memorymodels.LongTermMemory, error)
	Create(ctx context.Context, memory *memorymodels.LongTermMemory) error
	Update(ctx context.Context, id uint64, memory *memorymodels.LongTermMemory) error
	Delete(ctx context.Context, id uint64) error
	Search(ctx context.Context, userCode string, agentCode string, query string, offset int, limit int) ([]memorymodels.LongTermMemory, int64, error)
	GetRecent(ctx context.Context, userCode string, agentCode string, days int, offset int, limit int) ([]memorymodels.LongTermMemory, int64, error)
}

// === Stream Memory Handlers ===

func (h *Handler) handleStreamMemories(c *gin.Context) {
	userCode := c.Query("user_code")
	agentCode := c.Query("agent_code")
	offset, _ := strconv.Atoi(c.Query("offset"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit == 0 {
		limit = 50
	}

	memories, total, err := h.streamMemoryService.List(c.Request.Context(), userCode, agentCode, offset, limit)
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

// handleBuildStreamMemory 从对话记录构建短期记忆
// 将选中的对话记录按用户+Agent+日期聚合为短期记忆
func (h *Handler) handleBuildStreamMemory(c *gin.Context) {
	var req struct {
		UserCode        string   `json:"user_code" binding:"required"`
		AgentCode       string   `json:"agent_code"`
		Date            string   `json:"date" binding:"required"`
		ConversationIDs []string `json:"conversation_ids" binding:"required"`
		Contents        []string `json:"contents" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request: " + err.Error()})
		return
	}

	ctx := c.Request.Context()

	// 构建短期记忆（按用户+Agent+日期聚合）
	if err := h.streamMemoryService.BuildFromConversations(ctx, req.UserCode, req.AgentCode, req.Date, req.ConversationIDs, req.Contents); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "构建短期记忆失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "短期记忆构建成功",
		Data: map[string]interface{}{
			"user_code":        req.UserCode,
			"agent_code":       req.AgentCode,
			"date":             req.Date,
			"conversation_count": len(req.ConversationIDs),
		},
	})
}

// handleUpgradeMemories 手动触发记忆升级
// 将指定日期的未处理流水记忆升级为长期记忆（按用户+Agent分组）
func (h *Handler) handleUpgradeMemories(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "date parameter is required, format: YYYY-MM-DD"})
		return
	}

	ctx := c.Request.Context()

	// 获取未处理的流水记忆
	unprocessedMemories, err := h.streamMemoryService.GetUnprocessed(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "获取未处理记忆失败: " + err.Error()})
		return
	}

	if len(unprocessedMemories) == 0 {
		c.JSON(http.StatusOK, SuccessResponse{Message: "没有需要升级的记忆"})
		return
	}

	// 统计信息
	totalCount := len(unprocessedMemories)
	upgradedCount := 0

	// 按 UserCode + AgentCode 分组（用于用户和Agent隔离）
	userAgentMemories := make(map[string][]memorymodels.StreamMemory)
	for _, memory := range unprocessedMemories {
		userCode := memory.UserCode
		if userCode == "" {
			userCode = "default"
		}
		agentCode := memory.AgentCode
		if agentCode == "" {
			agentCode = "default"
		}
		key := userCode + ":" + agentCode
		userAgentMemories[key] = append(userAgentMemories[key], memory)
	}

	// 为每个用户+Agent组合创建长期记忆
	for key, memories := range userAgentMemories {
		parts := strings.SplitN(key, ":", 2)
		userCode, agentCode := parts[0], parts[1]

		// 构建总结内容
		var summary strings.Builder
		summary.WriteString(fmt.Sprintf("日期: %s 的记忆汇总\n\n", date))
		summary.WriteString(fmt.Sprintf("用户: %s\n", userCode))
		summary.WriteString(fmt.Sprintf("Agent: %s\n", agentCode))
		summary.WriteString(fmt.Sprintf("共 %d 条流水记忆\n\n", len(memories)))

		for i, m := range memories {
			summary.WriteString(fmt.Sprintf("--- 记忆 %d ---\n", i+1))
			summary.WriteString(fmt.Sprintf("日期: %s\n", m.Date))
			if m.Summary != "" {
				summary.WriteString(fmt.Sprintf("总结: %s\n", m.Summary))
			}
			if m.Content != "" {
				summary.WriteString(fmt.Sprintf("内容摘要: %s\n", truncateString(m.Content, 300)))
			}
			if m.SourceIDs != "" {
				summary.WriteString(fmt.Sprintf("来源对话: %s\n", m.SourceIDs))
			}
			summary.WriteString("\n")
		}

		// 创建长期记忆
		longTermMemory := &memorymodels.LongTermMemory{
			Date:         date,
			UserCode:     userCode,
			AgentCode:    agentCode,
			Summary:      fmt.Sprintf("%s 的记忆汇总 (%d 条)", date, len(memories)),
			WhatHappened: summary.String(),
			Conclusion:   fmt.Sprintf("共处理 %d 条流水记忆", len(memories)),
			Value:        "通过手动触发升级按钮生成",
		}

		// 检查是否已存在（按用户+Agent+日期）
		existing, err := h.longTermMemoryService.GetByDate(ctx, userCode, agentCode, date)
		if err != nil {
			// 查询失败，继续创建新的
		}

		if existing != nil {
			// 更新现有记录
			longTermMemory.ID = existing.ID
			if err := h.longTermMemoryService.Update(ctx, existing.ID, longTermMemory); err != nil {
				c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "更新长期记忆失败: " + err.Error()})
				return
			}
		} else {
			// 创建新记录
			if err := h.longTermMemoryService.Create(ctx, longTermMemory); err != nil {
				c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "创建长期记忆失败: " + err.Error()})
				return
			}
		}

		// 标记流水记忆为已处理
		for _, m := range memories {
			if err := h.streamMemoryService.MarkProcessed(ctx, m.ID); err != nil {
				// 记录错误但继续处理
			}
		}

		upgradedCount += len(memories)
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: fmt.Sprintf("记忆升级完成，共处理 %d 条流水记忆，涉及 %d 个用户+Agent组合", totalCount, len(userAgentMemories)),
		Data: map[string]interface{}{
			"date":              date,
			"total_count":       totalCount,
			"upgraded_count":    upgradedCount,
			"user_agent_count":  len(userAgentMemories),
		},
	})
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// === Long-term Memory Handlers ===

func (h *Handler) handleLongTermMemories(c *gin.Context) {
	userCode := c.Query("user_code")
	agentCode := c.Query("agent_code")
	offset, _ := strconv.Atoi(c.Query("offset"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit == 0 {
		limit = 50
	}

	memories, total, err := h.longTermMemoryService.List(c.Request.Context(), userCode, agentCode, offset, limit)
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
	userCode := c.Query("user_code")
	agentCode := c.Query("agent_code")

	memory, err := h.longTermMemoryService.GetByDate(c.Request.Context(), userCode, agentCode, date)
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

	userCode := c.Query("user_code")
	agentCode := c.Query("agent_code")
	offset, _ := strconv.Atoi(c.Query("offset"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit == 0 {
		limit = 50
	}

	memories, total, err := h.longTermMemoryService.Search(c.Request.Context(), userCode, agentCode, query, offset, limit)
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
	userCode := c.Query("user_code")
	agentCode := c.Query("agent_code")
	offset, _ := strconv.Atoi(c.Query("offset"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit == 0 {
		limit = 50
	}

	memories, total, err := h.longTermMemoryService.GetRecent(c.Request.Context(), userCode, agentCode, days, offset, limit)
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
