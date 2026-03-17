# 020-MCP-Tools管理-设计

## 1. 概述

本文档描述 MCP Tools 管理功能的技术设计方案。

## 2. k8m 设计借鉴

参考 k8m 项目的 `mcp_runtime` 模块，借鉴以下设计：

### 2.1 架构设计

| 设计点 | k8m 实现 | nanobot-go 采用 |
|--------|----------|-----------------|
| **工具缓存** | `Tools map[string][]mcp.Tool` | 添加内存缓存，避免重复获取 |
| **工具命名** | `tool@server` 格式 | 多 MCP 时避免工具名冲突 |
| **连接策略** | 每次调用创建新客户端 | 简单可靠，避免连接污染 |
| **执行日志** | `MCPToolLog` 表记录调用 | 新增 `mcp_tool_logs` 表 |
| **主机管理** | `MCPHost` 统一管理 | 参考 `MCPHost` 设计模式 |

### 2.2 代码模式借鉴

```go
// MCPHost 主机管理模式
type MCPHost struct {
    configs map[string]ServerConfig  // 服务器配置
    mutex   sync.RWMutex             // 并发保护
    Tools   map[string][]mcp.Tool    // 工具缓存（按服务器分组）
}

// 工具列表获取（带缓存）
func (m *MCPHost) GetTools(ctx context.Context, serverName string) ([]mcp.Tool, error) {
    // 检查缓存
    if tools, ok := m.Tools[serverName]; ok {
        return tools, nil
    }
    // 获取并缓存
    cli := m.getClient(serverName)
    tools, _ := cli.ListTools(ctx, mcp.ListToolsRequest{})
    m.Tools[serverName] = tools.Tools
    return tools.Tools, nil
}

// 工具调用（每次新建客户端）
func (m *MCPHost) ExecTools(ctx, toolCalls) {
    cli, _ := client.NewSSEMCPClient(config.URL, ...)
    defer cli.Close()
    result, _ := cli.CallTool(ctx, mcp.CallToolRequest{...})
    // 记录日志
    m.LogToolExecution(ctx, toolName, serverName, params, result, duration)
}
```

## 3. 数据库设计

### 3.1 新增表：mcp_tools

```sql
CREATE TABLE mcp_tools (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    mcp_server_id BIGINT UNSIGNED NOT NULL COMMENT '关联的 MCP Server ID',
    name VARCHAR(255) NOT NULL COMMENT '工具名称',
    description TEXT COMMENT '工具描述',
    input_schema JSON COMMENT '输入参数定义（JSON Schema）',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME DEFAULT NULL,

    INDEX idx_mcp_server_id (mcp_server_id),
    INDEX idx_name (name),
    UNIQUE INDEX uk_mcp_server_name (mcp_server_id, name, deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='MCP工具表';
```

### 3.2 新增表：mcp_tool_logs（工具调用日志）

```sql
CREATE TABLE mcp_tool_logs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    session_key VARCHAR(64) NOT NULL COMMENT '会话标识',
    mcp_server_id BIGINT UNSIGNED NOT NULL COMMENT 'MCP Server ID',
    tool_name VARCHAR(255) NOT NULL COMMENT '工具名称',
    parameters JSON COMMENT '调用参数',
    result TEXT COMMENT '返回结果',
    error_message TEXT COMMENT '错误信息',
    execute_time INT UNSIGNED DEFAULT 0 COMMENT '执行耗时(ms)',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_session_key (session_key),
    INDEX idx_mcp_server_id (mcp_server_id),
    INDEX idx_tool_name (tool_name),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='MCP工具调用日志';
```

### 3.3 修改表：agents（新增 mcp_list 字段）

Agent 已有 `skills_list` 和 `tools_list` 字段，新增 `mcp_list` 字段：

```sql
ALTER TABLE agents ADD COLUMN mcp_list TEXT COMMENT '可用MCP列表，JSON数组格式' AFTER tools_list;
```

**字段格式说明（三种能力统一格式）：**
- `skills_list`: `["skill1", "skill2"]` - 可用的 Skill 编码列表
- `tools_list`: `["tool1", "tool2"]` - 可用的内置工具名称列表
- `mcp_list`: `["opdk", "filesystem"]` - 可用的 MCP Server 编码列表（启用该 MCP 的所有工具）

### 3.4 数据关系图

