package repository

// 为了向后兼容，从 internal/service/conversation 子包导出类型
// 新代码应该直接使用 internal/service/conversation 包

import (
	"github.com/weibaohui/nanobot-go/internal/service/conversation"
)

// ConversationRecordRepository 对话记录仓储接口（向后兼容）
type ConversationRecordRepository = conversation.Repository

// NewConversationRecordRepository 创建仓储实例（向后兼容）
var NewConversationRecordRepository = conversation.NewRepository
