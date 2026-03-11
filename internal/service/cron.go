package service

import (
	"context"
	"fmt"
	"time"

	"github.com/weibaohui/nanobot-go/internal/models"
	"gorm.io/gorm"
)

// CreateCronJobRequest 创建定时任务请求
type CreateCronJobRequest struct {
	Name              string  `json:"name"`
	Description       string  `json:"description,omitempty"`
	ChannelID         uint    `json:"channel_id"`
	CronExpression    string  `json:"cron_expression"`
	Timezone          string  `json:"timezone,omitempty"`
	Prompt            string  `json:"prompt"`
	ModelSelectionMode string  `json:"model_selection_mode,omitempty"`
	ModelID           string  `json:"model_id,omitempty"`
	ModelName         string  `json:"model_name,omitempty"`
	TargetChannelID   *uint   `json:"target_channel_id,omitempty"`
	TargetUserID      string  `json:"target_user_id,omitempty"`
}

// UpdateCronJobRequest 更新定时任务请求
type UpdateCronJobRequest struct {
	Name              string  `json:"name,omitempty"`
	Description       string  `json:"description,omitempty"`
	ChannelID         uint    `json:"channel_id,omitempty"`
	CronExpression    string  `json:"cron_expression,omitempty"`
	Timezone          string  `json:"timezone,omitempty"`
	Prompt            string  `json:"prompt,omitempty"`
	ModelSelectionMode string  `json:"model_selection_mode,omitempty"`
	ModelID           string  `json:"model_id,omitempty"`
	ModelName         string  `json:"model_name,omitempty"`
	TargetChannelID   *uint   `json:"target_channel_id,omitempty"`
	TargetUserID      string  `json:"target_user_id,omitempty"`
	IsActive          *bool   `json:"is_active,omitempty"`
}

// CronJobService 定时任务服务接口
type CronJobService interface {
	List(ctx context.Context, userID uint, offset int, limit int) ([]models.CronJob, int64, error)
	Get(ctx context.Context, id uint) (*models.CronJob, error)
	Create(ctx context.Context, userID uint, req CreateCronJobRequest) (*models.CronJob, error)
	Update(ctx context.Context, id uint, req UpdateCronJobRequest) error
	Delete(ctx context.Context, id uint) error
	Enable(ctx context.Context, id uint) error
	Disable(ctx context.Context, id uint) error
	Execute(ctx context.Context, id uint) error
	GetPending(ctx context.Context) ([]models.CronJob, error)
}

// cronJobService 定时任务服务实现
type cronJobService struct {
	db *gorm.DB
}

// NewCronJobService 创建定时任务服务
func NewCronJobService(db *gorm.DB) CronJobService {
	return &cronJobService{db: db}
}

