package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/nanobot-go/internal/models"
	"github.com/weibaohui/nanobot-go/internal/service/conversation"
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

// ConversationRecordResponse 对话记录响应（包含名称信息）
type ConversationRecordResponse struct {
	models.ConversationRecord
	AgentName   string `json:"agent_name,omitempty"`
	ChannelName string `json:"channel_name,omitempty"`
}

// enrichConversationRecords 为对话记录添加 Agent 和 Channel 名称
func (h *Handler) enrichConversationRecords(records []models.ConversationRecord) []ConversationRecordResponse {
	// 收集唯一的 Code
	agentCodes := make(map[string]bool)
	channelCodes := make(map[string]bool)
	for _, r := range records {
		if r.AgentCode != "" {
			agentCodes[r.AgentCode] = true
		}
		if r.ChannelCode != "" {
			channelCodes[r.ChannelCode] = true
		}
	}

	// 批量查询名称
	agentNames := make(map[string]string)
	channelNames := make(map[string]string)

	for code := range agentCodes {
		if agent, err := h.agentService.GetAgentByCode(code); err == nil && agent != nil {
			agentNames[code] = agent.Name
		}
	}

	for code := range channelCodes {
		if channel, err := h.channelService.GetChannelByCode(code); err == nil && channel != nil {
			channelNames[code] = channel.Name
		}
	}

	// 组装响应
	result := make([]ConversationRecordResponse, len(records))
	for i, r := range records {
		result[i] = ConversationRecordResponse{
			ConversationRecord: r,
			AgentName:          agentNames[r.AgentCode],
			ChannelName:        channelNames[r.ChannelCode],
		}
	}

	return result
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
		Items:    h.enrichConversationRecords(records),
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
		Items:    h.enrichConversationRecords(records),
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

	c.JSON(http.StatusOK, h.enrichConversationRecords(records))
}

// handleConversationByUserAndDate 根据用户编码和日期查询对话记录
func (h *Handler) handleConversationByUserAndDate(c *gin.Context) {
	userCode := c.Param("userCode")
	date := c.Param("date")

	if userCode == "" || date == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "userCode and date are required"})
		return
	}

	records, err := h.conversationService.ListByUserAndDate(c.Request.Context(), userCode, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	// Convert ConversationDTO slice to ConversationRecordResponse slice
	recordDTOs := make([]ConversationDTO, len(records))
	for i, r := range records {
		recordDTOs[i] = ConversationDTO(r)
	}

	// Enrich with agent and channel names
	c.JSON(http.StatusOK, h.enrichConversationDTOs(recordDTOs))
}

// enrichConversationDTOs 为对话DTO添加 Agent 和 Channel 名称
func (h *Handler) enrichConversationDTOs(dtos []conversation.ConversationDTO) []ConversationRecordResponse {
	// Collect unique Codes
	agentCodes := make(map[string]bool)
	channelCodes := make(map[string]bool)
	for _, r := range dtos {
		if r.AgentCode != "" {
			agentCodes[r.AgentCode] = true
		}
		if r.ChannelCode != "" {
			channelCodes[r.ChannelCode] = true
		}
	}

	// Batch query names
	agentNames := make(map[string]string)
	channelNames := make(map[string]string)

	for code := range agentCodes {
		if agent, err := h.agentService.GetAgentByCode(code); err == nil && agent != nil {
			agentNames[code] = agent.Name
		}
	}

	for code := range channelCodes {
		if channel, err := h.channelService.GetChannelByCode(code); err == nil && channel != nil {
			channelNames[code] = channel.Name
		}
	}

	// Assemble response
	result := make([]ConversationRecordResponse, len(dtos))
	for i, r := range dtos {
		result[i] = ConversationRecordResponse{
			ConversationRecord: models.ConversationRecord{
				ID:           r.ID,
				TraceID:      r.TraceID,
				SpanID:       r.SpanID,
				ParentSpanID: r.ParentSpanID,
				EventType:    r.EventType,
				Timestamp:    r.Timestamp,
				SessionKey:   r.SessionKey,
				Role:         r.Role,
				Content:      r.Content,
				CreatedAt:    r.CreatedAt,
				UserCode:     r.UserCode,
				AgentCode:    r.AgentCode,
				ChannelCode:  r.ChannelCode,
				ChannelType:  r.ChannelType,
			},
			AgentName:   agentNames[r.AgentCode],
			ChannelName: channelNames[r.ChannelCode],
		}
		if r.TokenUsage != nil {
			result[i].PromptTokens = r.TokenUsage.PromptTokens
			result[i].CompletionTokens = r.TokenUsage.CompletionTokens
			result[i].TotalTokens = r.TokenUsage.TotalTokens
			result[i].ReasoningTokens = r.TokenUsage.ReasoningTokens
			result[i].CachedTokens = r.TokenUsage.CachedTokens
		}
	}

	return result
}

// ConversationDTO is an alias for conversation.ConversationDTO for local usage
type ConversationDTO = conversation.ConversationDTO

// handleConversationStats 处理对话记录统计请求
func (h *Handler) handleConversationStats(c *gin.Context) {
	// 解析时间范围
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	var startTime, endTime time.Time
	if startTimeStr != "" {
		startTime, _ = time.Parse(time.RFC3339, startTimeStr)
	}
	if endTimeStr != "" {
		endTime, _ = time.Parse(time.RFC3339, endTimeStr)
	}

	// 解析其他筛选条件
	agentCodes := parseQueryArray(c.Query("agent_codes"))
	channelCodes := parseQueryArray(c.Query("channel_codes"))
	roles := parseQueryArray(c.Query("roles"))

	// 如果没有指定时间范围，默认使用最近7天
	if startTime.IsZero() {
		startTime = time.Now().AddDate(0, 0, -7)
	}
	if endTime.IsZero() {
		endTime = time.Now()
	}

	req := &conversation.StatsRequest{
		StartTime:    startTime,
		EndTime:      endTime,
		AgentCodes:   agentCodes,
		ChannelCodes: channelCodes,
		Roles:        roles,
	}

	stats, err := h.conversationService.GetStats(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	// 填充 Agent 名称
	for i := range stats.AgentDistribution {
		code := stats.AgentDistribution[i].Code
		if code == "" {
			// agent_code 为空时显示"未知"
			stats.AgentDistribution[i].Name = "未知"
		} else if agent, err := h.agentService.GetAgentByCode(code); err == nil && agent != nil {
			stats.AgentDistribution[i].Name = agent.Name
		} else {
			// agent_code 不为空但找不到对应 Agent 时显示 code
			stats.AgentDistribution[i].Name = code
		}
	}

	c.JSON(http.StatusOK, stats)
}

// parseQueryArray 解析逗号分隔的查询参数
func parseQueryArray(s string) []string {
	if s == "" {
		return nil
	}
	parts := make([]string, 0)
	for _, p := range splitAndTrim(s, ",") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

// splitAndTrim 分割字符串并去除空白
func splitAndTrim(s, sep string) []string {
	parts := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if i < len(s)-len(sep)+1 && s[i:i+len(sep)] == sep {
			part := trimSpace(s[start:i])
			parts = append(parts, part)
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	if start < len(s) {
		parts = append(parts, trimSpace(s[start:]))
	}
	return parts
}

// trimSpace 去除字符串两端空白
func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	if start >= end {
		return ""
	}
	return s[start:end]
}
