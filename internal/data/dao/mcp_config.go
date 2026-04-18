package dao

import (
	"context"
	"time"
)

type MCPConfig struct {
	ID          int64      `gorm:"column:id;primaryKey" json:"id"`
	Name        string     `gorm:"column:name" json:"name"`
	Server      string     `gorm:"column:server" json:"server"`
	Endpoint    string     `gorm:"column:endpoint" json:"endpoint"`
	HeadersJSON string     `gorm:"column:headers_json" json:"headersJson"`
	Enabled     bool       `gorm:"column:enabled" json:"enabled"`
	Description string     `gorm:"column:description" json:"description"`
	CreatedAt   *time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt   *time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

func (MCPConfig) TableName() string {
	return "mcp_config"
}

func NewMCPConfig() *MCPConfig {
	return &MCPConfig{}
}

func (m *MCPConfig) Create(ctx context.Context) error {
	return db.WithContext(ctx).Create(m).Error
}

func (m *MCPConfig) FindAll(ctx context.Context) ([]MCPConfig, error) {
	var list []MCPConfig
	err := db.WithContext(ctx).Model(m).Order("id desc").Find(&list).Error
	return list, err
}

func (m *MCPConfig) FindByIDs(ctx context.Context, ids []int64) ([]MCPConfig, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []MCPConfig
	err := db.WithContext(ctx).Model(m).Where("id IN ?", ids).Find(&list).Error
	return list, err
}

func (m *MCPConfig) Updates(ctx context.Context, where map[string]interface{}, data map[string]interface{}) error {
	return db.WithContext(ctx).Model(m).Where(where).Updates(data).Error
}

func (m *MCPConfig) Delete(ctx context.Context, where map[string]interface{}) error {
	return db.WithContext(ctx).Where(where).Delete(m).Error
}
