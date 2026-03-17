-- Migration: 添加 agent_mcp_bindings 表的 auto_load 字段
-- 用于 MCP 渐进式加载功能

-- SQLite
ALTER TABLE agent_mcp_bindings ADD COLUMN auto_load BOOLEAN DEFAULT 0;

-- MySQL (如果使用)
-- ALTER TABLE agent_mcp_bindings ADD COLUMN auto_load TINYINT(1) DEFAULT 0 COMMENT '是否在对话开始时自动加载该 MCP 服务器的工具';

-- PostgreSQL (如果使用)
-- ALTER TABLE agent_mcp_bindings ADD COLUMN auto_load BOOLEAN DEFAULT FALSE;

-- 更新说明
-- 现有绑定的 auto_load 默认为 false，即不自动加载
-- 这保持了向后兼容，不影响现有行为
