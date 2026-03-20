# 023-WebSocket渠道与AI交互界面-设计

# 0. 文件修改记录表

| 修改人 | 修改时间 | 修改内容 |
| ------ | -------- | -------- |
| AI Assistant | 2026-03-18 | 初始版本 |

# 1. 概述

## 1.1 设计目标

本文档描述 WebSocket 渠道和配套前端 AI 对话界面的技术设计方案，实现用户通过浏览器与 AI Agent 进行实时交互的能力。

## 1.2 核心设计原则

1. **复用优先**：最大化复用现有渠道框架、消息总线和 Agent 处理流程
2. **接口兼容**：WebSocket 消息格式与飞书渠道保持一致，复用同一套消息处理逻辑
3. **前后端分离**：前端独立实现，通过 WebSocket 协议与后端通信
4. **渐进增强**：从基础文本对话开始，预留功能扩展空间

# 2. 系统架构

## 2.1 整体架构图

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                    前端                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ Chat 页面    │  │ Ant Design X │  │ WebSocket    │  │ 用户选择器   │      │
│  │              │──│ 组件库       │──│ Client       │──│              │      │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────────────────────┘
                                        │
                                        │ WebSocket (ws://host/ws/chat)
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                                    后端                                      │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                    pkg/channels/websocket/                           │    │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐               │    │
│  │  │ Channel      │  │ Connection   │  │ Protocol     │               │    │
│  │  │ 实现         │──│ Manager      │──│ Handler      │               │    │
│  │  └──────────────┘  └──────────────┘  └──────────────┘               │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                        │                                     │
│                                        ▼                                     │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                    pkg/bus/MessageBus                                │    │
│  │                      (入站/出站消息路由)                               │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                        │                                     │
│                    ┌───────────────────┼───────────────────┐                 │
│                    ▼                   ▼                   ▼                 │
│  ┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐  │
│  │ pkg/agent/          │  │ internal/service/   │  │ internal/api/       │  │
│  │ Agent 处理流程       │  │ conversation/       │  │ handlers            │  │
│  │                     │  │ 对话记录服务         │  │ (用户列表等)         │  │
│  └─────────────────────┘  └─────────────────────┘  └─────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 2.2 核心模块划分

| 模块 | 职责 | 关键文件 |
|------|------|----------|
| WebSocket Channel | WebSocket 服务端实现 | `pkg/channels/websocket/channel.go` |
| Connection Manager | 连接生命周期管理 | `pkg/channels/websocket/conn_manager.go` |
| Protocol Handler | 消息编码解码 | `pkg/channels/websocket/protocol.go` |
| Chat Page | 前端对话界面 | `web/src/pages/Chat.tsx` |
| WebSocket Client | 前端 WebSocket 封装 | `web/src/hooks/useWebSocket.ts` |

# 3. 后端设计

## 3.1 文件结构

```
pkg/channels/websocket/
├── channel.go          # Channel 接口实现
├── config.go           # 配置结构体
├── connection.go       # 单个连接管理
├── manager.go          # 连接管理器
├── protocol.go         # 消息协议定义
└── handler.go          # HTTP/WebSocket 处理器
```

## 3.2 Channel 实现

### 3.2.1 类型定义

```go
// Channel WebSocket 渠道实现
type Channel struct {
    name            string
    config          *Config
    bus             *bus.MessageBus
    logger          *zap.Logger

    // WebSocket 服务
    upgrader        websocket.Upgrader
    connManager     *ConnectionManager

    // 生命周期
    ctx             context.Context
    cancel          context.CancelFunc
    running         bool
}

// Config WebSocket 渠道配置
type Config struct {
    Addr string `json:"addr"`  // 监听地址，如 ":8080"
    Path string `json:"path"`  // WebSocket 路径，如 "/ws/chat"
}
```

### 3.2.2 核心方法

```go
// NewChannel 创建 WebSocket 渠道
func NewChannel(config *Config, messageBus *bus.MessageBus, logger *zap.Logger) *Channel

// Name 返回渠道名称
func (c *Channel) Name() string

// Start 启动 WebSocket 服务端
func (c *Channel) Start(ctx context.Context) error

// Stop 停止 WebSocket 服务端
func (c *Channel) Stop()

// HandleWebSocket HTTP 处理器，处理 WebSocket 升级
func (c *Channel) HandleWebSocket(w http.ResponseWriter, r *http.Request)
```

## 3.3 连接管理器

### 3.3.1 职责

- 管理所有活跃的 WebSocket 连接
- 根据 user_code 和 session_id 路由消息
- 处理连接断开和清理

### 3.3.2 类型定义

```go
// ConnectionManager WebSocket 连接管理器
type ConnectionManager struct {
    connections     map[string]*Connection  // key: connID
    userIndex       map[string][]string     // user_code -> []connID
    channelCode     string
    logger          *zap.Logger
    mu              sync.RWMutex
}

// Connection 单个 WebSocket 连接封装
type Connection struct {
    id          string
    conn        *websocket.Conn
    userCode    string
    channelCode string
    sessionID   string
    send        chan []byte
    logger      *zap.Logger
}
```

### 3.3.3 核心方法

```go
// Add 添加新连接
func (m *ConnectionManager) Add(conn *Connection)

// Remove 移除连接
func (m *ConnectionManager) Remove(connID string)

// SendToUser 向指定用户的所有连接发送消息
func (m *ConnectionManager) SendToUser(userCode string, data []byte)

// SendToConnection 向指定连接发送消息
func (m *ConnectionManager) SendToConnection(connID string, data []byte)

// Broadcast 广播消息给所有连接
func (m *ConnectionManager) Broadcast(data []byte)
```

## 3.4 消息协议

### 3.4.1 协议类型定义

```go
// MessageType 消息类型
type MessageType string

const (
    MessageTypePing     MessageType = "ping"
    MessageTypePong     MessageType = "pong"
    MessageTypeMessage  MessageType = "message"  // 用户消息
    MessageTypeChunk    MessageType = "chunk"    // AI 流式回复片段
    MessageTypeError    MessageType = "error"
    MessageTypeSystem   MessageType = "system"   // 系统消息
)

// Message 通用消息结构
type Message struct {
    Type      MessageType    `json:"type"`
    Payload   json.RawMessage `json:"payload"`
    Timestamp int64          `json:"timestamp"`
}

// MessagePayload 入站消息负载
type MessagePayload struct {
    Content   string `json:"content"`
    UserCode  string `json:"user_code"`
    SessionID string `json:"session_id,omitempty"`
}

// ChunkPayload 流式响应负载
type ChunkPayload struct {
    Content   string `json:"content"`
    SessionID string `json:"session_id"`
    IsEnd     bool   `json:"is_end"`
}

// ErrorPayload 错误消息负载
type ErrorPayload struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}
```

### 3.4.2 与 MessageBus 的映射

```go
// ToInboundMessage 将 WebSocket 消息转换为 MessageBus 入站消息
func (m *Message) ToInboundMessage(channelCode string) *bus.InboundMessage {
    return &bus.InboundMessage{
        ChannelType: "websocket",
        ChannelCode: channelCode,
        From:        payload.UserCode,
        Content:     payload.Content,
        SessionID:   payload.SessionID,
        Metadata: map[string]interface{}{
            "websocket": true,
        },
    }
}

// FromOutboundMessage 将 MessageBus 出站消息转换为 WebSocket 消息
func FromOutboundMessage(msg *bus.OutboundMessage) *Message {
    return &Message{
        Type:      MessageTypeChunk,
        Payload:   mustMarshal(ChunkPayload{
            Content:   msg.Content,
            SessionID: msg.SessionID,
            IsEnd:     msg.IsEnd,
        }),
        Timestamp: time.Now().UnixMilli(),
    }
}
```

## 3.5 与现有系统集成

### 3.5.1 MessageBus 订阅

```go
func (c *Channel) subscribeOutbound() {
    c.bus.SubscribeOutbound("websocket", func(msg *bus.OutboundMessage) error {
        // 检查消息是否属于当前渠道
        if msg.ChannelCode != c.config.ChannelCode {
            return nil
        }

        // 转换为 WebSocket 消息并发送
        wsMsg := FromOutboundMessage(msg)
        data, _ := json.Marshal(wsMsg)

        // 根据 user_code 路由到对应连接
        c.connManager.SendToUser(msg.To, data)
        return nil
    })
}
```

### 3.5.2 模型扩展

```go
// internal/models/channel.go

const (
    ChannelTypeFeishu   ChannelType = "feishu"
    ChannelTypeWebSocket ChannelType = "websocket"  // 新增
)
```

## 3.6 API 路由

```go
// internal/api/server.go

func (s *Server) setupRoutes() {
    // ... 现有路由

    // WebSocket 端点
    s.router.GET("/ws/chat", s.handleWebSocket)
}

func (s *Server) handleWebSocket(c *gin.Context) {
    channelCode := c.Query("channel_code")
    token := c.Query("token")

    // 验证 JWT Token
    claims, err := s.auth.ValidateToken(token)
    if err != nil {
        c.AbortWithStatus(401)
        return
    }

    // 获取 WebSocket Channel 实例
    wsChannel := s.channelManager.Get("websocket_" + channelCode)
    if wsChannel == nil {
        c.AbortWithStatus(404)
        return
    }

    // 升级到 WebSocket
    wsChannel.HandleWebSocket(c.Writer, c.Request)
}
```

# 4. 前端设计

## 4.1 文件结构

```
web/src/
├── pages/
│   └── Chat.tsx              # 对话主页面
├── components/
│   └── chat/
│       ├── ChatWindow.tsx    # 对话窗口组件
│       ├── UserSelector.tsx  # 用户选择器
│       └── MessageList.tsx   # 消息列表
├── hooks/
│   └── useWebSocket.ts       # WebSocket Hook
├── types/
│   └── chat.ts               # 类型定义
└── api/
    └── users.ts              # 用户相关 API
```

## 4.2 依赖安装

```bash
cd web
npm install @ant-design/x
```

## 4.3 Chat 页面组件

### 4.3.1 组件结构

```tsx
// web/src/pages/Chat.tsx

import React, { useState, useCallback } from 'react';
import { useParams } from 'react-router-dom';
import { Card, Select, Space, message as antMessage } from 'antd';
import { Bubble } from '@ant-design/x';
import { Sender } from '@ant-design/x';
import { Welcome } from '@ant-design/x';
import { useWebSocket } from '../hooks/useWebSocket';
import { useUserList } from '../hooks/useUserList';
import { getCurrentUserCode, getCurrentUserRole } from '../api';

interface ChatMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  isStreaming?: boolean;
}

const Chat: React.FC = () => {
  const { channelCode } = useParams<{ channelCode: string }>();
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [inputValue, setInputValue] = useState('');
  const [selectedUser, setSelectedUser] = useState<string>('');
  const [sessionId, setSessionId] = useState<string>('');

  const currentUser = getCurrentUserCode();
  const isAdmin = getCurrentUserRole() === 'admin';
  const { users } = useUserList();

  // WebSocket 连接
  const { sendMessage, isConnected } = useWebSocket({
    url: `/ws/chat?channel_code=${channelCode}`,
    onMessage: handleMessage,
    onError: handleError,
  });

  // 处理收到的消息
  const handleMessage = useCallback((msg: WebSocketMessage) => {
    switch (msg.type) {
      case 'chunk':
        handleChunk(msg.payload);
        break;
      case 'error':
        antMessage.error(msg.payload.message);
        break;
      case 'system':
        if (msg.payload.session_id) {
          setSessionId(msg.payload.session_id);
        }
        break;
    }
  }, []);

  // 处理流式响应片段
  const handleChunk = (payload: ChunkPayload) => {
    setMessages(prev => {
      const lastMsg = prev[prev.length - 1];
      if (lastMsg?.role === 'assistant' && lastMsg.isStreaming && !payload.is_end) {
        // 追加到现有消息
        return [
          ...prev.slice(0, -1),
          { ...lastMsg, content: lastMsg.content + payload.content }
        ];
      } else if (payload.is_end) {
        // 标记流结束
        return [
          ...prev.slice(0, -1),
          { ...lastMsg, isStreaming: false }
        ];
      } else {
        // 新消息开始
        return [...prev, {
          id: Date.now().toString(),
          role: 'assistant',
          content: payload.content,
          isStreaming: true
        }];
      }
    });
  };

  // 发送消息
  const handleSend = () => {
    if (!inputValue.trim() || !isConnected) return;

    const userCode = isAdmin && selectedUser ? selectedUser : currentUser;

    // 添加到本地消息列表
    setMessages(prev => [...prev, {
      id: Date.now().toString(),
      role: 'user',
      content: inputValue
    }]);

    // 发送 WebSocket 消息
    sendMessage({
      type: 'message',
      payload: {
        content: inputValue,
        user_code: userCode,
        session_id: sessionId
      }
    });

    setInputValue('');
  };

  return (
    <Card className="chat-page">
      <Space direction="vertical" style={{ width: '100%', height: 'calc(100vh - 200px)' }}>
        {/* 用户选择器 - 仅管理员可见 */}
        {isAdmin && (
          <UserSelector
            users={users}
            value={selectedUser}
            onChange={setSelectedUser}
          />
        )}

        {/* 欢迎界面 */}
        {messages.length === 0 && (
          <Welcome
            title="AI 助手"
            description="有什么可以帮助您的吗？"
          />
        )}

        {/* 消息列表 */}
        <div className="message-list" style={{ flex: 1, overflow: 'auto' }}>
          {messages.map(msg => (
            <Bubble
              key={msg.id}
              placement={msg.role === 'user' ? 'end' : 'start'}
              content={msg.content}
              typing={msg.isStreaming}
            />
          ))}
        </div>

        {/* 输入框 */}
        <Sender
          value={inputValue}
          onChange={setInputValue}
          onSubmit={handleSend}
          placeholder="输入消息..."
          disabled={!isConnected}
        />
      </Space>
    </Card>
  );
};

export default Chat;
```

## 4.4 WebSocket Hook

```tsx
// web/src/hooks/useWebSocket.ts

import { useEffect, useRef, useState, useCallback } from 'react';
import { getToken } from '../api';

interface UseWebSocketOptions {
  url: string;
  onMessage?: (msg: WebSocketMessage) => void;
  onError?: (error: Error) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
}

export function useWebSocket(options: UseWebSocketOptions) {
  const { url, onMessage, onError, onConnect, onDisconnect } = options;
  const [isConnected, setIsConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<NodeJS.Timeout>();
  const heartbeatTimerRef = useRef<NodeJS.Timeout>();

  const connect = useCallback(() => {
    const token = getToken();
    const wsUrl = `${url}&token=${token}`;

    const ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      setIsConnected(true);
      onConnect?.();
      startHeartbeat();
    };

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        onMessage?.(msg);
      } catch (e) {
        onError?.(new Error('Invalid message format'));
      }
    };

    ws.onerror = (error) => {
      onError?.(new Error('WebSocket error'));
    };

    ws.onclose = () => {
      setIsConnected(false);
      onDisconnect?.();
      stopHeartbeat();
      // 自动重连
      reconnectTimerRef.current = setTimeout(connect, 3000);
    };

    wsRef.current = ws;
  }, [url, onMessage, onError, onConnect, onDisconnect]);

  const disconnect = useCallback(() => {
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current);
    }
    stopHeartbeat();
    wsRef.current?.close();
  }, []);

  const sendMessage = useCallback((msg: object) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({
        ...msg,
        timestamp: Date.now()
      }));
    }
  }, []);

  const startHeartbeat = () => {
    heartbeatTimerRef.current = setInterval(() => {
      sendMessage({ type: 'ping' });
    }, 30000);
  };

  const stopHeartbeat = () => {
    if (heartbeatTimerRef.current) {
      clearInterval(heartbeatTimerRef.current);
    }
  };

  useEffect(() => {
    connect();
    return disconnect;
  }, [connect, disconnect]);

  return {
    isConnected,
    sendMessage,
    disconnect
  };
}
```

## 4.5 路由配置

```tsx
// web/src/App.tsx

import Chat from './pages/Chat';

// 在 Routes 中添加
<Route path="chat/:channelCode?" element={<Chat />} />

// 在菜单中添加
{
  key: 'chat',
  icon: <MessageOutlined />,
  label: 'AI 对话',
  path: '/chat'
}
```

# 5. 数据流设计

## 5.1 消息处理流程

```
┌──────────┐     ┌──────────────┐     ┌──────────────┐
│ 前端用户  │────▶│  WebSocket   │────▶│  Protocol    │
│ 输入消息  │     │   Client     │     │  Handler     │
└──────────┘     └──────────────┘     └──────┬───────┘
                                             │
                                             ▼
                              ┌────────────────────────────┐
                              │     MessageBus.PublishInbound │
                              └──────────────┬─────────────┘
                                             │
                         ┌───────────────────┼───────────────────┐
                         ▼                   ▼                   ▼
              ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
              │  Session Service │  │  Agent Runner   │  │ Conversation    │
              │  (创建/获取会话) │─▶│  (处理消息)     │─▶│ Service (记录) │
              └─────────────────┘  └────────┬────────┘  └─────────────────┘
                                            │
                                            ▼
                              ┌────────────────────────────┐
                              │    MessageBus.PublishOutbound │
                              └──────────────┬─────────────┘
                                             │
                              ┌──────────────┼──────────────┐
                              ▼              ▼              ▼
                    ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
                    │  WebSocket  │  │   Feishu    │  │   Other     │
                    │   Channel   │  │   Channel   │  │  Channels   │
                    └──────┬──────┘  └─────────────┘  └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │    前端      │
                    │  (流式展示) │
                    └─────────────┘
```

## 5.2 会话管理

```go
// WebSocket 渠道复用现有 Session 机制
// Session ID 由前端传入（继续对话）或由服务端生成（新对话）

func (c *Channel) handleInboundMessage(wsMsg *Message, conn *Connection) {
    // 转换为 MessageBus 入站消息
    inbound := wsMsg.ToInboundMessage(c.config.ChannelCode)

    // 如果没有 session_id，创建新会话
    if inbound.SessionID == "" {
        session := c.sessionService.CreateSession(inbound.UserCode, c.config.AgentCode)
        inbound.SessionID = session.SessionID
    }

    // 发布到消息总线
    c.bus.PublishInbound(inbound)
}
```

# 6. 安全设计

## 6.1 认证流程

```
1. 用户登录 → 获取 JWT Token
2. 前端连接 WebSocket → 在 Query 参数中携带 token
3. 服务端验证 Token → 提取 user_code
4. 建立连接 → 关联 user_code 与连接
```

## 6.2 权限控制

| 操作 | 管理员 | 普通用户 |
|------|--------|----------|
| 创建 WebSocket 渠道 | ✅ | ❌ |
| 查看所有渠道 | ✅ | 仅自己的 |
| 切换用户身份 | ✅ | ❌ |
| 发送消息 | ✅ | 仅自己身份 |

## 6.3 安全措施

- WebSocket 连接数限制（防止 DoS）
- 消息大小限制（10KB）
- Token 有效期验证
- 用户与 Channel 绑定关系验证

# 7. 错误处理

## 7.1 错误码定义

```go
const (
    ErrInvalidToken     = "INVALID_TOKEN"
    ErrChannelNotFound  = "CHANNEL_NOT_FOUND"
    ErrAgentNotBound    = "AGENT_NOT_BOUND"
    ErrRateLimited      = "RATE_LIMITED"
    ErrMessageTooLarge  = "MESSAGE_TOO_LARGE"
    ErrInternalError    = "INTERNAL_ERROR"
)
```

## 7.2 前端错误处理

```tsx
const handleError = (error: WebSocketError) => {
  switch (error.code) {
    case 'INVALID_TOKEN':
      // 跳转到登录页
      navigate('/login');
      break;
    case 'CHANNEL_NOT_FOUND':
      message.error('渠道不存在');
      break;
    case 'AGENT_NOT_BOUND':
      message.error('该渠道未绑定 Agent');
      break;
    default:
      message.error(error.message || '连接错误');
  }
};
```

# 8. 测试策略

## 8.1 单元测试

- WebSocket 协议编解码测试
- ConnectionManager 连接管理测试
- MessageBus 集成测试

## 8.2 集成测试

- 端到端消息流转测试
- 多客户端并发测试
- 断线重连测试

## 8.3 E2E 测试

- 前端界面交互测试
- 用户身份切换测试
- 流式响应展示测试

# 9. 部署配置

## 9.1 环境变量

```bash
# WebSocket 配置（可选，默认使用配置文件）
WEBSOCKET_ADDR=:8080
WEBSOCKET_PATH=/ws/chat
WEBSOCKET_MAX_CONNECTIONS=1000
```

## 9.2 Nginx 配置（如果使用）

```nginx
location /ws/chat {
    proxy_pass http://backend;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_read_timeout 86400;
}
```

# 10. 变更记录表（相对现有系统）

| 模块 | 变更类型 | 变更内容 |
|------|----------|----------|
| internal/models | 修改 | ChannelType 新增 websocket 枚举值 |
| pkg/channels | 新增 | websocket/ 目录及所有实现文件 |
| pkg/bus | 修改 | 如有需要，扩展消息格式 |
| web/package.json | 修改 | 新增 @ant-design/x 依赖 |
| web/src/pages | 新增 | Chat.tsx 页面 |
| web/src/hooks | 新增 | useWebSocket.ts 等 hooks |
| web/src/App.tsx | 修改 | 新增路由和菜单 |
| internal/api | 新增 | WebSocket 端点处理器 |

# 11. 风险评估与回滚方案

| 风险 | 影响 | 回滚方案 |
|------|------|----------|
| WebSocket 性能问题 | 系统资源耗尽 | 关闭 WebSocket 端口，前端降级为轮询 |
| Ant Design X 兼容性问题 | 前端构建失败 | 移除该依赖，使用原生 Ant Design 组件 |
| 消息格式不兼容 | 飞书渠道受影响 | 回滚到上一版本，修复格式问题 |
