package dao

import (
	"context"
	"time"
)

type LLMConfig struct {
	ID          int64      `gorm:"column:id;primaryKey" json:"id"`
	Name        string     `gorm:"column:name" json:"name"`
	Provider    string     `gorm:"column:provider" json:"provider"`
	BaseURL     string     `gorm:"column:base_url" json:"baseUrl"`
	APIKey      string     `gorm:"column:api_key" json:"apiKey"`
	Enabled     bool       `gorm:"column:enabled" json:"enabled"`
	Description string     `gorm:"column:description" json:"description"`
	CreatedAt   *time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt   *time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

func (LLMConfig) TableName() string {
	return "llm_config"
}

func NewLLMConfig() *LLMConfig {
	return &LLMConfig{}
}

func (m *LLMConfig) Create(ctx context.Context) error {
	return db.WithContext(ctx).Create(m).Error
}

func (m *LLMConfig) FindAll(ctx context.Context) ([]LLMConfig, error) {
	var list []LLMConfig
	err := db.WithContext(ctx).Model(m).Order("id desc").Find(&list).Error
	return list, err
}

func (m *LLMConfig) FindByID(ctx context.Context, id int64) (*LLMConfig, error) {
	var row LLMConfig
	err := db.WithContext(ctx).Model(m).Where("id = ?", id).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (m *LLMConfig) Updates(ctx context.Context, where map[string]interface{}, data map[string]interface{}) error {
	return db.WithContext(ctx).Model(m).Where(where).Updates(data).Error
}

func (m *LLMConfig) Delete(ctx context.Context, where map[string]interface{}) error {
	return db.WithContext(ctx).Where(where).Delete(m).Error
}
