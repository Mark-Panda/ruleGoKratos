package dao

import (
	"context"
	"time"
)

type LLMModelEntry struct {
	ID          int64      `gorm:"column:id;primaryKey" json:"id"`
	ConfigID    int64      `gorm:"column:config_id" json:"configId"`
	ModelName   string     `gorm:"column:model_name" json:"modelName"`
	Description string     `gorm:"column:description" json:"description"`
	Enabled     bool       `gorm:"column:enabled" json:"enabled"`
	CreatedAt   *time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt   *time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

func (LLMModelEntry) TableName() string {
	return "llm_model_entry"
}

func NewLLMModelEntry() *LLMModelEntry {
	return &LLMModelEntry{}
}

func (m *LLMModelEntry) Create(ctx context.Context) error {
	return db.WithContext(ctx).Create(m).Error
}

func (m *LLMModelEntry) FindByConfigIDs(ctx context.Context, configIDs []int64) ([]LLMModelEntry, error) {
	if len(configIDs) == 0 {
		return nil, nil
	}
	var list []LLMModelEntry
	err := db.WithContext(ctx).Model(m).
		Where("config_id IN ?", configIDs).
		Order("config_id asc, id asc").
		Find(&list).Error
	return list, err
}

func (m *LLMModelEntry) FindByID(ctx context.Context, id int64) (*LLMModelEntry, error) {
	var row LLMModelEntry
	err := db.WithContext(ctx).Model(m).Where("id = ?", id).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (m *LLMModelEntry) Updates(ctx context.Context, where map[string]interface{}, data map[string]interface{}) error {
	return db.WithContext(ctx).Model(m).Where(where).Updates(data).Error
}

func (m *LLMModelEntry) Delete(ctx context.Context, where map[string]interface{}) error {
	return db.WithContext(ctx).Where(where).Delete(m).Error
}

func (m *LLMModelEntry) CountByConfigAndModelName(ctx context.Context, configID int64, modelName string, excludeID int64) (int64, error) {
	q := db.WithContext(ctx).Model(m).
		Where("config_id = ? AND model_name = ?", configID, modelName)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var n int64
	err := q.Count(&n).Error
	return n, err
}
