package models

import (
	"time"
)

// User 用户模型
// 存储系统用户信息，支持多租户架构
type User struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	UserCode     string    `gorm:"type:varchar(16);uniqueIndex" json:"user_code"`
	Username     string    `gorm:"type:text;not null;uniqueIndex" json:"username"`
	Email        string    `gorm:"type:text;uniqueIndex" json:"email"`
	PasswordHash string    `gorm:"type:text" json:"-"` // 不序列化到 JSON
	DisplayName  string    `gorm:"type:text" json:"display_name"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt    time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}
