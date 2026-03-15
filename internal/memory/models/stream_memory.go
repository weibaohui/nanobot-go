package models

import "time"

// StreamMemory 流水记忆（短期记忆）
// 按用户+Agent+日期聚合，每天每条Agent一条记录，包含当天所有对话的摘要
type StreamMemory struct {
	ID        uint64     `gorm:"primaryKey" json:"id"`
	UserCode  string     `gorm:"type:text;index:idx_stream_user_agent_date,unique;not null" json:"user_code"` // 用户编码
	AgentCode string     `gorm:"type:text;index:idx_stream_user_agent_date,unique" json:"agent_code"` // Agent编码，区分不同Agent，默认为空
	Date      string     `gorm:"type:text;index:idx_stream_user_agent_date,unique;not null" json:"date"`      // 日期 YYYY-MM-DD，联合唯一索引
	Content   string     `gorm:"type:text" json:"content"`                                              // 当天所有对话的内容聚合
	Summary   string     `gorm:"type:text" json:"summary"`                                              // AI生成的当天总结
	SourceIDs string     `gorm:"type:text" json:"source_ids"`                                           // 来源对话ID列表，逗号分隔
	CreatedAt time.Time  `gorm:"type:datetime;not null" json:"created_at"`
	UpdatedAt time.Time  `gorm:"type:datetime" json:"updated_at"`
	Processed bool       `gorm:"type:integer;default:0;index:idx_stream_processed" json:"processed"` // 是否已升级为长期记忆
	ProcessedAt *time.Time `gorm:"type:datetime" json:"processed_at,omitempty"`
}

// TableName 指定表名
func (StreamMemory) TableName() string {
	return "stream_memories"
}
