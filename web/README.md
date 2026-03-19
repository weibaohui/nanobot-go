# Nanobot Web 管理界面

基于 React + TypeScript + Ant Design 的前端管理界面。

## 功能模块

- **仪表盘** - 系统概览统计
- **Agent 管理** - 创建、编辑、删除 Agent，配置模型选择模式（auto/specific）
- **渠道管理** - 飞书、钉钉、Matrix、WebSocket 渠道配置
- **LLM 提供商** - API 密钥管理，支持多提供商配置
- **定时任务** - Cron 任务管理，支持自动/指定模型选择
- **用户管理** - 多租户用户管理

## 技术栈

- React 18
- TypeScript
- Ant Design 5
- React Router 6
- Axios

## 开发

```bash
# 安装依赖
npm install

# 启动开发服务器
npm run dev

# 构建
npm run build
```

## 环境变量

```bash
VITE_API_URL=http://localhost:8080/api/v1
```

## 项目结构

```
src/
├── api/           # API 调用
│   ├── client.ts
│   ├── agents.ts
│   ├── channels.ts
│   ├── cron.ts
│   ├── providers.ts
│   └── users.ts
├── components/    # 通用组件
├── layouts/       # 布局组件
│   └── MainLayout.tsx
├── pages/         # 页面组件
│   ├── Dashboard.tsx
│   ├── Agents.tsx
│   ├── Channels.tsx
│   ├── CronJobs.tsx
│   ├── Providers.tsx
│   └── Users.tsx
├── types/         # TypeScript 类型
│   └── index.ts
└── utils/         # 工具函数
```
