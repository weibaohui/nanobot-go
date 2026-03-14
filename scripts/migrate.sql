-- 数据迁移脚本：为旧数据添加 Code 字段
-- 注意：此脚本假设 Code 字段已存在（nullable）

-- 为用户生成 Code
UPDATE users SET user_code = 'usr_' || substr(hex(randomblob(8)), 1, 10) WHERE user_code IS NULL OR user_code = '';

-- 为 Agents 生成 Code
UPDATE agents SET agent_code = 'agt_' || substr(hex(randomblob(8)), 1, 10) WHERE agent_code IS NULL OR agent_code = '';

-- 为 Agents 设置 user_code（基于 user_id）
UPDATE agents SET user_code = (SELECT user_code FROM users WHERE users.id = agents.user_id) WHERE user_code IS NULL OR user_code = '';

-- 为 Channels 生成 Code
UPDATE channels SET channel_code = 'chn_' || substr(hex(randomblob(8)), 1, 10) WHERE channel_code IS NULL OR channel_code = '';

-- 为 Channels 设置 user_code（基于 user_id）
UPDATE channels SET user_code = (SELECT user_code FROM users WHERE users.id = channels.user_id) WHERE user_code IS NULL OR user_code = '';

-- 为 Channels 设置 agent_code（基于 agent_id）
UPDATE channels SET agent_code = (SELECT agent_code FROM agents WHERE agents.id = channels.agent_id) WHERE agent_code IS NULL OR agent_code = '';

-- 为 Providers 设置 user_code
UPDATE llm_providers SET user_code = (SELECT user_code FROM users WHERE users.id = llm_providers.user_id) WHERE user_code IS NULL OR user_code = '';

-- 为 Sessions 设置 user_code, channel_code, agent_code
UPDATE sessions SET user_code = (SELECT user_code FROM users WHERE users.id = sessions.user_id) WHERE user_code IS NULL OR user_code = '';
UPDATE sessions SET channel_code = (SELECT channel_code FROM channels WHERE channels.id = sessions.channel_id) WHERE channel_code IS NULL OR channel_code = '';
UPDATE sessions SET agent_code = (SELECT agent_code FROM agents WHERE agents.id = sessions.agent_id) WHERE agent_code IS NULL OR agent_code = '';

-- 为对话记录设置 user_code, channel_code, agent_code
UPDATE conversation_records SET user_code = (SELECT user_code FROM users WHERE users.id = conversation_records.user_id) WHERE user_code IS NULL OR user_code = '';
UPDATE conversation_records SET channel_code = (SELECT channel_code FROM channels WHERE channels.id = conversation_records.channel_id) WHERE channel_code IS NULL OR channel_code = '';
UPDATE conversation_records SET agent_code = (SELECT agent_code FROM agents WHERE agents.id = conversation_records.agent_id) WHERE agent_code IS NULL OR agent_code = '';

-- 验证结果
SELECT 'Users with Code:' as check_item, COUNT(*) as count FROM users WHERE user_code IS NOT NULL AND user_code != '';
SELECT 'Agents with Code:' as check_item, COUNT(*) as count FROM agents WHERE agent_code IS NOT NULL AND agent_code != '';
SELECT 'Channels with Code:' as check_item, COUNT(*) as count FROM channels WHERE channel_code IS NOT NULL AND channel_code != '';
