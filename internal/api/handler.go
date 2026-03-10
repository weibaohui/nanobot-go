package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/weibaohui/nanobot-go/internal/service"
)

// Handler API 处理器
type Handler struct {
	userService    service.UserService
	agentService   service.AgentService
	channelService service.ChannelService
	sessionService service.SessionService
}

// NewHandler 创建 API 处理器
func NewHandler(
	userService service.UserService,
	agentService service.AgentService,
	channelService service.ChannelService,
	sessionService service.SessionService,
) *Handler {
	return &Handler{
		userService:    userService,
		agentService:   agentService,
		channelService: channelService,
		sessionService: sessionService,
	}
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// User API
	mux.HandleFunc("/api/v1/users", h.handleUsers)
	mux.HandleFunc("/api/v1/users/", h.handleUserByID)

	// Agent API
	mux.HandleFunc("/api/v1/agents", h.handleAgents)
	mux.HandleFunc("/api/v1/agents/", h.handleAgentByID)

	// Channel API
	mux.HandleFunc("/api/v1/channels", h.handleChannels)
	mux.HandleFunc("/api/v1/channels/", h.handleChannelByID)

	// Session API
	mux.HandleFunc("/api/v1/sessions", h.handleSessions)
	mux.HandleFunc("/api/v1/sessions/", h.handleSessionByKey)
}

// === Helper Functions ===

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func parseID(path string, prefix string) (uint, bool) {
	idStr := strings.TrimPrefix(path, prefix)
	idStr = strings.Split(idStr, "/")[0]
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, false
	}
	return uint(id), true
}

// === Response Types ===

// ListResponse 列表响应
type ListResponse struct {
	Data  interface{} `json:"data"`
	Total int64       `json:"total,omitempty"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error string `json:"error"`
}

// SuccessResponse 成功响应
type SuccessResponse struct {
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}
