package agent

import (
	"context"
	"fmt"

	"github.com/weibaohui/nanobot-go/agent/hooks/trace"
	"github.com/weibaohui/nanobot-go/bus"
	"go.uber.org/zap"
)

// Run 运行代理循环
func (l *Loop) Run(ctx context.Context) error {
	l.running = true
	l.logger.Info("消息监听循环处理功能已启动")

	for l.running {
		// 等待消息
		msg, err := l.bus.ConsumeInbound(ctx)
		if err != nil {
			if err == context.DeadlineExceeded {
				continue
			}
			if err == context.Canceled {
				return nil
			}
			return err
		}

		// 处理消息
		if err := l.processMessage(ctx, msg); err != nil {
			l.logger.Error("处理消息失败", zap.Error(err))
			outMsg := bus.NewOutboundMessage(msg.Channel, msg.ChatID, fmt.Sprintf("抱歉，我遇到了错误: %s", err))
			// 传递原始消息的 message_id
			if msg.Metadata != nil {
				if msgID, ok := msg.Metadata["message_id"].(string); ok {
					outMsg.Metadata["reply_to_message_id"] = msgID
				}
			}
			l.bus.PublishOutbound(outMsg)
		}
	}

	return nil
}

// processMessage 处理单条消息
func (l *Loop) processMessage(ctx context.Context, msg *bus.InboundMessage) error {
	preview := msg.Content
	if len(preview) > 80 {
		preview = preview[:80] + "..."
	}
	l.logger.Info("处理消息",
		zap.String("渠道", msg.Channel),
		zap.String("发送者", msg.SenderID),
		zap.String("内容", preview),
	)

	// 为每条消息创建根 span，建立完整的调用链
	ctx = trace.WithTraceID(ctx, trace.NewTraceID())
	ctx = trace.WithSpanID(ctx, trace.NewSpanID())
	// 根 span 没有 parentSpanID

	// 注入会话信息到 context，用于事件分发时获取
	sessionKey := msg.SessionKey()
	ctx = trace.WithSessionInfo(ctx, sessionKey, msg.Channel)

	// 触发收到消息事件
	if l.hookManager != nil {
		l.hookManager.OnMessageReceived(ctx, msg)
	}

	// 加载渠道绑定的 Agent 配置
	ctx, err := l.loadChannelAgentConfig(ctx, msg)
	if err != nil {
		l.logger.Warn("加载渠道 Agent 配置失败，将使用默认配置", zap.Error(err))
	}

	// 使用 Master Agent 处理消息（包括中断恢复和正常处理）
	if l.masterAgent == nil {
		return fmt.Errorf("Master Agent 未初始化，无法处理消息")
	}
	l.logger.Info("使用 Master Agent 处理消息")
	response, err := l.masterAgent.Process(ctx, msg)

	if err != nil {
		// 检查是否是中断
		if IsInterruptError(err) {
			return nil
		}
		// 非中断错误：如果 response 包含错误信息（由 interruptible 构造），直接发送
		// 否则构造默认错误消息
		outMsg := bus.NewOutboundMessage(msg.Channel, msg.ChatID, response)
		if response != "" {
			l.logger.Error("Master Agent 处理失败", zap.Error(err), zap.String("response", response))
		} else {
			l.logger.Error("Master Agent 处理失败", zap.Error(err))
			outMsg.Content = fmt.Sprintf("抱歉，处理消息时遇到错误: %v", err)
		}
		// 传递原始消息的 metadata 用于渠道特定功能
		if msg.Metadata != nil {
			// 复制 message_id 用于删除反应表情等功能
			if msgID, ok := msg.Metadata["message_id"].(string); ok {
				outMsg.Metadata["reply_to_message_id"] = msgID
			}
			// 复制 app_id 用于飞书多渠道路由
			if appID, ok := msg.Metadata["app_id"].(string); ok {
				outMsg.Metadata["app_id"] = appID
			}
		}
		l.bus.PublishOutbound(outMsg)
		return nil
	}

	// 发布响应
	outMsg := bus.NewOutboundMessage(msg.Channel, msg.ChatID, response)
	// 传递原始消息的 metadata 用于渠道特定功能
	if msg.Metadata != nil {
		// 复制 message_id 用于删除反应表情等功能
		if msgID, ok := msg.Metadata["message_id"].(string); ok {
			outMsg.Metadata["reply_to_message_id"] = msgID
		}
		// 复制 app_id 用于飞书多渠道路由
		if appID, ok := msg.Metadata["app_id"].(string); ok {
			outMsg.Metadata["app_id"] = appID
		}
	}
	l.bus.PublishOutbound(outMsg)
	return nil
}