// List 获取定时任务列表
func (s *cronJobService) List(ctx context.Context, userID uint, offset int, limit int) ([]models.CronJob, int64, error) {
	var jobs []models.CronJob
	var total int64

	query := s.db.WithContext(ctx).Where("user_id = ?", userID)

	if err := query.Model(&models.CronJob{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Offset(offset).Limit(limit).Find(&jobs).Error; err != nil {
		return nil, 0, err
	}

	return jobs, total, nil
}

// Get 获取单个定时任务
func (s *cronJobService) Get(ctx context.Context, id uint) (*models.CronJob, error) {
	var job models.CronJob
	if err := s.db.WithContext(ctx).First(&job, id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// Create 创建定时任务
func (s *cronJobService) Create(ctx context.Context, userID uint, req CreateCronJobRequest) (*models.CronJob, error) {
	// 验证 channel 是否存在
	var channel models.Channel
	if err := s.db.WithContext(ctx).First(&channel, req.ChannelID).Error; err != nil {
		return nil, fmt.Errorf("channel not found: %w", err)
	}

	job := &models.CronJob{
		UserID:           userID,
		ChannelID:        req.ChannelID,
		Name:             req.Name,
		Description:      req.Description,
		CronExpression:   req.CronExpression,
		Timezone:         req.Timezone,
		Prompt:           req.Prompt,
		ModelSelectionMode: req.ModelSelectionMode,
		ModelID:          req.ModelID,
		ModelName:        req.ModelName,
		TargetChannelID:  req.TargetChannelID,
		TargetUserID:     req.TargetUserID,
		IsActive:         true,
		RunCount:         0,
		FailCount:        0,
	}

	if req.Timezone == "" {
		job.Timezone = "Asia/Shanghai"
	}
	if req.ModelSelectionMode == "" {
		job.ModelSelectionMode = "auto"
	}

	if err := s.db.WithContext(ctx).Create(job).Error; err != nil {
		return nil, err
	}

	return job, nil
}

// Update 更新定时任务
func (s *cronJobService) Update(ctx context.Context, id uint, req UpdateCronJobRequest) error {
	var job models.CronJob
	if err := s.db.WithContext(ctx).First(&job, id).Error; err != nil {
		return err
	}

	updates := make(map[string]interface{})

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.ChannelID != 0 {
		// 验证 channel 是否存在
		var channel models.Channel
		if err := s.db.WithContext(ctx).First(&channel, req.ChannelID).Error; err != nil {
			return fmt.Errorf("channel not found: %w", err)
		}
		updates["channel_id"] = req.ChannelID
	}
	if req.CronExpression != "" {
		updates["cron_expression"] = req.CronExpression
	}
	if req.Timezone != "" {
		updates["timezone"] = req.Timezone
	}
	if req.Prompt != "" {
		updates["prompt"] = req.Prompt
	}
	if req.ModelSelectionMode != "" {
		updates["model_selection_mode"] = req.ModelSelectionMode
	}
	if req.ModelID != "" {
		updates["model_id"] = req.ModelID
	}
	if req.ModelName != "" {
		updates["model_name"] = req.ModelName
	}
	if req.TargetChannelID != nil {
		if *req.TargetChannelID != 0 {
			// 验证 channel 是否存在
			var channel models.Channel
			if err := s.db.WithContext(ctx).First(&channel, *req.TargetChannelID).Error; err != nil {
				return fmt.Errorf("target channel not found: %w", err)
			}
		}
		updates["target_channel_id"] = req.TargetChannelID
	}
	if req.TargetUserID != "" {
		updates["target_user_id"] = req.TargetUserID
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if len(updates) == 0 {
		return nil
	}

	return s.db.WithContext(ctx).Model(&job).Updates(updates).Error
}

// Delete 删除定时任务
func (s *cronJobService) Delete(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.CronJob{}, id).Error
}

// Enable 启用定时任务
func (s *cronJobService) Enable(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Model(&models.CronJob{}).
		Where("id = ?", id).
		Update("is_active", true).Error
}

// Disable 禁用定时任务
func (s *cronJobService) Disable(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Model(&models.CronJob{}).
		Where("id = ?", id).
		Update("is_active", false).Error
}

// Execute 立即执行定时任务
func (s *cronJobService) Execute(ctx context.Context, id uint) error {
	var job models.CronJob
	if err := s.db.WithContext(ctx).First(&job, id).Error; err != nil {
		return err
	}

	// TODO: 实现实际的执行逻辑
	// 这里只是一个占位符，实际需要:
	// 1. 根据 ModelSelectionMode 选择模型
	// 2. 调用 LLM API
	// 3. 将结果发送到目标渠道
	// 4. 更新执行状态和结果

	// 更新执行状态
	now := time.Now()
	job.LastRunAt = &now
	job.LastRunStatus = "running"
	s.db.WithContext(ctx).Save(&job)

	return nil
}

// GetPending 获取待执行的定时任务
func (s *cronJobService) GetPending(ctx context.Context) ([]models.CronJob, error) {
	var jobs []models.CronJob
	now := time.Now()

	if err := s.db.WithContext(ctx).
		Where("is_active = ? AND next_run_at <= ?", true, now).
		Find(&jobs).Error; err != nil {
		return nil, err
	}

	return jobs, nil
}
