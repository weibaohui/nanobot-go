package feishu

import (
	"context"
	"fmt"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	"github.com/larksuite/oapi-sdk-go/v3/ws"
	"github.com/weibaohui/nanobot-go/pkg/bus"
	"go.uber.org/zap"
)

// NewChannel 创建飞书渠道
func NewChannel(config *Config, messageBus *bus.MessageBus, logger *zap.Logger) *Channel {
	if logger == nil {
		logger = zap.NewNop()
	}
	// 使用 app_id 作为渠道名称的一部分，确保唯一性
	channelName := "feishu"
	if config.AppID != "" {
		channelName = fmt.Sprintf("feishu_%s", config.AppID)
	}
	return &Channel{
		bus:             messageBus,
		name:            channelName,
		config:          config,
		logger:          logger,
		processedMsgIDs: newSyncMap(1000),
		reactionCache:   make(map[string]*reactionInfo),
	}
}

// Name 返回渠道名称
func (c *Channel) Name() string {
	return c.name
}

// Bus 返回消息总线
func (c *Channel) Bus() *bus.MessageBus {
	return c.bus
}

// Start 启动飞书渠道
func (c *Channel) Start(ctx context.Context) error {
	if c.config.AppID == "" || c.config.AppSecret == "" {
		c.logger.Error("飞书 app_id 和 app_secret 未配置")
		return fmt.Errorf("飞书配置不完整")
	}

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.running = true

	// 创建飞书客户端（用于发送消息）
	c.client = lark.NewClient(c.config.AppID, c.config.AppSecret)

	// 创建事件处理器
	handler := newMessageHandler(c)
	c.eventHandler = dispatcher.NewEventDispatcher(
		c.config.VerificationToken,
		c.config.EncryptKey,
	).OnP2MessageReceiveV1(handler.onMessageReceive).
		OnP2MessageReactionCreatedV1(handler.onReactionCreated).
		OnP2MessageReactionDeletedV1(handler.onReactionDeleted)

	// 创建 WebSocket 客户端
	c.wsClient = ws.NewClient(c.config.AppID, c.config.AppSecret,
		ws.WithEventHandler(c.eventHandler),
		ws.WithLogLevel(larkcore.LogLevelInfo),
	)

	// 订阅出站消息
	c.bus.SubscribeOutbound("feishu", func(msg *bus.OutboundMessage) error {
		// 检查消息是否属于当前渠道（通过 app_id 匹配）
		if msg.Metadata != nil {
			if targetAppID, ok := msg.Metadata["app_id"].(string); ok && targetAppID != "" {
				if targetAppID != c.config.AppID {
					// 消息属于其他飞书渠道，跳过
					return nil
				}
			}
		}
		if err := c.Send(msg); err != nil {
			c.logger.Error("发送飞书消息失败", zap.Error(err))
			return err
		}
		return nil
	})

	c.logger.Info("飞书渠道已启动",
		zap.String("app_id", c.config.AppID),
	)

	// 启动 WebSocket 客户端（带重连）
	c.bgTasks.Add(1)
	go c.runWebSocketClient()

	return nil
}

// runWebSocketClient 运行 WebSocket 客户端（带重连）
func (c *Channel) runWebSocketClient() {
	defer c.bgTasks.Done()

	for c.running {
		err := c.wsClient.Start(c.ctx)
		if err != nil {
			if c.ctx.Err() != nil {
				// 上下文被取消，正常退出
				return
			}
			c.logger.Warn("飞书 WebSocket 连接错误", zap.Error(err))
		}

		if !c.running {
			break
		}

		// 等待 5 秒后重连
		select {
		case <-time.After(5 * time.Second):
		case <-c.ctx.Done():
			return
		}
	}
}

// Stop 停止飞书渠道
func (c *Channel) Stop() {
	c.running = false
	if c.cancel != nil {
		c.cancel()
	}

	// WebSocket 客户端会在上下文取消后自动关闭
	// 等待后台任务完成
	c.bgTasks.Wait()

	c.logger.Info("飞书渠道已停止")
}
