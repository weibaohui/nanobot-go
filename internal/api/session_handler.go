package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/nanobot-go/internal/models"
	"github.com/weibaohui/nanobot-go/internal/service"
)

// SessionResponse 带名称的 Session 响应
type SessionResponse struct {
	ID          uint   `json:"id"`
	SessionKey  string `json:"session_key"`
	UserCode    string `json:"user_code"`
	ChannelCode string `json:"channel_code"`
	AgentCode   string `json:"agent_code,omitempty"`
	ExternalID  string `json:"external_id,omitempty"`
	Metadata    string `json:"metadata,omitempty"`

	// 名称字段
	UserName    string `json:"user_name,omitempty"`
	ChannelName string `json:"channel_name,omitempty"`
	AgentName   string `json:"agent_name,omitempty"`

	LastActiveAt *string `json:"last_active_at"`
	CreatedAt    string  `json:"created_at"`
}

// toSessionResponse 将 Session 转换为包含名称的响应
func toSessionResponse(session *models.Session, svc CodeLookupService) SessionResponse {
	resp := SessionResponse{
		ID:          session.ID,
		SessionKey:  session.SessionKey,
		UserCode:    session.UserCode,
		ChannelCode: session.ChannelCode,
		AgentCode:   session.AgentCode,
		ExternalID:  session.ExternalID,
		Metadata:    session.Metadata,
		CreatedAt:   session.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if session.LastActiveAt != nil {
		t := session.LastActiveAt.Format("2006-01-02T15:04:05Z07:00")
		resp.LastActiveAt = &t
	}

	// 查询用户名称
	if session.UserCode != "" {
		if user, err := svc.GetUserByCode(session.UserCode); err == nil && user != nil {
			resp.UserName = user.DisplayName
			if resp.UserName == "" {
				resp.UserName = user.Username
			}
		}
	}

	// 查询渠道名称
	if session.ChannelCode != "" {
		if ch, err := svc.GetChannelByCode(session.ChannelCode); err == nil && ch != nil {
			resp.ChannelName = ch.Name
		}
	}

	// 查询 Agent 名称
	if session.AgentCode != "" {
		if agent, err := svc.GetAgentByCode(session.AgentCode); err == nil && agent != nil {
			resp.AgentName = agent.Name
		}
	}

	return resp
}

// handleSessions 处理 /api/v1/sessions
func (h *Handler) handleSessions(c *gin.Context) {
	switch c.Request.Method {
	case http.MethodGet:
		h.listSessions(c)
	case http.MethodPost:
		h.createSession(c)
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "method not allowed"})
	}
}

// handleSessionByKey 处理 /api/v1/sessions/{session_key}
func (h *Handler) handleSessionByKey(c *gin.Context) {
	sessionKey := c.Param("id")
	if sessionKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session key is required"})
		return
	}

	// Check URL path for sub-routes
	path := c.Request.URL.Path
	if strings.Contains(path, "/touch") {
		h.handleSessionTouch(c, sessionKey)
		return
	}
	if strings.Contains(path, "/metadata") {
		h.handleSessionMetadata(c, sessionKey)
		return
	}

	switch c.Request.Method {
	case http.MethodGet:
		h.getSession(c, sessionKey)
	case http.MethodDelete:
		h.deleteSession(c, sessionKey)
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "method not allowed"})
	}
}

// listSessions 获取 Session 列表
func (h *Handler) listSessions(c *gin.Context) {
	// 支持按 user_code 或 channel_code 查询
	userCode := c.Query("user_code")
	channelCode := c.Query("channel_code")

	var sessions []models.Session
	var err error

	if userCode != "" {
		sessions, err = h.sessionService.GetUserSessions(userCode)
	} else if channelCode != "" {
		sessions, err = h.sessionService.GetChannelSessions(channelCode)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_code or channel_code is required"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 转换为包含名称的响应
	items := make([]SessionResponse, 0, len(sessions))
	for i := range sessions {
		items = append(items, toSessionResponse(&sessions[i], h.codeLookupService))
	}

	c.JSON(http.StatusOK, ListResponse{Items: items})
}

// createSession 创建 Session
func (h *Handler) createSession(c *gin.Context) {
	var req struct {
		UserCode    string                 `json:"user_code"`
		ChannelCode string                 `json:"channel_code"`
		AgentCode   string                 `json:"agent_code,omitempty"`
		SessionKey  string                 `json:"session_key"`
		ExternalID  string                 `json:"external_id,omitempty"`
		Metadata    map[string]interface{} `json:"metadata,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.UserCode == "" || req.ChannelCode == "" || req.SessionKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_code, channel_code and session_key are required"})
		return
	}

	sessionReq := service.CreateSessionRequest{
		SessionKey:  req.SessionKey,
		ExternalID:  req.ExternalID,
		AgentCode:   req.AgentCode,
		ChannelCode: req.ChannelCode,
		Metadata:    req.Metadata,
	}

	session, err := h.sessionService.CreateSession(req.UserCode, sessionReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, session)
}

// getSession 获取 Session
func (h *Handler) getSession(c *gin.Context, sessionKey string) {
	session, err := h.sessionService.GetSessionByKey(sessionKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if session == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	c.JSON(http.StatusOK, session)
}

// deleteSession 删除 Session
func (h *Handler) deleteSession(c *gin.Context, sessionKey string) {
	if err := h.sessionService.DeleteSession(sessionKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "session deleted"})
}

// handleSessionTouch 更新 Session 活跃时间
func (h *Handler) handleSessionTouch(c *gin.Context, sessionKey string) {
	if c.Request.Method != http.MethodPost {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "method not allowed"})
		return
	}

	if err := h.sessionService.TouchSession(sessionKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "session touched"})
}

// handleSessionMetadata 处理 Session 元数据
func (h *Handler) handleSessionMetadata(c *gin.Context, sessionKey string) {
	switch c.Request.Method {
	case http.MethodGet:
		metadata, err := h.sessionService.GetSessionMetadata(sessionKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, metadata)
	case http.MethodPut:
		var metadata map[string]interface{}
		if err := c.ShouldBindJSON(&metadata); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		if err := h.sessionService.UpdateSessionMetadata(sessionKey, metadata); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, SuccessResponse{Message: "metadata updated"})
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "method not allowed"})
	}
}

// cancelSession 取消正在执行的会话
// 直接尝试执行 cancel，有 context 就取消，没有就返回提示
func (h *Handler) cancelSession(c *gin.Context) {
	sessionKey := c.Param("id")
	if sessionKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session key is required"})
		return
	}

	if h.sessionManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "session manager not available"})
		return
	}

	// 直接尝试取消，让底层判断是否有可取消的执行
	success := h.sessionManager.CancelSession(sessionKey)
	if !success {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "当前没有正在执行的任务",
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "会话已取消",
	})
}

// checkSessionActive 检查会话是否活跃（正在执行中）
func (h *Handler) checkSessionActive(c *gin.Context) {
	sessionKey := c.Param("id")
	if sessionKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session key is required"})
		return
	}

	if h.sessionManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "session manager not available"})
		return
	}

	isActive := h.sessionManager.IsSessionActive(sessionKey)
	c.JSON(http.StatusOK, gin.H{
		"session_key": sessionKey,
		"is_active":   isActive,
	})
}
