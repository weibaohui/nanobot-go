package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	mcpsvc "github.com/weibaohui/nanobot-go/internal/service/mcp"
)

// listMCPServers 获取 MCP 服务器列表
func (h *Handler) listMCPServers(c *gin.Context) {
	servers, err := h.mcpService.ListServers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, ListResponse{
		Items: servers,
		Total: int64(len(servers)),
	})
}

// createMCPServer 创建 MCP 服务器
func (h *Handler) createMCPServer(c *gin.Context) {
	var req mcpsvc.CreateMCPServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body: " + err.Error()})
		return
	}

	server, err := h.mcpService.CreateServer(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, server)
}

// getMCPServer 获取指定 MCP 服务器
func (h *Handler) getMCPServer(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid mcp server id"})
		return
	}

	server, err := h.mcpService.GetServer(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	if server == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "mcp server not found"})
		return
	}

	c.JSON(http.StatusOK, server)
}

// updateMCPServer 更新 MCP 服务器
func (h *Handler) updateMCPServer(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid mcp server id"})
		return
	}

	var req mcpsvc.UpdateMCPServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body: " + err.Error()})
		return
	}

	server, err := h.mcpService.UpdateServer(uint(id), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, server)
}

// deleteMCPServer 删除 MCP 服务器
func (h *Handler) deleteMCPServer(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid mcp server id"})
		return
	}

	if err := h.mcpService.DeleteServer(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "mcp server deleted"})
}

// testMCPServer 测试 MCP 服务器连接
func (h *Handler) testMCPServer(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid mcp server id"})
		return
	}

	if err := h.mcpService.TestServer(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "mcp server connection test passed"})
}

// refreshMCPServerCapabilities 刷新 MCP 服务器能力
func (h *Handler) refreshMCPServerCapabilities(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid mcp server id"})
		return
	}

	if err := h.mcpService.RefreshCapabilities(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "mcp server capabilities refreshed"})
}

// listAgentMCPBindings 获取 Agent 的 MCP 绑定列表
func (h *Handler) listAgentMCPBindings(c *gin.Context) {
	agentID, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid agent id"})
		return
	}

	bindings, err := h.mcpService.GetAgentBindings(uint(agentID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ListResponse{
		Items: bindings,
		Total: int64(len(bindings)),
	})
}

// createAgentMCPBinding 创建 Agent MCP 绑定
func (h *Handler) createAgentMCPBinding(c *gin.Context) {
	agentID, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid agent id"})
		return
	}

	var req mcpsvc.CreateAgentMCPBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body: " + err.Error()})
		return
	}

	binding, err := h.mcpService.CreateAgentBinding(uint(agentID), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, binding)
}

// getAgentMCPBinding 获取 Agent MCP 绑定
func (h *Handler) getAgentMCPBinding(c *gin.Context) {
	bindingID, ok := parseID(c, "binding_id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid binding id"})
		return
	}

	binding, err := h.mcpService.GetAgentBindingByID(uint(bindingID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	if binding == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "binding not found"})
		return
	}

	c.JSON(http.StatusOK, binding)
}

// updateAgentMCPBinding 更新 Agent MCP 绑定
func (h *Handler) updateAgentMCPBinding(c *gin.Context) {
	bindingID, ok := parseID(c, "binding_id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid binding id"})
		return
	}

	var req mcpsvc.UpdateAgentMCPBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body: " + err.Error()})
		return
	}

	binding, err := h.mcpService.UpdateAgentBinding(uint(bindingID), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, binding)
}

// deleteAgentMCPBinding 删除 Agent MCP 绑定
func (h *Handler) deleteAgentMCPBinding(c *gin.Context) {
	bindingID, ok := parseID(c, "binding_id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid binding id"})
		return
	}

	if err := h.mcpService.DeleteAgentBinding(uint(bindingID)); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "binding deleted"})
}

// listMCPTools 获取指定 MCP 服务器的工具列表
func (h *Handler) listMCPTools(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid mcp server id"})
		return
	}

	tools, err := h.mcpService.ListTools(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ListResponse{
		Items: tools,
		Total: int64(len(tools)),
	})
}

// getAgentMCPTools 获取 Agent 可用的 MCP 工具
func (h *Handler) getAgentMCPTools(c *gin.Context) {
	agentCode := c.Param("id")
	// 如果是数字 ID，尝试通过 agent service 获取 code
	if id, err := strconv.ParseUint(agentCode, 10, 32); err == nil {
		agent, err := h.agentService.GetAgent(uint(id))
		if err != nil || agent == nil {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "agent not found"})
			return
		}
		agentCode = agent.AgentCode
	}

	tools, err := h.mcpService.GetAgentMCPTools(agentCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ListResponse{
		Items: tools,
		Total: int64(len(tools)),
	})
}
