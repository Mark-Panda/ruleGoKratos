package dao

import (
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
