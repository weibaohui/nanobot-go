# 022-Session架构简化-设计

## 设计目标

将 `pkg/session` 包从「消息缓存 + Context 管理」双职责，简化为仅「Context 管理」单职责。消息存储完全交由 `conversation_records` 表。

## 当前架构分析

### 双层架构现状

```
┌─────────────────────────────────────────────────────────────┐
│                      内存层 (pkg/session)                    │
│  ┌──────────────┐    Messages []Message                     │
│  │   Session    │    AddMessage() / Clear()                 │
│  │              │    SetContext() / Cancel()                │
│  └──────────────┘                                           │
│           │                                                 │
│  ┌─────────────────┐                                        │
│  │ SessionManager  │  GetOrCreate() / GetHistory()          │
│  └─────────────────┘                                        │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      数据库层                               │
│  ┌──────────────┐         ┌──────────────────────┐         │
│  │   sessions   │         │ conversation_records │         │
│  │   (元数据)    │         │     (消息内容)        │         │
│  └──────────────┘         └──────────────────────┘         │
└─────────────────────────────────────────────────────────────┘
```

### 问题

1. **数据冗余**：消息同时存在于内存 `Messages` 数组和数据库 `conversation_records` 表
2. **内存泄漏风险**：长会话消息持续增长
3. **状态不一致**：内存与数据库可能不同步

## 目标架构

```
┌─────────────────────────────────────────────────────────────┐
│              运行时管理层 (pkg/session)                      │
│  ┌─────────────────┐                                        │
│  │ SessionManager  │  GetOrCreate() / CancelSession()       │
│  └─────────────────┘                                        │
│           │                                                 │
│           ▼                                                 │
│  ┌──────────────┐    cancel context.CancelFunc              │
│  │   Session    │    ctx    context.Context                 │
│  │   (运行时)    │    SetContext() / Cancel() / IsActive()   │
│  └──────────────┘                                           │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ GetHistory()
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      数据库层                               │
│  ┌──────────────┐         ┌──────────────────────┐         │
│  │   sessions   │         │ conversation_records │         │
│  │   (元数据)    │         │     (消息内容)        │         │
│  └──────────────┘         └──────────────────────┘         │
└─────────────────────────────────────────────────────────────┘
```

## 核心设计决策

### 决策 1：Session 结构体重定义

**决定**：移除 `Messages []Message`，仅保留运行时相关字段。

```go
type Session struct {
    Key       string    // 会话标识
    CreatedAt time.Time
    UpdatedAt time.Time

    // 运行时状态（不序列化）
    cancel context.CancelFunc
    ctx    context.Context
    mu     sync.RWMutex
}
```

**理由**：
- 明确职责边界
- 消除内存泄漏风险
- 为后续多实例部署做准备

### 决策 2：消息存储职责转移

**决定**：所有消息直接写入 `conversation_records` 表，不再经过内存缓存。

**当前流程**：
```
Agent Loop -> session.AddMessage() -> 内存缓存 -> Hook 写入数据库
```

**目标流程**：
```
Agent Loop -> ConversationRecordRepository.Save() -> 数据库
          -> Hook 记录（如有需要）
```

**理由**：
- 单一数据源
- 消除同步复杂性
- 服务重启不丢数据

### 决策 3：GetHistory 实现方式

**决定**：`GetHistory()` 保持现有实现，它已从数据库查询，但移除与内存缓存的合并逻辑。

当前实现（已符合目标）：
```go
func (m *Manager) GetHistory(ctx context.Context, sessionKey string, maxMessages int) []map[string]any {
    // 1. 从数据库查询 conversation_records
    records, err := m.convRepo.FindBySessionKey(...)
    // 2. 转换格式
    // 3. 返回（不再合并内存缓存）
}
```

### 决策 4：Context 管理保留

**决定**：保留 `SetContext()` / `Cancel()` / `IsActive()` 机制。

**理由**：
- `context.CancelFunc` 是运行时对象，无法持久化
- 取消正在进行的 AI 请求是核心功能
- 内存占用极小（仅几个指针）

## 变更明细

### 文件变更表

| 文件 | 变更类型 | 变更内容 |
|------|----------|----------|
| `pkg/session/manager.go` | 修改 | 移除 `Message` 类型，简化 `Session` 结构体 |
| `pkg/agent/loop.go` | 修改 | 替换 `AddMessage` 调用为直接写库 |
| `pkg/agent/interruptible.go` | 修改 | 如有 `AddMessage` 调用需替换 |
| `pkg/agent/provider/adapter.go` | 审查 | 检查是否有消息操作 |

### 代码变更详情

#### 1. pkg/session/manager.go

```go
// 移除整个 Message 类型
type Message struct {
    Role         string
    Content      string
    Timestamp    time.Time
    TraceID      string
    SpanID       string
    ParentSpanID string
}

// Session 结构体简化
type Session struct {
    Key       string
    CreatedAt time.Time
    UpdatedAt time.Time

    cancel context.CancelFunc
    ctx    context.Context
    mu     sync.RWMutex
}

// 移除方法
// - AddMessage()
// - AddMessageWithTrace()
// - Clear()
```

#### 2. 调用方改造示例

以 `pkg/agent/loop.go` 为例（假设）：

```go
// 改造前
session.AddMessageWithTrace(role, content, traceID, spanID, parentSpanID)

// 改造后
record := &models.ConversationRecord{
    SessionKey:   sessionKey,
    Role:         role,
    Content:      content,
    TraceID:      traceID,
    SpanID:       spanID,
    ParentSpanID: parentSpanID,
    Timestamp:    time.Now(),
}
convRepo.Save(ctx, record)
```

## 风险评估

| 风险 | 可能性 | 影响 | 缓解措施 |
|------|--------|------|----------|
| 调用方遗漏 | 中 | 高 | 全局搜索 `AddMessage`，逐个替换 |
| 性能下降 | 低 | 中 | `conversation_records` 已有索引 |
| 行为变更 | 低 | 高 | 保持 `GetHistory()` 接口不变 |

## 测试策略

1. **单元测试**：确保 `Session` 的 Context 管理功能正常
2. **集成测试**：验证消息写入和查询流程
3. **回归测试**：确认对话功能行为不变

## 实现顺序

1. 修改 `pkg/session/manager.go` 移除消息相关代码
2. 全局搜索并替换 `AddMessage` 调用
3. 运行测试验证
4. 编译验证
