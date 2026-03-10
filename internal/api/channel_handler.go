package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/weibaohui/nanobot-go/internal/service"
)

// handleChannels 处理 /api/v1/channels
func (h *Handler) handleChannels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listChannels(w, r)
	case http.MethodPost:
		h.createChannel(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleChannelByID 处理 /api/v1/channels/{id} 及其子路径
func (h *Handler) handleChannelByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/channels/")
	parts := strings.Split(path, "/")
	idStr := parts[0]

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	// 处理子路径
	if len(parts) > 1 {
		subPath := parts[1]
		switch subPath {
		case "bind":
			h.handleChannelBind(w, r, uint(id))
			return
		case "unbind":
			h.handleChannelUnbind(w, r, uint(id))
			return
		case "config":
			h.handleChannelConfig(w, r, uint(id))
			return
		case "allowlist":
			h.handleChannelAllowList(w, r, uint(id))
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		h.getChannel(w, r, uint(id))
	case http.MethodPut:
		h.updateChannel(w, r, uint(id))
	case http.MethodDelete:
		h.deleteChannel(w, r, uint(id))
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// listChannels 获取 Channel 列表
func (h *Handler) listChannels(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	channels, err := h.channelService.GetUserChannels(uint(userID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ListResponse{
		Data: channels,
	})
}

// createChannel 创建 Channel
func (h *Handler) createChannel(w http.ResponseWriter, r *http.Request) {
	var req service.CreateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	channel, err := h.channelService.CreateChannel(uint(userID), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, channel)
}

// getChannel 获取 Channel
func (h *Handler) getChannel(w http.ResponseWriter, r *http.Request, id uint) {
	channel, err := h.channelService.GetChannel(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if channel == nil {
		writeError(w, http.StatusNotFound, "channel not found")
		return
	}

	writeJSON(w, http.StatusOK, channel)
}

// updateChannel 更新 Channel
func (h *Handler) updateChannel(w http.ResponseWriter, r *http.Request, id uint) {
	var req service.UpdateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	channel, err := h.channelService.UpdateChannel(id, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, channel)
}

// deleteChannel 删除 Channel
func (h *Handler) deleteChannel(w http.ResponseWriter, r *http.Request, id uint) {
	if err := h.channelService.DeleteChannel(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Message: "channel deleted"})
}

// handleChannelBind 处理 Channel 绑定 Agent
func (h *Handler) handleChannelBind(w http.ResponseWriter, r *http.Request, channelID uint) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		AgentID uint `json:"agent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.channelService.BindAgent(channelID, req.AgentID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Message: "agent bound to channel"})
}

// handleChannelUnbind 处理 Channel 解绑 Agent
func (h *Handler) handleChannelUnbind(w http.ResponseWriter, r *http.Request, channelID uint) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if err := h.channelService.UnbindAgent(channelID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Message: "agent unbound from channel"})
}

// handleChannelConfig 处理 Channel 配置
func (h *Handler) handleChannelConfig(w http.ResponseWriter, r *http.Request, channelID uint) {
	switch r.Method {
	case http.MethodGet:
		config, err := h.channelService.GetChannelConfig(channelID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, config)
	case http.MethodPut:
		var config map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := h.channelService.UpdateChannelConfig(channelID, config); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, SuccessResponse{Message: "config updated"})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleChannelAllowList 处理 Channel 白名单
func (h *Handler) handleChannelAllowList(w http.ResponseWriter, r *http.Request, channelID uint) {
	switch r.Method {
	case http.MethodGet:
		allowList, err := h.channelService.GetAllowList(channelID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"allow_list": allowList})
	case http.MethodPut:
		var req struct {
			AllowList []string `json:"allow_list"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := h.channelService.SetAllowList(channelID, req.AllowList); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, SuccessResponse{Message: "allow list updated"})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
