package dao

import (
	"context"
	"time"
)

type Regulation struct {
	ID          int64      `gorm:"column:id" json:"id"`
	UserName    string     `gorm:"column:user_name" json:"userName"`
	Root        bool       `gorm:"column:root" json:"root"`
	Disabled    bool       `gorm:"column:disabled" json:"disabled"`
	Name        string     `gorm:"column:name" json:"name"`
	RuleChainID string     `gorm:"column:rule_chain_id" json:"ruleChainId"`
	RuleVersion int        `gorm:"column:rule_version" json:"ruleVersion"`
	RuleConfig  string     `gorm:"column:rule_config" json:"ruleConfig"`
	CreatedAt   *time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt   *time.Time `gorm:"column:updated_at" json:"updatedAt"`
	DeletedAt   *time.Time `gorm:"column:deleted_at" json:"deletedAt"`
}

func (Regulation) TableName() string {
	return "regulation"
}

func NewRegulation() *Regulation {
	return &Regulation{}
}

func (r *Regulation) Create(ctx context.Context) error {
	return db.WithContext(ctx).Create(r).Error
}
