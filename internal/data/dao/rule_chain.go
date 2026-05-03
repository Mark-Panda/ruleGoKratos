package dao

import (
	"context"
	"time"
)

type RuleChain struct {
	ID             int64      `gorm:"column:id;primaryKey" json:"id"`
	UserName       string     `gorm:"column:user_name" json:"userName"`
	Root           bool       `gorm:"column:root" json:"root"`
	Disabled       bool       `gorm:"column:disabled" json:"disabled"`
	DebugMode      bool       `gorm:"column:debug_mode" json:"debugMode"`
	Name           string     `gorm:"column:name" json:"name"`
	RuleChainID    string     `gorm:"column:rule_chain_id" json:"ruleChainId"`
	RuleVersion    int        `gorm:"column:rule_version" json:"ruleVersion"`
	Configuration  *string    `gorm:"column:configuration;type:json" json:"configuration"`
	Metadata       *string    `gorm:"column:metadata;type:json" json:"metadata"`
	AdditionalInfo *string    `gorm:"column:additional_info;type:json" json:"additionalInfo"`
	CreatedAt      *time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt      *time.Time `gorm:"column:updated_at" json:"updatedAt"`
	DeletedAt      *time.Time `gorm:"column:deleted_at" json:"deletedAt"`
}

func (RuleChain) TableName() string {
	return "rule_chain"
}

func NewRuleChain() *RuleChain {
	return &RuleChain{}
}

func (r *RuleChain) Create(ctx context.Context) error {
	return db.WithContext(ctx).Create(r).Error
}

func (r *RuleChain) Updates(ctx context.Context, date map[string]interface{}, where map[string]interface{}) error {
	return db.WithContext(ctx).Model(r).Where(where).Updates(date).Error
}

func (r *RuleChain) Delete(ctx context.Context, where map[string]interface{}) error {
	return db.WithContext(ctx).Model(r).Where(where).Unscoped().Delete(r).Error
}

func (r *RuleChain) FindOne(ctx context.Context, where map[string]interface{}) (*RuleChain, error) {
	var ruleChain RuleChain
	err := db.WithContext(ctx).Model(r).Where(where).First(&ruleChain).Error
	return &ruleChain, err
}

// 分页查询
func (r *RuleChain) FindList(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]RuleChain, int64, error) {
	var ruleChains []RuleChain
	var count int64
	db := db.WithContext(ctx).Model(r).Where(where)
	_ = db.Count(&count).Error
	err := db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&ruleChains).Error
	if err != nil {
		return nil, 0, err
	}
	return ruleChains, count, err
}

func (r *RuleChain) FindAll(ctx context.Context, where map[string]interface{}) ([]RuleChain, error) {
	var ruleChains []RuleChain
	err := db.WithContext(ctx).Model(r).Where(where).Find(&ruleChains).Error
	return ruleChains, err
}
