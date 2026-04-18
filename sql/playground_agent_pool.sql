-- Agent Playground：Agent 池持久化（与 internal/data/dao/playground_agent_pool.go 一致）
-- agents_json：池内 []*AgentDefinition 的 JSON，空池可为 []
-- 运行时仍由 dao.Init → migratePlaygroundAgentPoolTable 执行 AutoMigrate；此文件便于手工初始化与版本对照

CREATE TABLE IF NOT EXISTS playground_agent_pool (
    id VARCHAR(64) NOT NULL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    agents_json TEXT NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);

COMMENT ON TABLE playground_agent_pool IS 'Agent Playground Agent 池';
COMMENT ON COLUMN playground_agent_pool.agents_json IS 'JSON 数组：池内 Agent 定义（含 managed_agent_id 等）';
