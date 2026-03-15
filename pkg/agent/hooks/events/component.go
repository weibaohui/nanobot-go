package events

import "github.com/cloudwego/eino/callbacks"

// ComponentStartEvent 组件开始执行事件 (来自 Eino callbacks)
type ComponentStartEvent struct {
	*BaseEvent
	Component string `json:"component"` // 组件类型
	Type      string `json:"type"`      // 组件类型
	Name      string `json:"name"`      // 组件名称
}

// NewComponentStartEvent 创建组件开始执行事件
func NewComponentStartEvent(traceID, spanID, parentSpanID string, info *callbacks.RunInfo) *ComponentStartEvent {
	return &ComponentStartEvent{
		BaseEvent: NewBaseEvent(traceID, spanID, parentSpanID, EventComponentStart),
		Component: string(info.Component),
		Type:      info.Type,
		Name:      info.Name,
	}
}

// ComponentEndEvent 组件执行完成事件 (来自 Eino callbacks)
type ComponentEndEvent struct {
	*BaseEvent
	Component  string `json:"component"`   // 组件类型
	Type       string `json:"type"`        // 组件类型
	Name       string `json:"name"`        // 组件名称
	DurationMs int64  `json:"duration_ms"` // 持续时间 (毫秒)
}

// NewComponentEndEvent 创建组件执行完成事件
func NewComponentEndEvent(traceID, spanID, parentSpanID string, info *callbacks.RunInfo, durationMs int64) *ComponentEndEvent {
	return &ComponentEndEvent{
		BaseEvent:  NewBaseEvent(traceID, spanID, parentSpanID, EventComponentEnd),
		Component:  string(info.Component),
		Type:       info.Type,
		Name:       info.Name,
		DurationMs: durationMs,
	}
}

// ComponentErrorEvent 组件执行错误事件 (来自 Eino callbacks)
type ComponentErrorEvent struct {
	*BaseEvent
	Component  string `json:"component"`   // 组件类型
	Type       string `json:"type"`        // 组件类型
	Name       string `json:"name"`        // 组件名称
	Error      string `json:"error"`       // 错误信息
	DurationMs int64  `json:"duration_ms"` // 持续时间 (毫秒)
}

// NewComponentErrorEvent 创建组件执行错误事件
func NewComponentErrorEvent(traceID, spanID, parentSpanID string, info *callbacks.RunInfo, err error, durationMs int64) *ComponentErrorEvent {
	return &ComponentErrorEvent{
		BaseEvent:  NewBaseEvent(traceID, spanID, parentSpanID, EventComponentError),
		Component:  string(info.Component),
		Type:       info.Type,
		Name:       info.Name,
		Error:      err.Error(),
		DurationMs: durationMs,
	}
}
