# 023-WebSocket渠道与AI交互界面-需求

# 0. 文件修改记录表

| 修改人 | 修改时间 | 修改内容 |
| ------ | -------- | -------- |
| AI Assistant | 2026-03-18 | 初始版本 |

# 1. 背景（Why）

当前系统已支持飞书渠道与 AI 进行交互，但缺少一个直接面向 Web 用户的实时交互界面。为了提供更直观的 AI 对话体验，需要新增 WebSocket 渠道，并配套一个类似 ChatGPT 的前端交互界面。用户可以通过浏览器直接与 AI Agent 进行实时对话，同时支持选择以特定用户身份发起对话，以便测试不同用户的上下文和权限。

# 2. 目标（What，必须可验证）

- [ ] 后端 WebSocket 渠道实现，支持实时消息收发
- [ ] WebSocket 渠道支持 Agent 绑定（复用现有 Channel- Agent 绑定机制）
- [ ] 前端 AI 对话界面，基于 Ant Design X 组件库实现
- [ ] 前端界面支持用户身份选择功能
- [ ] 前端界面支持实时流式响应展示
- [ ] 对话记录与现有 Conversation 服务集成

# 3. 非目标（Explicitly Out of Scope）

- 不支持多用户同时在一个 WebSocket 连接中切换身份
- 不支持文件上传/图片发送（仅文本消息）
- 不支持语音输入/输出
- 不实现复杂的权限控制（复用现有用户体系）
- 不替代飞书渠道，两者独立运行

# 4. 使用场景 / 用户路径

## 场景 1：管理员配置 WebSocket 渠道

1. 管理员登录系统
2. 进入"渠道管理"页面
3. 点击"新建渠道"
4. 选择类型为"WebSocket"
5. 配置监听地址和路径（或使用默认）
6. 选择要绑定的 Agent
7. 保存配置

## 场景 2：用户通过前端界面与 AI 对话

1. 用户打开浏览器访问系统
2. 登录后进入"AI 对话"页面
3. 页面自动连接 WebSocket
4. 用户从下拉菜单选择要模拟的用户身份（可选）
5. 用户在输入框中输入消息
6. 系统实时显示 AI 的流式回复
7. 对话历史自动保存到 Conversation 服务

# 5. 功能需求清单（Checklist）

## 后端功能

- [ ] 实现 WebSocketChannel 类型，满足 Channel 接口
- [ ] WebSocket 服务端支持多客户端连接管理
- [ ] 消息协议设计（入站/出站消息格式）
- [ ] 用户身份验证与权限校验
- [ ] 与 MessageBus 集成，接入现有消息处理流程
- [ ] 流式响应支持（SSE 风格通过 WebSocket 传输）

## 前端功能

- [ ] 安装 @ant-design/x 依赖
- [ ] 创建 Chat 页面路由 `/chat`
- [ ] 使用 Ant Design X 的 Bubble 组件显示对话气泡
- [ ] 使用 Ant Design X 的 Sender 组件作为消息输入框
- [ ] 使用 Ant Design X 的 Welcome 组件显示欢迎界面
- [ ] 用户选择器组件（Select 下拉框）
- [ ] WebSocket 连接管理（自动重连、心跳检测）
- [ ] 流式消息渲染（打字机效果）
- [ ] 对话历史展示

## 集成功能

- [ ] 渠道类型注册（ChannelTypeWebSocket）
- [ ] 前端 API 获取用户列表
- [ ] 消息格式与现有飞书渠道兼容

# 6. 约束条件

## 技术约束

- 后端使用 Go 的 gorilla/websocket 库实现 WebSocket 服务端
- 前端使用 React + TypeScript + Ant Design X
- WebSocket 消息格式必须为 JSON
- 必须复用现有的 Agent 处理流程和 Session 管理

## 架构约束

- WebSocket Channel 必须实现 `pkg/channels.Channel` 接口
- 消息必须通过 `pkg/bus.MessageBus` 进行路由
- 对话记录必须通过 `internal/service/conversation` 服务保存
- 用户身份验证复用现有的 JWT 机制

