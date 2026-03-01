# Hook 系统架构文档

## 概述

Hook 系统提供了一个全面的、只读的观察机制，用于追踪和记录 nanobot 处理过程中的各种事件。每个请求都会被分配一个唯一的 TraceID，该 ID 会贯穿整个请求生命周期，便于关联所有相关事件。

## 核心组件

### 1. Trace Context (`agent/hooks/trace/`)

负责生成和传播 TraceID，实现请求追踪。

```go
// 生成新的 TraceID
traceID := trace.NewTraceID()

// 注入到 context
ctx := trace.WithTraceID(ctx, traceID)

// 获取 TraceID
traceID := trace.GetTraceID(ctx)  // 不存在则自动生成
traceID := trace.MustGetTraceID(ctx)  // 不存在则返回空
```

### 2. Events (`agent/hooks/events/`)

定义了所有可能的 Hook 事件类型。

#### 事件类型分类

| 分类 | 事件类型 | 说明 |
|------|----------|------|
| 消息相关 | `message_received` | 收到消息 |
| | `message_sent` | 发送消息 |
| | `prompt_submitted` | 提交用户 prompt |
| | `system_prompt_built` | 生成系统 prompt |
| 工具相关 | `tool_used` | 使用工具 |
| | `tool_completed` | 工具执行完成 |
| | `tool_error` | 工具执行错误 |
| 技能相关 | `skill_lookup` | 查找技能 |
| | `skill_used` | 使用技能 |
| LLM 相关 | `llm_call_start` | LLM 调用开始 |
| | `llm_call_end` | LLM 调用结束 |
| | `llm_call_error` | LLM 调用错误 |
| 通用 | `component_start` | 组件开始执行 |
| | `component_end` | 组件执行完成 |
| | `component_error` | 组件执行错误 |

#### 事件结构

所有事件都继承自 `BaseEvent`：

```go
type BaseEvent struct {
    TraceID   string      `json:"trace_id"`   // 追踪 ID
    EventType EventType   `json:"event_type"` // 事件类型
    Timestamp time.Time   `json:"timestamp"`  // 时间戳
    Metadata  interface{} `json:"metadata,omitempty"` // 额外元数据
}
```

### 3. Observer (`agent/hooks/observer/`)

观察器接口定义，用于实现自定义的事件处理逻辑。

```go
type Observer interface {
    Name() string
    OnEvent(ctx context.Context, event *BaseEvent) error
    Enabled() bool
}
```

#### 过滤器

观察器可以通过过滤器筛选感兴趣的事件：

```go
type ObserverFilter struct {
    EventTypes []EventType  // 按事件类型过滤
    Channels   []string     // 按渠道过滤
    SessionKeys []string    // 按会话过滤
}
```

### 4. Dispatcher (`agent/hooks/dispatcher/`)

事件分发器，负责将事件分发给所有注册的观察器。

```go
disp := dispatcher.NewDispatcher(logger)
disp.Register(myObserver)
disp.Dispatch(ctx, event, channel, sessionKey)
```

### 5. Eino Bridge (`agent/hooks/eino/`)

将 Eino 框架的回调转换为 Hook 系统事件的桥接器。

```go
einoBridge := eino.NewEinoCallbackBridge(disp, logger)
handler := einoBridge.Handler()  // 可用于全局注册
```

### 6. Hook Manager (`agent/hooks/manager.go`)

统一的 Hook 系统管理器。

```go
hm := hooks.NewHookManager(logger, true)

// 注册观察器
hm.Register(myObserver)

// 分发事件
hm.Dispatch(ctx, event, channel, sessionKey)

// 获取 Eino Handler
handler := hm.EinoHandler()
```

## Hook 点分布

### 消息流程

```
收到消息 → 生成 TraceID → 处理消息 → 发送响应
    ↓            ↓           ↓         ↓
   Hook①       Context     Hook②    Hook③
```

1. **Loop.processMessage()** - 收到消息时
2. **Interruptible.processNormal()** - 提交 prompt 时
3. **Loop.processMessage()** - 发送响应时

### Eino Callback 集成

Eino 框架的回调通过 `EinoCallbackBridge` 自动转换为 Hook 事件：

- `OnStart` → `component_start` + 特定组件事件
- `OnEnd` → `component_end` + 特定组件事件
- `OnError` → `component_error` + 特定组件事件

## 内置观察器

### LoggingObserver

将事件记录到日志，结构化输出。

```go
obs := logging.NewLoggingObserver(logger, nil)
hm.Register(obs)
```

### JSONLogger

将事件以 JSON 格式输出。

```go
obs := logging.NewJSONLogger(logger, nil)
hm.Register(obs)
```

## 集成指南

### 1. 启用 Hook 系统

```go
// 创建 Hook Manager
hookManager := hooks.NewHookManager(logger, true)

// 注册默认观察器
loggingObs := logging.NewLoggingObserver(logger, nil)
hookManager.Register(loggingObs)

// 注册 Eino Handler
callbacks.AppendGlobalHandlers(hookManager.EinoHandler())
```

### 2. 在代码中触发事件

```go
// 收到消息时
traceID := trace.GetTraceID(ctx)
event := events.NewMessageReceivedEvent(traceID, msg)
hookManager.Dispatch(ctx, event.BaseEvent, msg.Channel, msg.SessionKey())

// 发送消息时
event := events.NewMessageSentEvent(traceID, outMsg, sessionKey)
hookManager.Dispatch(ctx, event.BaseEvent, channel, sessionKey)
```

### 3. 实现自定义观察器

```go
type MyObserver struct {
    *observer.BaseObserver
    logger *zap.Logger
}

func (o *MyObserver) OnEvent(ctx context.Context, event *events.BaseEvent) error {
    switch event.EventType {
    case events.EventMessageReceived:
        // 处理收到消息事件
    case events.EventToolUsed:
        // 处理工具使用事件
    }
    return nil
}

func (o *MyObserver) Enabled() bool {
    return true
}

func (o *MyObserver) Name() string {
    return "my_observer"
}
```

## 追踪流程示例

```
用户消息 → TraceID: abc-123-def
    ├─ [Hook] message_received
    ├─ [Eino] component_start (ChatModel)
    ├─ [Hook] prompt_submitted
    ├─ [Eino] llm_call_start
    ├─ [Eino] tool_used (read_file)
    ├─ [Eino] tool_completed (read_file)
    ├─ [Eino] llm_call_end
    ├─ [Eino] component_end (ChatModel)
    └─ [Hook] message_sent
```

所有事件都包含相同的 TraceID，可以轻松关联整个处理流程。

## 设计原则

1. **只读观察**：Hook 系统只用于观察和记录，不修改原始数据
2. **非阻塞**：事件分发是异步的，不会影响主流程性能
3. **可扩展**：通过实现 Observer 接口，可以轻松添加新的事件处理逻辑
4. **完整追踪**：TraceID 贯穿整个请求生命周期，实现端到端追踪

## 性能考虑

- 事件分发采用异步方式，避免阻塞主流程
- 观察器可以独立开关，不影响其他组件
- 过滤器机制可以减少不必要的事件处理
- 日志级别控制可以减少日志输出

## 后续扩展

1. **持久化存储**：将事件存储到数据库或文件
2. **实时监控**：集成 Prometheus/Grafana 监控
3. **调试工具**：提供交互式调试界面
4. **性能分析**：基于事件数据的性能分析工具