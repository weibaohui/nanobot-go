# 019 - Task 管理界面设计文档

## 核心设计思路

### 架构设计

采用前后端分离设计：

1. **后端**：新增 Task Service 层，包装 `pkg/agent/task.Manager`，提供 REST API
2. **前端**：新建 Tasks 管理页面，使用 Table + Modal 展示列表和详情

### 数据模型

复用已有的 Task 数据结构：

```go
// task.Info - 任务查询结果
type Info struct {
    ID            string
    Status        Status
    ResultSummary string
}

// task.PersistedTask - 持久化任务结构
type PersistedTask struct {
    ID          string    `yaml:"id"`
    Work        string    `yaml:"work,omitempty"`
    Status      Status    `yaml:"status"`
    Result      string    `yaml:"result,omitempty"`
    Channel     string    `yaml:"channel,omitempty"`
    ChatID      string    `yaml:"chat_id,omitempty"`
    CreatedAt   time.Time `yaml:"created_at"`
    CompletedAt time.Time `yaml:"completed_at,omitempty"`
}
```

扩展 API 响应结构：

```go
// TaskResponse - API 响应结构
type TaskResponse struct {
    ID          string   `json:"id"`
    Status      string   `json:"status"`
    Work        string   `json:"work"`
    Channel     string   `json:"channel,omitempty"`
    ChatID      string   `json:"chat_id,omitempty"`
    CreatedAt   string   `json:"created_at"`
    CompletedAt string   `json:"completed_at,omitempty"`
    Result      string   `json:"result,omitempty"`
    Logs        []string `json:"logs,omitempty"` // 仅详情接口返回
}
```

前端 TypeScript 类型：

```typescript
interface Task {
  id: string;
  status: 'pending' | 'running' | 'finished' | 'failed' | 'stopped';
  work: string;
  channel?: string;
  chat_id?: string;
  created_at: string;
  completed_at?: string;
  result?: string;
  logs?: string[];
}
```

## 影响模块

| 模块 | 变更类型 | 说明 |
|------|----------|------|
| `internal/service/task` | 新增 | 新增 Task Service 层 |
| `internal/api/task_handler.go` | 新增 | 新增 Task API Handler |
| `internal/api/handler.go` | 修改 | 添加 TaskService 接口和字段 |
| `internal/api/providers.go` | 修改 | 初始化 TaskService |
| `internal/api/server.go` | 修改 | 传递 TaskService 参数 |
| `web/src/types/index.ts` | 修改 | 添加 Task 类型 |
| `web/src/api/tasks.ts` | 新增 | 新增 Tasks API 客户端 |
| `web/src/api/index.ts` | 修改 | 导出 Tasks API |
| `web/src/pages/Tasks.tsx` | 新增 | 新增 Tasks 管理页面 |
| `web/src/App.tsx` | 修改 | 添加 Tasks 路由 |
| `web/src/layouts/MainLayout.tsx` | 修改 | 添加 Tasks 菜单项 |

## 变更记录表

| 序号 | 文件路径 | 变更内容 | 优先级 |
|------|----------|----------|--------|
| 1 | `internal/service/task/service.go` | 新建 Task Service | P0 |
| 2 | `internal/api/task_handler.go` | 新建 Task API Handler | P0 |
| 3 | `internal/api/handler.go` | 添加 TaskService 接口 | P0 |
| 4 | `internal/api/providers.go` | 初始化 TaskService | P0 |
| 5 | `internal/api/server.go` | 传递 TaskManager 参数 | P0 |
| 6 | `web/src/types/index.ts` | 添加 Task 类型 | P0 |
| 7 | `web/src/api/tasks.ts` | 新建 Tasks API | P0 |
| 8 | `web/src/api/index.ts` | 导出 Tasks API | P0 |
| 9 | `web/src/pages/Tasks.tsx` | 新建 Tasks 管理页面 | P0 |
| 10 | `web/src/App.tsx` | 添加 Tasks 路由 | P0 |
| 11 | `web/src/layouts/MainLayout.tsx` | 添加 Tasks 菜单 | P0 |

## API 设计

### GET /api/v1/tasks

获取所有任务列表（运行中 + 今日已完成）

**响应**：
```json
{
  "items": [
    {
      "id": "000001",
      "status": "running",
      "work": "分析代码并生成报告",
      "channel": "wechat",
      "chat_id": "chat_123",
      "created_at": "2026-03-17T10:00:00Z",
      "completed_at": null,
      "result": ""
    }
  ],
  "total": 1
}
```

### GET /api/v1/tasks/:id

获取单个任务详情（包含日志）

**响应**：
```json
{
  "id": "000001",
  "status": "running",
  "work": "分析代码并生成报告",
  "channel": "wechat",
  "chat_id": "chat_123",
  "created_at": "2026-03-17T10:00:00Z",
  "completed_at": null,
  "result": "",
  "logs": ["任务已创建", "任务启动", "正在分析..."]
}
```

### POST /api/v1/tasks/:id/stop

停止运行中的任务

**响应**：
```json
{
  "message": "任务已停止",
  "data": {
    "id": "000001",
    "status": "stopped"
  }
}
```

## 界面设计

### Tasks 列表页

- 表格列：ID、状态（带颜色标签）、任务内容、渠道、创建时间、操作
- 操作按钮：查看详情、停止（仅运行中任务显示）
- 顶部：刷新按钮

### Task 详情弹窗

- 基本信息：ID、状态、渠道、ChatID、创建时间、完成时间
- 任务内容：完整的 work 描述
- 执行日志：日志列表
- 执行结果：任务结果摘要

### 状态颜色映射

| 状态 | 颜色 | 标签文字 |
|------|------|----------|
| `pending` | default | 等待中 |
| `running` | processing | 运行中 |
| `finished` | success | 已完成 |
| `failed` | error | 失败 |
| `stopped` | warning | 已停止 |

## Service 层设计

```go
// Service Task 服务接口
type Service interface {
    // ListTasks 获取所有任务列表
    ListTasks() ([]*TaskResponse, error)
    // GetTask 获取任务详情
    GetTask(id string) (*TaskDetailResponse, error)
    // StopTask 停止任务
    StopTask(id string) (*TaskResponse, error)
}
```

## 安全考虑

1. 所有 API 需要登录认证（复用现有 AuthMiddleware）
2. 停止任务前需要用户确认
3. 只能停止运行中的任务，其他状态返回错误提示

## 依赖注入

Task Manager 在应用启动时创建，通过 Providers 结构传递给 API 层：

```go
// Providers 需要新增
TaskManager *task.Manager

// 初始化时
providers.TaskManager = taskManager
```
