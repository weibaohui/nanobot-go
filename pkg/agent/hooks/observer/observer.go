package observer

import (
	"context"

	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/events"
)

// Observer 观察型 Hook 接口
// 只读观察系统事件，不修改任何数据
type Observer interface {
	// Name 返回观察器名称
	Name() string

	// OnEvent 处理事件
	// ctx: 包含 traceID 的上下文
	// event: 事件对象
	OnEvent(ctx context.Context, event events.Event) error

	// Enabled 返回是否启用此观察器
	Enabled() bool
}

// ObserverFilter 观察器过滤器
type ObserverFilter struct {
	// EventTypes 只监听指定的事件类型，为空则监听所有事件
	EventTypes []events.EventType

	// Channels 只监听指定渠道的消息，为空则监听所有渠道
	Channels []string

	// SessionKeys 只监听指定会话的消息，为空则监听所有会话
	SessionKeys []string
}

// ShouldNotify 判断是否应该通知此观察器
func (f *ObserverFilter) ShouldNotify(eventType events.EventType, channel, sessionKey string) bool {
	// 检查事件类型
	if len(f.EventTypes) > 0 {
		found := false
		for _, t := range f.EventTypes {
			if t == eventType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// 检查渠道
	if len(f.Channels) > 0 && channel != "" {
		found := false
		for _, c := range f.Channels {
			if c == channel {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// 检查会话
	if len(f.SessionKeys) > 0 && sessionKey != "" {
		found := false
		for _, s := range f.SessionKeys {
			if s == sessionKey {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// BaseObserver 基础观察器实现
// 抽象基类，需要子类实现 OnEvent 方法
type BaseObserver struct {
	name    string
	filter  *ObserverFilter
	enabled bool
}

// NewBaseObserver 创建基础观察器
func NewBaseObserver(name string, filter *ObserverFilter) *BaseObserver {
	if filter == nil {
		filter = &ObserverFilter{}
	}
	return &BaseObserver{
		name:    name,
		filter:  filter,
		enabled: true,
	}
}

// Name 返回观察器名称
func (o *BaseObserver) Name() string {
	return o.name
}

// Enabled 返回是否启用
func (o *BaseObserver) Enabled() bool {
	return o.enabled
}

// SetEnabled 设置是否启用
func (o *BaseObserver) SetEnabled(enabled bool) {
	o.enabled = enabled
}

// Filter 返回过滤器
func (o *BaseObserver) Filter() *ObserverFilter {
	return o.filter
}

// ShouldNotify 判断是否应该通知
func (o *BaseObserver) ShouldNotify(eventType events.EventType, channel, sessionKey string) bool {
	return o.filter.ShouldNotify(eventType, channel, sessionKey)
}

// OnEvent 需要子类实现
func (o *BaseObserver) OnEvent(ctx context.Context, event events.Event) error {
	return nil
}