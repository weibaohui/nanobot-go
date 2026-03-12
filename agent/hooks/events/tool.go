package events

// ToolUsedEvent 使用工具事件
type ToolUsedEvent struct {
	*BaseEvent
	ToolName      string `json:"tool_name"`      // 工具名称
	ToolArguments string `json:"tool_arguments"` // 工具参数 (JSON)
	ArgumentsRaw  string `json:"arguments_raw"`  // 原始参数
}

// NewToolUsedEvent 创建使用工具事件
func NewToolUsedEvent(traceID, spanID, parentSpanID string, toolName, toolArguments string) *ToolUsedEvent {
	return &ToolUsedEvent{
		BaseEvent:     NewBaseEvent(traceID, spanID, parentSpanID, EventToolUsed),
		ToolName:      toolName,
		ToolArguments: toolArguments,
		ArgumentsRaw:  toolArguments,
	}
}

// ToolCompletedEvent 工具执行完成事件
type ToolCompletedEvent struct {
	*BaseEvent
	ToolName       string `json:"tool_name"`       // 工具名称
	Response       string `json:"response"`        // 响应内容
	ResponseLength int    `json:"response_length"` // 响应长度
	Success        bool   `json:"success"`         // 是否成功
}

// NewToolCompletedEvent 创建工具执行完成事件
func NewToolCompletedEvent(traceID, spanID, parentSpanID string, toolName, response string, success bool) *ToolCompletedEvent {
	return &ToolCompletedEvent{
		BaseEvent:      NewBaseEvent(traceID, spanID, parentSpanID, EventToolCompleted),
		ToolName:       toolName,
		Response:       response,
		ResponseLength: len(response),
		Success:        success,
	}
}

// ToolErrorEvent 工具执行错误事件
type ToolErrorEvent struct {
	*BaseEvent
	ToolName string `json:"tool_name"` // 工具名称
	Error    string `json:"error"`     // 错误信息
}

// NewToolErrorEvent 创建工具执行错误事件
func NewToolErrorEvent(traceID, spanID, parentSpanID, toolName, error string) *ToolErrorEvent {
	return &ToolErrorEvent{
		BaseEvent: NewBaseEvent(traceID, spanID, parentSpanID, EventToolError),
		ToolName:  toolName,
		Error:     error,
	}
}
