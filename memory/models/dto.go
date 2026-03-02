package models

import "time"

// MemoryDTO 记忆数据传输对象
// 用于 Service 层对外暴露的统一数据结构
type MemoryDTO struct {
	ID          uint64    `json:"id"`
	Type        string    `json:"type"` // "stream" | "longterm"
	TraceID     string    `json:"trace_id,omitempty"`
	SessionKey  string    `json:"session_key,omitempty"`
	ChannelType string    `json:"channel_type,omitempty"`
	Content     string    `json:"content"`
	Summary     string    `json:"summary,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// SearchFilters 搜索过滤条件
// 支持时间范围、trace_id、session_key 等多种过滤方式
type SearchFilters struct {
	StartTime       *time.Time `json:"start_time,omitempty"`
	EndTime         *time.Time `json:"end_time,omitempty"`
	TraceID         string     `json:"trace_id,omitempty"`
	SessionKey      string     `json:"session_key,omitempty"`
	ChannelType     string     `json:"channel_type,omitempty"`
	Limit           int        `json:"limit,omitempty"`             // 默认 20，最大 100
	IncludeStream   bool       `json:"include_stream,omitempty"`    // 是否包含流水记忆，默认 true
	IncludeLongTerm bool       `json:"include_long_term,omitempty"` // 是否包含长期记忆，默认 true
}

// SearchResult 搜索结果
// 包含流水记忆和长期记忆两部分，按时间排序
type SearchResult struct {
	StreamMemories   []MemoryDTO   `json:"stream_memories"`
	LongTermMemories []MemoryDTO   `json:"long_term_memories"`
	Total            int           `json:"total"`
	QueryTime        time.Duration `json:"query_time"`
}

// ConversationSummary 对话总结结果
// 由 Summarizer 对单条对话进行初步总结
type ConversationSummary struct {
	Summary   string `json:"summary"`    // 一句话总结
	KeyPoints string `json:"key_points"` // 关键要点（换行分隔）
}

// LongTermSummary 长期记忆提炼结果
// 由 Summarizer 将多条流水记忆提炼为结构化长期记忆
type LongTermSummary struct {
	WhatHappened string   `json:"what_happened"` // 发生了什么
	Conclusion   string   `json:"conclusion"`    // 结论/结果
	Value        string   `json:"value"`         // 价值与用途
	Highlights   []string `json:"highlights"`    // 高印象事件列表
}

// QueryOptions 查询选项
// 用于 Repository 层的通用查询参数
type QueryOptions struct {
	OrderBy string   // 排序字段
	Order   string   // ASC 或 DESC
	Limit   int      // 限制数量
	Offset  int      // 偏移量
	Roles   []string // 筛选角色（可选）
}

// Message 对话消息
// 用于 Summarizer 的输入
type Message struct {
	Role      string    `json:"role"`      // user/assistant/system
	Content   string    `json:"content"`   // 消息内容
	Timestamp time.Time `json:"timestamp"` // 时间戳
}

// ConversationCompletedEvent 对话完成事件
// 由对话系统发布，记忆模块订阅处理
type ConversationCompletedEvent struct {
	TraceID     string    `json:"trace_id"`
	SessionKey  string    `json:"session_key"`
	ChannelType string    `json:"channel_type"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Messages    []Message `json:"messages"`
}

// EventName 返回事件名称
func (e ConversationCompletedEvent) EventName() string {
	return "conversation.completed"
}
