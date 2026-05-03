package dao

import (
	"context"
	"time"
)

type ComponentRegulation struct {
	ID          int64      `gorm:"column:id" json:"id"`
	UserName    string     `gorm:"column:user_name" json:"userName"`
	Root        bool       `gorm:"column:root" json:"root"`
	Disabled    bool       `gorm:"column:disabled" json:"disabled"`
	Name        string     `gorm:"column:name" json:"name"`
	RuleChainID string     `gorm:"column:rule_chain_id" json:"ruleChainId"`
	RuleConfig  string     `gorm:"column:rule_config" json:"ruleConfig"`
	CreatedAt   *time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt   *time.Time `gorm:"column:updated_at" json:"updatedAt"`
	DeletedAt   *time.Time `gorm:"column:deleted_at" json:"deletedAt"`
}

func (ComponentRegulation) TableName() string {
	return "component_regulation"
}

func NewComponentRegulation() *ComponentRegulation {
	return &ComponentRegulation{}
}

func (c *ComponentRegulation) Create(ctx context.Context) error {
	return db.WithContext(ctx).Create(c).Error
}

func (c *ComponentRegulation) Updates(ctx context.Context, date map[string]interface{}, where map[string]interface{}) error {
	return db.WithContext(ctx).Model(c).Where(where).Updates(date).Error
}

func (c *ComponentRegulation) Delete(ctx context.Context, where map[string]interface{}) error {
	return db.WithContext(ctx).Model(c).Where(where).Delete(c).Error
}

func (c *ComponentRegulation) FindOne(ctx context.Context, where map[string]interface{}) (*ComponentRegulation, error) {
	var componentRegulation ComponentRegulation
	err := db.WithContext(ctx).Model(c).Where(where).First(&componentRegulation).Error
	return &componentRegulation, err
}

// 分页查询
func (c *ComponentRegulation) FindList(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]ComponentRegulation, int64, error) {
	var componentRegulations []ComponentRegulation
	var count int64
	db := db.WithContext(ctx).Model(c).Where(where)
	_ = db.Count(&count).Error
	err := db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&componentRegulations).Error
	if err != nil {
		return nil, 0, err
	}

	return componentRegulations, count, err
}

func (c *ComponentRegulation) FindAll(ctx context.Context, where map[string]interface{}) ([]ComponentRegulation, error) {
	var componentRegulations []ComponentRegulation
	err := db.WithContext(ctx).Model(c).Where(where).Find(&componentRegulations).Error
	return componentRegulations, err
}
