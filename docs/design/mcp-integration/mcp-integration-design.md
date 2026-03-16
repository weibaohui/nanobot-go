# MCP 服务器接入功能设计方案

## 1. 需求概述

### 1.1 背景
引入 MCP (Model Context Protocol) 服务器管理机制，允许外部 MCP Server 注册到平台，为 Agent 提供可调用的工具服务。

### 1.2 核心需求
- **MCP 服务器注册**：支持外部 MCP Server 注册到平台
- **全局管理界面**：统一后台管理 MCP 连接
- **Agent 级别控制**：精细化控制每个 Agent 可使用的 MCP 工具
- **上下文优化**：避免一次性暴露所有 MCP 方法，节省上下文空间

### 1.3 设计原则
- 类似现有 tools/skills 的管理模式
- 支持动态添加/移除 MCP 服务器
- 支持为每个 Agent 配置可用的 MCP 工具子集

## 2. 数据库设计

### 2.1 MCP 服务器表 (mcp_servers)

```sql
CREATE TABLE mcp_servers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code VARCHAR(64) UNIQUE NOT NULL,        -- MCP 服务器编码 (如: github, fetch)
    name VARCHAR(128) NOT NULL,               -- 显示名称
    description TEXT,                         -- 描述
    transport_type VARCHAR(32) NOT NULL,      -- 传输类型: stdio, http, sse
    command TEXT,                             -- 启动命令 (stdio 类型)
    args TEXT,                                -- 启动参数 JSON 数组 (stdio 类型)
    url VARCHAR(512),                         -- 服务 URL (http/sse 类型)
    env_vars TEXT,                            -- 环境变量 JSON
    status VARCHAR(32) DEFAULT 'inactive',    -- 状态: active, inactive, error
    capabilities TEXT,                        -- 能力列表 JSON (从服务器获取的工具列表)
    last_connected_at DATETIME,               -- 最后连接时间
    error_message TEXT,                       -- 错误信息
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 2.2 Agent MCP 关联表 (agent_mcp_bindings)

```sql
CREATE TABLE agent_mcp_bindings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    agent_id INTEGER NOT NULL,
    mcp_server_id INTEGER NOT NULL,
    enabled_tools TEXT,                       -- 启用的工具列表 JSON (null 表示全部启用)
    is_enabled BOOLEAN DEFAULT TRUE,          -- 是否启用该 MCP 服务器
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE,
    FOREIGN KEY (mcp_server_id) REFERENCES mcp_servers(id) ON DELETE CASCADE,
    UNIQUE(agent_id, mcp_server_id)
);
```

### 2.3 索引

```sql
CREATE INDEX idx_mcp_servers_code ON mcp_servers(code);
CREATE INDEX idx_mcp_servers_status ON mcp_servers(status);
CREATE INDEX idx_agent_mcp_bindings_agent_id ON agent_mcp_bindings(agent_id);
CREATE INDEX idx_agent_mcp_bindings_mcp_server_id ON agent_mcp_bindings(mcp_server_id);
```

## 3. 架构设计

### 3.1 整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        前端管理界面                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │ MCP 服务器列表 │  │ 新增/编辑 MCP │  │ Agent-MCP 绑定配置   │   │
│  └──────────────┘  └──────────────┘  └──────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                         API 层                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │ MCP Server   │  │ MCP Tool     │  │ Agent MCP Binding    │   │
│  │ Controller   │  │ Controller   │  │ Controller           │   │
│  └──────────────┘  └──────────────┘  └──────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                       Service 层                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │ MCPManager   │  │ MCPClient    │  │ AgentMCPBinding      │   │
│  │ Service      │  │ Pool         │  │ Service              │   │
│  └──────────────┘  └──────────────┘  └──────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      MCP 协议层                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │ Stdio        │  │ HTTP         │  │ SSE                  │   │
│  │ Transport    │  │ Transport    │  │ Transport            │   │
│  └──────────────┘  └──────────────┘  └──────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     外部 MCP Servers                             │
│         (GitHub MCP, Fetch MCP, FileSystem MCP, ...)            │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 核心组件

#### MCPManager
- 管理所有 MCP 服务器的生命周期
- 维护 MCP 客户端连接池
- 提供工具调用接口

#### MCPClient
- 封装单个 MCP 服务器的连接
- 处理协议通信 (stdio/http/sse)
- 缓存服务器 capabilities

#### AgentMCPBindingService
- 管理 Agent 与 MCP 的绑定关系
- 为 Agent 加载可用的 MCP 工具
- 工具过滤和权限控制

## 4. API 设计

### 4.1 MCP 服务器管理

```go
// 创建 MCP 服务器
POST /api/mcp/servers
{
    "code": "github",
    "name": "GitHub MCP",
    "description": "GitHub 操作工具",
    "transport_type": "stdio",
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-github"],
    "env_vars": {"GITHUB_TOKEN": "xxx"}
}

