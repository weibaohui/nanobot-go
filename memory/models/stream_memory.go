package models

import "time"

// StreamMemory 流水记忆（短期记忆）
// 存储对话的初步总结，按日期归类，直到被定时任务升级为长期记忆
type StreamMemory struct {
	ID          uint64     `gorm:"primaryKey" json:"id"`
	TraceID     string     `gorm:"type:text;index:idx_stream_trace_id;not null" json:"trace_id"`
	SessionKey  string     `gorm:"type:text;index:idx_stream_session_key" json:"session_key"`
	ChannelType string     `gorm:"type:text;index:idx_stream_channel" json:"channel_type"`
	Content     string     `gorm:"type:text" json:"content"`                      // 原始对话内容摘要
	Summary     string     `gorm:"type:text" json:"summary"`                      // AI生成的初步总结
	EventType   string     `gorm:"type:text;index:idx_stream_event_type" json:"event_type"`
	CreatedAt   time.Time  `gorm:"type:datetime;index:idx_stream_created_at;not null" json:"created_at"`
	Processed   bool       `gorm:"type:integer;default:0;index:idx_stream_processed" json:"processed"` // 是否已升级为长期记忆
	ProcessedAt *time.Time `gorm:"type:datetime" json:"processed_at,omitempty"`
}

// TableName 指定表名
func (StreamMemory) TableName() string {
	return "stream_memories"
}
