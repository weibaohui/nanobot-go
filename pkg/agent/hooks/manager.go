package hooks

import (
	"context"
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/dispatcher"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/eino"
	hookevents "github.com/weibaohui/nanobot-go/pkg/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/observer"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/trace"
	"github.com/weibaohui/nanobot-go/pkg/bus"
	"go.uber.org/zap"
)

// HookManager Hook 系统管理器
// 统一管理所有观察器和事件分发
type HookManager struct {
	dispatcher *dispatcher.Dispatcher
	einoBridge *eino.EinoCallbackBridge
	logger     *zap.Logger
	enabled    bool
}

// NewHookManager 创建 Hook 管理器
func NewHookManager(logger *zap.Logger, enabled bool) *HookManager {
	if logger == nil {
		logger = zap.NewNop()
	}

	disp := dispatcher.NewDispatcher(logger)
	einoBridge := eino.NewEinoCallbackBridge(disp, logger)

	return &HookManager{
		dispatcher: disp,
		einoBridge: einoBridge,
		logger:     logger,
		enabled:    enabled,
	}
}

// Register 注册观察器
func (hm *HookManager) Register(obs observer.Observer) {
	hm.dispatcher.Register(obs)
}

// Unregister 注销观察器
func (hm *HookManager) Unregister(name string) {
	hm.dispatcher.Unregister(name)
}

// Dispatch 分发事件
func (hm *HookManager) Dispatch(ctx context.Context, event hookevents.Event, channel, sessionKey string) {
	if !hm.enabled {
		return
	}
	hm.dispatcher.Dispatch(ctx, event, channel, sessionKey)
}

// EinoHandler 获取 Eino Callback Handler
func (hm *HookManager) EinoHandler() callbacks.Handler {
	return hm.einoBridge.Handler()
}

// EinoHandlerForGlobal 获取用于全局注册的 Handler
// 替代现有的 RegisterGlobalCallbacks
func (hm *HookManager) EinoHandlerForGlobal() *callbacks.Handler {
	if !hm.enabled {
		return nil
	}
	handler := hm.einoBridge.Handler()
	return &handler
}

// Enabled 返回是否启用
func (hm *HookManager) Enabled() bool {
	return hm.enabled
}

// SetEnabled 设置是否启用
func (hm *HookManager) SetEnabled(enabled bool) {
	hm.enabled = enabled
}

// List 列出所有观察器
func (hm *HookManager) List() []string {
	return hm.dispatcher.List()
}

// Count 返回观察器数量
func (hm *HookManager) Count() int {
	return hm.dispatcher.Count()
}

// ========== 便捷方法：触发常见事件 ==========

// OnMessageReceived 收到消息
func (hm *HookManager) OnMessageReceived(ctx context.Context, msg *bus.InboundMessage) {
	if !hm.enabled {
		return
	}
	traceID, spanID, parentSpanID := hm.extractTraceInfo(ctx)
	event := hookevents.NewMessageReceivedEvent(traceID, spanID, parentSpanID, msg)
	hm.Dispatch(ctx, event, msg.Channel, msg.SessionKey())
}

// OnMessageSent 发送消息
func (hm *HookManager) OnMessageSent(ctx context.Context, msg *bus.OutboundMessage, sessionKey string) {
	if !hm.enabled {
		return
	}
	traceID, spanID, parentSpanID := hm.extractTraceInfo(ctx)
	event := hookevents.NewMessageSentEvent(traceID, spanID, parentSpanID, msg, sessionKey)
	hm.Dispatch(ctx, event, msg.Channel, sessionKey)
}

// OnPromptSubmitted 提交 Prompt
func (hm *HookManager) OnPromptSubmitted(ctx context.Context, userInput string, messages []*schema.Message, sessionKey string) {
	if !hm.enabled {
		return
	}
	traceID, spanID, parentSpanID := hm.extractTraceInfo(ctx)
	event := hookevents.NewPromptSubmittedEvent(traceID, spanID, parentSpanID, userInput, messages, sessionKey)
	hm.Dispatch(ctx, event, "", sessionKey)
}

