package service

// 为了向后兼容，从 conversation 子包导出类型
// 新代码应该直接使用 internal/service/conversation 包

import (
	"github.com/weibaohui/nanobot-go/internal/service/conversation"
)

// 类型别名（向后兼容）
type (
	ConversationDTO              = conversation.ConversationDTO
	TokenUsageDTO                = conversation.TokenUsageDTO
	ConversationListResult       = conversation.ConversationListResult
	ConversationRecordService    = conversation.Service
	ConversationRecordRepository = conversation.Repository
)

// 错误（向后兼容）
var (
	ErrRecordNotFound    = conversation.ErrRecordNotFound
	ErrInvalidParameter  = conversation.ErrInvalidParameter
	ErrDatabaseOperation = conversation.ErrDatabaseOperation
)

// 函数（向后兼容）
var NewConversationRecordService = conversation.NewService
