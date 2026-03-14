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

// EnableThinkingProcessKey 是 context 中存储是否启用思考过程的 key
type EnableThinkingProcessKey struct{}

// ChannelIDKey 是 context 中存储 ChannelID 的 key
type ChannelIDKey struct{}

// AgentIDKey 是 context 中存储 AgentID 的 key
type AgentIDKey struct{}

// UserCodeKey 是 context 中存储 UserCode 的 key
type UserCodeKey struct{}

// ChannelCodeKey 是 context 中存储 ChannelCode 的 key
type ChannelCodeKey struct{}

// AgentCodeKey 是 context 中存储 AgentCode 的 key
type AgentCodeKey struct{}

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

// WithEnableThinkingProcess 将是否启用思考过程注入到 context 中
func WithEnableThinkingProcess(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, EnableThinkingProcessKey{}, enabled)
}

// GetEnableThinkingProcess 从 context 中获取是否启用思考过程
func GetEnableThinkingProcess(ctx context.Context) bool {
	if enabled, ok := ctx.Value(EnableThinkingProcessKey{}).(bool); ok {
		return enabled
	}
	return false
}

// WithChannelID 将 ChannelID 注入到 context 中
func WithChannelID(ctx context.Context, channelID uint) context.Context {
	return context.WithValue(ctx, ChannelIDKey{}, channelID)
}

// GetChannelID 从 context 中获取 ChannelID，如果不存在则返回 0
func GetChannelID(ctx context.Context) uint {
	if channelID, ok := ctx.Value(ChannelIDKey{}).(uint); ok {
		return channelID
	}
	return 0
}

// WithAgentID 将 AgentID 注入到 context 中
func WithAgentID(ctx context.Context, agentID uint) context.Context {
	return context.WithValue(ctx, AgentIDKey{}, agentID)
}

// GetAgentID 从 context 中获取 AgentID，如果不存在则返回 0
func GetAgentID(ctx context.Context) uint {
	if agentID, ok := ctx.Value(AgentIDKey{}).(uint); ok {
		return agentID
	}
	return 0
}

// WithUserCode 将 UserCode 注入到 context 中
func WithUserCode(ctx context.Context, userCode string) context.Context {
	return context.WithValue(ctx, UserCodeKey{}, userCode)
}

// GetUserCode 从 context 中获取 UserCode，如果不存在则返回空字符串
func GetUserCode(ctx context.Context) string {
	if userCode, ok := ctx.Value(UserCodeKey{}).(string); ok {
		return userCode
	}
	return ""
}

// WithChannelCode 将 ChannelCode 注入到 context 中
func WithChannelCode(ctx context.Context, channelCode string) context.Context {
	return context.WithValue(ctx, ChannelCodeKey{}, channelCode)
}

// GetChannelCode 从 context 中获取 ChannelCode，如果不存在则返回空字符串
func GetChannelCode(ctx context.Context) string {
	if channelCode, ok := ctx.Value(ChannelCodeKey{}).(string); ok {
		return channelCode
	}
	return ""
}

// WithAgentCode 将 AgentCode 注入到 context 中
func WithAgentCode(ctx context.Context, agentCode string) context.Context {
	return context.WithValue(ctx, AgentCodeKey{}, agentCode)
}

// GetAgentCode 从 context 中获取 AgentCode，如果不存在则返回空字符串
func GetAgentCode(ctx context.Context) string {
	if agentCode, ok := ctx.Value(AgentCodeKey{}).(string); ok {
		return agentCode
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