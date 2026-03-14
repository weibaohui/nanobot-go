# 需求 013: 嵌入模型配置合并到 LLMProvider

## 背景

当前项目中存在两种 AI 模型配置方式：
1. **LLMProvider 表** - 存储对话模型配置（API Key、BaseURL、模型列表等）
2. **EmbeddingConfig** - 配置文件中的向量化模型配置

这种分离的配置方式导致：
- 维护复杂度高，需要在多个地方配置 AI 服务
- 环境变量中的 AI Provider 配置实际未被使用
- 嵌入模型的维度等信息分散在不同地方

## 目标

将 Embedding 模型配置合并到 LLMProvider 表中，实现 AI 配置的集中管理，简化系统架构。

## 需求列表

### 1. 数据库模型扩展

**修改文件**: `internal/models/llm_provider.go`

**新增字段**:

| 字段名 | 类型 | 说明 |
|--------|------|------|
| `EmbeddingModels` | string (JSON) | 嵌入模型列表，包含维度信息 |
| `DefaultEmbeddingModel` | string | 默认嵌入模型ID |

**EmbeddingModel 结构**:
```go
type EmbeddingModelInfo struct {
    ID        string `json:"id"`        // 模型ID，如 "text-embedding-3-small"
    Name      string `json:"name"`      // 模型名称
    Dimensions int   `json:"dimensions"` // 向量维度，如 1536
}
```

---

### 2. 配置文件简化

**修改文件**: `config/schema.go`

**移除字段**:
- `EmbeddingConfig.APIKey` - 不再从配置文件读取
- `EmbeddingConfig.BaseURL` - 不再从配置文件读取

**保留字段**:
- `EmbeddingConfig.Enabled` - 是否启用向量化
- `EmbeddingConfig.Model` - 默认模型（可选，留空则从数据库读取）
- `EmbeddingConfig.Dimensions` - 维度（可选，从数据库模型信息读取）

---

### 3. 嵌入配置服务

**新增文件**: `internal/service/embedding/config.go`

**功能需求**:
- 从数据库查询支持嵌入模型的 Provider
- 获取默认嵌入模型的配置（API Key、Base URL、维度）
- 提供与现有 LLM 配置获取类似的接口

**接口设计**:
```go
type EmbeddingConfigService interface {
    // GetEmbeddingConfig 获取嵌入模型配置
    // 优先使用指定模型，否则使用默认模型
    GetEmbeddingConfig(ctx context.Context, modelID string) (*EmbeddingConfig, error)

    // GetDefaultEmbeddingModel 获取默认嵌入模型信息
    GetDefaultEmbeddingModel(ctx context.Context) (*EmbeddingModelInfo, error)

    // ListEmbeddingProviders 列出支持嵌入的 Provider 列表
    ListEmbeddingProviders(ctx context.Context) ([]*LLMProvider, error)
}

type EmbeddingConfig struct {
    APIKey     string
    BaseURL    string
    Model      string
    Dimensions int
}
```

---

### 4. 环境变量清理

**修改文件**: `.env.example`

**移除配置项**:
- `OPENAI_API_KEY`
- `ANTHROPIC_API_KEY`

**理由**: 所有 AI Provider 配置统一从数据库 LLMProvider 表读取

---

### 5. 数据库迁移脚本

**新增文件**: `scripts/migrate_embedding.sql`

**迁移内容**:
```sql
-- 新增字段
ALTER TABLE llm_providers ADD COLUMN embedding_models TEXT;
ALTER TABLE llm_providers ADD COLUMN default_embedding_model TEXT;

-- 示例数据更新
UPDATE llm_providers
SET embedding_models = '[{"id":"text-embedding-3-small","name":"Text Embedding 3 Small","dimensions":1536}]',
    default_embedding_model = 'text-embedding-3-small'
WHERE provider_key = 'openai';
```

---

### 6. 前端适配

**修改文件**: `web/src/types/index.ts`, `web/src/pages/Providers.tsx`

**新增内容**:
- Provider 类型添加 `embedding_models` 和 `default_embedding_model` 字段
- Provider 管理页面支持编辑嵌入模型配置

## 验收标准

- [ ] LLMProvider 模型新增 `embedding_models` 和 `default_embedding_model` 字段
- [ ] 新增 `EmbeddingModelInfo` 结构体，支持维度信息
- [ ] 实现 `EmbeddingConfigService` 接口
- [ ] 简化 `EmbeddingConfig`，移除 APIKey 和 BaseURL 字段
- [ ] 移除 `.env.example` 中的 AI Provider 环境变量配置
- [ ] 创建数据库迁移脚本
- [ ] 前端 Provider 管理支持嵌入模型配置
- [ ] 向量化功能正常从数据库读取配置
- [ ] 向后兼容：配置文件中的旧配置可以平滑迁移

## 优先级

| 需求 | 优先级 | 原因 |
|------|--------|------|
| 数据库模型扩展 | P0 | 基础依赖 |
| 嵌入配置服务 | P0 | 核心功能 |
| 配置文件简化 | P1 | 清理冗余 |
| 环境变量清理 | P1 | 配置统一 |
| 前端适配 | P2 | 管理界面 |
| 迁移脚本 | P2 | 数据迁移 |

## 相关文件

- `internal/models/llm_provider.go`
- `config/schema.go`
- `internal/service/embedding/` (新增)
- `.env.example`
- `web/src/types/index.ts`
- `web/src/pages/Providers.tsx`

## 影响范围

| 模块 | 影响 | 说明 |
|------|------|------|
| 数据库 | 新增字段 | 需要迁移 |
| 配置系统 | 简化 | 移除重复配置 |
| 向量化功能 | 变更 | 从数据库读取配置 |
| 前端管理 | 新增 | Provider 配置扩展 |

## 注意事项

1. **向后兼容**: 已有配置需要迁移，不能破坏现有功能
2. **配置优先级**: 数据库配置优先于配置文件
3. **多 Provider 支持**: 支持多个 Provider 提供嵌入模型
4. **模型选择**: 用户可指定模型，否则使用默认
