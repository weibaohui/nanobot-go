package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/nanobot-go/internal/service"
)

// handleChannels 处理 GET/POST /api/v1/channels
func (h *Handler) handleChannels(c *gin.Context) {
	switch c.Request.Method {
	case http.MethodGet:
		userIDStr := c.Query("user_id")
		if userIDStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
			return
		}
		userID, err := strconv.ParseUint(userIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
			return
		}
		channels, err := h.channelService.GetUserChannels(uint(userID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, ListResponse{
			Items: channels,
			Total: int64(len(channels)),
		})

	case http.MethodPost:
		var req service.CreateChannelRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		userIDStr := c.Query("user_id")
		if userIDStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
			return
		}
		userID, err := strconv.ParseUint(userIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
			return
		}
		channel, err := h.channelService.CreateChannel(uint(userID), req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, channel)

	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "method not allowed"})
	}
}

// handleChannelByID 处理 GET/PUT/DELETE /api/v1/channels/:id
func (h *Handler) handleChannelByID(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	switch c.Request.Method {
	case http.MethodGet:
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

	case http.MethodPut:
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

	case http.MethodDelete:
		if err := h.channelService.DeleteChannel(uint(id)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, SuccessResponse{Message: "channel deleted"})

	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "method not allowed"})
	}
}

// createChannel 处理 POST /api/v1/channels
func (h *Handler) createChannel(c *gin.Context) {
	var req service.CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	channel, err := h.channelService.CreateChannel(uint(userID), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, channel)
}

// updateChannel 处理 PUT /api/v1/channels/:id
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

// deleteChannel 处理 DELETE /api/v1/channels/:id
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