```
┌─────────────────────────────────────────────────────────────────┐
│                           agents                                │
├─────────────────────────────────────────────────────────────────┤
│  id, agent_code, name, ...                                      │
│  skills_list (JSON)  - 可用技能列表 ["skill1", "skill2"]         │
│  tools_list (JSON)   - 可用内置工具列表 ["tool1", "tool2"]       │
│  mcp_list (JSON)     - 可用MCP列表 ["opdk", "filesystem"]        │
└────────────────────────────┬────────────────────────────────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
              ▼              ▼              ▼
     ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐
     │   skills    │  │ 内置tools   │  │  mcp_servers    │
     └─────────────┘  └─────────────┘  └────────┬────────┘
                                                │ 1:N
                                                ▼
                                       ┌─────────────────┐
                                       │   mcp_tools     │
                                       ├─────────────────┤
                                       │  mcp_server_id  │
                                       │  name           │
                                       │  description    │
                                       │  input_schema   │
                                       └─────────────────┘
```

### 3.5 移除 agent_mcp_bindings 表

**决定移除 `agent_mcp_bindings` 表**，理由：
1. 避免数据冗余和同步问题
2. 绑定关系直接存在 Agent 上更直观
3. 只控制到 MCP 级别，不需要复杂的工具级别控制
4. 减少一张表，简化系统

**迁移策略：**
- 将现有 `agent_mcp_bindings` 数据迁移到 `agents.mcp_list` 字段
- 迁移完成后删除 `agent_mcp_bindings` 表

## 4. API 设计

### 4.1 新增接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/mcp-servers/:id/tools` | 获取 MCP Server 的工具列表 |
| GET | `/api/v1/mcp-servers/:id/tool-logs` | 获取 MCP Server 的调用日志 |
| GET | `/api/v1/agents/:id/available-tools` | 获取 Agent 可用的所有工具（含 MCP） |

### 4.2 响应格式

**工具列表：**
```json
{
    "items": [
        {
            "id": 1,
            "mcp_server_id": 1,
            "name": "read_file",
            "description": "读取文件内容",
            "input_schema": {
                "type": "object",
                "properties": {
                    "path": { "type": "string", "description": "文件路径" }
                },
                "required": ["path"]
            },
            "created_at": "2024-01-01T00:00:00Z",
            "updated_at": "2024-01-01T00:00:00Z"
        }
    ],
    "total": 1
}
```

**调用日志：**
```json
{
    "items": [
        {
            "id": 1,
            "session_key": "abc123",
            "mcp_server_id": 1,
            "tool_name": "read_file",
            "parameters": {"path": "/tmp/test.txt"},
            "result": "file content...",
            "error_message": null,
            "execute_time": 150,
            "created_at": "2024-01-01T00:00:00Z"
        }
    ],
    "total": 1
}
```

### 4.3 现有接口复用

- 刷新工具列表：`POST /api/v1/mcp-servers/:id/refresh`
- Agent 绑定：现有 API 已支持 `enabled_tools` 参数

## 5. 后端实现

### 5.1 Model 层

**文件：`internal/models/mcp_tool.go`**

```go
type MCPTool struct {
    ID          uint            `gorm:"primaryKey" json:"id"`
    MCPServerID uint            `gorm:"not null;index" json:"mcp_server_id"`
    Name        string          `gorm:"size:255;not null" json:"name"`
    Description string          `json:"description"`
    InputSchema datatypes.JSON  `json:"input_schema"`
    CreatedAt   time.Time       `json:"created_at"`
    UpdatedAt   time.Time       `json:"updated_at"`
    DeletedAt   gorm.DeletedAt  `gorm:"index" json:"-"`

    MCPServer   *MCPServer      `gorm:"foreignKey:MCPServerID" json:"-"`
}

func (MCPTool) TableName() string {
    return "mcp_tools"
}
```

**文件：`internal/models/mcp_tool_log.go`**

```go
type MCPToolLog struct {
    ID          uint            `gorm:"primaryKey" json:"id"`
    SessionKey  string          `gorm:"size:64;not null;index" json:"session_key"`
    MCPServerID uint            `gorm:"not null;index" json:"mcp_server_id"`
    ToolName    string          `gorm:"size:255;not null;index" json:"tool_name"`
    Parameters  datatypes.JSON  `json:"parameters"`
    Result      string          `gorm:"type:text" json:"result"`
    ErrorMessage string         `gorm:"type:text" json:"error_message"`
    ExecuteTime uint            `gorm:"default:0" json:"execute_time"` // ms
    CreatedAt   time.Time       `gorm:"index" json:"created_at"`
}

func (MCPToolLog) TableName() string {
    return "mcp_tool_logs"
}
```