// OnSystemPromptBuilt 生成系统 Prompt
func (hm *HookManager) OnSystemPromptBuilt(ctx context.Context, systemPrompt string) {
	if !hm.enabled {
		return
	}
	traceID, spanID, parentSpanID := hm.extractTraceInfo(ctx)
	event := hookevents.NewSystemPromptBuiltEvent(traceID, spanID, parentSpanID, systemPrompt)
	hm.Dispatch(ctx, event, "", "")
}

// OnToolUsed 使用工具
func (hm *HookManager) OnToolUsed(ctx context.Context, toolName, toolArguments string) {
	if !hm.enabled {
		return
	}
	traceID, spanID, parentSpanID := hm.extractTraceInfo(ctx)
	event := hookevents.NewToolUsedEvent(traceID, spanID, parentSpanID, toolName, toolArguments)
	hm.Dispatch(ctx, event, "", "")
}

// OnToolCompleted 工具执行完成
func (hm *HookManager) OnToolCompleted(ctx context.Context, toolName, response string, success bool) {
	if !hm.enabled {
		return
	}
	traceID, spanID, parentSpanID := hm.extractTraceInfo(ctx)
	event := hookevents.NewToolCompletedEvent(traceID, spanID, parentSpanID, toolName, response, success)
	hm.Dispatch(ctx, event, "", "")
}

// OnToolError 工具执行错误
func (hm *HookManager) OnToolError(ctx context.Context, toolName, error string) {
	if !hm.enabled {
		return
	}
	traceID, spanID, parentSpanID := hm.extractTraceInfo(ctx)
	event := hookevents.NewToolErrorEvent(traceID, spanID, parentSpanID, toolName, error)
	hm.Dispatch(ctx, event, "", "")
}

// OnSkillLookup 查找技能
func (hm *HookManager) OnSkillLookup(ctx context.Context, skillName string, found bool, source, path string) {
	if !hm.enabled {
		return
	}
	traceID, spanID, parentSpanID := hm.extractTraceInfo(ctx)
	event := hookevents.NewSkillLookupEvent(traceID, spanID, parentSpanID, skillName, found, source, path)
	hm.Dispatch(ctx, event, "", "")
}

// OnSkillUsed 使用技能
func (hm *HookManager) OnSkillUsed(ctx context.Context, skillName string, skillLength int) {
	if !hm.enabled {
		return
	}
	traceID, spanID, parentSpanID := hm.extractTraceInfo(ctx)
	event := hookevents.NewSkillUsedEvent(traceID, spanID, parentSpanID, skillName, skillLength)
	hm.Dispatch(ctx, event, "", "")
}

// WithTraceID 创建包含 TraceID 的上下文
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return trace.WithTraceID(ctx, traceID)
}

// GetTraceID 从上下文获取 TraceID
func GetTraceID(ctx context.Context) string {
	return trace.GetTraceID(ctx)
}

// NewTraceID 生成新的 TraceID
func NewTraceID() string {
	return trace.NewTraceID()
}

// MustGetTraceID 从上下文获取 TraceID (如果不存在返回空字符串)
func MustGetTraceID(ctx context.Context) string {
	return trace.MustGetTraceID(ctx)
}

// ========== 统计信息 ==========

// extractTraceInfo 从上下文提取链路追踪信息
func (hm *HookManager) extractTraceInfo(ctx context.Context) (traceID, spanID, parentSpanID string) {
	return trace.GetTraceID(ctx), trace.GetSpanID(ctx), trace.GetParentSpanID(ctx)
}

// Stats 统计信息
type Stats struct {
	ObserverCount int        `json:"observer_count"`
	Enabled       bool       `json:"enabled"`
	LastEvent     *LastEvent `json:"last_event,omitempty"`
}

// LastEvent 最后一个事件
type LastEvent struct {
	EventType string    `json:"event_type"`
	TraceID   string    `json:"trace_id"`
	Timestamp time.Time `json:"timestamp"`
}

// GetStats 获取统计信息
func (hm *HookManager) GetStats() *Stats {
	return &Stats{
		ObserverCount: hm.Count(),
		Enabled:       hm.enabled,
	}
}
