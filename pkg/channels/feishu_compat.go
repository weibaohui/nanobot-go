package channels

// 为了向后兼容，从 feishu 子包导出类型
// 新代码应该直接使用 channels/feishu 包

import (
	"github.com/weibaohui/nanobot-go/pkg/bus"
	"github.com/weibaohui/nanobot-go/pkg/channels/feishu"
	"go.uber.org/zap"
)

// FeishuConfig 飞书配置（向后兼容）
type FeishuConfig = feishu.Config

// NewFeishuChannel 创建飞书渠道（向后兼容）
func NewFeishuChannel(config *FeishuConfig, messageBus *bus.MessageBus, logger *zap.Logger) Channel {
	return feishu.NewChannel(config, messageBus, logger)
}
