package dao

import (
	"context"
	"log"
	"sync"

	"gorm.io/gorm"
)

var db *gorm.DB
var pgOnce sync.Once

// Init pg客户端初始化（含 managed_agent、mcp_config 等表自动迁移）
func Init(client *gorm.DB) {
	pgOnce.Do(func() {
		db = client
		if db != nil {
			if err := migrateManagedAgentTable(db); err != nil {
				log.Printf("dao: migrate managed_agent: %v", err)
			}
			if err := migrateMcpConfigTable(db); err != nil {
				log.Printf("dao: migrate mcp_config: %v", err)
			}
			if err := migratePlaygroundAgentPoolTable(db); err != nil {
				log.Printf("dao: migrate playground_agent_pool: %v", err)
			}
			if err := migratePlaygroundCollaborationSchemeTable(db); err != nil {
				log.Printf("dao: migrate playground_collaboration_scheme: %v", err)
			}
			// 自动迁移任务看板、服务管理和定时任务表
			if err := db.AutoMigrate(&TaskBoard{}, &ServiceManagement{}, &ScheduledTask{}, &ScheduledTaskRun{}); err != nil {
				log.Printf("dao: migrate task_board/service_management/scheduled_task: %v", err)
			}
			// 自动迁移 LLM Token 使用记录表
			if err := db.AutoMigrate(&LLMTokenUsage{}); err != nil {
				log.Printf("dao: migrate llm_token_usage: %v", err)
			}
		}
	})
	_ = db
}

// Transaction 执行事务（与 LLM 配置创建等多表写入共用）
func Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return db.WithContext(ctx).Transaction(fn)
}
