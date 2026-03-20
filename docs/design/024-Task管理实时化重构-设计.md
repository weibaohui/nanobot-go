# 024-Task管理实时化重构-设计

# 0. 文件修改记录表

| 修改人 | 修改时间 | 修改内容 |
| ------ | -------- | -------- |
| AI Assistant | 2026-03-19 | 初始版本 |

# 1. 设计概述

## 1.1 核心设计思想

本设计采用**事件驱动架构**，通过WebSocket实现Task状态变更的实时推送：

1. **消息总线统一路由**：所有Task状态变更事件通过MessageBus路由到WebSocket Handler
2. **增量更新策略**：只推送变更的数据，不推送完整列表
3. **前后端状态同步**：前端维护任务列表状态，WebSocket消息驱动状态更新
4. **渐进式增强**：WebSocket为增量功能，HTTP API保持完整向后兼容

## 1.2 架构全景

```
┌─────────────────────────────────────────────────────────────────────┐
│                            Frontend                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐               │
│  │   Tasks.tsx  │  │ useWebSocket │  │  TaskStore   │               │
│  │   (UI组件)    │  │   (Hook)     │  │  (状态管理)   │               │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘               │
│         │                 │                 │                        │
│         └─────────────────┴─────────────────┘                        │
│                           │                                          │
│                    ┌──────┴──────┐                                   │
│                    │  WebSocket  │                                   │
│                    │  Client     │                                   │
│                    └──────┬──────┘                                   │
└───────────────────────────┼─────────────────────────────────────────┘
                            │ WS Messages
┌───────────────────────────┼─────────────────────────────────────────┐
│                           │         Backend                          │
│                    ┌──────┴──────┐                                   │
│                    │  WebSocket  │                                   │
│                    │  Handler    │                                   │
│                    └──────┬──────┘                                   │
│                           │                                          │
│  ┌────────────────────────┼──────────────────────────────────────┐  │
│  │                        ▼          MessageBus                  │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐      │  │
│  │  │Channel   │  │  Task    │  │   Log    │  │  Other   │      │  │
│  │  │Events    │  │ Events   │  │  Events  │  │  Events  │      │  │
│  │  └──────────┘  └────┬─────┘  └──────────┘  └──────────┘      │  │
│  │                     │                                         │  │
│  └─────────────────────┼─────────────────────────────────────────┘  │
│                        │                                             │
│                 ┌──────┴──────┐                                      │
│                 │Task Manager │                                      │
│                 │  (核心逻辑)  │                                      │
│                 └──────┬──────┘                                      │
│                        │                                             │
│                 ┌──────┴──────┐                                      │
│                 │Persistence  │                                      │
│                 │ (YAML存储)  │                                      │
│                 └─────────────┘                                      │
└──────────────────────────────────────────────────────────────────────┘
```

# 2. 核心模块设计

## 2.1 后端模块

### 2.1.1 WebSocket消息路由扩展

**文件**: `internal/api/websocket.go`

扩展现有WebSocket Handler，支持按消息类型路由：

```go
// WebSocketMessageType 消息类型枚举
type WebSocketMessageType string

const (
    // 现有类型
    MsgTypePing    WebSocketMessageType = "ping"
    MsgTypePong    WebSocketMessageType = "pong"
    MsgTypeMessage WebSocketMessageType = "message"
    MsgTypeChunk   WebSocketMessageType = "chunk"
    MsgTypeError   WebSocketMessageType = "error"
    MsgTypeSystem  WebSocketMessageType = "system"

    // 新增Task相关类型
    MsgTypeTaskCreated   WebSocketMessageType = "task_created"
    MsgTypeTaskUpdated   WebSocketMessageType = "task_updated"
    MsgTypeTaskCompleted WebSocketMessageType = "task_completed"
    MsgTypeTaskLog       WebSocketMessageType = "task_log"
)
```

### 2.1.2 Task事件发布器

**文件**: `pkg/agent/task/publisher.go`（新增）

