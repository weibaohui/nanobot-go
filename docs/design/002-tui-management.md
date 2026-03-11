# TUI 管理系统设计文档

## 1. 需求概述

### 1.1 目标
为 nanobot-go 提供一个终端用户界面（TUI），用于管理核心数据：用户、Agent、Channel、Session。

### 1.2 功能范围
- **用户管理**: 查看、创建、编辑、删除用户
- **Agent 管理**: 查看、创建、编辑、删除 Agent，配置 Agent 参数
- **Channel 管理**: 查看、创建、编辑、删除 Channel，绑定/解绑 Agent
- **Session 管理**: 查看会话列表，查看会话详情和元数据

### 1.3 非功能需求
- 纯键盘操作，无需鼠标
- 界面响应流畅，支持实时刷新
- 数据操作有确认机制，防止误操作
- 错误提示清晰，操作有反馈

## 2. 技术选型

### 2.1 核心库
- **Bubble Tea**: Elm 架构的 TUI 框架
- **Lipgloss**: 样式和布局
- **Bubbles**: 现成组件（list, textinput, textarea, help, keybindings）

### 2.2 测试策略
- **单元测试**: 测试 Model/Update/View 逻辑（使用 bubble tea 的 test 包）
- **集成测试**: 使用 `expect` 类工具或模拟用户输入测试完整流程
- **组件测试**: 单独测试各个页面组件

### 2.3 项目结构
```
cmd/tui/
├── main.go                 # 入口

internal/tui/
├── app.go                  # 应用主结构和路由
├── styles.go               # 共享样式定义
├── keys.go                 # 快捷键定义
├── components/
│   ├── header.go           # 顶部标题栏
│   ├── sidebar.go          # 左侧导航栏
│   ├── statusbar.go        # 底部状态栏
│   ├── confirm.go          # 确认对话框
│   ├── form.go             # 通用表单组件
│   └── table.go            # 表格组件
├── pages/
│   ├── dashboard.go        # 首页仪表盘
│   ├── user/
│   │   ├── list.go         # 用户列表
│   │   ├── form.go         # 用户表单（创建/编辑）
│   │   └── detail.go       # 用户详情
│   ├── agent/
│   │   ├── list.go         # Agent 列表
│   │   ├── form.go         # Agent 表单
│   │   ├── config.go       # Agent 配置编辑
│   │   └── detail.go       # Agent 详情
│   ├── channel/
│   │   ├── list.go         # Channel 列表
│   │   ├── form.go         # Channel 表单
│   │   └── bind.go         # 绑定 Agent 界面
│   └── session/
│       ├── list.go         # Session 列表
│       └── detail.go       # Session 详情
├── client/
│   └── api.go              # HTTP API 客户端
└── tests/
    ├── app_test.go         # 应用级测试
    ├── pages_test.go       # 页面测试
    └── integration_test.go # 集成测试
```

## 3. 界面设计

### 3.1 整体布局
```
┌─────────────────────────────────────────────────────────────┐
│  🐈 nanobot TUI Management              [Tab] Navigate      │  Header
├──────────┬──────────────────────────────────────────────────┤
│          │                                                  │
│  🏠 Home │        Dashboard / Content Area                  │
│  👤 Users│                                                  │
│  🤖 Agent│                                                  │
│  📡 Chan │                                                  │
│  💬 Sess │                                                  │
│          │                                                  │
│          │                                                  │
├──────────┴──────────────────────────────────────────────────┤
│  ↑↓: Navigate  Enter: Select  q: Quit  ?: Help              │  StatusBar
└─────────────────────────────────────────────────────────────┘
```

### 3.2 导航设计
- **Tab / Shift+Tab**: 在 Sidebar 和 Content 之间切换
- **↑ / ↓ / k / j**: 列表上下导航
- **Enter**: 确认/进入
- **Esc / q**: 返回/退出
- **?**: 显示帮助

### 3.3 快捷键定义

全局快捷键：
| 按键 | 功能 |
|------|------|
| `q` | 退出/返回 |
| `?` | 显示帮助 |
| `Tab` | 切换焦点 |
| `1-4` | 直接跳转到对应页面 |
| `r` | 刷新数据 |

列表页面快捷键：
| 按键 | 功能 |
|------|------|
| `n` | 新建 |
| `e` | 编辑选中项 |
| `d` | 删除选中项 |
| `/` | 搜索过滤 |

表单页面快捷键：
| 按键 | 功能 |
|------|------|
| `Tab` | 下一个字段 |
| `Shift+Tab` | 上一个字段 |
| `Ctrl+S` | 保存 |
| `Esc` | 取消 |

## 4. 页面详细设计

### 4.1 Dashboard 页面
显示系统概览：
- 总用户数
- 总 Agent 数
- 总 Channel 数
- 活跃 Session 数
- 最近活动日志

### 4.2 用户列表页面
表格列：ID, 用户名, API Key(隐藏), 创建时间, 操作
- 支持搜索过滤
- 选中后按 `e` 编辑，`d` 删除

### 4.3 Agent 列表页面
表格列：ID, 名称, 所属用户, 模型, 是否默认, 状态
- 支持搜索过滤
- 选中后按 `Enter` 查看详情
- `e` 编辑，`d` 删除，`c` 配置

### 4.4 Channel 列表页面
表格列：ID, 名称, 类型, 所属用户, 绑定 Agent, 状态
- 支持搜索过滤
- `b` 绑定/解绑 Agent

### 4.5 Session 列表页面
表格列：Session Key, 所属 Channel, Agent, 最后活跃时间
- 只读界面，支持查看详情

## 5. 测试计划

### 5.1 单元测试
- 测试每个页面的 Model 初始化
- 测试 Update 函数处理各种 Msg
- 测试 View 函数渲染输出

### 5.2 集成测试
- 使用 `tea.Program` 的测试模式
- 模拟用户输入序列，验证最终状态
- 测试数据流的完整性

### 5.3 测试示例
```go
// 测试用户列表页面
func TestUserList_Update(t *testing.T) {
    model := NewUserListPage()

    // 测试加载数据
    model = model.LoadUsers(mockUsers)
    assert.Equal(t, 2, len(model.table.Rows()))

    // 测试选择变化
    msg := tea.KeyMsg{Type: tea.KeyDown}
    newModel, _ := model.Update(msg)
    assert.Equal(t, 1, newModel.(UserListPage).cursor)
}
```

## 6. 实现计划

### Phase 1: 基础框架
1. 创建项目结构
2. 实现基础组件（Header, Sidebar, StatusBar）
3. 实现页面路由机制
4. 实现 API 客户端

### Phase 2: Dashboard 页面
1. 实现数据概览展示
2. 集成 API 获取统计数据

### Phase 3: 用户管理
1. 用户列表页面
2. 用户表单（创建/编辑）
3. 用户删除确认

### Phase 4: Agent 管理
1. Agent 列表页面
2. Agent 表单
3. Agent 配置编辑

### Phase 5: Channel 管理
1. Channel 列表页面
2. Channel 表单
3. Agent 绑定功能

### Phase 6: Session 管理
1. Session 列表页面
2. Session 详情查看

### Phase 7: 测试与优化
1. 编写单元测试
2. 编写集成测试
3. 性能优化和 Bug 修复
