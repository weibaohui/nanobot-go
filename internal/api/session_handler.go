package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/nanobot-go/internal/service"
)

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

	if userCode != "" {
		sessions, err := h.sessionService.GetUserSessions(userCode)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, ListResponse{Items: sessions})
		return
	}

	if channelCode != "" {
		sessions, err := h.sessionService.GetChannelSessions(channelCode)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, ListResponse{Items: sessions})
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": "user_code or channel_code is required"})
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

	success := h.sessionManager.CancelSession(sessionKey)
	if !success {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "会话不存在或不在执行中",
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "会话已取消",
	})
}
