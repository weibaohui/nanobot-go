package interrupt

import (
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(&InterruptInfo{})
	gob.Register(&AskUserInterrupt{})
	gob.Register(&PlanApprovalInterrupt{})
	gob.Register(&ToolConfirmInterrupt{})
	gob.Register(&FileOperationInterrupt{})
	gob.Register(&CustomInterrupt{})
}

// InterruptType 中断类型
type InterruptType string

const (
	InterruptTypeAskUser       InterruptType = "ask_user"
	InterruptTypePlanApproval  InterruptType = "plan_approval"
	InterruptTypeToolConfirm   InterruptType = "tool_confirm"
	InterruptTypeFileOperation InterruptType = "file_operation"
	InterruptTypeCustom        InterruptType = "custom"
)

// InterruptStatus 中断状态
type InterruptStatus string

const (
	InterruptStatusPending   InterruptStatus = "pending"
	InterruptStatusResolved  InterruptStatus = "resolved"
	InterruptStatusCancelled InterruptStatus = "cancelled"
	InterruptStatusExpired   InterruptStatus = "expired"
)

// InterruptInfo 中断信息（基础结构）
type InterruptInfo struct {
	CheckpointID         string          `json:"checkpoint_id"`
	OriginalCheckpointID string          `json:"original_checkpoint_id"`
	InterruptID          string          `json:"interrupt_id"`
	Channel              string          `json:"channel"`
	ChatID               string          `json:"chat_id"`
	Question             string          `json:"question"`
	Options              []string        `json:"options,omitempty"`
	SessionKey           string          `json:"session_key"`
	IsAskUser            bool            `json:"is_ask_user"`
	IsPlan               bool            `json:"is_plan"`
	IsSupervisor         bool            `json:"is_supervisor"`
	IsMaster             bool            `json:"is_master"`
	Type                 InterruptType   `json:"type"`
	Status               InterruptStatus `json:"status"`
	CreatedAt            time.Time       `json:"created_at"`
	ExpiresAt            *time.Time      `json:"expires_at,omitempty"`
	Priority             int             `json:"priority"`
	Metadata             map[string]any  `json:"metadata,omitempty"`
}

// AskUserInterrupt 用户提问中断
type AskUserInterrupt struct {
	InterruptInfo
	Question     string   `json:"question"`
	Options      []string `json:"options,omitempty"`
	DefaultValue string   `json:"default_value,omitempty"`
	Validation   string   `json:"validation,omitempty"`
}

// PlanApprovalInterrupt 计划审批中断
type PlanApprovalInterrupt struct {
	InterruptInfo
	PlanID      string   `json:"plan_id"`
	PlanContent string   `json:"plan_content"`
	Steps       []string `json:"steps"`
	Requires    []string `json:"requires,omitempty"`
}

// ToolConfirmInterrupt 工具确认中断
type ToolConfirmInterrupt struct {
	InterruptInfo
	ToolName    string         `json:"tool_name"`
	ToolArgs    map[string]any `json:"tool_args"`
	RiskLevel   string         `json:"risk_level"`
	Description string         `json:"description"`
}

// FileOperationInterrupt 文件操作中断
type FileOperationInterrupt struct {
	InterruptInfo
	Operation string `json:"operation"`
	FilePath  string `json:"file_path"`
	Content   string `json:"content,omitempty"`
	Backup    bool   `json:"backup"`
}

// CustomInterrupt 自定义中断
type CustomInterrupt struct {
	InterruptInfo
	CustomType string         `json:"custom_type"`
	Data       map[string]any `json:"data"`
}

// UserResponse 用户响应
type UserResponse struct {
	CheckpointID string         `json:"checkpoint_id"`
	Answer       string         `json:"answer"`
	Approved     bool           `json:"approved,omitempty"`
	ModifiedData map[string]any `json:"modified_data,omitempty"`
	Timestamp    time.Time      `json:"timestamp"`
}