**文件：`internal/models/agent.go`（修改）**

```go
// 新增字段
MCPList string `gorm:"type:text" json:"mcp_list"` // 可用MCP列表，JSON数组

// 新增方法
func (a *Agent) GetAvailableMCPs() []string {
    if a.MCPList == "" || a.MCPList == "null" {
        return nil
    }
    var mcps []string
    if err := json.Unmarshal([]byte(a.MCPList), &mcps); err != nil {
        return nil
    }
    return mcps
}
```

### 5.2 Service 层

**文件：`internal/service/mcp/service.go`**

```go
// RefreshCapabilities 刷新 MCP 能力（删除旧工具，插入新工具）
func (s *Service) RefreshCapabilities(serverID uint) error {
    server, err := s.repo.GetServerByID(serverID)
    if err != nil {
        return err
    }

    // 调用 MCP 协议获取工具列表
    tools, err := s.mcpClient.ListTools(server)
    if err != nil {
        return err
    }

    // 删除旧工具
    s.repo.DeleteToolsByServerID(serverID)

    // 批量插入新工具
    for _, tool := range tools {
        mcpTool := &models.MCPTool{
            MCPServerID: serverID,
            Name:        tool.Name,
            Description: tool.Description,
            InputSchema: tool.InputSchema,
        }
        s.repo.CreateTool(mcpTool)
    }

    // 更新服务器状态
    s.repo.UpdateServerStatus(serverID, "active", "")
    return nil
}

// ListTools 获取 MCP Server 的工具列表
func (s *Service) ListTools(serverID uint) ([]models.MCPTool, error)

// LogToolExecution 记录工具调用日志
func (s *Service) LogToolExecution(sessionKey string, serverID uint, toolName string,
    params any, result string, errMsg string, executeTime int64) error
```

### 5.3 Handler 层

**文件：`internal/api/mcp_handler.go`**

```go
// listMCPServerTools 获取 MCP Server 的工具列表
func (h *Handler) listMCPServerTools(c *gin.Context) {
    id, ok := parseID(c, "id")
    if !ok {
        c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid mcp server id"})
        return
    }

    tools, err := h.mcpService.ListTools(uint(id))
    if err != nil {
        c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
        return
    }

    c.JSON(http.StatusOK, ListResponse{
        Items: tools,
        Total: int64(len(tools)),
    })
}

// listMCPServerToolLogs 获取 MCP Server 的调用日志
func (h *Handler) listMCPServerToolLogs(c *gin.Context) {
    // 实现日志查询
}
```

### 5.4 路由注册

**文件：`internal/api/handler.go`**

```go
mcpServers.GET("/:id/tools", h.listMCPServerTools)
mcpServers.GET("/:id/tool-logs", h.listMCPServerToolLogs)
```

## 6. 前端实现

### 6.1 类型定义

**文件：`web/src/types/index.ts`**

```typescript
// MCP 工具
export interface MCPToolItem {
  id: number;
  mcp_server_id: number;
  name: string;
  description: string;
  input_schema: Record<string, any> | null;
  created_at: string;
  updated_at: string;
}

// MCP 调用日志
export interface MCPToolLog {
  id: number;
  session_key: string;
  mcp_server_id: number;
  tool_name: string;
  parameters: Record<string, any> | null;
  result: string | null;
  error_message: string | null;
  execute_time: number;
  created_at: string;
}

// Agent 类型更新
export interface Agent {
  // ... 现有字段
  skills_list: string;  // JSON 数组
  tools_list: string;   // JSON 数组
  mcp_list: string;     // JSON 数组 - 新增
}
```

### 6.2 API 调用

**文件：`web/src/api/mcpServers.ts`**

```typescript
// 获取 MCP Server 的工具列表
getTools: (serverId: number) =>
  client.get<any, ListResponse<MCPToolItem>>(`/mcp-servers/${serverId}/tools`),

// 获取 MCP Server 的调用日志
getToolLogs: (serverId: number, params?: { session_key?: string; limit?: number }) =>
  client.get<any, ListResponse<MCPToolLog>>(`/mcp-servers/${serverId}/tool-logs`, { params }),
```

