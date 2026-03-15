package events

import (
	"time"
)

// Event 事件接口
type Event interface {
	// ToBaseEvent 转换为基础事件
	ToBaseEvent() *BaseEvent

	// GetTraceID 获取追踪 ID
	GetTraceID() string

	// GetSpanID 获取 Span ID
	GetSpanID() string

	// GetParentSpanID 获取父 Span ID
	GetParentSpanID() string

	// GetEventType 获取事件类型
	GetEventType() EventType

	// GetTimestamp 获取时间戳
	GetTimestamp() time.Time
}

// EventType 事件类型
type EventType string

const (
	// 消息相关事件
	EventMessageReceived   EventType = "message_received"    // 收到消息
	EventMessageSent       EventType = "message_sent"        // 发送消息
	EventPromptSubmitted   EventType = "prompt_submitted"    // 提交用户 prompt
	EventSystemPromptBuilt EventType = "system_prompt_built" // 生成系统 prompt

	// 工具相关事件
	EventToolCall        EventType = "tool_call"         // 工具调用
	EventToolIntercepted EventType = "tool_intercepted"  // 工具调用被拦截
	EventToolUsed        EventType = "tool_used"         // 使用工具
	EventToolCompleted   EventType = "tool_completed"    // 工具执行完成
	EventToolError       EventType = "tool_error"        // 工具执行错误

	// 技能相关事件
	EventSkillCall   EventType = "skill_call"    // 技能调用
	EventSkillLookup EventType = "skill_lookup"  // 查找技能
	EventSkillUsed   EventType = "skill_used"    // 使用技能

	// LLM 相关事件 (来自 Eino callbacks)
	EventLLMCallStart EventType = "llm_call_start" // LLM 调用开始
	EventLLMCallEnd   EventType = "llm_call_end"   // LLM 调用结束
	EventLLMCallError EventType = "llm_call_error" // LLM 调用错误

	// 通用事件
	EventComponentStart EventType = "component_start" // 组件开始执行
	EventComponentEnd   EventType = "component_end"   // 组件执行完成
	EventComponentError EventType = "component_error" // 组件执行错误
)

// BaseEvent 事件基类
type BaseEvent struct {
	TraceID      string    `json:"trace_id"`       // 追踪 ID
	SpanID       string    `json:"span_id"`        // Span ID
	ParentSpanID string    `json:"parent_span_id"` // 父 Span ID
	EventType    EventType `json:"event_type"`     // 事件类型
	Timestamp    time.Time `json:"timestamp"`      // 时间戳
}

// ToBaseEvent 实现 Event 接口
func (e *BaseEvent) ToBaseEvent() *BaseEvent {
	return e
}

// GetTraceID 实现 Event 接口
func (e *BaseEvent) GetTraceID() string {
	return e.TraceID
}

// GetSpanID 实现 Event 接口
func (e *BaseEvent) GetSpanID() string {
	return e.SpanID
}

// GetParentSpanID 实现 Event 接口
func (e *BaseEvent) GetParentSpanID() string {
	return e.ParentSpanID
}

// GetEventType 实现 Event 接口
func (e *BaseEvent) GetEventType() EventType {
	return e.EventType
}

// GetTimestamp 实现 Event 接口
func (e *BaseEvent) GetTimestamp() time.Time {
	return e.Timestamp
}

// NewBaseEvent 创建基础事件
func NewBaseEvent(traceID, spanID, parentSpanID string, eventType EventType) *BaseEvent {
	return &BaseEvent{
		TraceID:      traceID,
		SpanID:       spanID,
		ParentSpanID: parentSpanID,
		EventType:    eventType,
		Timestamp:    time.Now(),
	}
}
