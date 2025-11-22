package dao

import (
	"context"
	"time"
)

type ComponentUseRule struct {
	ID            int64      `gorm:"column:id" json:"id"`
	ComponentName string     `gorm:"column:component_name" json:"componentName"`
	ComponentType string     `gorm:"column:component_type" json:"componentType"`
	Disabled      bool       `gorm:"column:disabled" json:"disabled"`
	UseDesc       string     `gorm:"column:use_desc" json:"useDesc"`
	UseRuleDesc   string     `gorm:"column:use_rule_desc" json:"useRuleDesc"`
	CreatedAt     *time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt     *time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

func (ComponentUseRule) TableName() string {
	return "component_use_rule"
}

func NewComponentUseRule() *ComponentUseRule {
	return &ComponentUseRule{}
}

func (c *ComponentUseRule) Create(ctx context.Context) error {
	return db.WithContext(ctx).Create(c).Error
}

func (c *ComponentUseRule) Updates(ctx context.Context, date map[string]interface{}, where map[string]interface{}) error {
	return db.WithContext(ctx).Model(c).Where(where).Updates(date).Error
}

func (c *ComponentUseRule) Delete(ctx context.Context, where map[string]interface{}) error {
	return db.WithContext(ctx).Model(c).Where(where).Delete(c).Error
}

func (c *ComponentUseRule) FindOne(ctx context.Context, where map[string]interface{}) (*ComponentUseRule, error) {
	var componentUseRule ComponentUseRule
	err := db.WithContext(ctx).Model(c).Where(where).First(&componentUseRule).Error
	return &componentUseRule, err
}

// 分页查询
func (c *ComponentUseRule) FindList(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]ComponentUseRule, int64, error) {
	var componentUseRules []ComponentUseRule
	var count int64
	db := db.WithContext(ctx).Model(c).Where(where)
	err := db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&componentUseRules).Error
	if err != nil {
		return nil, 0, err
	}
	err = db.Count(&count).Error
	return componentUseRules, count, err
}

func (c *ComponentUseRule) FindAll(ctx context.Context, where map[string]interface{}) ([]ComponentUseRule, error) {
	var componentUseRules []ComponentUseRule
	err := db.WithContext(ctx).Model(c).Where(where).Find(&componentUseRules).Error
	return componentUseRules, err
}
