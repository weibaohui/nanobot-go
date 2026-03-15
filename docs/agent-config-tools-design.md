# Agent 配置管理 Tools 设计方案

## 背景

当前 Agent 的配置（SOUL.md, IDENTITY.md 等）已从文件系统迁移到数据库存储。需要设计专门的 Tools 来让 Agent 能够读取和更新自己的配置。

## 设计目标

1. **强制隔离**：Agent、Channel、User 三个参数必须通过 ctx 传入，缺一不可
2. **数据库优先**：所有操作直接读写 `agents` 表，不再接触文件系统
3. **权限内省**：Agent 只能修改自己的配置，不能越权访问其他 Agent
4. **接口精简**：用最少的工具覆盖所有需求

## 工具清单（共3个）

| Tool 名称 | 功能 |
|-----------|------|
| `read_agent_config` | 读取 Agent 配置项（identity/soul/agents/tools/user） |
| `update_agent_config` | 更新 Agent 配置项 |
| `manage_agent_memory` | 管理长期记忆（read/append/clear） |

## Context 设计

```go
// pkg/agent/tools/config/config_ctx.go

package config

// AgentConfigContext 配置工具上下文
type AgentConfigContext struct {
    UserCode    string // 用户编码（租户隔离）
    AgentCode   string // Agent 编码（操作目标）
    ChannelCode string // 渠道编码（审计追踪）
}

type contextKey struct{}
var agentConfigCtxKey = &contextKey{}

func WithAgentConfigContext(ctx context.Context, acc *AgentConfigContext) context.Context {
    return context.WithValue(ctx, agentConfigCtxKey, acc)
}

func GetAgentConfigContext(ctx context.Context) (*AgentConfigContext, error) {
    v := ctx.Value(agentConfigCtxKey)
    if v == nil {
        return nil, fmt.Errorf("agent config context required: must provide UserCode, AgentCode, ChannelCode via context")
    }
    acc, ok := v.(*AgentConfigContext)
    if !ok {
        return nil, fmt.Errorf("invalid agent config context type")
    }
    if acc.UserCode == "" || acc.AgentCode == "" || acc.ChannelCode == "" {
        return nil, fmt.Errorf("agent config context incomplete: UserCode, AgentCode, ChannelCode are all required")
    }
    return acc, nil
}
```

## 工具参数设计

### 1. read_agent_config

**输入参数：**
```json
{
    "config_type": "string"
}
```
- `config_type`: 必填，可选值: `identity`, `soul`, `agents`, `tools`, `user`

**输出结果：**
```json
{
    "success": true,
    "config_type": "soul",
    "content": "# Agent Soul\n\n你是一个...",
    "updated_at": "2026-03-15T10:30:00Z",
    "size_bytes": 1234
}
```

### 2. update_agent_config

**输入参数：**
```json
{
    "config_type": "string",
    "content": "string"
}
```
- `config_type`: 必填，同上
- `content`: 必填，完整替换内容

**输出结果：**
```json
{
    "success": true,
    "message": "配置已更新",
    "config_type": "soul",
    "bytes_written": 1234,
    "updated_at": "2026-03-15T10:30:00Z"
}
```

### 3. manage_agent_memory

**输入参数：**
```json
{
    "action": "string",
    "content": "string"
}
```
- `action`: 必填，可选值: `read`, `append`, `clear`
- `content`: `append` 时必填

**输出结果（read）：**
```json
{
    "success": true,
    "action": "read",
    "content": "# 长期记忆\n\n## 2026-03-14\n用户喜欢...",
    "size_bytes": 5678,
    "updated_at": "2026-03-15T10:30:00Z"
}
```

**输出结果（append）：**
```json
{
    "success": true,
    "action": "append",
    "message": "记忆已追加",
    "bytes_appended": 234,
    "total_size": 5912
}
```

**输出结果（clear）：**
```json
{
    "success": true,
    "action": "clear",
    "message": "记忆已清空"
}
```

## 配置类型映射

```go
var configTypeToField = map[string]string{
    "identity": "identity_content",  // IDENTITY.md
    "soul":     "soul_content",      // SOUL.md
    "agents":   "agents_content",    // AGENTS.md
    "tools":    "tools_content",     // TOOLS.md
    "user":     "user_content",      // USER.md
}

// 记忆字段单独处理
const memoryField = "memory_content"
```

## 权限与审计流程

每个 Tool 的执行流程：

1. **提取并验证上下文（强制）**：从 ctx 获取 UserCode, AgentCode, ChannelCode，缺一不可
2. **解析参数**：验证 config_type/action 合法性
3. **验证内容大小**：单字段最大 1MB
4. **权限检查**：验证 Agent 存在且属于当前 User
5. **执行操作**：数据库更新
6. **记录审计日志**：记录谁、何时、哪个渠道、改了什么

## Context 注入位置

在消息处理入口处注入 Context：

```go
func (h *Handler) processMessage(ctx context.Context, session *models.Session, msg *Message) {
    cfgCtx := &config.AgentConfigContext{
        UserCode:    session.UserCode,
        AgentCode:   session.AgentCode,
        ChannelCode: session.ChannelCode,
    }
    ctx = config.WithAgentConfigContext(ctx, cfgCtx)

    loop := h.agentManager.GetLoop(session.AgentCode)
    loop.Process(ctx, msg)
}
```

## 与现有记忆系统的区分

| 维度 | 长期记忆 (LongTermMemory 表) | Agent Memory 字段 |
|------|---------------------------|------------------|
| 存储位置 | `long_term_memories` 表 | `agents.memory_content` 字段 |
| 内容形式 | 结构化数据（JSON/分字段） | Markdown 自由文本 |
| 更新方式 | 定时任务自动提炼 | Agent 主动调用 Tool |
| 用途 | 语义检索、回忆 | 系统提示词的一部分 |
| 工具 | 无需 Tool（自动） | `manage_agent_memory` |

## 安全红线

1. **禁止跨用户访问**：UserCode 必须匹配
2. **禁止跨 Agent 访问**：AgentCode 必须匹配当前 Loop 绑定的 Agent
3. **禁止绕过 Context**：不提供 Context 的工具调用直接拒绝
4. **内容审计**：所有写操作记录审计日志

## 后续开发计划

1. 实现 `pkg/agent/tools/config/config_ctx.go` - Context 定义
2. 实现 `pkg/agent/tools/config/read_agent_config.go` - 读取工具
3. 实现 `pkg/agent/tools/config/update_agent_config.go` - 更新工具
4. 实现 `pkg/agent/tools/config/manage_agent_memory.go` - 记忆管理工具
5. 实现 `pkg/agent/tools/config/factory.go` - 工具工厂
6. 更新 `pkg/agent/tools.go` - 注册新工具
7. 更新消息处理器 - 注入 Context
8. 编写单元测试
