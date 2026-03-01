# 002-SQLite观察器-设计.md

## 变更记录表

| 日期       | 版本 | 变更内容                           | 变更人 |
| ---------- | ---- | ---------------------------------- | ------ |
| 2026-03-01 | v1.0 | 初始设计文档创建                   | AI     |
| 2026-03-01 | v1.1 | 简化事件类型，与 SessionObserver 一致 | AI     |
| 2026-03-01 | v1.2 | 添加 Token Usage 字段              | AI     |
| 2026-03-01 | v1.3 | 添加 ToolCompleted 事件监听        | AI     |
| 2026-03-01 | v1.4 | 添加去重逻辑，移除 Data 字段       | AI     |

---

## 1. 概述

### 1.1 背景

当前系统已有观察者模式实现（`SessionObserver`），用于将消息存储到 JSON 文件中。为了提供更强大的查询能力和数据持久化，需要新增一个 SQLite 观察器，将消息事件存储到 SQLite 数据库中。

### 1.2 目标

- 实现一个 SQLite 观察器，监听关键事件类型
- 使用纯 Go 实现的 SQLite 驱动（`modernc.org/sqlite`），无需 CGO
- 提供结构化的数据存储，便于后续查询和分析
- 提取 role 和 content 字段作为独立列存储
- 记录 Token Usage 信息，便于统计分析
- 记录工具执行结果，便于追踪和调试
- 实现数据去重，避免重复记录

### 1.3 边界

- 本观察器处理以下事件：
  - `PromptSubmitted` - 用户输入
  - `LLMCallEnd` - AI 回复
  - `ToolCompleted` - 工具执行结果
- 数据库文件存储在配置的 dataDir 目录下
- role 字段支持：user、assistant、tool、tool_result 四种类型

---

## 2. 技术设计

### 2.1 数据库设计

#### 2.1.1 events 表

存储消息事件：

```sql
CREATE TABLE IF NOT EXISTS events (
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
    reasoning_tokens INTEGER DEFAULT 0,   -- 推理 token 数量 (o1 等模型)
    cached_tokens INTEGER DEFAULT 0,      -- 缓存 token 数量 (缓存命中)
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_events_event_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_session_key ON events(session_key);
CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
CREATE INDEX IF NOT EXISTS idx_events_trace_id ON events(trace_id);
CREATE INDEX IF NOT EXISTS idx_events_role ON events(role);
```

### 2.2 事件处理

| 事件类型 | role | content | Token Usage |
|---------|------|---------|-------------|
| PromptSubmitted | user | e.UserInput | 无 |
| LLMCallEnd (无工具调用) | assistant | e.ResponseContent | 有 |
| LLMCallEnd (有工具调用) | tool | 拼接工具调用信息 | 有 |
| ToolCompleted | tool_result | 工具名: 响应内容 | 无 |

### 2.3 去重逻辑

对于相同 traceID、role、content 的记录：
1. 优先保留有 TokenUsage 信息（total_tokens > 0）的记录
2. 如果都没有 TokenUsage 信息，保留 ID 最小的（最早插入的）

---

## 3. 已知限制

1. 数据库文件大小会随时间增长，需要后续实现清理机制
2. 未实现查询接口，可根据需求后续添加