```go
// EventPublisher Task事件发布器
type EventPublisher struct {
    bus *bus.MessageBus
}

// PublishTaskCreated 发布任务创建事件
func (p *EventPublisher) PublishTaskCreated(task *Task)

// PublishTaskUpdated 发布任务更新事件
func (p *EventPublisher) PublishTaskUpdated(task *Task)

// PublishTaskCompleted 发布任务完成事件
func (p *EventPublisher) PublishTaskCompleted(task *Task, duration time.Duration)

// PublishTaskLog 发布任务日志事件（可选）
func (p *EventPublisher) PublishTaskLog(taskID string, log string)
```

### 2.1.3 Task Manager集成事件发布

**文件**: `pkg/agent/task/manager.go`

修改`runTask`方法，在关键节点发布事件：

```go
func (m *Manager) runTask(ctx context.Context, task *Task, channel, chatID string) {
    // ... 现有逻辑 ...

    // 任务启动时发布创建事件
    if m.publisher != nil {
        m.publisher.PublishTaskCreated(task)
    }

    // 任务执行中，定期发布更新事件（可选，用于长任务）
    // 或使用钩子机制在日志追加时发布

    // 任务结束时发布完成事件
    if m.publisher != nil {
        m.publisher.PublishTaskCompleted(task, duration)
    }
}
```

### 2.1.4 扩展Task Service

**文件**: `internal/service/task/service.go`

新增方法：

```go
// CreateTask 手动创建任务
func (s *service) CreateTask(work string, createdBy string) (*TaskResponse, error)

// RetryTask 重试任务
func (s *service) RetryTask(id string, createdBy string) (*TaskResponse, error)

// ListTasksWithFilter 带筛选的任务列表
func (s *service) ListTasksWithFilter(filter TaskFilter) ([]*TaskResponse, error)

// TaskFilter 任务筛选条件
type TaskFilter struct {
    Status    []string  // 状态列表
    Since     time.Time // 起始时间
    Until     time.Time // 结束时间
    Keyword   string    // 关键词
    CreatedBy string    // 创建者
    IsAdmin   bool      // 是否管理员（决定CreatedBy是否生效）
}
```

### 2.1.5 扩展API Handler

**文件**: `internal/api/task_handler.go`

新增接口：

```go
// createTask 手动创建任务
// POST /api/v1/tasks
func (h *Handler) createTask(c *gin.Context)

// retryTask 重试任务
// POST /api/v1/tasks/:id/retry
func (h *Handler) retryTask(c *gin.Context)

// 扩展listTasks，支持查询参数解析
// GET /api/v1/tasks?status=running&since=2026-03-18&keyword=xxx
func (h *Handler) listTasks(c *gin.Context)
```

## 2.2 前端模块

### 2.2.1 扩展WebSocket Hook

**文件**: `web/src/hooks/useWebSocket.ts`

扩展消息类型定义：

```typescript
export type WebSocketMessageType =
  | 'ping' | 'pong' | 'message' | 'chunk' | 'error' | 'system'
  | 'task_created' | 'task_updated' | 'task_completed' | 'task_log';

// Task消息Payload类型
export interface TaskCreatedPayload {
  id: string;
  status: TaskStatus;
  work: string;
  channel?: string;
  chat_id?: string;
  created_at: string;
  created_by?: string;
}

export interface TaskUpdatedPayload {
  id: string;
  status: TaskStatus;
  result?: string;
  logs?: string[];
  updated_at: string;
}

export interface TaskCompletedPayload {
  id: string;
  status: 'finished' | 'failed' | 'stopped';
  result?: string;
  logs?: string[];
  completed_at: string;
  duration_seconds: number;
}

// Hook选项扩展
export interface UseWebSocketOptions {
  channelCode: string;
  token: string;
  enabled?: boolean;
  // 新增Task消息回调
  onTaskCreated?: (payload: TaskCreatedPayload) => void;
  onTaskUpdated?: (payload: TaskUpdatedPayload) => void;
  onTaskCompleted?: (payload: TaskCompletedPayload) => void;
  onMessage?: (msg: WebSocketMessage) => void;
  onError?: (error: Error) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
}
```

