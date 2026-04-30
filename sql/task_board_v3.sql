-- 任务看板 v3: 增加最近执行记录ID，用于精确查询执行日志
ALTER TABLE task_board ADD COLUMN last_run_id varchar(64) DEFAULT NULL COMMENT '最近一次规则链执行的记录ID';
