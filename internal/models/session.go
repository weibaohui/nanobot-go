package models

import (
	"time"
)

// Session 会话模型
// 用于跟踪活跃的 Channel 会话
type Session struct {
	ID         uint   `gorm:"primarykey" json:"id"`
	UserID     uint   `gorm:"not null;index" json:"user_id"`
	AgentID    *uint  `gorm:"index" json:"agent_id"`
	ChannelID  uint   `gorm:"not null;index" json:"channel_id"`

	SessionKey   string `gorm:"type:text;not null;uniqueIndex" json:"session_key"`
	ExternalID   string `gorm:"type:text" json:"external_id"` // 外部系统的会话标识

	LastActiveAt *time.Time `gorm:"type:datetime" json:"last_active_at"`
	Metadata     string     `gorm:"type:text" json:"metadata"` // 会话元数据 JSON

	CreatedAt time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`

	// 关联
	User    User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Agent   *Agent   `gorm:"foreignKey:AgentID" json:"agent,omitempty"`
	Channel Channel `gorm:"foreignKey:ChannelID" json:"channel,omitempty"`
}

// TableName 指定表名
func (Session) TableName() string {
	return "sessions"
}
