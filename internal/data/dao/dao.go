package dao

import (
	"context"
	"sync"

	"gorm.io/gorm"
)

var db *gorm.DB
var pgOnce sync.Once

// Init pg客户端初始化
func Init(client *gorm.DB) {
	pgOnce.Do(func() { db = client })
	_ = db
}

// Transaction 执行事务（与 LLM 配置创建等多表写入共用）
func Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return db.WithContext(ctx).Transaction(fn)
}
