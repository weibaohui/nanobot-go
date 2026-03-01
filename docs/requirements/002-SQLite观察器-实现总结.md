# 002-SQLite观察器-实现总结.md

## 变更记录表

| 日期       | 版本 | 变更内容                           | 变更人 |
| ---------- | ---- | ---------------------------------- | ------ |
| 2026-03-01 | v1.0 | 初始实现总结创建                   | AI     |
| 2026-03-01 | v1.1 | 简化事件类型，添加 role/content 列 | AI     |
| 2026-03-01 | v1.2 | 添加 Token Usage 字段              | AI     |
| 2026-03-01 | v1.3 | 添加 ToolCompleted 事件监听        | AI     |

---

## 1. 实现概述

成功实现了一个 SQLite 观察器，将消息事件存储到 SQLite 数据库中，支持用户消息、AI 回复和工具执行结果的存储。

## 2. 实现内容

### 2.1 核心文件

| 文件 | 说明 |
|------|------|
| `agent/hooks/observers/sqlite.go` | SQLite 观察器实现 |
| `agent/hooks/observers/sqlite_test.go` | 单元测试 |
| `docs/design/002-SQLite观察器-设计.md` | 设计文档 |

### 2.2 依赖变更

- 新增依赖：`modernc.org/sqlite v1.46.1`（纯 Go SQLite 驱动）

### 2.3 数据库表结构

```sql
CREATE TABLE events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    trace_id TEXT NOT NULL,
    span_id TEXT,
    parent_span_id TEXT,
    event_type TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    session_key TEXT,
    role TEXT,               -- user / assistant / tool / tool_result
    content TEXT,            -- 消息内容
    prompt_tokens INTEGER DEFAULT 0,      -- 输入 token 数量
    completion_tokens INTEGER DEFAULT 0,  -- 输出 token 数量
    total_tokens INTEGER DEFAULT 0,       -- 总 token 数量
    reasoning_tokens INTEGER DEFAULT 0,   -- 推理 token 数量
    cached_tokens INTEGER DEFAULT 0,      -- 缓存 token 数量
    data TEXT,               -- JSON 格式完整数据
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## 3. 关键实现点

### 3.1 事件处理范围

处理以下事件类型：
- `PromptSubmitted` - 用户输入，role=user
- `LLMCallEnd` - AI 回复，role=assistant 或 tool
- `ToolCompleted` - 工具执行结果，role=tool_result

### 3.2 Role 和 Content 提取

| 事件类型 | role | content |
|---------|------|---------|
| PromptSubmitted | user | e.UserInput |
| LLMCallEnd (无工具调用) | assistant | e.ResponseContent |
| LLMCallEnd (有工具调用) | tool | 拼接工具调用信息 |
| ToolCompleted | tool_result | 工具名: 响应内容 |

### 3.3 Token Usage 提取

参考 TokenUsageObserver 实现，从 LLMCallEnd 事件中提取：
- `prompt_tokens`: 输入 token 数量
- `completion_tokens`: 输出 token 数量
- `total_tokens`: 总 token 数量
- `reasoning_tokens`: 推理 token 数量（如 o1 模型）
- `cached_tokens`: 缓存命中的 token 数量

### 3.4 并发控制

- 使用 `sync.RWMutex` 保护数据库写入
- SQLite 连接池设置为单连接（推荐配置）

## 4. 测试结果

```
=== RUN   TestSQLiteObserver_OnEvent_PromptSubmitted
--- PASS: TestSQLiteObserver_OnEvent_PromptSubmitted (0.01s)
=== RUN   TestSQLiteObserver_OnEvent_LLMCallEnd
--- PASS: TestSQLiteObserver_OnEvent_LLMCallEnd (0.01s)
=== RUN   TestSQLiteObserver_OnEvent_LLMCallEnd_WithToolCalls
--- PASS: TestSQLiteObserver_OnEvent_LLMCallEnd_WithToolCalls (0.01s)
=== RUN   TestSQLiteObserver_OnEvent_LLMCallEnd_WithTokenUsage
--- PASS: TestSQLiteObserver_OnEvent_LLMCallEnd_WithTokenUsage (0.01s)
=== RUN   TestSQLiteObserver_OnEvent_ToolCompleted
--- PASS: TestSQLiteObserver_OnEvent_ToolCompleted (0.01s)
=== RUN   TestSQLiteObserver_OnEvent_IgnoredEvents
--- PASS: TestSQLiteObserver_OnEvent_IgnoredEvents (0.00s)
=== RUN   TestSQLiteObserver_Filter
--- PASS: TestSQLiteObserver_Filter (0.00s)
=== RUN   TestSQLiteObserver_ConcurrentWrites
--- PASS: TestSQLiteObserver_ConcurrentWrites (0.01s)
PASS
```

## 5. 已知限制

1. 数据库文件会随时间增长，暂未实现清理机制
2. 未实现查询 API，可根据需求后续添加

## 6. 后续改进建议

1. 实现数据清理机制（TTL 或定期清理）
2. 实现查询 API 供其他模块使用
3. 实现 Token Usage 统计查询