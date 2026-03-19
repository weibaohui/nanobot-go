package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	tasksvc "github.com/weibaohui/nanobot-go/internal/service/task"
)

// TaskService 任务服务接口
type TaskService interface {
	ListTasks() ([]*tasksvc.TaskResponse, error)
	ListTasksWithFilter(filter *tasksvc.TaskFilter) ([]*tasksvc.TaskResponse, error)
	GetTask(id string) (*tasksvc.TaskDetailResponse, error)
	CreateTask(work, createdBy string) (*tasksvc.TaskResponse, error)
	StopTask(id string) (*tasksvc.TaskResponse, error)
	RetryTask(id, createdBy string) (*tasksvc.TaskResponse, error)
}

// listTasks 获取任务列表
// GET /api/v1/tasks?status=running,failed&since=2026-03-18&keyword=xxx
func (h *Handler) listTasks(c *gin.Context) {
	// 解析筛选参数
	filter := &tasksvc.TaskFilter{}

	// 状态筛选
	if status := c.Query("status"); status != "" {
		filter.Status = strings.Split(status, ",")
	}

	// 时间范围筛选
	if since := c.Query("since"); since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			filter.Since = t
		}
	}
	if until := c.Query("until"); until != "" {
		if t, err := time.Parse(time.RFC3339, until); err == nil {
			filter.Until = t
		}
	}

	// 关键词筛选
	filter.Keyword = c.Query("keyword")

	// 获取当前用户
	if claims, ok := c.Value("claims").(*Claims); ok && claims != nil {
		filter.CreatedBy = claims.Username
		filter.IsAdmin = claims.Username == "admin"
	}

	// 如果有筛选条件，使用带筛选的查询
	var tasks []*tasksvc.TaskResponse
	var err error

	if filter.Keyword != "" || len(filter.Status) > 0 || !filter.Since.IsZero() || !filter.Until.IsZero() {
		tasks, err = h.taskService.ListTasksWithFilter(filter)
	} else {
		tasks, err = h.taskService.ListTasks()
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ListResponse{
		Items: tasks,
		Total: int64(len(tasks)),
	})
}

// getTask 获取任务详情
// GET /api/v1/tasks/:id
func (h *Handler) getTask(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "任务ID不能为空"})
		return
	}

	task, err := h.taskService.GetTask(taskID)
	if err != nil {
		if errors.Is(err, tasksvc.ErrManagerNotInitialized) {
			c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	if task == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "任务不存在"})
		return
	}

	c.JSON(http.StatusOK, task)
}

// stopTask 停止任务
// POST /api/v1/tasks/:id/stop
func (h *Handler) stopTask(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "任务ID不能为空"})
		return
	}

	task, err := h.taskService.StopTask(taskID)
	if err != nil {
		switch {
		case errors.Is(err, tasksvc.ErrManagerNotInitialized):
			c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: err.Error()})
		case errors.Is(err, tasksvc.ErrTaskNotFound):
			c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		default:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "任务已停止",
		Data:    task,
	})
}

// createTaskRequest 创建任务请求
type createTaskRequest struct {
	Work string `json:"work" binding:"required"`
}

// createTask 手动创建任务
// POST /api/v1/tasks
func (h *Handler) createTask(c *gin.Context) {
	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "请求参数错误: " + err.Error()})
		return
	}

	// 获取当前用户
	createdBy := ""
	if claims, ok := c.Value("claims").(*Claims); ok && claims != nil {
		createdBy = claims.Username
	}

	task, err := h.taskService.CreateTask(req.Work, createdBy)
	if err != nil {
		switch {
		case errors.Is(err, tasksvc.ErrManagerNotInitialized):
			c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: err.Error()})
		default:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Message: "任务已创建",
		Data:    task,
	})
}

// retryTask 重试任务
// POST /api/v1/tasks/:id/retry
func (h *Handler) retryTask(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "任务ID不能为空"})
		return
	}

	// 获取当前用户
	createdBy := ""
	if claims, ok := c.Value("claims").(*Claims); ok && claims != nil {
		createdBy = claims.Username
	}

	task, err := h.taskService.RetryTask(taskID, createdBy)
	if err != nil {
		switch {
		case errors.Is(err, tasksvc.ErrManagerNotInitialized):
			c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: err.Error()})
		case errors.Is(err, tasksvc.ErrTaskNotFound):
			c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		default:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "任务已重试",
		Data:    task,
	})
}
