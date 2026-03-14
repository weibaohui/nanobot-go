package models

import "time"

// ConversationRecord 对话记录模型
// 存储完整的对话消息，包括用户输入、AI 响应、工具调用及结果
type ConversationRecord struct {
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

	// 新增字段：归属信息（用于多租户、多 Agent 架构）
	UserCode    string     `gorm:"type:varchar(16);index" json:"user_code,omitempty"`
	AgentCode   string     `gorm:"type:varchar(16);index" json:"agent_code,omitempty"`
	ChannelCode string     `gorm:"type:varchar(16);index" json:"channel_code,omitempty"`
	ChannelType string     `gorm:"type:text;index" json:"channel_type,omitempty"` // 渠道类型
}

// TableName 指定表名
func (ConversationRecord) TableName() string {
	return "conversation_records"
}

// QueryOptions 查询选项
type QueryOptions struct {
	OrderBy string
	Order   string
	Limit   int
	Offset  int
	Roles   []string
}
