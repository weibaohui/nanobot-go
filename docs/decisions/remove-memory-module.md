# 删除自研记忆模块，改用第三方方案

## 概述

本文档记录删除 nanobot-go 项目中自研的记忆模块，改用开源第三方方案（如 [qmd](https://github.com/tobi/qmd)）的架构决策。

---

## 一、变更原因

### 1.1 自研记忆模块的局限性

当前 `internal/memory/` 模块存在以下问题：

1. **功能局限性**：自研的短期记忆（Stream Memory）和长期记忆（Long Term Memory）功能相对简单，无法满足复杂场景下的记忆检索需求。

2. **维护成本**：需要单独维护记忆存储、索引构建、查询扩展等逻辑，增加开发负担。

3. **可扩展性差**：自研方案在向量检索、混合搜索等方面难以扩展。

### 1.2 第三方方案的优势

以 qmd 为例的开源方案具有以下优势：

1. **成熟的混合检索**：BM25 + 向量搜索 + RRF 融合
2. **内容去重机制**：通过 hash 去重，节省存储
3. **社区支持**：活跃的开源社区，持续迭代
4. **开箱即用**：集成度高，减少自研成本

---

## 二、需要删除的代码模块

### 2.1 后端核心模块

| 目录/文件 | 说明 |
|---------|------|
| `internal/memory/` | 完整记忆模块（handler、job、models、repository、service） |
| `internal/service/memory/` | 记忆服务封装 |
| `internal/service/agent/memory.go` | Agent 记忆 CRUD 操作 |
| `internal/app/memory.go` | 记忆模块初始化 |
| `pkg/agent/tools/config/manage_agent_memory.go` | Agent 记忆管理工具 |

### 2.2 API 处理器

| 文件 | 说明 |
|------|------|
| `internal/api/handler_memory.go` | 记忆 API 处理器 |

### 2.3 依赖修改

| 文件 | 说明 |
|------|------|
| `internal/api/providers.go` | 移除 MemoryService 依赖 |
| `internal/app/gateway.go` | 移除 Memory 初始化 |
| `internal/models/agent.go` | 移除 MemoryContent、MemorySummary 字段 |
| `config/schema.go` | 移除 Memory 配置结构 |
| `config/loader.go` | 移除 Memory 配置加载 |

### 2.4 数据库相关

| 文件 | 说明 |
|------|------|
| `scripts/migrate_drop_user_id.sql` | 检查并移除记忆相关表结构（如 agents 表的 memory 相关字段） |

### 2.5 前端相关

| 文件 | 说明 |
|------|------|
| `web/src/pages/StreamMemories.tsx` | 短期记忆页面 |
| `web/src/pages/LongTermMemories.tsx` | 长期记忆页面 |
| `web/src/api/streamMemories.ts` | 记忆 API 调用 |
| `web/src/types/index.ts` | 移除相关类型定义 |
| `web/src/layouts/MainLayout.tsx` | 移除记忆菜单入口 |

---

## 三、变更影响分析

### 3.1 功能影响

1. **Agent 记忆功能**：Agent 的 MemoryContent 和 MemorySummary 字段将被移除
2. **记忆管理工具**：`manage_agent_memory` 工具将不再可用
3. **记忆 API**：短期记忆和长期记忆的 API 将被移除
4. **记忆升级任务**：定时任务 `MemoryUpgradeJob` 将被移除

### 3.2 数据迁移

在删除记忆模块前，需要：

1. 确认是否有重要数据需要导出
2. 评估是否需要保留数据库表结构（可后续清理）

---

## 四、第三方方案集成说明

删除自研模块后，建议采用以下方式之一集成第三方记忆方案：

### 方案一：qmd（推荐）

```bash
# qmd 安装
go install github.com/tobi/qmd@latest

# 启动 qmd 服务
qmd serve --addr :8080
```

集成方式：
- 通过 HTTP API 与 qmd 交互
- qmd 提供 /collections/{collection}/documents 接口管理记忆
- 查询时使用 /collections/{collection}/query 接口

### 方案二：其他开源方案

根据实际需求，也可考虑：
- [mem0](https://github.com/mem0ai/mem0) - 专为 AI Agent 设计的记忆层
- [AnythingLLM](https://github.com/Mintplex-Labs/anything-llm) - 完整解决方案

---

## 五、实施步骤

### Phase 1: 代码删除

1. 删除 `internal/memory/` 目录
2. 删除 `internal/service/memory/` 目录
3. 删除 `internal/service/agent/memory.go`
4. 删除 `internal/app/memory.go`
5. 删除 `pkg/agent/tools/config/manage_agent_memory.go`
6. 删除 `internal/api/handler_memory.go`

### Phase 2: 引用清理

1. 修改 `internal/api/providers.go` - 移除记忆服务
2. 修改 `internal/app/gateway.go` - 移除 Memory 初始化
3. 修改 `internal/models/agent.go` - 移除记忆字段
4. 修改 `config/schema.go` - 移除 Memory 配置
5. 修改 `config/loader.go` - 移除配置加载
6. 清理 `pkg/agent/context.go` - 移除记忆上下文
7. 清理 `internal/service/agent/types.go` - 移除相关类型

### Phase 3: 前端清理

1. 删除 `web/src/pages/StreamMemories.tsx`
2. 删除 `web/src/pages/LongTermMemories.tsx`
3. 删除 `web/src/api/streamMemories.ts`
4. 修改 `web/src/types/index.ts` - 移除相关类型
5. 修改 `web/src/layouts/MainLayout.tsx` - 移除菜单

### Phase 4: 数据库清理

1. 创建数据库迁移脚本移除记忆相关表和字段
2. 执行数据迁移

---

## 六、回滚计划

如需回滚，可通过以下方式恢复：

1. 从 Git 历史恢复删除的文件
2. 恢复 `internal/models/agent.go` 中的字段定义
3. 恢复 `config/schema.go` 中的配置结构

---

## 七、变更记录表

| 日期 | 变更内容 | 负责人 |
|------|---------|--------|
| 2026-03-19 | 初始文档创建 | AI |
