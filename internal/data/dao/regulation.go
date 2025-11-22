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

func (r *Regulation) Updates(ctx context.Context, date map[string]interface{}, where map[string]interface{}) error {
	return db.WithContext(ctx).Model(r).Where(where).Updates(date).Error
}

func (r *Regulation) Delete(ctx context.Context, where map[string]interface{}) error {
	return db.WithContext(ctx).Model(r).Where(where).Delete(r).Error
}

func (r *Regulation) FindOne(ctx context.Context, where map[string]interface{}) (*Regulation, error) {
	var regulation Regulation
	err := db.WithContext(ctx).Model(r).Where(where).First(&regulation).Error
	return &regulation, err
}

// 分页查询
func (r *Regulation) FindList(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]Regulation, int64, error) {
	var regulations []Regulation
	var count int64
	db := db.WithContext(ctx).Model(r).Where(where)
	err := db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&regulations).Error
	if err != nil {
		return nil, 0, err
	}
	err = db.Count(&count).Error
	return regulations, count, err
}

func (r *Regulation) FindAll(ctx context.Context, where map[string]interface{}) ([]Regulation, error) {
	var regulations []Regulation
	err := db.WithContext(ctx).Model(r).Where(where).Find(&regulations).Error
	return regulations, err
}
