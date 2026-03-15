# 入口型 Agent 路由与编排设计文档

## 1. 概述

### 1.1 目标
构建一个"入口型 Agent"（Supervisor Agent），作为 nanobot-go 的统一入口，根据用户输入自动选择并调用不同风格的子 Agent，实现智能路由与编排。

### 1.2 适用场景
- 复杂任务需要多种能力协同
- 需要在简单对话、工具调用、任务规划执行之间智能切换
- 支持人在回路（Human-in-the-Loop）的中断与恢复

## 2. 架构设计

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                     用户消息 (InboundMessage)                 │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Supervisor Agent (入口)                    │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                   意图识别与路由                      │    │
│  │  - 分析用户请求类型                                    │    │
│  │  - 选择合适的子 Agent                                  │    │
│  │  - 编排多 Agent 协作                                   │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│  ReAct Agent  │    │  Plan Agent   │    │  Chat Agent   │
│               │    │               │    │               │
│ - 工具调用     │    │ - 任务规划     │    │ - 闲聊对话    │
│ - 推理链      │    │ - 分步执行     │    │ - 简单问答    │
│ - 长对话      │    │ - 动态重规划   │    │ - 信息查询    │
└───────────────┘    └───────────────┘    └───────────────┘
        │                     │                     │
        └─────────────────────┼─────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     响应输出 (OutboundMessage)                │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 组件说明

#### 2.2.1 Supervisor Agent（监督者）
- **职责**：作为统一入口，理解用户意图，选择并调用合适的子 Agent
- **能力**：
  - 意图识别：分析用户请求的类型和复杂度
  - 智能路由：将请求分发给最合适的子 Agent
  - 编排协调：管理多 Agent 协作流程
  - 结果聚合：汇总子 Agent 的执行结果

#### 2.2.2 ReAct Agent（推理行动型）
- **适用场景**：
  - 需要调用工具的任务（文件操作、网络请求等）
  - 需要多步推理的复杂问题
  - 长对话场景，需要保持上下文
- **特点**：
  - ReAct 模式：推理 → 行动 → 观察 → 再推理
  - 支持工具调用和状态管理
  - 支持中断与恢复

#### 2.2.3 Plan-Execute-Replan Agent（规划执行型）
- **适用场景**：
  - 复杂多步骤任务
  - 需要目标分解和规划
  - 执行过程中可能需要调整计划
- **特点**：
  - Planner：生成执行计划
  - Executor：逐步执行计划步骤
  - Replanner：根据执行结果动态调整计划
  - 支持人在回路的审核与确认

#### 2.2.4 Chat Agent（对话型）
- **适用场景**：
  - 简单闲聊
  - 快速问答
  - 信息查询
- **特点**：
  - 轻量级，快速响应
  - 不涉及复杂工具调用
  - 保持对话流畅性

## 3. 路由决策规则

### 3.1 决策流程

```
用户输入
    │
    ▼
┌─────────────────────────────┐
│ 是否包含复杂任务关键词？      │
│ (规划、帮我完成、分步骤等)    │
└─────────────────────────────┘
    │
    ├── 是 ──→ Plan-Execute-Replan Agent
    │
    └── 否
         │
         ▼
    ┌─────────────────────────────┐
    │ 是否需要工具调用？           │
    │ (文件操作、网络请求等)       │
    └─────────────────────────────┘
         │
         ├── 是 ──→ ReAct Agent
         │
         └── 否 ──→ Chat Agent
```

### 3.2 决策准则

| 用户需求类型 | 推荐子 Agent | 判断依据 |
|------------|-------------|---------|
| 检索、工具调用、推理与长文生成 | ReAct Agent | 包含工具相关关键词，需要多步推理 |
| 目标分解、执行与可能重规划 | Plan-Execute-Replan Agent | 包含规划关键词，多步骤任务 |
| 闲聊或简单问答 | Chat Agent | 简单对话，无需复杂处理 |

