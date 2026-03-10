package api

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Server API 服务器
type Server struct {
	handler *Handler
	server  *http.Server
	logger  *zap.Logger
}

// NewServer 创建 API 服务器
func NewServer(addr string, providers *Providers, logger *zap.Logger) *Server {
	handler := NewHandler(
		providers.UserService,
		providers.AgentService,
		providers.ChannelService,
		providers.SessionService,
	)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// 添加健康检查端点
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		handler: handler,
		server:  server,
		logger:  logger,
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
