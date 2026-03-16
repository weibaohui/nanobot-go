# MCP 服务器接入功能 - 实现总结

## 已实现功能

### 1. 数据库层
- **MCPServer 模型** (`internal/models/mcp_server.go`)
  - 支持 stdio/http/sse 三种传输类型
  - 存储命令、参数、环境变量、URL 等配置
  - 状态管理 (active/inactive/error)
  - 能力列表 (JSON 格式存储工具定义)

- **AgentMCPBinding 模型** (`internal/models/agent_mcp_binding.go`)
  - Agent 与 MCP 服务器的多对多关联
  - 精细化工具控制 (enabled_tools 字段)
  - 启用/禁用开关

### 2. Repository 层
- **MCPServerRepository** (`internal/repository/mcp_server.go`)
  - CRUD 操作
  - 按编码查询
  - 按状态筛选

- **AgentMCPBindingRepository** (`internal/repository/agent_mcp_binding.go`)
  - 绑定 CRUD
  - 按 Agent 查询
  - 按 AgentCode 查询 (关联查询)

### 3. Service 层
- **MCP Service** (`internal/service/mcp/`)
  - 服务器管理: 创建、更新、删除、测试连接
  - 能力刷新: 从 MCP 服务器获取工具列表
  - 绑定管理: Agent 与 MCP 服务器的关联
  - 工具获取: 为 Agent 加载可用的 MCP 工具

### 4. API 层
- **MCP 服务器 API** (`/api/v1/mcp-servers`)
  - `GET` - 列表
  - `POST` - 创建
  - `GET /:id` - 详情
  - `PUT /:id` - 更新
  - `DELETE /:id` - 删除
  - `POST /:id/test` - 测试连接
  - `POST /:id/refresh` - 刷新能力

- **Agent MCP 绑定 API** (`/api/v1/agents/:agent_id/mcp-bindings`)
  - `GET` - 绑定列表
  - `POST` - 创建绑定
  - `GET /:binding_id` - 绑定详情
  - `PUT /:binding_id` - 更新绑定
  - `DELETE /:binding_id` - 删除绑定
  - `GET /tools` - 可用工具列表

## 待实现功能

### 1. MCP 协议客户端
需要实现真正的 MCP 协议通信:
- stdio transport (子进程通信)
- HTTP transport (HTTP 请求)
- SSE transport (Server-Sent Events)
- JSON-RPC 消息封装

### 2. Agent 执行流程集成
- 在 Agent Loop 中加载 MCP 工具
- MCP Tool 包装器实现
- 工具调用路由到对应 MCP 服务器

### 3. 前端页面
- MCP 服务器管理页面
- Agent MCP 配置页面
- 工具选择和启用的交互界面

## 技术要点

### 代码结构
```
internal/
├── models/
│   ├── mcp_server.go          # MCP 服务器模型
│   └── agent_mcp_binding.go   # 绑定关系模型
├── repository/
│   ├── mcp_server.go          # MCP 仓库
│   └── agent_mcp_binding.go   # 绑定仓库
├── service/mcp/
│   ├── types.go               # 类型定义
│   ├── server.go              # 服务器服务
│   └── binding.go             # 绑定服务
└── api/
    ├── handler.go             # Handler 更新
    ├── mcp_handler.go         # MCP API
    ├── providers.go           # Providers 更新
    └── server.go              # Server 更新
```

### 关键设计决策
1. **传输类型抽象**: 支持 stdio/http/sse 三种 MCP 标准传输方式
2. **工具粒度控制**: enabled_tools 为 null 表示全部启用，数组表示白名单
3. **状态管理**: MCP 服务器有独立状态，避免影响 Agent 启动
4. **延迟加载**: 能力列表按需刷新，非实时同步

## API 使用示例

### 创建 MCP 服务器
```bash
curl -X POST http://localhost:8080/api/v1/mcp-servers \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "code": "github",
    "name": "GitHub MCP",
    "transport_type": "stdio",
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-github"],
    "env_vars": {"GITHUB_TOKEN": "xxx"}
  }'
```

### 绑定到 Agent
```bash
curl -X POST http://localhost:8080/api/v1/agents/1/mcp-bindings \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "mcp_server_id": 1,
    "enabled_tools": ["search_issues", "create_issue"]
  }'
```

### 获取 Agent 可用 MCP 工具
```bash
curl http://localhost:8080/api/v1/agents/agent-1/mcp-bindings/tools \
  -H "Authorization: Bearer $TOKEN"
```

## 后续开发建议

1. **MCP 客户端库**: 参考官方 MCP SDK 实现 Go 版本客户端
2. **连接池管理**: MCP stdio 类型需要进程池管理
3. **错误重试**: MCP 调用失败时的重试机制
4. **工具缓存**: 缓存工具 schema 减少重复获取
5. **健康检查**: 定时检查 MCP 服务器状态
