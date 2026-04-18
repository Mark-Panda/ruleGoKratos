package dao

import (
	"context"
	"log"
	"sync"

	"gorm.io/gorm"
)

var db *gorm.DB
var pgOnce sync.Once

// Init pg客户端初始化（含 managed_agent 表自动迁移）
func Init(client *gorm.DB) {
	pgOnce.Do(func() {
		db = client
		if db != nil {
			if err := migrateManagedAgentTable(db); err != nil {
				log.Printf("dao: migrate managed_agent: %v", err)
			}
			if err := migratePlaygroundAgentPoolTable(db); err != nil {
				log.Printf("dao: migrate playground_agent_pool: %v", err)
			}
			if err := migratePlaygroundCollaborationSchemeTable(db); err != nil {
				log.Printf("dao: migrate playground_collaboration_scheme: %v", err)
			}
		}
	})
	_ = db
}

// Transaction 执行事务（与 LLM 配置创建等多表写入共用）
func Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return db.WithContext(ctx).Transaction(fn)
}
