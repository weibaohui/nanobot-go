package models

import "time"

// LongTermMemory 长期记忆（精华记忆）
// 存储每日精华提炼后的结构化记忆，可长期保存和检索
type LongTermMemory struct {
	ID           uint64    `gorm:"primaryKey" json:"id"`
	Date         string    `gorm:"type:text;index:idx_longterm_date;not null" json:"date"` // 日期 YYYY-MM-DD
	Summary      string    `gorm:"type:text" json:"summary"`                               // 总体摘要
	WhatHappened string    `gorm:"type:text" json:"what_happened"`                         // 发生了什么
	Conclusion   string    `gorm:"type:text" json:"conclusion"`                            // 结论/结果
	Value        string    `gorm:"type:text" json:"value"`                                 // 价值与用途
	Highlights   string    `gorm:"type:text" json:"highlights"`                            // 高印象事件（JSON数组）
	SourceIDs    string    `gorm:"type:text" json:"source_ids"`                            // 来源流水记忆ID列表（逗号分隔）
	VectorID     string    `gorm:"type:text" json:"vector_id,omitempty"`                   // 向量库ID
	CreatedAt    time.Time `gorm:"type:datetime;not null" json:"created_at"`
	UpdatedAt    time.Time `gorm:"type:datetime;not null" json:"updated_at"`
}

// TableName 指定表名
func (LongTermMemory) TableName() string {
	return "long_term_memories"
}