// 获取 MCP 服务器列表
GET /api/mcp/servers

// 获取单个 MCP 服务器
GET /api/mcp/servers/:id

// 更新 MCP 服务器
PUT /api/mcp/servers/:id

// 删除 MCP 服务器
DELETE /api/mcp/servers/:id

// 测试连接
POST /api/mcp/servers/:id/test

// 刷新 capabilities
POST /api/mcp/servers/:id/refresh
```

### 4.2 Agent MCP 绑定

```go
// 获取 Agent 的 MCP 绑定
GET /api/agents/:agent_code/mcp-bindings

// 绑定 MCP 服务器到 Agent
POST /api/agents/:agent_code/mcp-bindings
{
    "mcp_server_id": 1,
    "enabled_tools": ["search_issues", "create_issue"],  // null 表示全部启用
    "is_enabled": true
}

// 更新 Agent 的 MCP 绑定
PUT /api/agents/:agent_code/mcp-bindings/:binding_id
{
    "enabled_tools": ["search_issues"],
    "is_enabled": true
}

// 解绑 MCP 服务器
DELETE /api/agents/:agent_code/mcp-bindings/:binding_id
```

### 4.3 MCP 工具调用 (内部)

```go
// Agent 执行时调用 MCP 工具
POST /api/internal/mcp/invoke
{
    "agent_code": "agent-1",
    "mcp_code": "github",
    "tool_name": "search_issues",
    "arguments": {"query": "bug"}
}
```

## 5. 前端设计

### 5.1 MCP 服务器管理页面
- 列表展示所有 MCP 服务器
- 状态指示 (活跃/断开/错误)
- 新增/编辑/删除/测试操作
- 查看 capabilities (工具列表)

### 5.2 Agent 配置页面扩展
- 在 Agent 编辑页面增加 "MCP 配置" Tab
- 已绑定 MCP 服务器列表
- 添加绑定对话框 (选择 MCP 服务器 + 选择启用工具)
- 工具开关控制

## 6. 集成到 Agent 执行流程

### 6.1 工具加载
1. Agent 启动时，加载绑定的 MCP 工具
2. 根据 `enabled_tools` 过滤可用工具
3. 将 MCP 工具转换为 Agent 可调用的 Tool 接口

### 6.2 工具调用
1. LLM 请求调用 MCP 工具
2. Agent 通过 MCPManager 路由到对应 MCPClient
3. MCPClient 执行实际调用
4. 返回结果给 LLM

## 7. 变更记录

| 日期 | 变更内容 | 作者 |
|------|----------|------|
| 2026-03-16 | 初始设计文档 | Claude |

## 8. 实现状态

| 模块 | 状态 | 说明 |
|------|------|------|
| 数据库设计 | ✅ 已完成 | mcp_servers, agent_mcp_bindings 表 |
| Repository 层 | ✅ 已完成 | MCPServerRepository, AgentMCPBindingRepository |
| Service 层 | ✅ 已完成 | 基础 CRUD, 绑定管理 |
| API Handler | ✅ 已完成 | RESTful API |
| MCP 协议客户端 | ⏳ 待实现 | stdio/http/sse transport |
| Agent 集成 | ⏳ 待实现 | 工具加载和调用 |
| 前端页面 | ⏳ 待实现 | React 管理界面 |

## 9. 待办事项

- [x] 数据库迁移脚本
- [x] 后端 API 开发
- [ ] MCP 协议客户端实现 (stdio/http/sse transport)
- [ ] Agent 执行流程集成
- [ ] 前端页面开发
- [ ] 集成测试