### 2.2.2 Task状态管理

**文件**: `web/src/store/taskStore.ts`（新增，可选）

使用Zustand或React Context管理Task列表状态：

```typescript
interface TaskState {
  tasks: Task[];
  filters: {
    status: TaskStatus[];
    keyword: string;
    timeRange: 'today' | 'week' | 'month' | 'all';
  };

  // Actions
  setTasks: (tasks: Task[]) => void;
  addTask: (task: Task) => void;
  updateTask: (id: string, updates: Partial<Task>) => void;
  setFilters: (filters: Partial<TaskState['filters']>) => void;
  getFilteredTasks: () => Task[];
}
```

### 2.2.3 Tasks页面重构

**文件**: `web/src/pages/Tasks.tsx`

组件结构：

```typescript
const Tasks: React.FC = () => {
  // 状态
  const [tasks, setTasks] = useState<Task[]>([]);
  const [filters, setFilters] = useState<FilterState>(...);
  const [createModalVisible, setCreateModalVisible] = useState(false);

  // WebSocket连接
  const { isConnected } = useWebSocket({
    channelCode: 'system', // 使用系统channel接收所有任务事件
    token: getToken(),
    onTaskCreated: handleTaskCreated,
    onTaskUpdated: handleTaskUpdated,
    onTaskCompleted: handleTaskCompleted,
  });

  // 事件处理
  const handleTaskCreated = (payload) => {
    setTasks(prev => [payload, ...prev]);
  };

  const handleTaskUpdated = (payload) => {
    setTasks(prev => prev.map(t =>
      t.id === payload.id ? { ...t, ...payload } : t
    ));
  };

  const handleTaskCompleted = (payload) => {
    setTasks(prev => prev.map(t =>
      t.id === payload.id ? { ...t, ...payload } : t
    ));
    notification.info({ message: `任务 ${payload.id} ${getStatusText(payload.status)}` });
  };

  // 渲染
  return (
    <div>
      <TaskFilterBar filters={filters} onChange={setFilters} />
      <TaskToolbar onCreate={() => setCreateModalVisible(true)} />
      <TaskTable
        tasks={filteredTasks}
        onStop={handleStopTask}
        onRetry={handleRetryTask}
        onViewDetail={handleViewDetail}
      />
      <CreateTaskModal
        visible={createModalVisible}
        onCancel={() => setCreateModalVisible(false)}
        onSubmit={handleCreateTask}
      />
    </div>
  );
};
```

### 2.2.4 组件拆分

**文件**: `web/src/components/tasks/`（新增目录）

```
tasks/
├── TaskFilterBar.tsx      # 筛选栏（搜索、状态、时间）
├── TaskToolbar.tsx        # 工具栏（新建按钮、刷新）
├── TaskTable.tsx          # 任务列表表格
├── TaskStatusTag.tsx      # 状态标签
├── CreateTaskModal.tsx    # 创建任务弹窗
├── TaskDetailModal.tsx    # 任务详情弹窗
├── TaskLogViewer.tsx      # 日志查看器（支持实时流）
└── index.ts               # 导出
```

# 3. 数据流设计

## 3.1 任务创建数据流

```
用户点击"新建任务"
    │
    ▼
前端调用 POST /api/v1/tasks
    │
    ▼
后端创建任务，启动执行
    │
    ▼
后端发布 task_created 事件到 MessageBus
    │
    ▼
WebSocket Handler 广播消息
    │
    ▼
所有连接的客户端收到 task_created
    │
    ▼
前端添加到列表顶部
```

## 3.2 任务状态更新数据流

```
Task Agent 执行中
    │
    ▼
Task Manager 更新状态/日志
    │
    ▼
后端发布 task_updated 事件
    │
    ▼
WebSocket Handler 广播消息
    │
    ▼
前端更新对应任务状态
    │
    ▼
UI 实时反映变化（状态标签、日志）
```

