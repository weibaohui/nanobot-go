package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/weibaohui/nanobot-go/internal/service"
)

// handleAgents 处理 /api/v1/agents
func (h *Handler) handleAgents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listAgents(w, r)
	case http.MethodPost:
		h.createAgent(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleAgentByID 处理 /api/v1/agents/{id} 及其子路径
func (h *Handler) handleAgentByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
	parts := strings.Split(path, "/")
	idStr := parts[0]

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid agent id")
		return
	}

	// 处理子路径
	if len(parts) > 1 {
		subPath := parts[1]
		switch subPath {
		case "config":
			h.handleAgentConfig(w, r, uint(id))
			return
		case "memory":
			h.handleAgentMemory(w, r, uint(id))
			return
		case "skills":
			h.handleAgentSkills(w, r, uint(id))
			return
		case "tools":
			h.handleAgentTools(w, r, uint(id))
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		h.getAgent(w, r, uint(id))
	case http.MethodPut:
		h.updateAgent(w, r, uint(id))
	case http.MethodDelete:
		h.deleteAgent(w, r, uint(id))
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// listAgents 获取 Agent 列表
func (h *Handler) listAgents(w http.ResponseWriter, r *http.Request) {
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

	agents, err := h.agentService.GetUserAgents(uint(userID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ListResponse{
		Data: agents,
	})
}

// createAgent 创建 Agent
func (h *Handler) createAgent(w http.ResponseWriter, r *http.Request) {
	var req service.CreateAgentRequest
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

	agent, err := h.agentService.CreateAgent(uint(userID), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, agent)
}

// getAgent 获取 Agent
func (h *Handler) getAgent(w http.ResponseWriter, r *http.Request, id uint) {
	agent, err := h.agentService.GetAgent(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if agent == nil {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}

	writeJSON(w, http.StatusOK, agent)
}

// updateAgent 更新 Agent
func (h *Handler) updateAgent(w http.ResponseWriter, r *http.Request, id uint) {
	var req service.UpdateAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	agent, err := h.agentService.UpdateAgent(id, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, agent)
}

// deleteAgent 删除 Agent
func (h *Handler) deleteAgent(w http.ResponseWriter, r *http.Request, id uint) {
	if err := h.agentService.DeleteAgent(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Message: "agent deleted"})
}

// handleAgentConfig 处理 Agent 配置
func (h *Handler) handleAgentConfig(w http.ResponseWriter, r *http.Request, agentID uint) {
	switch r.Method {
	case http.MethodGet:
		config, err := h.agentService.GetAgentConfig(agentID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, config)
	case http.MethodPut:
		var config service.AgentConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := h.agentService.UpdateAgentConfig(agentID, &config); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, SuccessResponse{Message: "config updated"})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleAgentMemory 处理 Agent 记忆
func (h *Handler) handleAgentMemory(w http.ResponseWriter, r *http.Request, agentID uint) {
	switch r.Method {
	case http.MethodGet:
		memory, err := h.agentService.GetMemory(agentID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"content": memory})
	case http.MethodPut:
		var req struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := h.agentService.UpdateMemory(agentID, req.Content); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, SuccessResponse{Message: "memory updated"})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleAgentSkills 处理 Agent 技能
func (h *Handler) handleAgentSkills(w http.ResponseWriter, r *http.Request, agentID uint) {
	switch r.Method {
	case http.MethodGet:
		skills, err := h.agentService.GetAvailableSkills(agentID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"skills": skills})
	case http.MethodPut:
		var req struct {
			Skills []string `json:"skills"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := h.agentService.SetAvailableSkills(agentID, req.Skills); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, SuccessResponse{Message: "skills updated"})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleAgentTools 处理 Agent 工具
func (h *Handler) handleAgentTools(w http.ResponseWriter, r *http.Request, agentID uint) {
	switch r.Method {
	case http.MethodGet:
		tools, err := h.agentService.GetAvailableTools(agentID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"tools": tools})
	case http.MethodPut:
		var req struct {
			Tools []string `json:"tools"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := h.agentService.SetAvailableTools(agentID, req.Tools); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, SuccessResponse{Message: "tools updated"})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