// loadChannelAgentConfig 加载渠道绑定的 Agent 配置
// 从数据库获取 Agent 的 markdown 配置内容，设置到 ContextBuilder
// 返回注入 Agent 设置后的 context
func (l *Loop) loadChannelAgentConfig(ctx context.Context, msg *bus.InboundMessage) (context.Context, error) {
	if l.channelService == nil || l.agentService == nil {
		return ctx, fmt.Errorf("channelService 或 agentService 未初始化")
	}

	// 从消息元数据中获取 channel_id
	var channelID uint
	if msg.Metadata != nil {
		if cid, ok := msg.Metadata["channel_id"].(float64); ok {
			channelID = uint(cid)
		} else if cid, ok := msg.Metadata["channel_id"].(uint); ok {
			channelID = cid
		}
	}

	if channelID == 0 {
		return ctx, fmt.Errorf("消息中未包含 channel_id")
	}

	// 获取渠道信息
	channel, err := l.channelService.GetChannel(channelID)
	if err != nil {
		return ctx, fmt.Errorf("获取渠道信息失败: %w", err)
	}
	if channel == nil {
		return ctx, fmt.Errorf("渠道不存在: %d", channelID)
	}

	// 检查渠道是否绑定了 Agent
	if channel.AgentCode == "" {
		l.logger.Info("渠道未绑定 Agent，使用默认配置",
			zap.Uint("channel_id", channelID),
		)
		// 清除之前的 Agent 配置，使用默认文件配置
		l.context.SetAgentConfig(nil)
		// 未绑定 Agent，思考过程默认关闭
		ctx = trace.WithEnableThinkingProcess(ctx, false)
		// 仍然存储 channel_id 和 channel_code
		ctx = trace.WithChannelID(ctx, channelID)
		ctx = trace.WithChannelCode(ctx, channel.ChannelCode)
		// 通过 Channel 反推 UserCode
		ctx = trace.WithUserCode(ctx, channel.UserCode)
		return ctx, nil
	}

	// 获取 Agent 完整信息（不是配置内容）
	agent, err := l.agentService.GetAgentByCode(channel.AgentCode)
	if err != nil {
		return ctx, fmt.Errorf("获取 Agent 信息失败: %w", err)
	}

	// 获取 Agent 配置内容
	agentConfig, err := l.agentService.GetAgentConfigByCode(channel.AgentCode)
	if err != nil {
		return ctx, fmt.Errorf("获取 Agent 配置失败: %w", err)
	}

	// 创建 AgentConfig 并设置到 ContextBuilder
	config := &AgentConfig{
		IdentityContent: agentConfig.IdentityContent,
		SoulContent:     agentConfig.SoulContent,
		AgentsContent:   agentConfig.AgentsContent,
		ToolsContent:    agentConfig.ToolsContent,
		UserContent:     agentConfig.UserContent,
	}
	l.context.SetAgentConfig(config)

	// 将思考过程设置注入到 context
	ctx = trace.WithEnableThinkingProcess(ctx, agent.EnableThinkingProcess)
	// 将 channel_id, channel_code, agent_code 注入到 context
	ctx = trace.WithChannelID(ctx, channelID)
	ctx = trace.WithChannelCode(ctx, channel.ChannelCode)
	ctx = trace.WithAgentCode(ctx, agent.AgentCode)
	// 通过 Agent 反推 UserCode
	ctx = trace.WithUserCode(ctx, agent.UserCode)

	l.logger.Info("已加载渠道绑定的 Agent 配置",
		zap.Uint("channel_id", channelID),
		zap.String("agent_code", agent.AgentCode),
		zap.String("agent_name", agent.Name),
		zap.Bool("enable_thinking_process", agent.EnableThinkingProcess),
	)

	return ctx, nil
}