## 安全约束

- WebSocket 连接必须验证 JWT Token
- 用户只能访问自己有权限的 Agent
- 用户选择器仅对管理员显示（普通用户固定为自己）

## 性能约束

- WebSocket 连接数上限：1000（可配置）
- 单条消息大小限制：10KB
- 心跳间隔：30秒

# 7. 可修改 / 不可修改项

- ❌ 不可修改：
  - 现有 Channel 接口定义
  - 现有 MessageBus 消息格式
  - 现有 Conversation 服务接口
  - 现有 Agent 处理流程

- ✅ 可调整：
  - WebSocket 消息协议字段命名
  - 前端界面布局和样式
  - 心跳间隔和重连策略参数

# 8. 接口与数据约定

## WebSocket 连接建立

```
WS /ws/chat?channel_code={channel_code}&token={jwt_token}
```

## 客户端发送消息（入站）

```json
{
  "type": "message",
  "payload": {
    "content": "用户消息内容",
    "user_code": "user_xxx",
    "session_id": "可选，用于继续对话"
  },
  "timestamp": 1710000000000
}
```

## 服务端推送消息（出站）

```json
{
  "type": "chunk",
  "payload": {
    "content": "AI 回复片段",
    "session_id": "sess_xxx",
    "is_end": false
  },
  "timestamp": 1710000000000
}
```

## 服务端结束标记

```json
{
  "type": "chunk",
  "payload": {
    "content": "",
    "session_id": "sess_xxx",
    "is_end": true
  },
  "timestamp": 1710000000000
}
```

## 错误消息

```json
{
  "type": "error",
  "payload": {
    "code": "ERROR_CODE",
    "message": "错误描述"
  },
  "timestamp": 1710000000000
}
```

## 心跳消息

```json
{
  "type": "ping"
}
```

```json
{
  "type": "pong"
}
```

# 9. 验收标准（Acceptance Criteria）

- **AC1**: 如果管理员创建了 WebSocket 渠道并绑定 Agent，那么前端可以通过 WebSocket 连接到该渠道并发送消息
- **AC2**: 如果用户发送消息，那么 AI 的回复以流式方式实时显示在对话界面
- **AC3**: 如果用户选择了特定用户身份，那么对话记录归属于该用户，并在 Conversation 服务中可查
- **AC4**: 如果 WebSocket 连接断开，那么前端自动重连，且对话上下文保持
- **AC5**: 如果管理员查看渠道列表，那么 WebSocket 渠道与飞书渠道一样正常显示和管理
- **AC6**: 如果普通用户访问对话页面，那么只能以自己的身份发起对话（不显示用户选择器）

# 10. 风险与已知不确定点

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| Ant Design X 与现有 Ant Design 版本冲突 | 前端构建失败 | 提前测试依赖兼容性，必要时升级 Ant Design |
| WebSocket 连接数过多导致性能问题 | 服务端资源耗尽 | 实现连接数限制和优雅降级 |
| 流式响应与现有飞书渠道格式不兼容 | 消息处理异常 | 统一消息格式，增加适配层 |

## 不确定点

- Ant Design X 的具体组件 API 需要根据官方文档确认
- WebSocket 是否需要支持集群部署（待后续评估）

# 11. 相关文档与资源

- Ant Design X 官方文档：https://x.ant.design/
- 现有飞书渠道实现：`pkg/channels/feishu/`
- 渠道模型定义：`internal/models/channel.go`
- MessageBus 实现：`pkg/bus/`

# 12. 开发顺序建议

1. 后端：定义 WebSocket 消息协议和 Channel 实现
2. 后端：集成 MessageBus 和现有 Agent 处理流程
3. 前端：安装 Ant Design X，创建基础 Chat 页面
4. 前端：实现 WebSocket 连接管理和消息收发
5. 前端：集成用户选择器和流式渲染
6. 集成测试：端到端验证完整流程
