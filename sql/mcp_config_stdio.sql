-- MCP 支持 stdio：本地进程型 MCP（command + args + env）
-- 在已有库上执行一次。

ALTER TABLE mcp_config
  ADD COLUMN IF NOT EXISTS transport varchar(16) NOT NULL DEFAULT 'http';

ALTER TABLE mcp_config
  ADD COLUMN IF NOT EXISTS stdio_command text NOT NULL DEFAULT '';

ALTER TABLE mcp_config
  ADD COLUMN IF NOT EXISTS stdio_args_json text NOT NULL DEFAULT '[]';

ALTER TABLE mcp_config
  ADD COLUMN IF NOT EXISTS stdio_env_json text NOT NULL DEFAULT '{}';

-- stdio 模式可不填 HTTP 地址：允许空字符串
ALTER TABLE mcp_config ALTER COLUMN endpoint SET DEFAULT '';
UPDATE mcp_config SET endpoint = '' WHERE endpoint IS NULL;
