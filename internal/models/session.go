package models

import (
	"time"
)

// Session 会话模型
// 用于跟踪活跃的 Channel 会话
type Session struct {
	ID          uint   `gorm:"primarykey" json:"id"`
	UserCode    string `gorm:"type:varchar(16);index" json:"user_code"`
	AgentCode   string `gorm:"type:varchar(16);index" json:"agent_code"`
	ChannelCode string `gorm:"type:varchar(16);index" json:"channel_code"`

	SessionKey   string `gorm:"type:text;not null;uniqueIndex" json:"session_key"`
	ExternalID   string `gorm:"type:text" json:"external_id"` // 外部系统的会话标识

	LastActiveAt *time.Time `gorm:"type:datetime" json:"last_active_at"`
	Metadata     string     `gorm:"type:text" json:"metadata"` // 会话元数据 JSON

	CreatedAt time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`
}

// TableName 指定表名
func (Session) TableName() string {
	return "sessions"
}
