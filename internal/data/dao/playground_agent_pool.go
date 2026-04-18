package dao

import (
	"time"

	"gorm.io/gorm"
)

// PlaygroundAgentPool Agent Playground 侧「Agent 池」持久化（池内多条 AgentDefinition 序列化于 agents_json）。
type PlaygroundAgentPool struct {
	ID           string     `gorm:"column:id;primaryKey;size:64"`
	Name         string     `gorm:"column:name;size:255;not null"`
	Description  string     `gorm:"column:description;type:text"`
	AgentsJSON   string     `gorm:"column:agents_json;type:text;not null"` // []*entity.AgentDefinition JSON
	CreatedAt    *time.Time `gorm:"column:created_at"`
	UpdatedAt    *time.Time `gorm:"column:updated_at"`
}

func (PlaygroundAgentPool) TableName() string {
	return "playground_agent_pool"
}

func migratePlaygroundAgentPoolTable(db *gorm.DB) error {
	return db.AutoMigrate(&PlaygroundAgentPool{})
}
