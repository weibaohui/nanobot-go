package trace

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// TraceIDKey 是 context 中存储 TraceID 的 key
type TraceIDKey struct{}

// NewTraceID 生成新的 TraceID
func NewTraceID() string {
	return uuid.New().String()
}

// WithTraceID 将 TraceID 注入到 context 中
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey{}, traceID)
}

// GetTraceID 从 context 中获取 TraceID，如果不存在则生成新的
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey{}).(string); ok && traceID != "" {
		return traceID
	}
	return NewTraceID()
}

// MustGetTraceID 从 context 中获取 TraceID，如果不存在则返回空字符串
func MustGetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey{}).(string); ok {
		return traceID
	}
	return ""
}

// EventTime 事件时间戳
type EventTime struct {
	Timestamp time.Time `json:"timestamp"`
}

// NewEventTime 创建事件时间戳
func NewEventTime() *EventTime {
	return &EventTime{
		Timestamp: time.Now(),
	}
}