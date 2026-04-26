CREATE TABLE IF NOT EXISTS scheduled_tasks (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  rule_chain_id VARCHAR(255) NOT NULL,
  cron_expr VARCHAR(255) NOT NULL,
  schedule_type VARCHAR(64) NOT NULL,
  schedule_config TEXT,
  disabled BOOLEAN NOT NULL DEFAULT TRUE,
  last_run_at TIMESTAMP,
  last_status INTEGER,
  last_error TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_scheduled_tasks_rule_chain_id
  ON scheduled_tasks (rule_chain_id);

CREATE INDEX IF NOT EXISTS idx_scheduled_tasks_disabled_deleted_at
  ON scheduled_tasks (disabled, deleted_at);

CREATE INDEX IF NOT EXISTS idx_scheduled_tasks_deleted_at
  ON scheduled_tasks (deleted_at);

CREATE INDEX IF NOT EXISTS idx_scheduled_tasks_created_at_id
  ON scheduled_tasks (created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS scheduled_task_runs (
  id BIGSERIAL PRIMARY KEY,
  task_id BIGINT NOT NULL,
  rule_chain_id VARCHAR(255) NOT NULL,
  status INTEGER NOT NULL,
  trigger_payload TEXT,
  error_message TEXT,
  started_at TIMESTAMP NOT NULL,
  finished_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_scheduled_task_runs_task_id_created_at_id
  ON scheduled_task_runs (task_id, created_at DESC, id DESC);
