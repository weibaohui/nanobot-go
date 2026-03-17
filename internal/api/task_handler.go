package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	tasksvc "github.com/weibaohui/nanobot-go/internal/service/task"
)

// TaskService 任务服务接口
type TaskService interface {
	ListTasks() ([]*tasksvc.TaskResponse, error)
	GetTask(id string) (*tasksvc.TaskDetailResponse, error)
	StopTask(id string) (*tasksvc.TaskResponse, error)
}

// listTasks 获取任务列表
// GET /api/v1/tasks
func (h *Handler) listTasks(c *gin.Context) {
	tasks, err := h.taskService.ListTasks()
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
