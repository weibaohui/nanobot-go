package dispatcher

import (
	"context"
	"sync"

	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/observer"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/trace"
	"go.uber.org/zap"
)

// Dispatcher 事件分发器
// 负责将事件分发给所有注册的观察器
type Dispatcher struct {
	observers []observer.Observer
	logger    *zap.Logger
	mu        sync.RWMutex
}

// NewDispatcher 创建事件分发器
func NewDispatcher(logger *zap.Logger) *Dispatcher {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Dispatcher{
		observers: make([]observer.Observer, 0),
		logger:    logger,
	}
}

// Register 注册观察器
func (d *Dispatcher) Register(obs observer.Observer) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if obs == nil {
		d.logger.Warn("尝试注册 nil 观察器")
		return
	}

	d.observers = append(d.observers, obs)
	d.logger.Info("注册观察器",
		zap.String("name", obs.Name()),
		zap.Bool("enabled", obs.Enabled()),
	)
}

// Unregister 注销观察器
func (d *Dispatcher) Unregister(name string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for i, obs := range d.observers {
		if obs.Name() == name {
			d.observers = append(d.observers[:i], d.observers[i+1:]...)
			d.logger.Info("注销观察器", zap.String("name", name))
			return
		}
	}
}

// Dispatch 分发事件
// channel 和 sessionKey 会被注入到 context 中，供 Observer 获取
func (d *Dispatcher) Dispatch(ctx context.Context, event events.Event, channel, sessionKey string) {
	d.mu.RLock()
	observers := make([]observer.Observer, len(d.observers))
	copy(observers, d.observers)
	d.mu.RUnlock()

	// 如果 context 中没有 sessionKey/channel，使用 Dispatch 参数补充
	// 这确保从 loop.go 传递的会话信息能被所有事件使用
	existingSessionKey := trace.GetSessionKey(ctx)
	existingChannel := trace.GetChannel(ctx)
	if existingSessionKey == "" && sessionKey != "" {
		ctx = trace.WithSessionKey(ctx, sessionKey)
	}
	if existingChannel == "" && channel != "" {
		ctx = trace.WithChannel(ctx, channel)
	}

	// 提取 trace 信息用于异步处理
	traceID := event.GetTraceID()
	spanID := trace.GetSpanID(ctx)
	parentSpanID := trace.GetParentSpanID(ctx)

	for _, obs := range observers {
		if !obs.Enabled() {
			continue
		}

		// 异步执行，避免阻塞主流程
		// 使用 background context 避免父 context 取消导致写入失败
		go func(o observer.Observer) {
			// 创建新的 context，复制必要的 trace 信息
			bgCtx := context.Background()
			bgCtx = trace.WithSessionKey(bgCtx, trace.GetSessionKey(ctx))
			bgCtx = trace.WithChannel(bgCtx, trace.GetChannel(ctx))
			bgCtx = trace.WithTraceID(bgCtx, traceID)
			bgCtx = trace.WithSpanID(bgCtx, spanID)
			bgCtx = trace.WithParentSpanID(bgCtx, parentSpanID)
			bgCtx = trace.WithEnableThinkingProcess(bgCtx, trace.GetEnableThinkingProcess(ctx))
			bgCtx = trace.WithChannelID(bgCtx, trace.GetChannelID(ctx))
			bgCtx = trace.WithChannelCode(bgCtx, trace.GetChannelCode(ctx))
			bgCtx = trace.WithAgentID(bgCtx, trace.GetAgentID(ctx))
			bgCtx = trace.WithAgentCode(bgCtx, trace.GetAgentCode(ctx))
			bgCtx = trace.WithUserCode(bgCtx, trace.GetUserCode(ctx))

			if err := o.OnEvent(bgCtx, event); err != nil {
				d.logger.Error("观察器处理事件失败",
					zap.String("observer", o.Name()),
					zap.String("event_type", string(event.GetEventType())),
					zap.String("trace_id", traceID),
					zap.Error(err),
				)
			}
		}(obs)
	}
}

// List 列出所有观察器
func (d *Dispatcher) List() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	names := make([]string, len(d.observers))
	for i, obs := range d.observers {
		names[i] = obs.Name()
	}
	return names
}

// Count 返回观察器数量
func (d *Dispatcher) Count() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.observers)
}