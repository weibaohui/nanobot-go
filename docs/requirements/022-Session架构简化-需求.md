# 022-Session架构简化-需求

## 需求概述

将当前双层 Session 管理机制（内存 + 数据库）简化为以数据库为主、内存仅保留运行时状态的混合架构，解决会话消息在内存中累积导致的内存膨胀问题，同时支持多实例部署。

## 问题背景

当前 `pkg/session/manager.go` 中的 `Session` 结构体同时承担了两种职责：

1. **消息缓存**：`Messages []Message` 存储完整对话历史
2. **运行时控制**：`cancel context.CancelFunc` 用于取消正在进行的 AI 请求

这导致以下问题：

| 问题 | 影响 |
|------|------|
| 内存膨胀 | 长会话消息累积，占用大量内存 |
| 数据不一致 | 内存消息与数据库 `conversation_records` 表可能不同步 |
| 多实例障碍 | 内存缓存无法共享，阻碍水平扩展 |
| 会话丢失 | 服务重启后内存消息全部丢失 |

## 需求目标

1. 移除内存中的消息缓存，消息实时读写数据库
2. 保留内存 Manager 仅用于管理运行时 context（取消请求）
3. 简化 `Session` 结构体，明确职责边界
4. 保持现有 API 和行为不变（向后兼容）

## 功能需求

### FR1: 移除消息缓存

**描述**: `pkg/session.Session` 结构体移除 `Messages []Message` 字段。

**验收标准**:
- [ ] `Session` 结构体不再包含 `Messages` 字段
- [ ] `AddMessage()` 方法移除或改为空实现
- [ ] `AddMessageWithTrace()` 方法移除或改为空实现
- [ ] `Clear()` 方法移除或改为空实现

### FR2: 实时历史加载

**描述**: `SessionManager.GetHistory()` 改为每次从数据库实时查询。

**验收标准**:
- [ ] `GetHistory()` 直接从 `conversation_records` 表查询
- [ ] 移除内存缓存相关的历史合并逻辑
- [ ] 支持按 `maxMessages` 参数限制返回数量
- [ ] 结果按时间正序排列（最旧在前）

### FR3: 保留 Context 管理

**描述**: 内存 Manager 保留 `context.CancelFunc` 管理能力，用于取消正在进行的 AI 请求。

**验收标准**:
- [ ] `Session` 保留 `cancel` 和 `ctx` 字段
- [ ] `SetContext()` / `GetContext()` 方法正常工作
- [ ] `Cancel()` 方法正常工作
- [ ] `IsActive()` 方法正常工作
- [ ] `CancelSession()` 方法正常工作

### FR4: 适配现有调用方

**描述**: 修改所有调用 `session.AddMessage()` 的代码，改为直接写入数据库。

**验收标准**:
- [ ] 识别所有调用 `AddMessage` / `AddMessageWithTrace` 的代码位置
- [ ] 修改为通过 `ConversationRecordRepository` 直接写入数据库
- [ ] 确保消息格式和字段与现有逻辑一致

### FR5: 向后兼容

**描述**: 现有 API 和行为保持不变。

**验收标准**:
- [ ] 会话历史查询接口返回格式不变
- [ ] 取消会话接口行为不变
- [ ] 现有单元测试通过

## 非功能需求

### NFR1: 性能
- 单次 `GetHistory()` 查询耗时 < 50ms（假设 100 条消息）
- 消息写入延迟与之前持平（直接写库）

### NFR2: 资源占用
- 内存中不再累积会话消息
- 长会话的内存占用保持稳定

### NFR3: 可靠性
- 服务重启不会丢失消息历史
- 消息写入失败返回错误，不静默丢弃

## 数据结构变更

### 移除的字段

```go
// pkg/session/manager.go

// 从 Session 结构体移除
- Messages []Message

// 从 Message 结构体（可考虑移除整个类型）
- type Message struct { ... }
```

### 保留的字段

```go
// pkg/session/manager.go

type Session struct {
    Key       string             // 会话标识
    CreatedAt time.Time
    UpdatedAt time.Time
    cancel    context.CancelFunc // 运行时状态
    ctx       context.Context    // 运行时状态
    mu        sync.RWMutex       // 保护运行时字段
}
```

## 影响模块

| 模块 | 变更内容 |
|------|----------|
| `pkg/session/manager.go` | 移除消息相关字段和方法 |
| `pkg/agent/loop.go` | 修改消息存储逻辑 |
| `pkg/agent/interruptible.go` | 可能需要调整 |
| `pkg/agent/provider/adapter.go` | 可能需要调整 |
| `internal/api/handler_conversation.go` | 如有直接调用需调整 |

## 任务清单

- [ ] 创建功能分支
- [ ] 编写设计文档
- [ ] 修改 `pkg/session/manager.go` 移除消息缓存
- [ ] 适配所有调用 `AddMessage` 的代码
- [ ] 运行单元测试
- [ ] 编译验证
- [ ] 编写实现总结
- [ ] 提交 PR

## 相关文档

- 设计文档: `docs/design/022-Session架构简化-设计.md`
- `conversation_records` 表模型: `internal/models/conversation_record.go`