## 3.3 任务完成数据流

```
Task Agent 执行完成/失败/停止
    │
    ▼
Task Manager 设置最终状态
    │
    ▼
后端发布 task_completed 事件
    │
    ▼
WebSocket Handler 广播消息
    │
    ▼
前端更新状态 + 显示通知
```

# 4. 接口详细设计

## 4.1 后端API变更

### 现有接口变更

| 接口 | 变更 | 说明 |
|------|------|------|
| GET /api/v1/tasks | 新增查询参数 | status, since, until, keyword |
| GET /api/v1/tasks/:id | 无变更 | 保持兼容 |
| POST /api/v1/tasks/:id/stop | 无变更 | 保持兼容 |

### 新增接口

| 接口 | 方法 | 说明 |
|------|------|------|
| POST /api/v1/tasks | POST | 手动创建任务 |
| POST /api/v1/tasks/:id/retry | POST | 重试任务 |

## 4.2 WebSocket消息格式

详见需求文档第8节。

关键设计决策：
- `task_updated` 只推送变更字段，不推送完整任务对象
- `task_completed` 包含执行时长（duration_seconds），便于展示
- `task_log` 可选实现，用于大日志任务的实时流式展示

# 5. 状态管理设计

## 5.1 前端状态结构

```typescript
// 任务列表状态
interface TasksState {
  // 数据
  tasks: Task[];
  loading: boolean;
  error: string | null;

  // 筛选
  filters: {
    keyword: string;
    status: TaskStatus[];
    timeRange: 'today' | 'week' | 'month' | 'all';
  };

  // 分页（如需要）
  pagination: {
    page: number;
    pageSize: number;
    total: number;
  };

  // 选中/展开
  selectedTaskId: string | null;
  expandedTaskIds: string[];
}

// 派生状态（通过selector计算）
const filteredTasks = (state: TasksState) => {
  return state.tasks
    .filter(t => matchesKeyword(t, state.filters.keyword))
    .filter(t => matchesStatus(t, state.filters.status))
    .filter(t => matchesTimeRange(t, state.filters.timeRange));
};
```

## 5.2 状态更新策略

| 场景 | 更新方式 | 说明 |
|------|----------|------|
| 初始加载 | HTTP GET | 获取当前所有任务 |
| 新任务创建 | WebSocket push | 前置插入到列表 |
| 任务更新 | WebSocket push | 替换对应任务数据 |
| 手动刷新 | HTTP GET | 重新获取完整列表 |
| 筛选条件变化 | 前端计算 | 不请求后端，本地过滤 |

# 6. 错误处理策略

## 6.1 WebSocket错误处理

```typescript
// 连接断开
onDisconnect: () => {
  // 显示连接中断提示
  // 自动重连由Hook内部处理
  message.warning('连接中断，正在重连...');
}

// 重连成功
onConnect: () => {
  // 刷新数据以获取期间可能丢失的更新
  fetchTasks();
}
```

## 6.2 API错误处理

| 场景 | 错误码 | 前端处理 |
|------|--------|----------|
| 任务不存在 | 404 | 显示"任务不存在或已删除" |
| 无权限停止 | 403 | 显示"只能停止自己创建的任务" |
| 任务已完成 | 400 | 提示"任务已结束，无需停止" |
| 服务器错误 | 500 | 显示"操作失败，请重试" |

# 7. 性能优化策略

## 7.1 列表性能

- **虚拟滚动**：如果任务数量超过100，使用react-window实现虚拟滚动
- **增量更新**：WebSocket只推送变更，不重渲染整个列表
- **防抖筛选**：关键词搜索使用防抖（300ms）

## 7.2 WebSocket性能

- **消息节流**：任务更新消息最多1秒推送一次（避免高频日志导致消息过多）
- **批量推送**：后端可以累积变更，定期批量推送
- **连接池管理**：限制单用户连接数

## 7.3 日志性能

