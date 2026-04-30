-- 任务看板 v2: 增加规则链关联和父子看板关系
ALTER TABLE task_board ADD COLUMN rule_chain_id varchar(64) DEFAULT NULL COMMENT '关联的规则链ID';
ALTER TABLE task_board ADD COLUMN parent_id bigint DEFAULT NULL COMMENT '父任务ID';
CREATE INDEX idx_task_board_rule_chain_id ON task_board(rule_chain_id);
CREATE INDEX idx_task_board_parent_id ON task_board(parent_id);
