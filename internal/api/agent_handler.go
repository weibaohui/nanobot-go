package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	agentsvc "github.com/weibaohui/nanobot-go/internal/service/agent"
)

// getAgentByID 获取指定 Agent
func (h *Handler) getAgentByID(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent id"})
		return
	}

	agent, err := h.agentService.GetAgent(uint(id))
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

// updateAgentByID 更新指定 Agent
func (h *Handler) updateAgentByID(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent id"})
		return
	}

	var req agentsvc.UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	agent, err := h.agentService.UpdateAgent(uint(id), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, agent)
}

// deleteAgentByID 删除指定 Agent
func (h *Handler) deleteAgentByID(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent id"})
		return
	}

	if err := h.agentService.DeleteAgent(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "agent deleted"})
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

// ToolInfo 工具信息
type ToolInfo struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// listAvailableTools 获取可用的内置工具列表
func (h *Handler) listAvailableTools(c *gin.Context) {
	tools := []ToolInfo{
		{Value: "readfile", Label: "readfile - 读取文件", Description: "读取文件内容"},
		{Value: "writefile", Label: "writefile - 写入文件", Description: "写入文件内容"},
		{Value: "editfile", Label: "editfile - 编辑文件", Description: "编辑文件内容"},
		{Value: "listdir", Label: "listdir - 列出目录", Description: "列出目录内容"},
		{Value: "exec", Label: "exec - 执行命令", Description: "执行 Shell 命令"},
		{Value: "websearch", Label: "websearch - 网页搜索", Description: "搜索网页内容"},
		{Value: "webfetch", Label: "webfetch - 网页获取", Description: "获取网页内容"},
		{Value: "message", Label: "message - 发送消息", Description: "发送消息到渠道"},
		{Value: "cron", Label: "cron - 定时任务", Description: "管理定时任务"},
		{Value: "askuser", Label: "askuser - 询问用户", Description: "向用户提问"},
		{Value: "skill", Label: "skill - 技能调用", Description: "调用技能"},
		{Value: "task_start", Label: "task_start - 启动任务", Description: "启动后台任务"},
		{Value: "task_get", Label: "task_get - 获取任务", Description: "获取任务状态"},
		{Value: "task_stop", Label: "task_stop - 停止任务", Description: "停止后台任务"},
		{Value: "task_list", Label: "task_list - 列出任务", Description: "列出所有任务"},
	}

	c.JSON(http.StatusOK, ListResponse{
		Items: tools,
		Total: int64(len(tools)),
	})
}
