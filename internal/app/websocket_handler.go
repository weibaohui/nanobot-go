package app

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/nanobot-go/internal/api"
	"github.com/weibaohui/nanobot-go/internal/models"
	"github.com/weibaohui/nanobot-go/pkg/channels/websocket"
	"go.uber.org/zap"
)

// WebSocketHandler WebSocket 连接处理器
type WebSocketHandler struct {
	gateway *Gateway
	logger  *zap.Logger
}

// NewWebSocketHandler 创建 WebSocket 处理器
func NewWebSocketHandler(gateway *Gateway, logger *zap.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		gateway: gateway,
		logger:  logger.With(zap.String("component", "websocket_handler")),
	}
}

// Handle 处理 WebSocket 连接请求
func (h *WebSocketHandler) Handle(c *gin.Context) {
	channelCode := c.Query("channel_code")
	if channelCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 channel_code 参数"})
		return
	}

	// 从 URL 参数获取 token 并解析用户身份
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少 token 参数"})
		return
	}

	claims, err := api.ParseToken(token)
	if err != nil {
		h.logger.Warn("解析 token 失败", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的 token"})
		return
	}

	// 使用 username 作为 user_code
	userCode := claims.Username
	if userCode == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无法获取用户信息"})
		return
	}

	h.logger.Debug("收到 WebSocket 连接请求",
		zap.String("channel_code", channelCode),
		zap.String("user_code", userCode),
	)

	// 从数据库验证渠道
	channel, err := h.gateway.Providers.ChannelService.GetChannelByCode(channelCode)
	if err != nil {
		h.logger.Error("获取渠道信息失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取渠道信息失败"})
		return
	}
	if channel == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "渠道不存在"})
		return
	}

	// 验证渠道类型
	if channel.Type != models.ChannelTypeWebSocket {
		c.JSON(http.StatusBadRequest, gin.H{"error": "渠道类型不是 WebSocket"})
		return
	}

	// 验证渠道是否启用
	if !channel.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "渠道已禁用"})
		return
	}

	// 检查白名单
	if err := h.checkAllowList(channel, userCode); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// 获取 WebSocket Channel 实例
	wsChannel := h.getWebSocketChannel(channelCode)
	if wsChannel == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "WebSocket 渠道未初始化"})
		return
	}

	// 处理 WebSocket 升级
	wsChannel.HandleWebSocketWithGin(c, userCode)
}

// checkAllowList 检查用户是否在白名单中
func (h *WebSocketHandler) checkAllowList(channel *models.Channel, userCode string) error {
	if channel.AllowFrom == "" || channel.AllowFrom == "null" {
		return nil
	}

	allowList, err := h.gateway.Providers.ChannelService.GetAllowList(channel.ID)
	if err != nil {
		h.logger.Error("获取白名单失败", zap.Error(err))
		return nil // 白名单获取失败时允许访问（降级处理）
	}

	if len(allowList) == 0 {
		return nil
	}

	for _, u := range allowList {
		if u == userCode {
			return nil
		}
	}

	return http.ErrBodyNotAllowed
}

// getWebSocketChannel 从 ChannelManager 获取 WebSocket Channel
func (h *WebSocketHandler) getWebSocketChannel(channelCode string) *websocket.Channel {
	channelName := "websocket_" + channelCode
	ch := h.gateway.ChannelManager.Get(channelName)
	if ch == nil {
		return nil
	}

	// 类型断言
	if wsCh, ok := ch.(*websocket.Channel); ok {
		return wsCh
	}

	return nil
}

