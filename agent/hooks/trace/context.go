package trace

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// TraceIDKey 是 context 中存储 TraceID 的 key
type TraceIDKey struct{}

// SpanIDKey 是 context 中存储 SpanID 的 key
type SpanIDKey struct{}

// ParentSpanIDKey 是 context 中存储 ParentSpanID 的 key
type ParentSpanIDKey struct{}

// SessionKeyKey 是 context 中存储 SessionKey 的 key
type SessionKeyKey struct{}

// ChannelKey 是 context 中存储 Channel 的 key
type ChannelKey struct{}

// NewTraceID 生成新的 TraceID
func NewTraceID() string {
	return uuid.New().String()
}

// NewSpanID 生成新的 SpanID
func NewSpanID() string {
	return uuid.New().String()
}

// WithTraceID 将 TraceID 注入到 context 中
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey{}, traceID)
}

// WithSpanID 将 SpanID 注入到 context 中
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, SpanIDKey{}, spanID)
}

// WithParentSpanID 将 ParentSpanID 注入到 context 中
func WithParentSpanID(ctx context.Context, parentSpanID string) context.Context {
	return context.WithValue(ctx, ParentSpanIDKey{}, parentSpanID)
}

// WithSessionKey 将 SessionKey 注入到 context 中
func WithSessionKey(ctx context.Context, sessionKey string) context.Context {
	return context.WithValue(ctx, SessionKeyKey{}, sessionKey)
}

// WithChannel 将 Channel 注入到 context 中
func WithChannel(ctx context.Context, channel string) context.Context {
	return context.WithValue(ctx, ChannelKey{}, channel)
}

// WithSessionInfo 将会话信息（sessionKey 和 channel）注入到 context 中
func WithSessionInfo(ctx context.Context, sessionKey, channel string) context.Context {
	ctx = WithSessionKey(ctx, sessionKey)
	ctx = WithChannel(ctx, channel)
	return ctx
}

// GetTraceID 从 context 中获取 TraceID，如果不存在则生成新的
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey{}).(string); ok && traceID != "" {
		return traceID
	}
	return NewTraceID()
}

// GetSpanID 从 context 中获取 SpanID，如果不存在则生成新的
func GetSpanID(ctx context.Context) string {
	if spanID, ok := ctx.Value(SpanIDKey{}).(string); ok && spanID != "" {
		return spanID
	}
	return NewSpanID()
}

// GetParentSpanID 从 context 中获取 ParentSpanID，如果不存在则返回空字符串
func GetParentSpanID(ctx context.Context) string {
	if parentSpanID, ok := ctx.Value(ParentSpanIDKey{}).(string); ok {
		return parentSpanID
	}
	return ""
}

// GetSessionKey 从 context 中获取 SessionKey，如果不存在则返回空字符串
func GetSessionKey(ctx context.Context) string {
	if sessionKey, ok := ctx.Value(SessionKeyKey{}).(string); ok {
		return sessionKey
	}
	return ""
}

// GetChannel 从 context 中获取 Channel，如果不存在则返回空字符串
func GetChannel(ctx context.Context) string {
	if channel, ok := ctx.Value(ChannelKey{}).(string); ok {
		return channel
	}
	return ""
}

// MustGetTraceID 从 context 中获取 TraceID，如果不存在则返回空字符串
func MustGetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey{}).(string); ok {
		return traceID
	}
	return ""
}

// MustGetSpanID 从 context 中获取 SpanID，如果不存在则返回空字符串
func MustGetSpanID(ctx context.Context) string {
	if spanID, ok := ctx.Value(SpanIDKey{}).(string); ok {
		return spanID
	}
	return ""
}

// StartSpan 开始一个新的 Span，继承父 Span 的 TraceID，并设置 ParentSpanID
func StartSpan(ctx context.Context) (context.Context, string) {
	parentSpanID := MustGetSpanID(ctx) // 使用 MustGetSpanID 来检测是否真的有父 Span
	newSpanID := NewSpanID()

	ctx = WithSpanID(ctx, newSpanID)
	if parentSpanID != "" {
		ctx = WithParentSpanID(ctx, parentSpanID)
	}

	return ctx, newSpanID
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