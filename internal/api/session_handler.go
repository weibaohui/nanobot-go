package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/weibaohui/nanobot-go/internal/service"
)

// handleSessions 处理 /api/v1/sessions
func (h *Handler) handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listSessions(w, r)
	case http.MethodPost:
		h.createSession(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleSessionByKey 处理 /api/v1/sessions/{session_key}
func (h *Handler) handleSessionByKey(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/sessions/")
	parts := strings.Split(path, "/")
	sessionKey := parts[0]

	if sessionKey == "" {
		writeError(w, http.StatusBadRequest, "session key is required")
		return
	}

	// 处理子路径
	if len(parts) > 1 {
		subPath := parts[1]
		switch subPath {
		case "touch":
			h.handleSessionTouch(w, r, sessionKey)
			return
		case "metadata":
			h.handleSessionMetadata(w, r, sessionKey)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		h.getSession(w, r, sessionKey)
	case http.MethodDelete:
		h.deleteSession(w, r, sessionKey)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// listSessions 获取 Session 列表
func (h *Handler) listSessions(w http.ResponseWriter, r *http.Request) {
	// 支持按 user_id 或 channel_id 查询
	userIDStr := r.URL.Query().Get("user_id")
	channelIDStr := r.URL.Query().Get("channel_id")

	if userIDStr != "" {
		userID, err := strconv.ParseUint(userIDStr, 10, 32)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid user_id")
			return
		}
		sessions, err := h.sessionService.GetUserSessions(uint(userID))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, ListResponse{Data: sessions})
		return
	}

	if channelIDStr != "" {
		channelID, err := strconv.ParseUint(channelIDStr, 10, 32)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid channel_id")
			return
		}
		sessions, err := h.sessionService.GetChannelSessions(uint(channelID))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, ListResponse{Data: sessions})
		return
	}

	writeError(w, http.StatusBadRequest, "user_id or channel_id is required")
}

// createSession 创建 Session
func (h *Handler) createSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID     uint                   `json:"user_id"`
		ChannelID  uint                   `json:"channel_id"`
		AgentID    *uint                  `json:"agent_id,omitempty"`
		SessionKey string                 `json:"session_key"`
		ExternalID string                 `json:"external_id,omitempty"`
		Metadata   map[string]interface{} `json:"metadata,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID == 0 || req.ChannelID == 0 || req.SessionKey == "" {
		writeError(w, http.StatusBadRequest, "user_id, channel_id and session_key are required")
		return
	}

	sessionReq := service.CreateSessionRequest{
		SessionKey: req.SessionKey,
		ExternalID: req.ExternalID,
		AgentID:    req.AgentID,
		ChannelID:  req.ChannelID,
		Metadata:   req.Metadata,
	}

	session, err := h.sessionService.CreateSession(req.UserID, sessionReq)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, session)
}

// getSession 获取 Session
func (h *Handler) getSession(w http.ResponseWriter, r *http.Request, sessionKey string) {
	session, err := h.sessionService.GetSessionByKey(sessionKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if session == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	writeJSON(w, http.StatusOK, session)
}

// deleteSession 删除 Session
func (h *Handler) deleteSession(w http.ResponseWriter, r *http.Request, sessionKey string) {
	if err := h.sessionService.DeleteSession(sessionKey); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Message: "session deleted"})
}

// handleSessionTouch 更新 Session 活跃时间
func (h *Handler) handleSessionTouch(w http.ResponseWriter, r *http.Request, sessionKey string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if err := h.sessionService.TouchSession(sessionKey); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Message: "session touched"})
}

// handleSessionMetadata 处理 Session 元数据
func (h *Handler) handleSessionMetadata(w http.ResponseWriter, r *http.Request, sessionKey string) {
	switch r.Method {
	case http.MethodGet:
		metadata, err := h.sessionService.GetSessionMetadata(sessionKey)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, metadata)
	case http.MethodPut:
		var metadata map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&metadata); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := h.sessionService.UpdateSessionMetadata(sessionKey, metadata); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, SuccessResponse{Message: "metadata updated"})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
