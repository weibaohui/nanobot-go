package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status string `json:"status"`
}

// WebSocketHandler WebSocket 处理器接口
type WebSocketHandler interface {
	Handle(c *gin.Context)
}

// Server API 服务器
type Server struct {
	handler   *Handler
	server    *http.Server
	logger    *zap.Logger
	router    *gin.Engine
	providers *Providers
	wsHandler WebSocketHandler
}

// NewServer 创建 API 服务器
func NewServer(addr string, providers *Providers, logger *zap.Logger) *Server {
	handler := NewHandler(
		providers.UserService,
		providers.AgentService,
		providers.ChannelService,
		providers.SessionService,
		providers.ProviderService,
		providers.CronJobService,
		providers.ConversationRecordService,
		providers.ConversationService,
		providers.SessionManager,
		providers.MCPService,
		providers.SkillService,
		providers.CodeLookupService,
	)

	// 创建 Gin 路由
	router := gin.Default()

	// 添加 CORS 中间件
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Content-Length", "Accept-Encoding", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * 3600,
	}))

	// 注册 API 路由
	handler.RegisterRoutes(router)

	// 添加健康检查端点
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
	})

	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		handler:   handler,
		server:    server,
		logger:    logger,
		router:    router,
		providers: providers,
	}
}

// Start 启动 API 服务器
func (s *Server) Start() error {
	s.logger.Info("API 服务器启动", zap.String("addr", s.server.Addr))
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("API 服务器错误", zap.Error(err))
		}
	}()
	return nil
}

// Stop 停止 API 服务器
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

// SetWebSocketHandler 设置 WebSocket 处理器
func (s *Server) SetWebSocketHandler(handler WebSocketHandler) {
	s.wsHandler = handler
	// 注册 WebSocket 路由
	s.router.GET("/ws/chat", handler.Handle)
}
