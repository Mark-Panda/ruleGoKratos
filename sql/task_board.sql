CREATE TABLE `task_board` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '任务ID',
  `name` varchar(255) NOT NULL COMMENT '任务名称',
  `priority` int NOT NULL DEFAULT '99' COMMENT '任务优先级 0-99',
  `status` int NOT NULL DEFAULT '1' COMMENT '任务状态 1:待处理 2:处理中 3:已完成 4:处理失败',
  `type` int NOT NULL DEFAULT '4' COMMENT '任务类型 1:缺陷 2:需求 3:功能 4:其他',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` datetime DEFAULT NULL COMMENT '删除时间',
  `handler_user_id` varchar(64) DEFAULT NULL COMMENT '处理用户ID',
  `description` text COMMENT '任务描述',
  PRIMARY KEY (`id`),
  KEY `idx_deleted_at` (`deleted_at`),
  KEY `idx_status` (`status`),
  KEY `idx_type` (`type`),
  KEY `idx_handler_user_id` (`handler_user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='任务看板表';
