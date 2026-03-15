package agent

// 为了向后兼容，从 callbacks 子包导出类型
// 新代码应该直接使用 agent/callbacks 包

import (
	"github.com/weibaohui/nanobot-go/pkg/agent/callbacks"
)

// EinoCallbacks Eino 回调处理器（向后兼容）
type EinoCallbacks = callbacks.EinoCallbacks

// NewEinoCallbacks 创建新的 Eino 回调处理器（向后兼容）
var NewEinoCallbacks = callbacks.NewEinoCallbacks

// RegisterGlobalCallbacks 注册全局回调处理器（向后兼容）
var RegisterGlobalCallbacks = callbacks.RegisterGlobalCallbacks
