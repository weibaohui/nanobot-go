# Agent 管理系统设计文档

## 变更记录表

| 版本 | 日期       | 变更内容           | 作者   |
|------|------------|--------------------|--------|
| v1.0 | 2026-03-10 | 初始版本                      | Claude |
| v1.1 | 2026-03-11 | 增加LLM提供商表、Cron任务表、Agent模型选择模式 | Claude |

---

## 1. 设计目标

将系统从单 Agent 架构升级为多租户、多 Agent、多 Channel 的数据库驱动架构。

## 2. 架构设计

### 2.1 核心实体关系

```
┌─────────────┐       ┌─────────────┐       ┌─────────────┐
│    User     │ 1:N   │    Agent    │ 1:N   │   Channel   │
│   (用户)     │──────▶│   (代理)     │◀──────│   (渠道)     │
└──────┬──────┘       └─────────────┘       └──────┬──────┘
       │                     │                      │
       │                     │ 1:N                  │ 1:N
       │                     ▼                      ▼
       │              ┌─────────────┐       ┌─────────────┐
       │              │Conversation │       │   Session   │
       │              │  (对话记录)  │       │   (会话)     │
       │              └─────────────┘       └─────────────┘
       │
       │ 1:N              1:N
       ▼                  ▼
┌─────────────┐     ┌─────────────┐
│ LLMProvider │     │   CronJob   │
│ (LLM提供商)  │     │  (定时任务)  │
└─────────────┘     └─────────────┘
```

### 2.2 实体关系说明

- **User (1) : Agent (N)** - 一个用户可以拥有多个 Agent
- **User (1) : LLMProvider (N)** - 一个用户可以配置多个 LLM 提供商
- **User (1) : CronJob (N)** - 一个用户可以创建多个定时任务
- **Agent (N) : Channel (M)** - 一个 Agent 可绑定多个 Channel，一个 Channel 归属一个 Agent（当前阶段先实现 1:1）
- **Agent (1) : Conversation (N)** - 一个 Agent 有多条对话记录
- **Channel (1) : Session (N)** - 一个 Channel 有多个活跃会话
- **Channel (1) : CronJob (N)** - 一个 Channel 可以有多个定时任务（来源渠道）

## 3. 数据库设计

### 3.1 用户表 (users)

```sql
CREATE TABLE users (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    username        TEXT NOT NULL UNIQUE,           -- 用户名
    email           TEXT UNIQUE,                     -- 邮箱
    password_hash   TEXT,                            -- 密码哈希
    display_name    TEXT,                            -- 显示名称
    is_active       BOOLEAN DEFAULT 1,               -- 是否激活
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
```

**Go Model:**
```go
type User struct {
    ID           uint      `gorm:"primarykey" json:"id"`
    Username     string    `gorm:"type:text;not null;uniqueIndex" json:"username"`
    Email        string    `gorm:"type:text;uniqueIndex" json:"email"`
    PasswordHash string    `gorm:"type:text" json:"-"` // 不序列化
    DisplayName  string    `gorm:"type:text" json:"display_name"`
    IsActive     bool      `gorm:"default:true" json:"is_active"`
    CreatedAt    time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`
    UpdatedAt    time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"updated_at"`
}
```

### 3.2 Agent 表 (agents)

```sql
CREATE TABLE agents (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL,               -- 所属用户
    name            TEXT NOT NULL,                   -- Agent 名称
    description     TEXT,                            -- 描述

    -- Markdown 配置内容（每个文档对应一个列）
    identity_content   TEXT,                         -- IDENTITY.md - Agent 身份信息（名称、头像、气质等）
    soul_content       TEXT,                         -- SOUL.md - Agent 灵魂/个性定义
    agents_content     TEXT,                         -- AGENTS.md - Agent 指令配置
    user_content       TEXT,                         -- USER.md - 用户信息（主人偏好）
    tools_content      TEXT,                         -- TOOLS.md - 工具本地备注/环境配置

    -- 长期记忆
    memory_content     TEXT,                         -- MEMORY.md - 长期记忆内容
    memory_summary     TEXT,                         -- 记忆摘要

    -- 能力配置
    skills_list        TEXT,                         -- 可用技能列表，JSON 数组
    tools_list         TEXT,                         -- 可用工具列表，JSON 数组

    -- 模型配置
    model_selection_mode TEXT DEFAULT 'auto',       -- 模型选择模式: 'auto'(自动选择) / 'specific'(指定模型)
    model_id        TEXT,                           -- 模型ID (如: 'claude-opus-4', 'gpt-4')
    model_name      TEXT,                           -- 模型显示名称 (如: 'Claude Opus 4')
    max_tokens      INTEGER DEFAULT 4096,
    temperature     REAL DEFAULT 0.7,
    max_iterations  INTEGER DEFAULT 15,

    is_active       BOOLEAN DEFAULT 1,
    is_default      BOOLEAN DEFAULT 0,               -- 是否默认 Agent
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_agents_user_id ON agents(user_id);
CREATE INDEX idx_agents_name ON agents(name);
CREATE INDEX idx_agents_active ON agents(is_active);
```