### 3.3 关键词分类

**复杂任务关键词（触发 Plan 模式）**：
- 中文：规划、帮我、完成以下任务、帮我完成、请帮我、计划、安排、组织、设计、实现、一步步、分步骤
- 英文：plan, help me, schedule, organize, design, implement, step by step

**工具调用关键词（触发 ReAct 模式）**：
- 文件操作：读取、写入、编辑、创建文件
- 网络操作：搜索、获取网页、下载
- 系统操作：执行命令、运行脚本
- 消息发送：发送消息、通知

## 4. 实现方案

### 4.1 文件结构

```
agent/
├── supervisor.go          # Supervisor Agent 核心实现
├── supervisor_config.go   # Supervisor 配置
├── subagent_react.go      # ReAct 子 Agent
├── subagent_plan.go       # Plan-Execute-Replan 子 Agent
├── subagent_chat.go       # Chat 子 Agent
├── router.go              # 路由决策器
├── loop.go                # 主循环（已修改，集成 Supervisor）
└── ...
```

### 4.2 核心接口设计

```go
// SubAgent 子 Agent 接口
type SubAgent interface {
    // Name 返回 Agent 名称
    Name() string
    // Description 返回 Agent 描述
    Description() string
    // CanHandle 判断是否能处理该请求
    CanHandle(ctx context.Context, input string) bool
    // Execute 执行任务
    Execute(ctx context.Context, input string, history []*schema.Message) (string, error)
    // Stream 流式执行
    Stream(ctx context.Context, input string, history []*schema.Message) (*adk.AsyncIterator[*adk.AgentEvent], error)
}

// SupervisorAgent 监督者 Agent
type SupervisorAgent struct {
    subAgents    []SubAgent
    router       *Router
    model        model.ToolCallingChatModel
    checkpointer compose.CheckPointStore
    logger       *zap.Logger
}

// Router 路由决策器
type Router struct {
    modeSelector *eino_adapter.ModeSelector
    logger       *zap.Logger
}
```

### 4.3 与现有代码集成

#### 4.3.1 Loop 结构体扩展

```go
type Loop struct {
    // ... 现有字段 ...
    
    // Supervisor 入口 Agent
    supervisor *SupervisorAgent
    
    // 子 Agent
    reactAgent  *ReActSubAgent
    planAgent   *PlanSubAgent
    chatAgent   *ChatSubAgent
}
```

#### 4.3.2 消息处理流程

```go
func (l *Loop) processMessage(ctx context.Context, msg *bus.InboundMessage) error {
    // 1. 检查是否有待处理的中断
    if pendingInterrupt := l.interruptManager.GetPendingInterrupt(sessionKey); pendingInterrupt != nil {
        return l.resumeExecution(ctx, msg, pendingInterrupt)
    }
    
    // 2. 通过 Supervisor 处理消息
    return l.supervisor.Process(ctx, msg)
}
```

## 5. 中断与恢复机制

### 5.1 中断类型

| 中断类型 | 触发场景 | 处理方式 |
|---------|---------|---------|
| AskUser | 需要用户输入 | 等待用户响应后恢复 |
| PlanApproval | 计划审核 | 用户确认后执行 |
| ToolConfirm | 敏感操作确认 | 用户确认后执行 |

### 5.2 检查点管理

```go
// 中断信息注册（确保 Gob 序列化正确）
func init() {
    gob.Register(&AskUserInfo{})
    gob.Register(&PlanApprovalInfo{})
    gob.Register(&ToolConfirmInfo{})
}
```

## 6. 配置选项

### 6.1 Supervisor 配置

```go
type SupervisorConfig struct {
    // 最大迭代次数
    MaxIterations int
    // 是否启用流式输出
    EnableStreaming bool
    // 是否启用中断恢复
    EnableInterrupt bool
    // 子 Agent 配置
    ReActConfig  *ReActConfig
    PlanConfig   *PlanConfig
    ChatConfig   *ChatConfig
}
```

