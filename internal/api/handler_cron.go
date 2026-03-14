package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/nanobot-go/internal/models"
	"github.com/weibaohui/nanobot-go/internal/service"
)

// CronJobService 定时任务服务接口
type CronJobService interface {
	List(ctx context.Context, userCode string, offset int, limit int) ([]models.CronJob, int64, error)
	Get(ctx context.Context, id uint) (*models.CronJob, error)
	Create(ctx context.Context, userCode string, req service.CreateCronJobRequest) (*models.CronJob, error)
	Update(ctx context.Context, id uint, req service.UpdateCronJobRequest) error
	Delete(ctx context.Context, id uint) error
	Enable(ctx context.Context, id uint) error
	Disable(ctx context.Context, id uint) error
	Execute(ctx context.Context, id uint) error
	GetPending(ctx context.Context) ([]models.CronJob, error)
}

// === Cron Job Handlers ===

func (h *Handler) handleCronJobs(c *gin.Context) {
	userCode := c.Query("user_code")
	offset, _ := strconv.Atoi(c.Query("offset"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit == 0 {
		limit = 20
	}

	jobs, total, err := h.cronJobService.List(c.Request.Context(), userCode, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ListResponse{
		Items:    jobs,
		Total:    total,
		Page:     offset/limit + 1,
		PageSize: limit,
	})
}

func (h *Handler) handlePendingCronJobs(c *gin.Context) {
	jobs, err := h.cronJobService.GetPending(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, jobs)
}

func (h *Handler) handleCronJobByID(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	job, err := h.cronJobService.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, job)
}

func (h *Handler) createCronJob(c *gin.Context) {
	userCode := c.Query("user_code")
	var req service.CreateCronJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	job, err := h.cronJobService.Create(c.Request.Context(), userCode, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, job)
}

func (h *Handler) updateCronJob(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	var req service.UpdateCronJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.cronJobService.Update(c.Request.Context(), id, req); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "更新成功"})
}

func (h *Handler) deleteCronJob(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	if err := h.cronJobService.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "删除成功"})
}

func (h *Handler) enableCronJob(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	if err := h.cronJobService.Enable(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "已启用"})
}

func (h *Handler) disableCronJob(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	if err := h.cronJobService.Disable(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "已禁用"})
}

func (h *Handler) executeCronJob(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	if err := h.cronJobService.Execute(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "已开始执行"})
}
