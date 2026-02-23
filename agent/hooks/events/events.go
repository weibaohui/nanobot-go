package events

import (
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/bus"
)

// Event 事件接口
type Event interface {
	// ToBaseEvent 转换为基础事件
	ToBaseEvent() *BaseEvent

	// GetTraceID 获取追踪 ID
	GetTraceID() string

	// GetEventType 获取事件类型
	GetEventType() EventType

	// GetTimestamp 获取时间戳
	GetTimestamp() time.Time
}

// EventType 事件类型
type EventType string

const (
	// 消息相关事件
	EventMessageReceived    EventType = "message_received"     // 收到消息
	EventMessageSent        EventType = "message_sent"         // 发送消息
	EventPromptSubmitted    EventType = "prompt_submitted"     // 提交用户 prompt
	EventSystemPromptBuilt  EventType = "system_prompt_built"  // 生成系统 prompt

	// 工具相关事件
	EventToolCall           EventType = "tool_call"            // 工具调用
	EventToolIntercepted    EventType = "tool_intercepted"     // 工具调用被拦截
	EventToolUsed           EventType = "tool_used"            // 使用工具
	EventToolCompleted      EventType = "tool_completed"       // 工具执行完成
	EventToolError          EventType = "tool_error"           // 工具执行错误

	// 技能相关事件
	EventSkillCall          EventType = "skill_call"           // 技能调用
	EventSkillLookup        EventType = "skill_lookup"        // 查找技能
	EventSkillUsed          EventType = "skill_used"          // 使用技能

	// LLM 相关事件 (来自 Eino callbacks)
	EventLLMCallStart       EventType = "llm_call_start"       // LLM 调用开始
	EventLLMCallEnd         EventType = "llm_call_end"         // LLM 调用结束
	EventLLMCallError       EventType = "llm_call_error"       // LLM 调用错误

	// 通用事件
	EventComponentStart     EventType = "component_start"      // 组件开始执行
	EventComponentEnd       EventType = "component_end"        // 组件执行完成
	EventComponentError     EventType = "component_error"      // 组件执行错误
)

// BaseEvent 事件基类
type BaseEvent struct {
	TraceID   string    `json:"trace_id"`   // 追踪 ID
	EventType EventType `json:"event_type"` // 事件类型
	Timestamp time.Time `json:"timestamp"`  // 时间戳
	Data      map[string]interface{} `json:"data,omitempty"` // 事件数据
}

// ToBaseEvent 实现 Event 接口
func (e *BaseEvent) ToBaseEvent() *BaseEvent {
	return e
}

