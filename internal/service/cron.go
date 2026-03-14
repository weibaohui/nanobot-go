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
	Name               string `json:"name"`
	Description        string `json:"description,omitempty"`
	ChannelCode        string `json:"channel_code"`
	CronExpression     string `json:"cron_expression"`
	Timezone           string `json:"timezone,omitempty"`
	Prompt             string `json:"prompt"`
	ModelSelectionMode string `json:"model_selection_mode,omitempty"`
	ModelID            string `json:"model_id,omitempty"`
	ModelName          string `json:"model_name,omitempty"`
	TargetChannelCode  string `json:"target_channel_code,omitempty"`
	TargetUserCode     string `json:"target_user_code,omitempty"`
}

// UpdateCronJobRequest 更新定时任务请求
type UpdateCronJobRequest struct {
	Name               string `json:"name,omitempty"`
	Description        string `json:"description,omitempty"`
	ChannelCode        string `json:"channel_code,omitempty"`
	CronExpression     string `json:"cron_expression,omitempty"`
	Timezone           string `json:"timezone,omitempty"`
	Prompt             string `json:"prompt,omitempty"`
	ModelSelectionMode string `json:"model_selection_mode,omitempty"`
	ModelID            string `json:"model_id,omitempty"`
	ModelName          string `json:"model_name,omitempty"`
	TargetChannelCode  string `json:"target_channel_code,omitempty"`
	TargetUserCode     string `json:"target_user_code,omitempty"`
	IsActive           *bool  `json:"is_active,omitempty"`
}

// CronJobService 定时任务服务接口
type CronJobService interface {
	List(ctx context.Context, userCode string, offset int, limit int) ([]models.CronJob, int64, error)
	Get(ctx context.Context, id uint) (*models.CronJob, error)
	Create(ctx context.Context, userCode string, req CreateCronJobRequest) (*models.CronJob, error)
	Update(ctx context.Context, id uint, req UpdateCronJobRequest) error
	Delete(ctx context.Context, id uint) error
	Enable(ctx context.Context, id uint) error
	Disable(ctx context.Context, id uint) error
	Execute(ctx context.Context, id uint) error
	GetPending(ctx context.Context) ([]models.CronJob, error)
}

// CodeLookupService 用于查询实体 Code 的接口
type CodeLookupService interface {
	GetUserByCode(code string) (*models.User, error)
	GetChannelByCode(code string) (*models.Channel, error)
}

// cronJobService 定时任务服务实现
type cronJobService struct {
	db        *gorm.DB
	lookupSvc CodeLookupService
}

// NewCronJobService 创建定时任务服务
func NewCronJobService(db *gorm.DB, lookupSvc CodeLookupService) CronJobService {
	return &cronJobService{db: db, lookupSvc: lookupSvc}
}

// List 获取定时任务列表
func (s *cronJobService) List(ctx context.Context, userCode string, offset int, limit int) ([]models.CronJob, int64, error) {
	var jobs []models.CronJob
	var total int64

	query := s.db.WithContext(ctx).Where("user_code = ?", userCode)

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
func (s *cronJobService) Create(ctx context.Context, userCode string, req CreateCronJobRequest) (*models.CronJob, error) {
	job := &models.CronJob{
		UserCode:           userCode,
		ChannelCode:        req.ChannelCode,
		Name:               req.Name,
		Description:        req.Description,
		CronExpression:     req.CronExpression,
		Timezone:           req.Timezone,
		Prompt:             req.Prompt,
		ModelSelectionMode: req.ModelSelectionMode,
		ModelID:            req.ModelID,
		ModelName:          req.ModelName,
		TargetChannelCode:  req.TargetChannelCode,
		TargetUserCode:     req.TargetUserCode,
		IsActive:           true,
		RunCount:           0,
		FailCount:          0,
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
	if req.ChannelCode != "" {
		updates["channel_code"] = req.ChannelCode
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
	if req.TargetChannelCode != "" {
		updates["target_channel_code"] = req.TargetChannelCode
	}
	if req.TargetUserCode != "" {
		updates["target_user_code"] = req.TargetUserCode
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

	// 更新任务执行状态
	now := time.Now()
	job.LastRunAt = &now
	job.LastRunStatus = "running"
	job.RunCount++
	if err := s.db.WithContext(ctx).Save(&job).Error; err != nil {
		return err
	}

	// 异步执行实际任务
	go s.executeJob(context.Background(), &job)

	return nil
}

// executeJob 实际执行任务（异步）
func (s *cronJobService) executeJob(ctx context.Context, job *models.CronJob) {
	// 获取 Channel
	channel, err := s.lookupSvc.GetChannelByCode(job.ChannelCode)
	if err != nil || channel == nil {
		s.updateJobFailure(ctx, job, fmt.Sprintf("获取渠道失败: %v", err))
		return
	}

	// TODO: 实现实际的 LLM 调用
	// 1. 根据 ModelSelectionMode 选择模型
	// 2. 调用 LLM API 处理 Prompt
	// 3. 将结果发送到 TargetChannelCode 或原 Channel

	// 标记任务完成
	job.LastRunStatus = "success"
	nextRun := s.calculateNextRun(job.CronExpression, job.Timezone)
	job.NextRunAt = &nextRun
	s.db.WithContext(ctx).Save(job)
}

// updateJobFailure 更新任务失败状态
func (s *cronJobService) updateJobFailure(ctx context.Context, job *models.CronJob, errMsg string) {
	job.LastRunStatus = "failed"
	job.FailCount++
	s.db.WithContext(ctx).Save(job)
}

// calculateNextRun 计算下次执行时间
func (s *cronJobService) calculateNextRun(cronExpr, timezone string) time.Time {
	// 简化实现，实际应该使用 cron 解析库
	return time.Now().Add(1 * time.Hour)
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
