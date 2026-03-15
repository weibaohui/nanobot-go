package agent

// 为了向后兼容，从 provider 子包导出类型
// 新代码应该直接使用 agent/provider 包

import (
	"github.com/weibaohui/nanobot-go/pkg/agent/provider"
)

// 类型别名（向后兼容）
type (
	ContextKey          = provider.ContextKey
	SkillLoader         = provider.SkillLoader
	HookCallback        = provider.HookCallback
	LLMConfig           = provider.LLMConfig
	LLMConfigLoader     = provider.LLMConfigLoader
	ChatModelAdapter    = provider.ChatModelAdapter
)

// 常量（向后兼容）
const SessionKeyContextKey = provider.SessionKeyContextKey

// 错误（向后兼容）
var (
	ErrNilConfig       = provider.ErrNilConfig
	ErrCreateChatModel = provider.ErrCreateChatModel
	ErrNilAPIKey       = provider.ErrNilAPIKey
)

// 函数（向后兼容）
var NewChatModelAdapter = provider.NewChatModelAdapter
