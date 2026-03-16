-- SQLite 删除 agents 表 user_id 字段的迁移脚本
-- 由于 SQLite 不支持 DROP COLUMN，需要使用表重建方式

-- 开始事务
BEGIN TRANSACTION;

-- 1. 创建临时表（不包含 user_id 字段）
CREATE TABLE agents_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    agent_code VARCHAR(16) UNIQUE,
    user_code VARCHAR(16),
    name TEXT NOT NULL,
    description TEXT,
    identity_content TEXT,
    soul_content TEXT,
    agents_content TEXT,
    user_content TEXT,
    tools_content TEXT,
    memory_content TEXT,
    memory_summary TEXT,
    skills_list TEXT,
    tools_list TEXT,
    model TEXT,
    max_tokens INTEGER DEFAULT 4096,
    temperature REAL DEFAULT 0.7,
    max_iterations INTEGER DEFAULT 15,
    is_active BOOLEAN DEFAULT true,
    is_default BOOLEAN DEFAULT false,
    enable_thinking_process BOOLEAN DEFAULT false,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 2. 复制数据（跳过 user_id 字段）
INSERT INTO agents_new (
    id, agent_code, user_code, name, description,
    identity_content, soul_content, agents_content, user_content, tools_content,
    memory_content, memory_summary,
    skills_list, tools_list,
    model, max_tokens, temperature, max_iterations,
    is_active, is_default, enable_thinking_process,
    created_at, updated_at
)
SELECT
    id, agent_code, user_code, name, description,
    identity_content, soul_content, agents_content, user_content, tools_content,
    memory_content, memory_summary,
    skills_list, tools_list,
    model, max_tokens, temperature, max_iterations,
    is_active, is_default, enable_thinking_process,
    created_at, updated_at
FROM agents;

-- 3. 删除旧表
DROP TABLE agents;

-- 4. 重命名新表
ALTER TABLE agents_new RENAME TO agents;

-- 5. 创建索引
CREATE INDEX IF NOT EXISTS idx_agents_agent_code ON agents(agent_code);
CREATE INDEX IF NOT EXISTS idx_agents_user_code ON agents(user_code);

-- 提交事务
COMMIT;

-- 验证
-- SELECT sql FROM sqlite_master WHERE type='table' AND name='agents';