// GetTraceID 实现 Event 接口
func (e *BaseEvent) GetTraceID() string {
	return e.TraceID
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
func NewBaseEvent(traceID string, eventType EventType) *BaseEvent {
	return &BaseEvent{
		TraceID:   traceID,
		EventType: eventType,
		Timestamp: time.Now(),
		Data:      make(map[string]interface{}),
	}
}

// MessageReceivedEvent 收到消息事件
type MessageReceivedEvent struct {
	*BaseEvent
	Message    *bus.InboundMessage `json:"message"`     // 原始消息
	Preview    string             `json:"preview"`     // 内容预览
	SenderID   string             `json:"sender_id"`   // 发送者 ID
	ChatID     string             `json:"chat_id"`     // 聊天 ID
	Channel    string             `json:"channel"`     // 渠道名称
	SessionKey string             `json:"session_key"` // 会话键
}

// NewMessageReceivedEvent 创建收到消息事件
func NewMessageReceivedEvent(traceID string, msg *bus.InboundMessage) *MessageReceivedEvent {
	preview := msg.Content
	if len(preview) > 100 {
		preview = preview[:100] + "..."
	}

	return &MessageReceivedEvent{
		BaseEvent:  NewBaseEvent(traceID, EventMessageReceived),
		Message:    msg,
		Preview:    preview,
		SenderID:   msg.SenderID,
		ChatID:     msg.ChatID,
		Channel:    msg.Channel,
		SessionKey: msg.SessionKey(),
	}
}

// MessageSentEvent 发送消息事件
type MessageSentEvent struct {
	*BaseEvent
	Message    *bus.OutboundMessage `json:"message"`     // 输出消息
	Content    string               `json:"content"`     // 消息内容
	Preview    string               `json:"preview"`     // 内容预览
	Channel    string               `json:"channel"`     // 渠道名称
	ChatID     string               `json:"chat_id"`     // 聊天 ID
	SessionKey string               `json:"session_key"` // 会话键
}

// NewMessageSentEvent 创建发送消息事件
func NewMessageSentEvent(traceID string, msg *bus.OutboundMessage, sessionKey string) *MessageSentEvent {
	preview := msg.Content
	if len(preview) > 100 {
		preview = preview[:100] + "..."
	}

	return &MessageSentEvent{
		BaseEvent:  NewBaseEvent(traceID, EventMessageSent),
		Message:    msg,
		Content:    msg.Content,
		Preview:    preview,
		Channel:    msg.Channel,
		ChatID:     msg.ChatID,
		SessionKey: sessionKey,
	}
}

// PromptSubmittedEvent 提交 Prompt 事件
type PromptSubmittedEvent struct {
	*BaseEvent
	UserInput  string               `json:"user_input"`  // 用户输入
	Messages   []*schema.Message    `json:"messages"`    // 完整消息列表
	Count      int                  `json:"count"`       // 消息数量
	SessionKey string               `json:"session_key"` // 会话键
}

// NewPromptSubmittedEvent 创建提交 Prompt 事件
func NewPromptSubmittedEvent(traceID string, userInput string, messages []*schema.Message, sessionKey string) *PromptSubmittedEvent {
	return &PromptSubmittedEvent{
		BaseEvent:  NewBaseEvent(traceID, EventPromptSubmitted),
		UserInput:  userInput,
		Messages:   messages,
		Count:      len(messages),
		SessionKey: sessionKey,
	}
}

// SystemPromptBuiltEvent 生成系统 Prompt 事件
type SystemPromptBuiltEvent struct {
	*BaseEvent
	SystemPrompt string `json:"system_prompt"` // 系统提示词内容
	Length       int    `json:"length"`        // 提示词长度
}

// NewSystemPromptBuiltEvent 创建生成系统 Prompt 事件
func NewSystemPromptBuiltEvent(traceID string, systemPrompt string) *SystemPromptBuiltEvent {
	return &SystemPromptBuiltEvent{
		BaseEvent:   NewBaseEvent(traceID, EventSystemPromptBuilt),
		SystemPrompt: systemPrompt,
		Length:      len(systemPrompt),
	}
}

// ToolUsedEvent 使用工具事件
type ToolUsedEvent struct {
	*BaseEvent
	ToolName        string `json:"tool_name"`         // 工具名称
	ToolArguments   string `json:"tool_arguments"`    // 工具参数 (JSON)
	ArgumentsRaw    string `json:"arguments_raw"`     // 原始参数
}

// NewToolUsedEvent 创建使用工具事件
func NewToolUsedEvent(traceID string, toolName, toolArguments string) *ToolUsedEvent {
	return &ToolUsedEvent{
		BaseEvent:      NewBaseEvent(traceID, EventToolUsed),
		ToolName:       toolName,
		ToolArguments:  toolArguments,
		ArgumentsRaw:   toolArguments,
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
func NewToolCompletedEvent(traceID string, toolName, response string, success bool) *ToolCompletedEvent {
	return &ToolCompletedEvent{
		BaseEvent:      NewBaseEvent(traceID, EventToolCompleted),
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
func NewToolErrorEvent(traceID, toolName, error string) *ToolErrorEvent {
	return &ToolErrorEvent{
		BaseEvent: NewBaseEvent(traceID, EventToolError),
		ToolName:  toolName,
		Error:     error,
	}
}

// SkillLookupEvent 查找技能事件
type SkillLookupEvent struct {
	*BaseEvent
	SkillName string `json:"skill_name"` // 技能名称
	Found     bool   `json:"found"`      // 是否找到
	Source    string `json:"source"`     // 来源 (workspace/builtin)
	Path      string `json:"path"`       // 技能文件路径
}

// NewSkillLookupEvent 创建查找技能事件
func NewSkillLookupEvent(traceID, skillName string, found bool, source, path string) *SkillLookupEvent {
	return &SkillLookupEvent{
		BaseEvent: NewBaseEvent(traceID, EventSkillLookup),
		SkillName: skillName,
		Found:     found,
		Source:    source,
		Path:      path,
	}
}

// SkillUsedEvent 使用技能事件
type SkillUsedEvent struct {
	*BaseEvent
	SkillName   string `json:"skill_name"`   // 技能名称
	SkillLength int    `json:"skill_length"` // 技能内容长度
}

// NewSkillUsedEvent 创建使用技能事件
func NewSkillUsedEvent(traceID, skillName string, skillLength int) *SkillUsedEvent {
	return &SkillUsedEvent{
		BaseEvent:  NewBaseEvent(traceID, EventSkillUsed),
		SkillName:  skillName,
		SkillLength: skillLength,
	}
}

// LLMCallStartEvent LLM 调用开始事件 (来自 Eino callbacks)
type LLMCallStartEvent struct {
	*BaseEvent
	Component  string                 `json:"component"`  // 组件名称
	Model      string                 `json:"model"`      // 模型名称
	Messages   []*schema.Message      `json:"messages"`   // 消息列表
	ToolNames  []string               `json:"tool_names"` // 工具名称列表
	Config     map[string]interface{} `json:"config"`     // 配置
}

// NewLLMCallStartEvent 创建 LLM 调用开始事件
func NewLLMCallStartEvent(traceID string, info *callbacks.RunInfo, input *model.CallbackInput) *LLMCallStartEvent {
	toolNames := make([]string, 0, len(input.Tools))
	for _, t := range input.Tools {
		if t != nil {
			toolNames = append(toolNames, t.Name)
		}
	}

	config := make(map[string]interface{})
	if input.Config != nil {
		config["model"] = input.Config.Model
		config["max_tokens"] = input.Config.MaxTokens
		config["temperature"] = input.Config.Temperature
		config["top_p"] = input.Config.TopP
	}

	return &LLMCallStartEvent{
		BaseEvent: NewBaseEvent(traceID, EventLLMCallStart),
		Component: string(info.Component),
		Model:     info.Name,
		Messages:  input.Messages,
		ToolNames: toolNames,
		Config:    config,
	}
}

// LLMCallEndEvent LLM 调用结束事件 (来自 Eino callbacks)
type LLMCallEndEvent struct {
	*BaseEvent
	Component       string             `json:"component"`        // 组件名称
	Model           string             `json:"model"`            // 模型名称
	ResponseContent string             `json:"response_content"` // 响应内容
	ToolCalls       []schema.ToolCall  `json:"tool_calls"`       // 工具调用列表
	TokenUsage      *model.TokenUsage  `json:"token_usage"`      // Token 使用情况
	DurationMs      int64              `json:"duration_ms"`      // 持续时间 (毫秒)
}

// NewLLMCallEndEvent 创建 LLM 调用结束事件
func NewLLMCallEndEvent(traceID string, info *callbacks.RunInfo, output *model.CallbackOutput, durationMs int64) *LLMCallEndEvent {
	responseContent := ""
	if output.Message != nil {
		responseContent = output.Message.Content
	}

	return &LLMCallEndEvent{
		BaseEvent:       NewBaseEvent(traceID, EventLLMCallEnd),
		Component:       string(info.Component),
		Model:           info.Name,
		ResponseContent: responseContent,
		ToolCalls:       nil, // 可选：从 output.Message.ToolCalls 填充
		TokenUsage:      output.TokenUsage,
		DurationMs:      durationMs,
	}
}

// LLMCallErrorEvent LLM 调用错误事件 (来自 Eino callbacks)
type LLMCallErrorEvent struct {
	*BaseEvent
	Component  string `json:"component"` // 组件名称
	Model      string `json:"model"`     // 模型名称
	Error      string `json:"error"`     // 错误信息
	DurationMs int64  `json:"duration_ms"` // 持续时间 (毫秒)
}

// NewLLMCallErrorEvent 创建 LLM 调用错误事件
func NewLLMCallErrorEvent(traceID string, info *callbacks.RunInfo, err error, durationMs int64) *LLMCallErrorEvent {
	return &LLMCallErrorEvent{
		BaseEvent:  NewBaseEvent(traceID, EventLLMCallError),
		Component:  string(info.Component),
		Model:      info.Name,
		Error:      err.Error(),
		DurationMs: durationMs,
	}
}

// ComponentStartEvent 组件开始执行事件 (来自 Eino callbacks)
type ComponentStartEvent struct {
	*BaseEvent
	Component string `json:"component"` // 组件类型
	Type      string `json:"type"`      // 组件类型
	Name      string `json:"name"`      // 组件名称
}

// NewComponentStartEvent 创建组件开始执行事件
func NewComponentStartEvent(traceID string, info *callbacks.RunInfo) *ComponentStartEvent {
	return &ComponentStartEvent{
		BaseEvent: NewBaseEvent(traceID, EventComponentStart),
		Component: string(info.Component),
		Type:      info.Type,
		Name:      info.Name,
	}
}

// ComponentEndEvent 组件执行完成事件 (来自 Eino callbacks)
type ComponentEndEvent struct {
	*BaseEvent
	Component  string `json:"component"`  // 组件类型
	Type       string `json:"type"`       // 组件类型
	Name       string `json:"name"`       // 组件名称
	DurationMs int64  `json:"duration_ms"` // 持续时间 (毫秒)
}

// NewComponentEndEvent 创建组件执行完成事件
func NewComponentEndEvent(traceID string, info *callbacks.RunInfo, durationMs int64) *ComponentEndEvent {
	return &ComponentEndEvent{
		BaseEvent:  NewBaseEvent(traceID, EventComponentEnd),
		Component:  string(info.Component),
		Type:       info.Type,
		Name:       info.Name,
		DurationMs: durationMs,
	}
}

// ComponentErrorEvent 组件执行错误事件 (来自 Eino callbacks)
type ComponentErrorEvent struct {
	*BaseEvent
	Component  string `json:"component"`  // 组件类型
	Type       string `json:"type"`       // 组件类型
	Name       string `json:"name"`       // 组件名称
	Error      string `json:"error"`      // 错误信息
	DurationMs int64  `json:"duration_ms"` // 持续时间 (毫秒)
}

// NewComponentErrorEvent 创建组件执行错误事件
func NewComponentErrorEvent(traceID string, info *callbacks.RunInfo, err error, durationMs int64) *ComponentErrorEvent {
	return &ComponentErrorEvent{
		BaseEvent:  NewBaseEvent(traceID, EventComponentError),
		Component:  string(info.Component),
		Type:       info.Type,
		Name:       info.Name,
		Error:      err.Error(),
		DurationMs: durationMs,
	}
}