### 6.3 页面改造

#### MCPServers.tsx

1. **表格改造**：添加可展开行显示工具列表
2. **工具列表展示**：工具名称、描述、参数定义
3. **调用日志入口**：查看该 MCP 的调用历史
4. **刷新工具按钮**：调用 `/mcp-servers/:id/refresh`

#### Agents.tsx

1. **新增 MCP 配置区域**：
   - 可用 MCP Server 列表（多选）
   - 每选中的 MCP 可进一步选择工具

2. **统一能力配置界面**：
```
┌─────────────────────────────────────────────────────────────┐
│ Agent 能力配置                                              │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ 【Skills】已选择 2 个                                        │
│ ☑ skill_search    ☑ skill_analyze    ☐ skill_translate    │
│                                                             │
│ 【内置工具】已选择 3 个                                      │
│ ☑ web_search      ☑ file_read        ☑ shell_exec        │
│                                                             │
│ 【MCP Server】已选择 1 个                                    │
│ ☑ opdk (5个工具)                                            │
│     └─ ☑ read_file  ☑ write_file  ☐ delete_file ...      │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 6.4 UI 组件

```
MCP Server 管理页面：

┌─────────────────────────────────────────────────────────────┐
│ MCP Server 管理                                    [+ 新建] │
├─────────────────────────────────────────────────────────────┤
│ ID │ 名称   │ 传输类型 │ 状态   │ 工具数 │ 调用次数 │ 操作  │
├────┼────────┼──────────┼────────┼────────┼──────────┼───────┤
│ ▶ 1│ opdk   │ SSE      │ 已连接 │ 5      │ 128      │ 🔄📋✏🗑│
└────┴────────┴──────────┴────────┴────────┴──────────┴───────┘

展开后显示工具列表：
┌────┬────────────────────────────────────────────────────────┐
│ ▼ 1│ opdk   │ SSE      │ 已连接 │ 5      │ 128      │ 🔄📋✏🗑│
│    │ ┌─────────────────────────────────────────────────────┐│
│    │ │ 工具名称      │ 描述          │ 参数               ││
│    │ ├────────────────┼───────────────┼────────────────────┤│
│    │ │ read_file      │ 读取文件内容  │ {"path":"string"}  ││
│    │ │ write_file     │ 写入文件内容  │ {"path":"string"}  ││
│    │ └────────────────┴───────────────┴────────────────────┘│
└────┴─────────────────────────────────────────────────────────┘

📋 = 查看调用日志
```

## 7. 实施计划

### Phase 1: 后端基础
1. 创建 `mcp_tools` 模型和数据库迁移
2. 创建 `mcp_tool_logs` 模型
3. Agent 表新增 `mcp_list` 字段
4. 实现 Service 层方法
5. 实现 API 接口

### Phase 2: 前端 MCP 工具展示
1. 添加类型定义和 API 调用
2. 改造 MCPServers.tsx，添加工具列表展开行
3. 添加调用日志查看功能

### Phase 3: Agent 能力配置
1. 改造 Agent 配置页面
2. 统一 Skills/Tools/MCP 配置界面
3. 实现 MCP 工具多选功能

### Phase 4: 工具调用日志
1. 在 MCP 工具调用时记录日志
2. 实现日志查询 API
3. 前端日志展示页面

## 8. 变更记录

| 日期 | 版本 | 变更内容 | 作者 |
|------|------|----------|------|
| 2024-03-17 | v1.0 | 初始设计 | AI |
| 2024-03-17 | v1.1 | 补充 k8m 设计借鉴、调用日志表、Agent mcp_list 字段 | AI |

## 9. 风险与待确认

### 9.1 待确认
1. 工具刷新时是否需要保留工具的关联关系？
   - **决定**：不保留，全部重新插入（简单可靠）

2. `enabled_tools` 为空表示全部启用还是全部禁用？
   - **决定**：空 = 全部启用（保持不变）

3. 调用日志保留多久？
   - **建议**：默认保留 30 天，可配置

### 9.2 风险
1. 大量工具时的性能问题（分页加载）
2. 工具名称冲突处理（同一 MCP 下唯一）
3. 日志表数据量增长（定期清理）