- **懒加载**：任务详情Modal打开时才获取完整日志
- **分页加载**：日志支持分页，不一次性加载全部
- **实时日志流**：可选WebSocket推送或使用Server-Sent Events

# 8. 安全设计

## 8.1 权限控制

| 操作 | 权限要求 |
|------|----------|
| 查看任务列表 | 登录用户 |
| 查看他人任务 | 仅管理员 |
| 创建任务 | 登录用户 |
| 停止任务 | 任务创建者或管理员 |
| 重试任务 | 任务创建者或管理员 |

## 8.2 后端校验

```go
func (h *Handler) stopTask(c *gin.Context) {
    taskID := c.Param("id")
    user := getCurrentUser(c) // 从JWT获取

    task := h.taskService.GetTask(taskID)
    if task == nil {
        c.JSON(404, ...)
        return
    }

    // 权限校验
    if !user.IsAdmin && task.CreatedBy != user.ID {
        c.JSON(403, ErrorResponse{Error: "无权操作此任务"})
        return
    }

    // ...
}
```

# 9. 测试策略

## 9.1 单元测试

- Task Manager事件发布逻辑
- WebSocket消息路由
- 前端Hook消息处理

## 9.2 集成测试

- 创建任务→WebSocket推送→前端更新 完整链路
- 筛选条件→API响应 正确性
- 权限控制 边界情况

## 9.3 E2E测试

使用Playwright测试：
1. 用户登录→创建任务→验证任务出现
2. 多个浏览器窗口→验证实时同步
3. 筛选功能→验证结果正确

# 10. 变更记录表

| 模块 | 文件路径 | 变更类型 | 变更内容 |
|------|----------|----------|----------|
| Backend | `pkg/agent/task/publisher.go` | 新增 | Task事件发布器 |
| Backend | `pkg/agent/task/manager.go` | 修改 | 集成事件发布 |
| Backend | `internal/service/task/service.go` | 修改 | 新增创建/重试/筛选方法 |
| Backend | `internal/api/task_handler.go` | 修改 | 新增API端点 |
| Backend | `internal/api/websocket.go` | 修改 | 扩展消息类型定义 |
| Frontend | `web/src/hooks/useWebSocket.ts` | 修改 | 扩展Task消息类型和回调 |
| Frontend | `web/src/pages/Tasks.tsx` | 修改 | 重构，接入WebSocket |
| Frontend | `web/src/api/tasks.ts` | 修改 | 新增API调用方法 |
| Frontend | `web/src/components/tasks/*` | 新增 | 组件拆分 |
| Frontend | `web/src/store/taskStore.ts` | 新增 | 状态管理（可选） |

# 11. 开发顺序与里程碑

## 阶段1：后端基础（2-3天）
1. 创建Task事件发布器
2. Task Manager集成事件发布
3. WebSocket Handler支持Task消息广播

## 阶段2：后端API（1-2天）
1. 实现手动创建任务接口
2. 实现重试任务接口
3. 扩展列表接口支持筛选

## 阶段3：前端基础（2天）
1. 扩展useWebSocket支持Task消息
2. 重构Tasks页面结构
3. 组件拆分

## 阶段4：前端功能（2天）
1. 实现实时更新逻辑
2. 实现筛选功能
3. 实现手动创建任务
4. 实现重试功能
5. 添加通知

## 阶段5：测试与优化（2天）
1. 单元测试
2. 集成测试
3. E2E测试
4. 性能优化

总计：9-11天

# 12. 风险与回滚策略

## 风险点
1. WebSocket消息过多导致性能问题
2. 向后兼容性破坏

## 缓解措施
1. 实现消息节流和批量推送
2. HTTP API保持完全兼容
3. 使用Feature Flag控制新功能上线

## 回滚策略
如果WebSocket方案出现问题：
1. 关闭WebSocket推送（回退到HTTP轮询）
2. 前端降级为定时轮询（5秒间隔）
3. 保留所有功能（只是非实时）