**Go Model:**
```go
type Agent struct {
    ID             uint      `gorm:"primarykey" json:"id"`
    UserID         uint      `gorm:"not null;index" json:"user_id"`
    Name           string    `gorm:"type:text;not null" json:"name"`
    Description    string    `gorm:"type:text" json:"description"`

    // Markdown 配置内容（每个文档对应一个列）
    IdentityContent  string    `gorm:"type:text" json:"identity_content"`   // IDENTITY.md - Agent 身份信息（名称、头像、气质等）
    SoulContent      string    `gorm:"type:text" json:"soul_content"`       // SOUL.md - Agent 灵魂/个性定义
    AgentsContent    string    `gorm:"type:text" json:"agents_content"`     // AGENTS.md - Agent 指令配置
    UserContent      string    `gorm:"type:text" json:"user_content"`       // USER.md - 用户信息（主人偏好）
    ToolsContent     string    `gorm:"type:text" json:"tools_content"`      // TOOLS.md - 工具本地备注/环境配置

    // 长期记忆
    MemoryContent    string    `gorm:"type:text" json:"memory_content"`     // MEMORY.md - 长期记忆内容
    MemorySummary    string    `gorm:"type:text" json:"memory_summary"`     // 记忆摘要

    // 能力配置（技能列表和工具列表）
    SkillsList       string    `gorm:"type:text" json:"skills_list"`        // JSON 数组，如 ["cron", "github", "weather"]
    ToolsList        string    `gorm:"type:text" json:"tools_list"`         // JSON 数组，如 ["read_file", "write_file", "exec"]

    // 模型配置
    ModelSelectionMode string  `gorm:"type:text;default:'auto'" json:"model_selection_mode"` // 'auto'(自动选择) / 'specific'(指定模型)
    ModelID            string  `gorm:"type:text" json:"model_id"`                           // 模型ID
    ModelName          string  `gorm:"type:text" json:"model_name"`                         // 模型显示名称
    MaxTokens          int     `gorm:"default:4096" json:"max_tokens"`
    Temperature        float64 `gorm:"default:0.7" json:"temperature"`
    MaxIterations      int     `gorm:"default:15" json:"max_iterations"`

    IsActive       bool      `gorm:"default:true" json:"is_active"`
    IsDefault      bool      `gorm:"default:false" json:"is_default"`
    CreatedAt      time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`
    UpdatedAt      time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"updated_at"`

    // 关联
    User           User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
```

### 3.3 Channel 表 (channels)

```sql
CREATE TABLE channels (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL,               -- 所属用户
    agent_id        INTEGER,                        -- 绑定的 Agent（可为空）

    name            TEXT NOT NULL,                   -- Channel 名称
    type            TEXT NOT NULL,                   -- 类型: feishu, dingtalk, matrix, websocket

    -- 通用配置
    is_active       BOOLEAN DEFAULT 1,
    allow_from      TEXT,                           -- 允许的用户白名单，JSON 数组

    -- 类型特定配置（JSON 存储）
    config          TEXT,                           -- 渠道特定配置 JSON

    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE SET NULL
);

CREATE INDEX idx_channels_user_id ON channels(user_id);
CREATE INDEX idx_channels_agent_id ON channels(agent_id);
CREATE INDEX idx_channels_type ON channels(type);
CREATE INDEX idx_channels_active ON channels(is_active);
```

