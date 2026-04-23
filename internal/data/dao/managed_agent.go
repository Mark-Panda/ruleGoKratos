package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// ManagedAgent 可编排的 Agent 配置（系统提示、SKILL/MCP、模型站点与范围）。
type ManagedAgent struct {
	ID                int64      `gorm:"column:id;primaryKey;autoIncrement"`
	Name              string     `gorm:"column:name;size:255;not null"`
	Description       string     `gorm:"column:description;type:text"`
	SystemPrompt      string     `gorm:"column:system_prompt;type:text"`
	SkillPathsJSON    string     `gorm:"column:skill_paths;type:text"` // JSON []string，存技能包 id（路径首段）；读取时兼容旧版文件路径
	McpIDsJSON        string     `gorm:"column:mcp_ids;type:text"`     // JSON []int64
	LLMConfigID       int64      `gorm:"column:llm_config_id"`
	ModelScope        string     `gorm:"column:model_scope;size:16;not null"`              // all | explicit
	ModelEntryIDsJSON string     `gorm:"column:model_entry_ids;type:text"`                 // JSON []int64，explicit 时使用
	WorkspaceID       string     `gorm:"column:workspace_id;size:128;not null;default:''"` // 关联工作区 id（可空）
	Enabled           bool       `gorm:"column:enabled"`
	CreatedAt         *time.Time `gorm:"column:created_at"`
	UpdatedAt         *time.Time `gorm:"column:updated_at"`
}

func (ManagedAgent) TableName() string {
	return "managed_agent"
}

func NewManagedAgent() *ManagedAgent {
	return &ManagedAgent{}
}

func (m *ManagedAgent) Create(ctx context.Context) error {
	return db.WithContext(ctx).Create(m).Error
}

func (m *ManagedAgent) FindAll(ctx context.Context) ([]ManagedAgent, error) {
	var list []ManagedAgent
	err := db.WithContext(ctx).Model(m).Order("id desc").Find(&list).Error
	return list, err
}

func (m *ManagedAgent) FindByID(ctx context.Context, id int64) (*ManagedAgent, error) {
	var row ManagedAgent
	err := db.WithContext(ctx).Model(m).Where("id = ?", id).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (m *ManagedAgent) Updates(ctx context.Context, where map[string]interface{}, data map[string]interface{}) error {
	return db.WithContext(ctx).Model(m).Where(where).Updates(data).Error
}

func (m *ManagedAgent) Delete(ctx context.Context, where map[string]interface{}) error {
	return db.WithContext(ctx).Where(where).Delete(m).Error
}

func migrateManagedAgentTable(g *gorm.DB) error {
	return g.AutoMigrate(&ManagedAgent{})
}
