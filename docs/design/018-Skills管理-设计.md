# Skills 管理功能设计文档

## 核心设计思路

### 架构设计

采用前后端分离设计：

1. **后端**：提供 REST API，复用已有的 `SkillsLoader` 从文件系统读取技能
2. **前端**：新建 Skills 管理页面，使用 Table + Modal 展示列表和详情

### 数据模型

使用已有的 `SkillInfo` 结构：

```go
type SkillInfo struct {
    Name        string   // 技能名称（目录名）
    Path        string   // SKILL.md 文件路径
    Source      string   // "workspace" 或 "builtin"
    Description string   // 从 YAML frontmatter 解析
}
```

扩展前端展示类型：

```typescript
interface Skill {
  name: string;           // 技能名称
  description: string;    // 技能描述
  source: 'workspace' | 'builtin';  // 来源
  content?: string;       // SKILL.md 完整内容
  boundAgents?: Agent[];  // 绑定的 Agent 列表
}
```

## 影响模块

| 模块 | 变更类型 | 说明 |
|------|----------|------|
| `internal/api` | 新增 | 新增 Skills API 路由和 Handler |
| `internal/service` | 新增 | 新增 Skills Service 层（包装 SkillsLoader）|
| `web/src/pages` | 新增 | 新增 Skills.tsx 管理页面 |
| `web/src/api` | 新增 | 新增 skills.ts API 客户端 |
| `web/src/types` | 新增 | 新增 Skill 类型定义 |
| `web/src/App.tsx` | 修改 | 添加 Skills 路由 |
| `web/src/layouts/MainLayout.tsx` | 修改 | 添加 Skills 菜单项 |

## 变更记录表

| 序号 | 文件路径 | 变更内容 | 优先级 |
|------|----------|----------|--------|
| 1 | `internal/service/skill/service.go` | 新建 Skills Service | P0 |
| 2 | `internal/api/skill_handler.go` | 新建 Skills API Handler | P0 |
| 3 | `internal/api/handler.go` | 添加 SkillsService 接口 | P0 |
| 4 | `internal/api/providers.go` | 初始化 SkillService | P0 |
| 5 | `internal/api/server.go` | 传递 SkillService 参数 | P0 |
| 6 | `internal/repository/agent.go` | 添加 ListAll 方法 | P0 |
| 7 | `web/src/types/index.ts` | 添加 Skill 类型 | P0 |
| 8 | `web/src/api/skills.ts` | 新建 Skills API | P0 |
| 9 | `web/src/api/index.ts` | 导出 Skills API | P0 |
| 10 | `web/src/pages/Skills.tsx` | 新建 Skills 管理页面 | P0 |
| 11 | `web/src/App.tsx` | 添加 Skills 路由 | P0 |
| 12 | `web/src/layouts/MainLayout.tsx` | 添加 Skills 菜单 | P0 |

## API 设计

### GET /api/v1/skills

获取所有 Skills 列表

**响应**：
```json
{
  "items": [
    {
      "name": "git",
      "description": "Git 操作技能",
      "source": "builtin"
    }
  ],
  "total": 1
}
```

### GET /api/v1/skills/:name

获取单个 Skill 详情

**响应**：
```json
{
  "name": "git",
  "description": "Git 操作技能",
  "source": "builtin",
  "content": "...",
  "bound_agents": [
    {"id": 1, "name": "echo-1Agent", "agent_code": "agt_xxx"}
  ]
}
```

## 界面设计

### Skills 列表页

- 表格列：名称、描述、来源、操作
- 操作按钮：查看详情

### Skill 详情弹窗

- 基本信息：名称、描述、来源
- 内容展示：SKILL.md 内容（代码块展示）
- 绑定 Agent：显示使用该技能的 Agent 列表

## 安全考虑

1. Skills 读取限制在工作区和内置目录，防止路径遍历
2. 仅返回 SKILL.md 文件内容，不暴露其他文件
3. 权限控制：需要登录才能访问（复用现有中间件）
