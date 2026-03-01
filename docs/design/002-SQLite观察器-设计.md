# 002-SQLite观察器-设计.md

## 变更记录表

| 日期       | 版本 | 变更内容                           | 变更人 |
| ---------- | ---- | ---------------------------------- | ------ |
| 2026-03-01 | v1.0 | 初始设计文档创建                   | AI     |
| 2026-03-01 | v1.1 | 简化事件类型，与 SessionObserver 一致 | AI     |

---

## 1. 概述

### 1.1 背景

当前系统已有观察者模式实现（`SessionObserver`），用于将消息存储到 JSON 文件中。为了提供更强大的查询能力和数据持久化，需要新增一个 SQLite 观察器，将消息事件存储到 SQLite 数据库中。

### 1.2 目标

- 实现一个 SQLite 观察器，监听与 SessionObserver 相同的事件类型
- 使用纯 Go 实现的 SQLite 驱动（`modernc.org/sqlite`），无需 CGO
- 提供结构化的数据存储，便于后续查询和分析
- 提取 role 和 content 字段作为独立列存储

### 1.3 边界

- 本观察器只处理 `PromptSubmitted` 和 `LLMCallEnd` 事件，与 SessionObserver 保持一致
- 数据库文件存储在配置的 dataDir 目录下
- role 字段支持：user、assistant、tool 三种类型

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
    role TEXT,           -- user / assistant / tool
    content TEXT,        -- 消息内容
    data TEXT,           -- JSON 格式存储完整事件数据
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_events_event_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_session_key ON events(session_key);
CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
CREATE INDEX IF NOT EXISTS idx_events_trace_id ON events(trace_id);
CREATE INDEX IF NOT EXISTS idx_events_role ON events(role);
```

### 2.2 事件处理

| 事件类型 | role | content 来源 |
|---------|------|-------------|
| PromptSubmitted | user | e.UserInput |
| LLMCallEnd (无工具调用) | assistant | e.ResponseContent |
| LLMCallEnd (有工具调用) | tool | 拼接工具调用信息 |

### 2.3 核心接口设计

```go
// SQLiteObserver SQLite 观察器
type SQLiteObserver struct {
    *observer.BaseObserver
    db     *sql.DB
    dbPath string
    logger *zap.Logger
    mu     sync.RWMutex
}

// OnEvent 处理事件（只处理 PromptSubmitted 和 LLMCallEnd）
func (o *SQLiteObserver) OnEvent(ctx context.Context, event events.Event) error
```

---

## 3. 依赖

### 3.1 新增依赖

- `modernc.org/sqlite` - 纯 Go 实现的 SQLite 驱动

---

## 4. 已知限制

1. 数据库文件大小会随时间增长，需要后续实现清理机制
2. 未实现查询接口，可根据需求后续添加