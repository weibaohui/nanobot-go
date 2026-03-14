package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	agentsvc "github.com/weibaohui/nanobot-go/internal/service/agent"
)

// handleAgents 处理 /api/v1/agents
func (h *Handler) handleAgents(c *gin.Context) {
	switch c.Request.Method {
	case http.MethodGet:
		h.listAgents(c)
	case http.MethodPost:
		h.createAgent(c)
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "method not allowed"})
	}
}

// handleAgentByID 处理 /api/v1/agents/{id}
func (h *Handler) handleAgentByID(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent id"})
		return
	}

	switch c.Request.Method {
	case http.MethodGet:
		h.getAgent(c, uint(id))
	case http.MethodPut:
		h.updateAgent(c, uint(id))
	case http.MethodDelete:
		h.deleteAgent(c, uint(id))
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "method not allowed"})
	}
}

// listAgents 获取 Agent 列表
func (h *Handler) listAgents(c *gin.Context) {
	userCode := c.Query("user_code")
	if userCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_code is required"})
		return
	}

	agents, err := h.agentService.GetUserAgents(userCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ListResponse{
		Items: agents,
		Total: int64(len(agents)),
	})
}

// createAgent 创建 Agent
func (h *Handler) createAgent(c *gin.Context) {
	var req agentsvc.CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	userCode := c.Query("user_code")
	if userCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_code is required"})
		return
	}

	agent, err := h.agentService.CreateAgent(userCode, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, agent)
}

// getAgent 获取 Agent
func (h *Handler) getAgent(c *gin.Context, id uint) {
	agent, err := h.agentService.GetAgent(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if agent == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	c.JSON(http.StatusOK, agent)
}

// updateAgent 更新 Agent
func (h *Handler) updateAgent(c *gin.Context, id uint) {
	var req agentsvc.UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	agent, err := h.agentService.UpdateAgent(id, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, agent)
}

// deleteAgent 删除 Agent
func (h *Handler) deleteAgent(c *gin.Context, id uint) {
	if err := h.agentService.DeleteAgent(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "agent deleted"})
}

// getAgentByCode 根据 Code 获取 Agent
func (h *Handler) getAgentByCode(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	agent, err := h.agentService.GetAgentByCode(code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if agent == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	c.JSON(http.StatusOK, agent)
}
