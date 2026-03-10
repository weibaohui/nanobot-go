package models

import (
	"time"
)

// ChannelType 渠道类型
type ChannelType string

const (
	ChannelTypeFeishu    ChannelType = "feishu"
	ChannelTypeDingTalk  ChannelType = "dingtalk"
	ChannelTypeMatrix    ChannelType = "matrix"
	ChannelTypeWebSocket ChannelType = "websocket"
)

// Channel 渠道模型
// 存储渠道配置信息和 Agent 绑定关系
type Channel struct {
	ID       uint        `gorm:"primarykey" json:"id"`
	UserID   uint        `gorm:"not null;index" json:"user_id"`
	AgentID  *uint       `gorm:"index" json:"agent_id"` // 可为空

	Name string      `gorm:"type:text;not null" json:"name"`
	Type ChannelType `gorm:"type:text;not null" json:"type"`

	IsActive  bool   `gorm:"default:true" json:"is_active"`
	AllowFrom string `gorm:"type:text" json:"allow_from"` // JSON 数组，允许的用户白名单

	Config string `gorm:"type:text" json:"config"` // 渠道特定配置 JSON

	CreatedAt time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"updated_at"`

	// 关联
	User    User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Agent   *Agent    `gorm:"foreignKey:AgentID" json:"agent,omitempty"`
	Sessions []Session `gorm:"foreignKey:ChannelID" json:"sessions,omitempty"`
}

// TableName 指定表名
func (Channel) TableName() string {
	return "channels"
}

// FeishuChannelConfig 飞书渠道配置
type FeishuChannelConfig struct {
	AppID             string `json:"app_id"`
	AppSecret         string `json:"app_secret"`
	EncryptKey        string `json:"encrypt_key,omitempty"`
	VerificationToken string `json:"verification_token,omitempty"`
}

// DingTalkChannelConfig 钉钉渠道配置
type DingTalkChannelConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// MatrixChannelConfig Matrix 渠道配置
type MatrixChannelConfig struct {
	Homeserver string `json:"homeserver"`
	UserID     string `json:"user_id"`
	Token      string `json:"token"`
}

// WebSocketChannelConfig WebSocket 渠道配置
type WebSocketChannelConfig struct {
	Addr string `json:"addr"`
	Path string `json:"path"`
}
