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

import (
    "context"
    "fmt"
)

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
    "content": "# 长期记忆\n\n## 2026-03-14 10:30:00\n用户喜欢...",
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

#### 记忆格式规范

`append` 操作时，系统自动添加时间戳和格式分隔：

1. **格式规则**：
   - 每次追加前自动插入时间戳行（ISO 8601 格式：`YYYY-MM-DD HH:MM:SS`）
   - 时间戳前加 Markdown 二级标题 `##`
   - 内容后加双换行符分隔段落

2. **格式示例**：
   ```markdown
   ## 2026-03-15 14:30:00
   用户提到喜欢简洁的回答风格

   ## 2026-03-14 09:15:00
   用户要求所有代码必须有注释
   ```

3. **首次追加**：如果记忆为空，自动添加一级标题 `# Agent 长期记忆`

4. **大小计算**：`bytes_appended` 和 `total_size` 计算原始内容长度（不包含自动添加的时间戳格式开销）

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

**特殊字段说明：**

Agent 模型中的 `skills_list` 和 `tools_list` 字段以 JSON 数组的字符串形式存储（例如 `["skill1", "skill2"]`），读写时需做序列化/反序列化处理。参考 `internal/models/agent.go` 中的 `GetAvailableSkills()` 和 `GetAvailableTools()` 辅助方法。这两个字段**不在**当前工具的 `config_type` 范围内，如需扩展，必须复用相同的编解码逻辑以避免数据不一致。

## 权限集中化设计

为避免各工具重复权限检查逻辑，实现统一的权限验证层：

```go
// pkg/agent/tools/config/permission.go

package config

import (
    "context"
    "fmt"
    "github.com/weibaohui/nanobot-go/internal/models"
    agentsvc "github.com/weibaohui/nanobot-go/internal/service/agent"
)

// AccessValidator 权限验证器
type AccessValidator struct {
    agentService agentsvc.Service
}

// NewAccessValidator 创建权限验证器
func NewAccessValidator(agentService agentsvc.Service) *AccessValidator {
    return &AccessValidator{agentService: agentService}
}

// ValidateAccess 执行统一的权限检查
// 检查内容包括：Context 存在性、三参数完整性、Agent 存在性、UserCode 匹配
// 返回验证通过的 Agent 对象，或错误信息
func (v *AccessValidator) ValidateAccess(ctx context.Context) (*models.Agent, error) {
    // 1. 提取并验证上下文（强制）
    cfgCtx, err := GetAgentConfigContext(ctx)
    if err != nil {
        // 记录拒绝审计日志
        v.logDeniedAccess(ctx, "context_missing", err.Error())
        return nil, fmt.Errorf("security check failed: %w", err)
    }

    // 2. 验证 Agent 存在且属于当前用户
    agent, err := v.agentService.GetAgentByCode(cfgCtx.AgentCode)
    if err != nil {
        v.logDeniedAccess(ctx, "agent_lookup_failed", err.Error())
        return nil, fmt.Errorf("failed to get agent: %w", err)
    }
    if agent == nil {
        v.logDeniedAccess(ctx, "agent_not_found", cfgCtx.AgentCode)
        return nil, fmt.Errorf("agent not found: %s", cfgCtx.AgentCode)
    }
    if agent.UserCode != cfgCtx.UserCode {
        v.logDeniedAccess(ctx, "access_denied", fmt.Sprintf("agent %s does not belong to user %s", cfgCtx.AgentCode, cfgCtx.UserCode))
        return nil, fmt.Errorf("access denied: agent %s does not belong to user %s", cfgCtx.AgentCode, cfgCtx.UserCode)
    }

    return agent, nil
}

// logDeniedAccess 记录拒绝访问的审计日志
func (v *AccessValidator) logDeniedAccess(ctx context.Context, reason, detail string) {
    // 审计日志记录：尝试访问的标识、拒绝原因、时间戳等
    // 具体实现由审计日志模块处理
}
```

## 权限与审计流程

每个 Tool 的执行流程：

1. **提取并验证上下文（强制）**：调用 `ValidateAccess()` 统一检查，缺一不可
2. **解析参数**：验证 config_type/action 合法性
3. **验证内容大小**：单字段最大 1MB（UTF-8 字节长度），超限返回硬错误
4. **权限检查**：已通过 `ValidateAccess()` 完成
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

### user 配置类型的特殊安全考虑

`config_type: "user"` 映射到 `agents.user_content`（USER.md），涉及用户隐私信息，建议实施以下策略：

1. **默认只读策略**：在初期实现中，建议仅开放 `read_agent_config` 读取权限，`update_agent_config` 对 `user` 类型默认拒绝
2. **如需写入，需额外审批**：
   - 变更审计日志详细记录前后值
   - 管理员确认机制或速率限制（如每小时最多 3 次变更）
   - 敏感字段变更需二次确认
