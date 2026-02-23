package dispatcher

import (
	"context"
	"sync"

	"github.com/weibaohui/nanobot-go/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/agent/hooks/observer"
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
func (d *Dispatcher) Dispatch(ctx context.Context, event events.Event, channel, sessionKey string) {
	d.mu.RLock()
	observers := make([]observer.Observer, len(d.observers))
	copy(observers, d.observers)
	d.mu.RUnlock()

	for _, obs := range observers {
		if !obs.Enabled() {
			continue
		}

		// 异步执行，避免阻塞主流程
		go func(o observer.Observer) {
			if err := o.OnEvent(ctx, event); err != nil {
				d.logger.Error("观察器处理事件失败",
					zap.String("observer", o.Name()),
					zap.String("event_type", string(event.GetEventType())),
					zap.String("trace_id", event.GetTraceID()),
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