**Go Model:**
```go
type ChannelType string

const (
    ChannelTypeFeishu    ChannelType = "feishu"
    ChannelTypeDingTalk  ChannelType = "dingtalk"
    ChannelTypeMatrix    ChannelType = "matrix"
    ChannelTypeWebSocket ChannelType = "websocket"
)

type Channel struct {
    ID        uint        `gorm:"primarykey" json:"id"`
    UserID    uint        `gorm:"not null;index" json:"user_id"`
    AgentID   *uint       `gorm:"index" json:"agent_id"`  // 可为空

    Name      string      `gorm:"type:text;not null" json:"name"`
    Type      ChannelType `gorm:"type:text;not null" json:"type"`

    IsActive  bool        `gorm:"default:true" json:"is_active"`
    AllowFrom string      `gorm:"type:text" json:"allow_from"`  // JSON 数组

    Config    string      `gorm:"type:text" json:"config"`      // JSON 配置

    CreatedAt time.Time   `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`
    UpdatedAt time.Time   `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"updated_at"`

    // 关联
    User      User        `gorm:"foreignKey:UserID" json:"user,omitempty"`
    Agent     *Agent      `gorm:"foreignKey:AgentID" json:"agent,omitempty"`
}

// ChannelConfig 渠道配置接口
type ChannelConfig interface {
    Validate() error
}

// FeishuConfig 飞书配置
type FeishuChannelConfig struct {
    AppID             string   `json:"app_id"`
    AppSecret         string   `json:"app_secret"`
    EncryptKey        string   `json:"encrypt_key,omitempty"`
    VerificationToken string   `json:"verification_token,omitempty"`
}
```

### 3.4 扩展现有对话记录表

**修改后的 conversation_records 表：**

```sql
-- 添加新列
ALTER TABLE conversation_records ADD COLUMN user_id INTEGER;
ALTER TABLE conversation_records ADD COLUMN agent_id INTEGER;
ALTER TABLE conversation_records ADD COLUMN channel_id INTEGER;
ALTER TABLE conversation_records ADD COLUMN channel_type TEXT;

-- 创建新索引
CREATE INDEX idx_conv_records_user_id ON conversation_records(user_id);
CREATE INDEX idx_conv_records_agent_id ON conversation_records(agent_id);
CREATE INDEX idx_conv_records_channel_id ON conversation_records(channel_id);
```

