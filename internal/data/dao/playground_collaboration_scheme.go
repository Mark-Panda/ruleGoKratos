package dao

import (
	"time"

	"gorm.io/gorm"
)

// PlaygroundCollaborationScheme Agent Playground 协作编排方案持久化（完整实体 JSON）。
type PlaygroundCollaborationScheme struct {
	ID         string     `gorm:"column:id;primaryKey;size:64"`
	SchemeJSON string     `gorm:"column:scheme_json;type:text;not null"`
	CreatedAt  *time.Time `gorm:"column:created_at"`
	UpdatedAt  *time.Time `gorm:"column:updated_at"`
}

func (PlaygroundCollaborationScheme) TableName() string {
	return "playground_collaboration_scheme"
}

func migratePlaygroundCollaborationSchemeTable(db *gorm.DB) error {
	return db.AutoMigrate(&PlaygroundCollaborationScheme{})
}
