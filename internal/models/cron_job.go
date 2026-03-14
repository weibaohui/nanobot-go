package models

import "time"

// CronJob 定时任务模型
type CronJob struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	UserCode    string    `gorm:"type:varchar(16);index" json:"user_code"`
	ChannelCode string    `gorm:"type:varchar(16);index" json:"channel_code"`

	// 任务标识
	Name        string    `gorm:"type:text;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description,omitempty"`

	// Cron 配置
	CronExpression string `gorm:"type:text;not null" json:"cron_expression"`
	Timezone       string `gorm:"type:text;default:'Asia/Shanghai'" json:"timezone"`

	// 执行配置
	Prompt            string `gorm:"type:text;not null" json:"prompt"`
	ModelSelectionMode string `gorm:"type:text;default:'auto'" json:"model_selection_mode"` // 'auto' or 'specific'
	ModelID           string `gorm:"type:text" json:"model_id,omitempty"`
	ModelName         string `gorm:"type:text" json:"model_name,omitempty"`

	// 目标配置
	TargetChannelCode string `gorm:"type:varchar(16)" json:"target_channel_code,omitempty"`
	TargetUserCode    string `gorm:"type:varchar(16)" json:"target_user_code,omitempty"`

	// 状态
	IsActive     bool       `gorm:"default:true" json:"is_active"`
	LastRunAt    *time.Time `gorm:"type:datetime" json:"last_run_at,omitempty"`
	LastRunStatus string     `gorm:"type:text" json:"last_run_status,omitempty"` // 'success', 'failed', 'running'
	LastRunResult string    `gorm:"type:text" json:"last_run_result,omitempty"`
	NextRunAt    *time.Time `gorm:"type:datetime;index" json:"next_run_at,omitempty"`
	RunCount     int        `gorm:"default:0" json:"run_count"`
	FailCount    int        `gorm:"default:0" json:"fail_count"`

	CreatedAt time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName 指定表名
func (CronJob) TableName() string {
	return "cron_jobs"
}
