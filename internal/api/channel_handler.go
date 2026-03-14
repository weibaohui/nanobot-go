package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/nanobot-go/internal/service"
)

// listChannels 获取 Channel 列表
func (h *Handler) listChannels(c *gin.Context) {
	userCode := c.Query("user_code")
	if userCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_code is required"})
		return
	}

	channels, err := h.channelService.GetUserChannels(userCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ListResponse{
		Items: channels,
		Total: int64(len(channels)),
	})
}

// createChannel 创建 Channel
func (h *Handler) createChannel(c *gin.Context) {
	var req service.CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	userCode := c.Query("user_code")
	if userCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_code is required"})
		return
	}

	channel, err := h.channelService.CreateChannel(userCode, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, channel)
}

// getChannelByID 获取指定 Channel
func (h *Handler) getChannelByID(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	channel, err := h.channelService.GetChannel(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if channel == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}
	c.JSON(http.StatusOK, channel)
}

// updateChannel 更新指定 Channel
func (h *Handler) updateChannel(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	var req service.UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	channel, err := h.channelService.UpdateChannel(uint(id), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, channel)
}

// deleteChannel 删除指定 Channel
func (h *Handler) deleteChannel(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	if err := h.channelService.DeleteChannel(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "channel deleted"})
}

// getChannelByCode 根据 Code 获取 Channel
func (h *Handler) getChannelByCode(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	channel, err := h.channelService.GetChannelByCode(code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if channel == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	c.JSON(http.StatusOK, channel)
}
