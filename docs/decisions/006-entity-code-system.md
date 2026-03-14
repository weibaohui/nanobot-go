# 006-业务实体编码系统决策

## 背景

随着系统从单租户向多租户、多 Agent 架构演进，原有的基于数据库自增 ID 的实体关联方式存在以下问题：

1. **跨实例数据迁移困难**：自增 ID 在不同数据库实例间可能冲突
2. **API 暴露安全风险**：暴露自增 ID 容易被猜测和遍历攻击
3. **数据导入导出复杂**：外键依赖导致数据迁移顺序受限
4. **可读性差**：数字 ID 无法直观识别实体类型

## 决策

### 1. ID 与 Code 职责分离

| 字段 | 用途 | 说明 |
|------|------|------|
| **ID** | 表内主键 | 仅作为数据库表的主键使用，用于表内行的唯一标识 |
| **Code** | 业务关联 | 用于表与表之间的关联、API 交互、日志追踪等业务场景 |

**原则**：
- ID 仅在数据库内部使用，不对外暴露
- 所有业务关联（外键关系）使用 Code 字段
- API 接口统一使用 Code 进行实体的查询和关联

### 2. Code 字段设计规范

#### 2.1 格式定义

```
格式: {prefix}_{random}
示例: usr_aB3dEfGhKj, chn_MnPqRsTuVw, agt_XyZaBcDeFg

- prefix: 3位小写字母 + 下划线，标识实体类型
- random: 10位随机字符
- 总长度: 15字符
```

#### 2.2 实体前缀定义

| 实体类型 | 前缀 | 示例 |
|----------|------|------|
| User | `usr_` | usr_aB3dEfGhKj |
| Channel | `chn_` | chn_MnPqRsTuVw |
| Agent | `agt_` | agt_XyZaBcDeFg |

#### 2.3 字符集设计

```go
// 去除易混淆字符（0, O, 1, I, l）
CodeAlphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
// 共 57 个字符
```

**设计理由**：
- 去除 `0/O`、`1/I/l` 等视觉相似字符，降低人工抄写出错率
- 使用大小写混合，增加编码空间
- 不包含特殊字符，便于 URL 传递

### 3. Code 生成算法

#### 3.1 核心算法

```go
// 使用 crypto/rand 生成安全随机数
func Generate() (string, error) {
    var sb strings.Builder
    alphabetLen := big.NewInt(int64(len(g.alphabet)))

    for i := 0; i < 10; i++ {  // 10位随机字符
        idx, err := rand.Int(rand.Reader, alphabetLen)
        if err != nil {
            return "", err
        }
        sb.WriteRune(g.alphabet[idx.Int64()])
    }
    return sb.String(), nil
}
```

#### 3.2 唯一性保证

采用**生成时检测 + 重试机制**：

```go
func GenerateUniqueCodeWithRetry(
    generateFunc func() (string, error),
    checker func(string) (bool, error),  // 检查 Code 是否已存在
    maxRetries int,
) (string, error)
```

**重试策略**：
- 最大重试次数：10 次
- Code 空间：57^10 ≈ 3.6 × 10^17，冲突概率极低
- 冲突时立即重试生成新的 Code

### 4. 数据库设计规范

#### 4.1 表结构示例

```go
// User 表
type User struct {
    ID       uint   `gorm:"primarykey"`                    // 内部主键，不对外使用
    UserCode string `gorm:"type:varchar(16);uniqueIndex"`  // 业务编码，用于关联
    // ... 其他字段
}

// Channel 表
type Channel struct {
    ID          uint   `gorm:"primarykey"`
    ChannelCode string `gorm:"type:varchar(16);uniqueIndex"`
    UserCode    string `gorm:"type:varchar(16);index"`     // 使用 Code 关联 User
    AgentCode   string `gorm:"type:varchar(16);index"`     // 使用 Code 关联 Agent
    // ... 其他字段
}
```

#### 4.2 关联关系

**所有外键关系使用 Code 字段**：

```go
// ✅ 正确：使用 Code 关联
UserCode    string `gorm:"type:varchar(16);index"`
AgentCode   string `gorm:"type:varchar(16);index"`
ChannelCode string `gorm:"type:varchar(16);index"`

// ❌ 错误：不使用 ID 关联
UserID    uint  // 避免使用
AgentID   uint  // 避免使用
ChannelID uint  // 避免使用
```

### 5. API 设计规范

#### 5.1 接口统一使用 Code

```go
// ✅ 正确：使用 Code 查询
GET /api/channels?user_code=usr_xxx
GET /api/agents?channel_code=chn_xxx

// ❌ 错误：不使用 ID 查询
GET /api/channels?user_id=123
```

#### 5.2 DTO 定义

```go
type ChannelResponse struct {
    ID          uint   `json:"id"`           // 内部使用，不建议前端依赖
    ChannelCode string `json:"channel_code"` // ✅ 业务关联使用
    UserCode    string `json:"user_code"`    // ✅ 关联 User
    AgentCode   string `json:"agent_code"`   // ✅ 关联 Agent
    // ...
}
```

### 6. Context 传递规范

在服务层和 Hook 事件中，通过 Context 传递 Code：

```go
// 注入到 Context
ctx = trace.WithUserCode(ctx, user.UserCode)
ctx = trace.WithAgentCode(ctx, agent.AgentCode)
ctx = trace.WithChannelCode(ctx, channel.ChannelCode)

// 从 Context 获取
userCode := trace.GetUserCode(ctx)
agentCode := trace.GetAgentCode(ctx)
channelCode := trace.GetChannelCode(ctx)
```

### 7. 对话记录应用示例

```go
type ConversationRecord struct {
    ID          uint   `gorm:"primarykey"`

    // 归属信息（使用 Code 关联）
    UserCode    string `gorm:"type:varchar(16);index"`
    AgentCode   string `gorm:"type:varchar(16);index"`
    ChannelCode string `gorm:"type:varchar(16);index"`

    // 其他字段...
}
```

## 影响范围

| 模块 | 变更内容 |
|------|----------|
| 数据库表结构 | 新增 Code 字段，原有 ID 仅作为主键 |
| API 接口 | 统一使用 Code 作为查询和关联参数 |
| 服务层 | 使用 Code 进行实体关联和权限校验 |
| Hook 事件 | 传递 Code 字段用于对话记录归属 |
| 前端 | 使用 Code 进行实体关联展示 |

## 向后兼容性

- **数据库**：保留 ID 字段，新增 Code 字段，现有数据需通过迁移脚本添加 Code
- **API**：逐步迁移接口从 ID 到 Code，保持过渡期兼容
- **外部集成**：提供 Code 映射文档，协助外部系统适配

## 参考

- [006-业务实体编码系统-设计文档](../design/006-业务实体编码系统-设计.md)
- [006-业务实体编码系统-需求文档](../requirements/006-业务实体编码系统-需求.md)
