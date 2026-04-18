-- 主站「Agent 配置」（与 internal/data/dao/managed_agent.go 一致）
-- 运行时仍由 dao.Init → migrateManagedAgentTable 执行 AutoMigrate；此文件便于手工初始化与版本对照

CREATE TABLE IF NOT EXISTS managed_agent (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    system_prompt TEXT,
    skill_paths TEXT,
    mcp_ids TEXT,
    llm_config_id BIGINT NOT NULL DEFAULT 0,
    model_scope VARCHAR(16) NOT NULL DEFAULT 'all',
    model_entry_ids TEXT,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);

COMMENT ON TABLE managed_agent IS '可编排 Agent 配置（提示词、SKILL/MCP、模型站点与范围）';