**修改后的 Go Model：**
```go
type ConversationRecord struct {
    ID               uint      `gorm:"primarykey" json:"id"`
    TraceID          string    `gorm:"type:text;index;not null" json:"trace_id"`
    SpanID           string    `gorm:"type:text" json:"span_id,omitempty"`
    ParentSpanID     string    `gorm:"type:text" json:"parent_span_id,omitempty"`
    EventType        string    `gorm:"type:text;index;not null" json:"event_type"`
    Timestamp        time.Time `gorm:"type:datetime;index;not null" json:"timestamp"`
    SessionKey       string    `gorm:"type:text;index" json:"session_key"`
    Role             string    `gorm:"type:text;index" json:"role"`
    Content          string    `gorm:"type:text" json:"content"`
    PromptTokens     int       `gorm:"type:integer;default:0" json:"prompt_tokens"`
    CompletionTokens int       `gorm:"type:integer;default:0" json:"completion_tokens"`
    TotalTokens      int       `gorm:"type:integer;default:0" json:"total_tokens"`
    ReasoningTokens  int       `gorm:"type:integer;default:0" json:"reasoning_tokens"`
    CachedTokens     int       `gorm:"type:integer;default:0" json:"cached_tokens"`
    CreatedAt        time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`

    // 新增字段：归属信息
    UserID      *uint      `gorm:"index" json:"user_id,omitempty"`
    AgentID     *uint      `gorm:"index" json:"agent_id,omitempty"`
    ChannelID   *uint      `gorm:"index" json:"channel_id,omitempty"`
    ChannelType string     `gorm:"type:text;index" json:"channel_type,omitempty"`
}
```

### 3.5 会话表 (sessions)

用于跟踪活跃的 Channel 会话：

```sql
CREATE TABLE sessions (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL,
    agent_id        INTEGER,
    channel_id      INTEGER NOT NULL,

    session_key     TEXT NOT NULL UNIQUE,           -- 会话唯一标识
    external_id     TEXT,                           -- 外部系统的会话标识

    last_active_at  DATETIME,                       -- 最后活跃时间
    metadata        TEXT,                           -- 会话元数据 JSON

    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE SET NULL,
    FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_agent_id ON sessions(agent_id);
CREATE INDEX idx_sessions_channel_id ON sessions(channel_id);
CREATE INDEX idx_sessions_session_key ON sessions(session_key);
CREATE INDEX idx_sessions_last_active ON sessions(last_active_at);
```

**Go Model:**
```go
type Session struct {
    ID           uint      `gorm:"primarykey" json:"id"`
    UserID       uint      `gorm:"not null;index" json:"user_id"`
    AgentID      *uint     `gorm:"index" json:"agent_id"`
    ChannelID    uint      `gorm:"not null;index" json:"channel_id"`

    SessionKey   string    `gorm:"type:text;not null;uniqueIndex" json:"session_key"`
    ExternalID   string    `gorm:"type:text" json:"external_id"`

    LastActiveAt *time.Time `gorm:"type:datetime" json:"last_active_at"`
    Metadata     string    `gorm:"type:text" json:"metadata"`

    CreatedAt    time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`

    // 关联
    User         User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
    Agent        *Agent     `gorm:"foreignKey:AgentID" json:"agent,omitempty"`
    Channel      Channel   `gorm:"foreignKey:ChannelID" json:"channel,omitempty"`
}
```

### 3.6 LLM 提供商表 (llm_providers)

用于存储用户的 LLM API 密钥和配置：

```sql
CREATE TABLE llm_providers (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL,               -- 所属用户

    provider_key    TEXT NOT NULL,                  -- 提供商标识 (如: 'anthropic', 'openai', 'siliconflow')
    provider_name   TEXT,                           -- 提供商显示名称

    -- API 配置
    api_key         TEXT,                           -- API 密钥（加密存储）
    api_base        TEXT,                           -- API 基础地址
    extra_headers   TEXT,                           -- 额外请求头 JSON

    -- 支持的模型列表
    supported_models TEXT,                          -- 支持的模型JSON数组 [{"id": "xxx", "name": "xxx"}]

    -- 默认配置
    default_model   TEXT,                           -- 默认模型ID
    is_default      BOOLEAN DEFAULT 0,              -- 是否为默认提供商
    priority        INTEGER DEFAULT 0,              -- 优先级 (数值越大优先级越高，auto模式下使用)
    auto_merge      BOOLEAN DEFAULT 1,              -- 是否自动合并从API获取的模型列表

    is_active       BOOLEAN DEFAULT 1,              -- 是否启用
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_llm_providers_user_id ON llm_providers(user_id);
CREATE INDEX idx_llm_providers_provider_key ON llm_providers(provider_key);
CREATE INDEX idx_llm_providers_active ON llm_providers(is_active);
CREATE INDEX idx_llm_providers_default ON llm_providers(user_id, is_default);
```

**Go Model:**
```go
type LLMProvider struct {
    ID              uint      `gorm:"primarykey" json:"id"`
    UserID          uint      `gorm:"not null;index" json:"user_id"`

    ProviderKey     string    `gorm:"type:text;not null" json:"provider_key"`   // 如: 'anthropic', 'openai'
    ProviderName    string    `gorm:"type:text" json:"provider_name"`           // 如: 'Anthropic', 'OpenAI'

    // API 配置
    APIKey          string    `gorm:"type:text" json:"-"`                       // API密钥，不序列化到JSON
    APIBase         string    `gorm:"type:text" json:"api_base,omitempty"`      // API基础地址
    ExtraHeaders    string    `gorm:"type:text" json:"extra_headers,omitempty"` // JSON格式额外请求头

    // 支持的模型
    SupportedModels string    `gorm:"type:text" json:"supported_models,omitempty"` // JSON数组

    // 默认配置
    DefaultModel    string    `gorm:"type:text" json:"default_model,omitempty"`  // 默认模型ID
    IsDefault       bool      `gorm:"default:false" json:"is_default"`
    Priority        int       `gorm:"default:0" json:"priority"`
    AutoMerge       bool      `gorm:"default:true" json:"auto_merge"`            // 是否自动合并从API获取的模型列表

    IsActive        bool      `gorm:"default:true" json:"is_active"`
    CreatedAt       time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`
    UpdatedAt       time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"updated_at"`

    // 关联
    User            User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// ModelInfo 模型信息结构
type ModelInfo struct {
    ID       string `json:"id"`       // 模型ID
    Name     string `json:"name"`     // 模型名称
    MaxTokens int   `json:"max_tokens,omitempty"`
}
```

### 3.7 Cron 任务表 (cron_jobs)

用于存储用户订阅的定时任务：

```sql
CREATE TABLE cron_jobs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL,               -- 所属用户
    channel_id      INTEGER NOT NULL,               -- 来源渠道

    -- 任务标识
    name            TEXT NOT NULL,                  -- 任务名称
    description     TEXT,                           -- 任务描述

    -- Cron 表达式
    cron_expression TEXT NOT NULL,                  -- Cron 表达式 (如: '0 9 * * 1-5')
    timezone        TEXT DEFAULT 'Asia/Shanghai',   -- 时区

    -- 执行配置
    prompt          TEXT NOT NULL,                  -- 执行时发送给 LLM 的提示词

    -- 模型配置 (同 Agent 模型配置)
    model_selection_mode TEXT DEFAULT 'auto',       -- 'auto'(自动选择) / 'specific'(指定模型)
    model_id        TEXT,                           -- 指定模型的ID
    model_name      TEXT,                           -- 指定模型的名称

    -- 目标配置
    target_channel_id INTEGER,                      -- 输出到哪个渠道 (可为空，表示原渠道)
    target_user_id  TEXT,                           -- 目标用户ID (用于定向推送)

    -- 状态
    is_active       BOOLEAN DEFAULT 1,              -- 是否启用
    last_run_at     DATETIME,                       -- 最后执行时间
    last_run_status TEXT,                           -- 最后执行状态 (success/failed)
    last_run_result TEXT,                           -- 最后执行结果
    next_run_at     DATETIME,                       -- 下次执行时间
    run_count       INTEGER DEFAULT 0,              -- 执行次数
    fail_count      INTEGER DEFAULT 0,              -- 失败次数

    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE,
    FOREIGN KEY (target_channel_id) REFERENCES channels(id) ON DELETE SET NULL
);

CREATE INDEX idx_cron_jobs_user_id ON cron_jobs(user_id);
CREATE INDEX idx_cron_jobs_channel_id ON cron_jobs(channel_id);
CREATE INDEX idx_cron_jobs_active ON cron_jobs(is_active);
CREATE INDEX idx_cron_jobs_next_run ON cron_jobs(next_run_at);
```

**Go Model:**
```go
type CronJob struct {
    ID          uint      `gorm:"primarykey" json:"id"`
    UserID      uint      `gorm:"not null;index" json:"user_id"`
    ChannelID   uint      `gorm:"not null;index" json:"channel_id"`

    // 任务标识
    Name        string    `gorm:"type:text;not null" json:"name"`
    Description string    `gorm:"type:text" json:"description,omitempty"`

    // Cron 表达式
    CronExpression string `gorm:"type:text;not null" json:"cron_expression"`
    Timezone       string `gorm:"type:text;default:'Asia/Shanghai'" json:"timezone"`

    // 执行配置
    Prompt      string    `gorm:"type:text;not null" json:"prompt"`

    // 模型配置
    ModelSelectionMode string `gorm:"type:text;default:'auto'" json:"model_selection_mode"` // 'auto' / 'specific'
    ModelID            string `gorm:"type:text" json:"model_id,omitempty"`
    ModelName          string `gorm:"type:text" json:"model_name,omitempty"`

    // 目标配置
    TargetChannelID *uint     `gorm:"index" json:"target_channel_id,omitempty"`
    TargetUserID    string    `gorm:"type:text" json:"target_user_id,omitempty"`

    // 状态
    IsActive       bool       `gorm:"default:true" json:"is_active"`
    LastRunAt      *time.Time `gorm:"type:datetime" json:"last_run_at,omitempty"`
    LastRunStatus  string     `gorm:"type:text" json:"last_run_status,omitempty"`
    LastRunResult  string     `gorm:"type:text" json:"last_run_result,omitempty"`
    NextRunAt      *time.Time `gorm:"type:datetime;index" json:"next_run_at,omitempty"`
    RunCount       int        `gorm:"default:0" json:"run_count"`
    FailCount      int        `gorm:"default:0" json:"fail_count"`

    CreatedAt time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`
    UpdatedAt time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"updated_at"`

    // 关联
    User           User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
    Channel        Channel  `gorm:"foreignKey:ChannelID" json:"channel,omitempty"`
    TargetChannel  *Channel `gorm:"foreignKey:TargetChannelID" json:"target_channel,omitempty"`
}

// CronJobStatus 任务执行状态
const (
    CronJobStatusSuccess = "success"
    CronJobStatusFailed  = "failed"
    CronJobStatusRunning = "running"
)

// ModelSelectionMode 模型选择模式
const (
    ModelSelectionModeAuto      = "auto"      // 自动选择模型
    ModelSelectionModeSpecific  = "specific"  // 指定模型
)
```

## 4. 设计变更说明

### 4.1 去掉工作区目录

**原因：**
1. SOUL.md、AGENTS.md、USER.md 已存储在数据库中
2. HEARTBEAT.md 随心跳功能一起移除
3. MEMORY.md 改为数据库存储（见下方扩展）
4. 文件操作工具使用可配置的根目录或系统临时目录

**新结构：**
```
~/.nanobot/
├── config.json          # 全局配置（简化）
├── data/                # 数据库文件目录
│   └── events.db
└── skills/              # 技能文件（全局共享）
    └── {skill-name}/
        └── SKILL.md
```

### 4.2 Agent 配置文档说明

每个 Agent 拥有 6 个 Markdown 配置文件，全部存储在数据库中：

| 文档 | 数据库列 | 用途说明 |
|------|----------|----------|
| `IDENTITY.md` | `identity_content` | **Agent 身份信息** - 名称、头像、气质、表情符号等 |
| `SOUL.md` | `soul_content` | **Agent 灵魂/个性** - 核心人格、价值观、行为准则 |
| `AGENTS.md` | `agents_content` | **Agent 指令配置** - 系统提示词、工作指南 |
| `USER.md` | `user_content` | **用户信息** - 主人偏好、用户画像 |
| `TOOLS.md` | `tools_content` | **工具本地备注** - 环境特定的工具配置（SSH 主机、设备别名等） |
| `MEMORY.md` | `memory_content` | **长期记忆** - 跨会话的持久化记忆 |

**HEARTBEAT.md** - 已移除（心跳功能已删除）

### 4.3 长期记忆存储扩展

**扩展 Agent 表，添加记忆字段：**

```sql
ALTER TABLE agents ADD COLUMN memory_content TEXT;      -- MEMORY.md 内容
ALTER TABLE agents ADD COLUMN memory_summary TEXT;      -- 记忆摘要
ALTER TABLE agents ADD COLUMN memory_vector BLOB;       -- 向量表示（可选）
```

**Go Model 扩展：**
```go
type Agent struct {
    // ... 其他字段 ...

    // 长期记忆
    MemoryContent string `gorm:"type:text" json:"memory_content"`     // 完整记忆内容
    MemorySummary string `gorm:"type:text" json:"memory_summary"`     // 记忆摘要
    MemoryVector  []byte `gorm:"type:blob" json:"memory_vector,omitempty"` // 向量
}
```

### 4.3 文件操作工具适配

**方案：** 使用可配置的文件根目录

```go
// FileConfig 文件操作配置
type FileConfig struct {
    BaseDir string `json:"base_dir"`  // 文件操作根目录，默认为系统临时目录
}
```

- 如果不配置，使用系统临时目录（`os.TempDir()`）
- 如果配置，使用指定目录（如 `/data/files/{user_id}/`）

### 4.4 Agent 能力配置示例

**Agent A - 代码助手：**
```json
{
  "skills_list": "[\"github\", \"skill-creator\", \"tmux\"]",
  "tools_list": "[\"read_file\", \"write_file\", \"edit_file\", \"exec\", \"list_dir\"]"
}
```

**Agent B - 日常助手：**
```json
{
  "skills_list": "[\"weather\", \"cron\", \"summarize\"]",
  "tools_list": "[\"web_search\", \"web_fetch\", \"message\"]"
}
```

**特殊值：**
- `skills_list`: `"["*"]"` 或 `null` → 允许使用所有技能
- `tools_list`: `"["*"]"` 或 `null` → 允许使用所有工具
- `[]` → 禁用所有技能/工具

## 5. 核心接口设计

### 5.1 Repository 接口

```go
// UserRepository 用户仓库接口
type UserRepository interface {
    Create(user *User) error
    GetByID(id uint) (*User, error)
    GetByUsername(username string) (*User, error)
    Update(user *User) error
    Delete(id uint) error
}

// AgentRepository Agent 仓库接口
type AgentRepository interface {
    Create(agent *Agent) error
    GetByID(id uint) (*Agent, error)
    GetByUserID(userID uint) ([]Agent, error)
    GetDefaultByUserID(userID uint) (*Agent, error)
    Update(agent *Agent) error
    Delete(id uint) error
}

// ChannelRepository Channel 仓库接口
type ChannelRepository interface {
    Create(channel *Channel) error
    GetByID(id uint) (*Channel, error)
    GetByUserID(userID uint) ([]Channel, error)
    GetByAgentID(agentID uint) ([]Channel, error)
    Update(channel *Channel) error
    Delete(id uint) error
}

// SessionRepository 会话仓库接口
type SessionRepository interface {
    Create(session *Session) error
    GetBySessionKey(key string) (*Session, error)
    GetActiveByChannel(channelID uint) ([]Session, error)
    UpdateLastActive(sessionKey string) error
    Delete(sessionKey string) error
}

// LLMProviderRepository LLM提供商仓库接口
type LLMProviderRepository interface {
    Create(provider *LLMProvider) error
    GetByID(id uint) (*LLMProvider, error)
    GetByUserID(userID uint) ([]LLMProvider, error)
    GetDefaultByUserID(userID uint) (*LLMProvider, error)
    GetByProviderKey(userID uint, providerKey string) (*LLMProvider, error)
    Update(provider *LLMProvider) error
    Delete(id uint) error
    SetDefault(userID uint, providerID uint) error
}

// CronJobRepository Cron任务仓库接口
type CronJobRepository interface {
    Create(job *CronJob) error
    GetByID(id uint) (*CronJob, error)
    GetByUserID(userID uint) ([]CronJob, error)
    GetByChannelID(channelID uint) ([]CronJob, error)
    GetActiveJobs() ([]CronJob, error)
    GetPendingJobs(before time.Time) ([]CronJob, error)
    Update(job *CronJob) error
    Delete(id uint) error
    UpdateRunStatus(id uint, status string, result string) error
}
```

### 5.2 Service 接口

```go
// AgentService Agent 服务接口
type AgentService interface {
    // CRUD
    CreateAgent(userID uint, req CreateAgentRequest) (*Agent, error)
    GetAgent(id uint) (*Agent, error)
    GetUserAgents(userID uint) ([]Agent, error)
    UpdateAgent(id uint, req UpdateAgentRequest) (*Agent, error)
    DeleteAgent(id uint) error

    // 配置管理
    GetAgentConfig(agentID uint) (*AgentConfig, error)
    UpdateAgentConfig(agentID uint, config *AgentConfig) error

    // 模型配置管理
    GetModelConfig(agentID uint) (*ModelConfigResponse, error)
    UpdateModelConfig(agentID uint, req UpdateModelConfigRequest) (*Agent, error)
    ResolveModel(agentID uint) (*ModelSelectionResult, error) // 解析实际使用的模型

    // 记忆管理
    GetMemory(agentID uint) (string, error)
    UpdateMemory(agentID uint, content string) error

    // 能力管理
    GetAvailableSkills(agentID uint) ([]string, error)
    SetAvailableSkills(agentID uint, skills []string) error
    GetAvailableTools(agentID uint) ([]string, error)
    SetAvailableTools(agentID uint, tools []string) error

    // 默认 Agent
    GetDefaultAgent(userID uint) (*Agent, error)
    SetDefaultAgent(userID uint, agentID uint) error
}

// ModelConfigResponse 模型配置响应
type ModelConfigResponse struct {
    SelectionMode string `json:"selection_mode"` // 'auto' / 'specific'
    ModelID       string `json:"model_id,omitempty"`
    ModelName     string `json:"model_name,omitempty"`
    MaxTokens     int    `json:"max_tokens"`
    Temperature   float64 `json:"temperature"`
}

// UpdateModelConfigRequest 更新模型配置请求
type UpdateModelConfigRequest struct {
    SelectionMode string  `json:"selection_mode"` // 'auto' / 'specific'
    ModelID       string  `json:"model_id,omitempty"`
    ModelName     string  `json:"model_name,omitempty"`
    MaxTokens     int     `json:"max_tokens,omitempty"`
    Temperature   float64 `json:"temperature,omitempty"`
}

// ChannelService Channel 服务接口
type ChannelService interface {
    // CRUD
    CreateChannel(userID uint, req CreateChannelRequest) (*Channel, error)
    GetChannel(id uint) (*Channel, error)
    GetUserChannels(userID uint) ([]Channel, error)
    UpdateChannel(id uint, req UpdateChannelRequest) (*Channel, error)
    DeleteChannel(id uint) error

    // Agent 绑定
    BindAgent(channelID, agentID uint) error
    UnbindAgent(channelID uint) error
}

// LLMProviderService LLM提供商服务接口
type LLMProviderService interface {
    // CRUD
    CreateProvider(userID uint, req CreateProviderRequest) (*LLMProvider, error)
    GetProvider(id uint) (*LLMProvider, error)
    GetUserProviders(userID uint) ([]LLMProvider, error)
    UpdateProvider(id uint, req UpdateProviderRequest) (*LLMProvider, error)
    DeleteProvider(id uint) error

    // 默认提供商
    GetDefaultProvider(userID uint) (*LLMProvider, error)
    SetDefaultProvider(userID uint, providerID uint) error

    // 模型选择
    SelectModel(userID uint, selectionMode string, preferredModelID string) (*ModelSelectionResult, error)
}

// ModelSelectionResult 模型选择结果
type ModelSelectionResult struct {
    ProviderID   uint   `json:"provider_id"`
    ProviderKey  string `json:"provider_key"`
    ProviderName string `json:"provider_name"`
    ModelID      string `json:"model_id"`
    ModelName    string `json:"model_name"`
    APIKey       string `json:"-"` // 不序列化
    APIBase      string `json:"api_base,omitempty"`
}

// CronJobService Cron任务服务接口
type CronJobService interface {
    // CRUD
    CreateJob(userID uint, req CreateCronJobRequest) (*CronJob, error)
    GetJob(id uint) (*CronJob, error)
    GetUserJobs(userID uint) ([]CronJob, error)
    GetChannelJobs(channelID uint) ([]CronJob, error)
    UpdateJob(id uint, req UpdateCronJobRequest) (*CronJob, error)
    DeleteJob(id uint) error

    // 状态管理
    EnableJob(id uint) error
    DisableJob(id uint) error

    // 执行管理
    GetPendingJobs() ([]CronJob, error)
    ExecuteJob(jobID uint) error
    UpdateJobStatus(jobID uint, status string, result string) error

    // 调度计算
    CalculateNextRun(job *CronJob) (time.Time, error)
}
```

## 6. 迁移策略

### 6.1 数据迁移步骤

1. **创建新表**：
   - 创建 users、agents、channels、sessions 表
   - 创建 llm_providers 表（LLM 提供商配置）
   - 创建 cron_jobs 表（定时任务）
   - 扩展 agents 表添加记忆字段和模型选择字段

2. **扩展现有表**：
   - 为 conversation_records 添加新列 (user_id, agent_id, channel_id, channel_type)
   - 为 agents 表添加模型选择相关字段 (model_selection_mode, model_id, model_name)

3. **迁移现有数据**：
   - 创建默认用户（基于现有配置）
   - 将 workspace/SOUL.md 等文件内容读入数据库创建默认 Agent
   - 将现有 Channel 配置迁移到数据库
   - 将 memory/MEMORY.md 内容迁移到数据库
   - 将配置文件中的 providers 迁移到 llm_providers 表

4. **清理旧文件**：迁移完成后可选择删除旧的工作区目录
5. **代码适配**：修改代码使用新的数据库驱动配置

### 6.3 Agent 模型选择模式说明

**自动选择模式 (auto)**：
- 系统自动从用户配置的 LLM 提供商中选择合适的模型
- 优先级：默认提供商 > 优先级最高的提供商
- 适用于不需要特定模型的场景

**指定模式 (specific)**：
- 明确指定使用的模型 ID 和名称
- 系统会根据 model_id 找到对应的提供商和 API 密钥
- 适用于需要特定模型能力的场景

**配置示例：**
```json
// 自动选择模式
{
  "model_selection_mode": "auto",
  "max_tokens": 4096,
  "temperature": 0.7
}

// 指定模式
{
  "model_selection_mode": "specific",
  "model_id": "claude-opus-4",
  "model_name": "Claude Opus 4",
  "max_tokens": 4096,
  "temperature": 0.7
}
```

### 6.2 向后兼容

- 配置文件保留，作为数据库的初始数据源
- 首次启动时自动迁移配置到数据库
- 文件操作工具默认使用系统临时目录，可通过配置指定根目录

## 7. 安全考虑

1. **用户认证**：密码使用 bcrypt 加密存储
2. **数据隔离**：Repository 层强制过滤 user_id
3. **SQL 注入**：使用 GORM 参数化查询
4. **敏感信息**：AppSecret、Token 等加密存储

## 8. 实现阶段

### 阶段 1：数据库设计
- 创建所有表结构
- 编写 migration 脚本

### 阶段 2：Repository 层
- 实现 UserRepository
- 实现 AgentRepository
- 实现 ChannelRepository
- 实现 SessionRepository

### 阶段 3：Service 层
- 实现 AgentService（含模型选择逻辑）
- 实现 ChannelService
- 实现 LLMProviderService（API 密钥管理）
- 实现 CronJobService（定时任务调度）
- 修改现有代码使用新 Service

### 阶段 4：Cron 调度器实现
- 实现 Cron 表达式解析器
- 实现任务调度引擎
- 实现任务执行器（调用 LLM）
- 实现结果推送（到渠道）

### 阶段 5：集成测试
- 验证多租户隔离
- 验证 Agent-Channel 绑定
- 验证模型自动选择逻辑
- 验证 Cron 任务调度
- 数据迁移测试

## 9. Cron 任务配置示例

**每日早报任务：**
```json
{
  "name": "每日早报",
  "description": "每天早上9点推送新闻摘要",
  "cron_expression": "0 9 * * *",
  "timezone": "Asia/Shanghai",
  "prompt": "请搜索今天的重要新闻，生成一份简短的早报摘要",
  "model_selection_mode": "auto",
  "target_channel_id": 1,
  "target_user_id": "ou_xxx"
}
```

**使用特定模型的周报任务：**
```json
{
  "name": "周报生成",
  "description": "每周五下午6点生成工作周报",
  "cron_expression": "0 18 * * 5",
  "timezone": "Asia/Shanghai",
  "prompt": "根据本周的对话记录生成一份工作周报",
  "model_selection_mode": "specific",
  "model_id": "claude-opus-4",
  "model_name": "Claude Opus 4",
  "target_channel_id": 2
}
```
