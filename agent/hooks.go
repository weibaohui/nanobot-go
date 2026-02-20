package agent

import (
	"context"

	"github.com/weibaohui/nanobot-go/bus"
	"github.com/weibaohui/nanobot-go/session"
	"go.uber.org/zap"
)

// Hook 定义 Agent 处理过程中的钩子接口
type Hook interface {
	// AfterMessageProcess 在消息处理完成后调用
	AfterMessageProcess(ctx context.Context, msg *bus.InboundMessage, sess *session.Session, response string) error
	// Name 返回 Hook 的名称
	Name() string
}

// HookManager Hook 管理器
type HookManager struct {
	hooks  []Hook
	logger *zap.Logger
}

// NewHookManager 创建 Hook 管理器
func NewHookManager() *HookManager {
	return &HookManager{
		hooks:  make([]Hook, 0),
		logger: zap.NewNop(),
	}
}

// SetLogger 设置日志记录器
func (hm *HookManager) SetLogger(logger *zap.Logger) {
	hm.logger = logger
}

// Register 注册 Hook
func (hm *HookManager) Register(hook Hook) {
	if hook == nil {
		return
	}
	hm.hooks = append(hm.hooks, hook)
	hm.logger.Info("注册 Hook", zap.String("name", hook.Name()))
}

// ExecuteAfterMessageProcess 执行消息后处理 Hook
func (hm *HookManager) ExecuteAfterMessageProcess(ctx context.Context, msg *bus.InboundMessage, sess *session.Session, response string) error {
	for _, hook := range hm.hooks {
		if err := hook.AfterMessageProcess(ctx, msg, sess, response); err != nil {
			hm.logger.Error("Hook 执行失败",
				zap.String("hook", hook.Name()),
				zap.Error(err),
			)
			// 继续执行其他 Hook，不中断
		}
	}
	return nil
}
