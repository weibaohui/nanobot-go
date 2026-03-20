package agent

import (
	"context"
	"fmt"

	"github.com/weibaohui/nanobot-go/internal/service"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/trace"
	configtools "github.com/weibaohui/nanobot-go/pkg/agent/tools/config"
	"github.com/weibaohui/nanobot-go/pkg/bus"
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
func (l *Loop) processMessage(parentCtx context.Context, msg *bus.InboundMessage) error {
	preview := msg.Content
	if len(preview) > 80 {
		preview = preview[:80] + "..."
	}
	l.logger.Info("处理消息",
		zap.String("渠道", msg.Channel),
		zap.String("发送者", msg.SenderID),
		zap.String("内容", preview),
	)

	// 注入会话信息到 context，用于事件分发时获取
	sessionKey := msg.SessionKey()

	// 获取或创建会话
	sess := l.sessions.GetOrCreate(sessionKey)

	// 为当前会话创建独立的 cancellable context
	ctx, cancel := context.WithCancel(parentCtx)
	sess.SetContext(ctx, cancel)

	// 处理完成后清理 context
	defer func() {
		sess.SetContext(nil, nil)
	}()

	// 为每条消息创建根 span，建立完整的调用链
	ctx = trace.WithTraceID(ctx, trace.NewTraceID())
	ctx = trace.WithSpanID(ctx, trace.NewSpanID())
	// 根 span 没有 parentSpanID

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

	// 确保数据库中存在 Session 记录
	if l.sessionService != nil {
		if err := l.ensureSession(ctx, sessionKey, msg); err != nil {
			l.logger.Warn("确保 Session 存在失败", zap.Error(err))
		}
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

		// 检查是否是 context 取消（会话被强制停止）
		if ctx.Err() == context.Canceled {
			l.logger.Info("会话被取消",
				zap.String("session_key", sessionKey),
				zap.String("channel", msg.Channel),
			)
			outMsg := bus.NewOutboundMessage(msg.Channel, msg.ChatID, "对话已取消")
			l.bus.PublishOutbound(outMsg)
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
			// 复制 user_code 用于 WebSocket 路由
			if userCode, ok := msg.Metadata["user_code"].(string); ok {
				outMsg.Metadata["user_code"] = userCode
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
		// 复制 user_code 用于 WebSocket 路由
		if userCode, ok := msg.Metadata["user_code"].(string); ok {
			outMsg.Metadata["user_code"] = userCode
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
		// 清空 MCP Server 列表
		l.context.SetMCPServers(nil)
		// 未绑定 Agent，思考过程默认关闭
		ctx = trace.WithEnableThinkingProcess(ctx, false)
		// 仍然存储 channel_id 和 channel_code
		ctx = trace.WithChannelID(ctx, channelID)
		ctx = trace.WithChannelCode(ctx, channel.ChannelCode)
		// 通过 Channel 反推 UserCode
		ctx = trace.WithUserCode(ctx, channel.UserCode)
		// 注入配置工具上下文（未绑定 Agent，config tools 会因缺少 AgentCode 而失败）
		cfgCtx := &configtools.AgentConfigContext{
			UserCode:    channel.UserCode,
			AgentCode:   "",
			ChannelCode: channel.ChannelCode,
		}
		ctx = configtools.WithAgentConfigContext(ctx, cfgCtx)
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
		HistoryMessages: agentConfig.HistoryMessages,
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

	// 加载 Agent 绑定的 MCP Servers 信息
	if l.mcpService != nil {
		mcpServers, err := l.mcpService.GetAgentMCPServersWithBinding(agent.AgentCode)
		if err != nil {
			l.logger.Warn("获取 Agent MCP Servers 失败", zap.Error(err))
		} else {
			var mcpServerInfos []MCPServerInfo
			for _, info := range mcpServers {
				if info.Binding.IsActive && info.MCPServer != nil {
					mcpServerInfos = append(mcpServerInfos, MCPServerInfo{
						Code:        info.MCPServer.Code,
						Name:        info.MCPServer.Name,
						Description: info.MCPServer.Description,
					})
				}
			}
			l.context.SetMCPServers(mcpServerInfos)
			l.logger.Info("已加载 Agent MCP Servers",
				zap.String("agent_code", agent.AgentCode),
				zap.Int("server_count", len(mcpServerInfos)),
			)
		}
	}

	// 注入配置工具上下文（供 config tools 使用）
	cfgCtx := &configtools.AgentConfigContext{
		UserCode:    agent.UserCode,
		AgentCode:   agent.AgentCode,
		ChannelCode: channel.ChannelCode,
	}
	ctx = configtools.WithAgentConfigContext(ctx, cfgCtx)

	return ctx, nil
}

// ensureSession 确保数据库中存在 Session 记录
// 如果不存在则创建，如果存在则更新最后活跃时间
func (l *Loop) ensureSession(ctx context.Context, sessionKey string, msg *bus.InboundMessage) error {
	// 先检查 Session 是否已存在
	existingSession, err := l.sessionService.GetSessionByKey(sessionKey)
	if err != nil {
		return fmt.Errorf("查询 Session 失败: %w", err)
	}

	if existingSession != nil {
		// Session 已存在，更新最后活跃时间
		if err := l.sessionService.TouchSession(sessionKey); err != nil {
			l.logger.Warn("更新 Session 活跃时间失败", zap.Error(err))
		}
		l.logger.Debug("Session 已存在，更新活跃时间",
			zap.String("session_key", sessionKey))
		return nil
	}

	// 从 context 获取渠道信息（由 loadChannelAgentConfig 设置）
	channelCode := trace.GetChannelCode(ctx)
	userCode := trace.GetUserCode(ctx)
	agentCode := trace.GetAgentCode(ctx)

	// 如果 context 中没有，尝试从消息元数据和 channelService 获取
	if channelCode == "" || userCode == "" {
		if msg.Metadata != nil {
			if cid, ok := msg.Metadata["channel_id"].(float64); ok && cid > 0 {
				channel, err := l.channelService.GetChannel(uint(cid))
				if err == nil && channel != nil {
					channelCode = channel.ChannelCode
					userCode = channel.UserCode
					agentCode = channel.AgentCode
				}
			}
		}
	}

	// 如果仍然获取不到 channelCode，使用消息中的 channel 作为备选
	if channelCode == "" {
		channelCode = msg.Channel
	}

	// 如果 userCode 为空，使用默认值
	if userCode == "" {
		userCode = "default"
	}

	l.logger.Debug("准备创建 Session",
		zap.String("session_key", sessionKey),
		zap.String("user_code", userCode),
		zap.String("channel_code", channelCode),
		zap.String("agent_code", agentCode))

	// Session 不存在，创建新的
	req := service.CreateSessionRequest{
		SessionKey:  sessionKey,
		ChannelCode: channelCode,
		AgentCode:   agentCode,
		ExternalID:  msg.ChatID,
		Metadata: map[string]interface{}{
			"sender_id":  msg.SenderID,
			"channel":    msg.Channel,
			"created_by": "message_processor",
		},
	}

	session, err := l.sessionService.CreateSession(userCode, req)
	if err != nil {
		return fmt.Errorf("创建 Session 失败: %w", err)
	}

	l.logger.Info("创建新 Session",
		zap.String("session_key", sessionKey),
		zap.Uint("session_id", session.ID),
		zap.String("user_code", userCode),
		zap.String("channel_code", channelCode),
		zap.String("agent_code", agentCode))

	return nil
}
