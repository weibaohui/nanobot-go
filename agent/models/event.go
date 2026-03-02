package models

import "time"

// Event 对应 events 表的数据模型
// 用于存储所有对话事件，包括用户输入、AI 响应、工具调用等
type Event struct {
	ID               uint      `gorm:"primarykey" json:"id"`
	TraceID          string    `gorm:"type:text;index;not null" json:"trace_id"`
	SpanID           string    `gorm:"type:text" json:"span_id,omitempty"`
	ParentSpanID     string    `gorm:"type:text" json:"parent_span_id,omitempty"`
	EventType        string    `gorm:"type:text;index;not null" json:"event_type"`
	Timestamp        time.Time `gorm:"type:datetime;index;not null" json:"timestamp"`
	SessionKey       string    `gorm:"type:text;index" json:"session_key"`
	Role             string    `gorm:"type:text;index" json:"role"`
	Content          string    `gorm:"type:text" json:"content"`
	PromptTokens     int       `gorm:"type:integer;default:0" json:"prompt_tokens"`
	CompletionTokens int       `gorm:"type:integer;default:0" json:"completion_tokens"`
	TotalTokens      int       `gorm:"type:integer;default:0" json:"total_tokens"`
	ReasoningTokens  int       `gorm:"type:integer;default:0" json:"reasoning_tokens"`
	CachedTokens     int       `gorm:"type:integer;default:0" json:"cached_tokens"`
	CreatedAt        time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`
}

// TableName 指定表名
func (Event) TableName() string {
	return "events"
}