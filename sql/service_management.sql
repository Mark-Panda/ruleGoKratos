CREATE TABLE `service_management` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '服务ID',
  `name` varchar(255) NOT NULL COMMENT '服务名称',
  `status` int NOT NULL DEFAULT '2' COMMENT '服务状态 1:运行中 2:停止',
  `volc_log_service_id` varchar(128) DEFAULT NULL COMMENT '火山云日志服务ID',
  `git_repo_url` varchar(512) DEFAULT NULL COMMENT 'git仓库地址',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` datetime DEFAULT NULL COMMENT '删除时间',
  `description` text COMMENT '服务描述',
  PRIMARY KEY (`id`),
  KEY `idx_deleted_at` (`deleted_at`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='服务管理表';
