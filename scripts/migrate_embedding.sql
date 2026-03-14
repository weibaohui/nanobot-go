-- 迁移脚本: 将嵌入模型配置合并到 LLMProvider 表
-- 执行时间: 2024-XX-XX

-- 步骤1: 新增字段到 llm_providers 表
ALTER TABLE llm_providers
    ADD COLUMN embedding_models TEXT,
    ADD COLUMN default_embedding_model TEXT;

-- 步骤2: 为现有 Provider 添加默认嵌入模型配置
-- OpenAI 支持的嵌入模型
UPDATE llm_providers
SET
    embedding_models = '[{"id":"text-embedding-3-small","name":"Text Embedding 3 Small","dimensions":1536},{"id":"text-embedding-3-large","name":"Text Embedding 3 Large","dimensions":3072}]',
    default_embedding_model = 'text-embedding-3-small'
WHERE provider_key = 'openai';

-- 其他 Provider 可以按需添加
-- 例如: Ollama 本地嵌入模型
UPDATE llm_providers
SET
    embedding_models = '[{"id":"nomic-embed-text","name":"Nomic Embed Text","dimensions":768},{"id":"mxbai-embed-large","name":"MXBAI Embed Large","dimensions":1024}]',
    default_embedding_model = 'nomic-embed-text'
WHERE provider_key = 'ollama';

-- Anthropic 暂不支持嵌入模型，设置为空数组
UPDATE llm_providers
SET
    embedding_models = '[]',
    default_embedding_model = ''
WHERE provider_key = 'anthropic';

-- 步骤3: 创建索引（可选，提升查询性能）
-- 注意: SQLite 支持部分索引的语法可能有所不同，请根据实际数据库类型调整
CREATE INDEX IF NOT EXISTS idx_llm_providers_embedding
    ON llm_providers(embedding_models)
    WHERE embedding_models IS NOT NULL AND embedding_models != '' AND embedding_models != '[]';

-- 验证: 查看配置结果
-- SELECT id, provider_key, provider_name, default_embedding_model, embedding_models FROM llm_providers;