3. **访问控制**：`user_content` 只能由 Agent 自身读取，其他 Agent 无权访问

## 内容大小限制（1MB）

**实现细节：**

1. **检查时机**：在接收参数后立即检查，任何数据库写入操作之前
2. **计算方式**：原始 UTF-8 字节长度（`len([]byte(content))`）
3. **错误处理**：超限返回硬错误（validation error / HTTP 413），**禁止**静默截断
4. **统一限制**：所有 `config_type` 适用相同的 1MB 限制

**验证辅助函数：**
```go
// validateConfigFieldSize 验证配置字段大小
func validateConfigFieldSize(content string, fieldName string) error {
    const maxContentSize = 1024 * 1024 // 1MB
    if len([]byte(content)) > maxContentSize {
        return fmt.Errorf("field %s exceeds size limit: %d bytes (max %d)", fieldName, len([]byte(content)), maxContentSize)
    }
    return nil
}
```

## 审计日志设计

**存储方案：**

建议采用独立审计日志表 `audit_logs`，与业务数据分离，便于查询和归档。

**表结构设计：**
```sql
CREATE TABLE audit_logs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_code       VARCHAR(16) NOT NULL,      -- 操作者用户编码
    agent_code      VARCHAR(16) NOT NULL,      -- 目标 Agent 编码
    channel_code    VARCHAR(16) NOT NULL,      -- 渠道编码
    action          VARCHAR(50) NOT NULL,      -- 操作类型：read_config/update_config/manage_memory
    resource_type   VARCHAR(50) NOT NULL,      -- 资源类型：identity/soul/agents/tools/user/memory
    old_value_hash  VARCHAR(64),               -- 变更前值的哈希（可选）
    new_value_size  INTEGER,                   -- 新值大小（字节）
    request_id      VARCHAR(64),               -- 请求 ID，用于追踪
    status          VARCHAR(20) NOT NULL,      -- 状态：success/denied/error
    error_message   TEXT,                      -- 错误信息（失败时）
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_user_agent (user_code, agent_code),
    INDEX idx_created_at (created_at),
    INDEX idx_action (action)
);
```

**事务策略：**

1. **同步写入**：审计日志与主业务操作同步写入，确保一致性
2. **失败处理**：审计日志写入失败不应回滚主业务操作（避免阻塞正常功能），但应记录错误并告警
3. **异步补偿**：对于高并发场景，可采用异步批量写入，但需保证最终一致性

**保留与清理策略：**

- **保留期限**：默认保留 90 天，敏感操作日志保留 1 年
- **归档策略**：超期日志自动归档到冷存储（如对象存储）
- **访问控制**：审计日志查询需管理员权限，普通用户只能查看自己的操作记录

**查询接口：**

```go
// AuditQuery 审计日志查询条件
type AuditQuery struct {
    UserCode     string
    AgentCode    string
    ChannelCode  string
    Action       string
    StartTime    time.Time
    EndTime      time.Time
    Limit        int
    Offset       int
}
```

## 后续开发计划

### 核心实现（已完成）
1. ✅ 实现 `pkg/agent/tools/config/config_ctx.go` - Context 定义
2. ✅ 实现 `pkg/agent/tools/config/read_agent_config.go` - 读取工具
3. ✅ 实现 `pkg/agent/tools/config/update_agent_config.go` - 更新工具
4. ✅ 实现 `pkg/agent/tools/config/manage_agent_memory.go` - 记忆管理工具
5. ✅ 实现 `pkg/agent/tools/config/factory.go` - 工具工厂
6. ✅ 更新 `pkg/agent/tools.go` - 注册新工具
7. ✅ 更新 `pkg/agent/run.go` - 注入 Context

### 待补充任务
8. **数据库相关**
   - [ ] 创建 `audit_logs` 表及索引
   - [ ] 验证 `agents` 表现有索引是否满足查询性能要求

9. **测试相关**
   - [ ] 编写单元测试（覆盖率 ≥ 80%）
   - [ ] 编写集成测试：验证工具在真实消息流中的行为
   - [ ] 性能测试：1MB 内容读写、并发场景

10. **文档更新**
    - [ ] 更新 `AGENTS.md`：说明 Agent 如何使用配置工具
    - [ ] 更新用户文档：说明配置工具的使用场景和限制

11. **审计日志模块**
    - [ ] 实现审计日志写入接口
    - [ ] 在 config tools 中集成审计日志调用
    - [ ] 实现审计日志查询 API

12. **权限集中化（可选优化）**
    - [ ] 实现 `pkg/agent/tools/config/permission.go` - 统一权限验证层
