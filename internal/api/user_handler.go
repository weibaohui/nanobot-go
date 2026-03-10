package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/weibaohui/nanobot-go/internal/service"
)

// handleUsers 处理 /api/v1/users
func (h *Handler) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listUsers(w, r)
	case http.MethodPost:
		h.createUser(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleUserByID 处理 /api/v1/users/{id}
func (h *Handler) handleUserByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")
	parts := strings.Split(path, "/")
	idStr := parts[0]

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getUser(w, r, uint(id))
	case http.MethodPut:
		h.updateUser(w, r, uint(id))
	case http.MethodDelete:
		h.deleteUser(w, r, uint(id))
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// listUsers 获取用户列表
func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}

	users, total, err := h.userService.ListUsers(offset, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ListResponse{
		Data:  users,
		Total: total,
	})
}

// createUser 创建用户
func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var req service.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.userService.CreateUser(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

// getUser 获取用户
func (h *Handler) getUser(w http.ResponseWriter, r *http.Request, id uint) {
	user, err := h.userService.GetUser(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

// updateUser 更新用户
func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request, id uint) {
	var req service.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.userService.UpdateUser(id, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, user)
}

// deleteUser 删除用户
func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request, id uint) {
	if err := h.userService.DeleteUser(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Message: "user deleted"})
}
