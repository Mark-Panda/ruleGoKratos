-- Agent Playground：协作编排方案持久化（与 internal/data/dao/playground_collaboration_scheme.go 一致）
-- scheme_json：完整 entity.CollaborationScheme 的 JSON

CREATE TABLE IF NOT EXISTS playground_collaboration_scheme (
    id VARCHAR(64) NOT NULL PRIMARY KEY,
    scheme_json TEXT NOT NULL,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);

COMMENT ON TABLE playground_collaboration_scheme IS 'Agent Playground 协作编排方案';
COMMENT ON COLUMN playground_collaboration_scheme.scheme_json IS 'JSON：协作方案实体（模式、绑定 Agent、config 等）';