### 6.2 路由配置

```go
type RouterConfig struct {
    // 复杂任务关键词
    ComplexKeywords []string
    // 工具调用关键词
    ToolKeywords []string
    // 最小输入长度阈值
    MinInputLength int
    // 最小动作关键词数量
    MinActionCount int
}
```

## 7. 使用示例

### 7.1 基本使用

```go
// 创建 Supervisor
supervisor := NewSupervisorAgent(ctx, &SupervisorConfig{
    Provider: provider,
    Model:    model,
    Tools:    tools,
})

// 处理用户消息
response, err := supervisor.Process(ctx, &bus.InboundMessage{
    Channel: "whatsapp",
    ChatID:  "123456",
    Content: "帮我规划一次日本旅行",
})
```

### 7.2 路由示例

```go
// 示例 1: 复杂任务 → Plan Agent
input := "帮我规划一个项目，包括需求分析、设计、开发和测试"
// 路由到 Plan-Execute-Replan Agent

// 示例 2: 工具调用 → ReAct Agent
input := "读取 config.yaml 文件并修改其中的数据库配置"
// 路由到 ReAct Agent

// 示例 3: 简单对话 → Chat Agent
input := "你好，今天天气怎么样？"
// 路由到 Chat Agent
```

## 8. 扩展性设计

### 8.1 添加新的子 Agent

```go
// 1. 实现 SubAgent 接口
type CustomSubAgent struct {
    // ...
}

func (a *CustomSubAgent) Name() string { return "custom_agent" }
func (a *CustomSubAgent) Description() string { return "自定义 Agent" }
func (a *CustomSubAgent) CanHandle(ctx context.Context, input string) bool {
    // 自定义判断逻辑
    return strings.Contains(input, "特定关键词")
}
func (a *CustomSubAgent) Execute(ctx context.Context, input string, history []*schema.Message) (string, error) {
    // 自定义执行逻辑
    return "", nil
}

// 2. 注册到 Supervisor
supervisor.RegisterSubAgent(&CustomSubAgent{})
```

### 8.2 自定义路由规则

```go
// 自定义路由决策器
type CustomRouter struct {
    *Router
}

func (r *CustomRouter) Route(ctx context.Context, input string) SubAgent {
    // 自定义路由逻辑
    if strings.Contains(input, "特殊任务") {
        return r.customAgent
    }
    return r.Router.Route(ctx, input)
}
```

## 9. 性能考虑

### 9.1 资源管理
- 子 Agent 按需创建，避免预加载所有 Agent
- 使用连接池管理 LLM 调用
- 合理设置超时和重试机制

### 9.2 并发控制
- 限制并发执行的 Agent 数量
- 使用 context 进行超时控制
- 实现优雅的取消机制

## 10. 监控与日志

### 10.1 关键指标
- 路由决策时间
- 各子 Agent 调用频率
- 任务执行成功率
- 中断恢复次数

### 10.2 日志记录
```go
logger.Info("路由决策",
    zap.String("input", input),
    zap.String("selected_agent", agent.Name()),
    zap.Duration("decision_time", duration),
)
```

## 11. 测试策略

### 11.1 单元测试
- 路由决策测试
- 各子 Agent 独立测试
- 中断恢复测试

### 11.2 集成测试
- 端到端消息处理流程
- 多 Agent 协作场景
- 异常情况处理

## 12. 迁移计划

### 12.1 阶段一：基础实现
1. 实现 Supervisor 核心结构
2. 实现三个基础子 Agent
3. 集成到现有 Loop

### 12.2 阶段二：功能完善
1. 完善路由决策算法
2. 添加中断恢复机制
3. 优化性能

### 12.3 阶段三：扩展增强
1. 支持自定义子 Agent
2. 添加监控指标
3. 完善文档和示例
