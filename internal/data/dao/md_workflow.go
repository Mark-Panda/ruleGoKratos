package dao

import (
	"context"
	"time"
)

type MdWorkflow struct {
	ID           int64      `gorm:"column:id" json:"id"`
	Title        string     `gorm:"column:title" json:"title"`
	Content      string     `gorm:"column:content" json:"content"`
	Desc         string     `gorm:"column:desc" json:"desc"`
	ChainID      string     `gorm:"column:chain_id" json:"chainId"`
	ChainName    string     `gorm:"column:chain_name" json:"chainName"`
	ChainVersion int        `gorm:"column:chain_version" json:"chainVersion"`
	CreatedAt    *time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt    *time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

func (MdWorkflow) TableName() string {
	return "md_workflow"
}

func NewMdWorkflow() *MdWorkflow {
	return &MdWorkflow{}
}

func (m *MdWorkflow) Create(ctx context.Context) error {
	return db.WithContext(ctx).Create(m).Error
}

func (m *MdWorkflow) Updates(ctx context.Context, date map[string]interface{}, where map[string]interface{}) error {
	return db.WithContext(ctx).Model(m).Where(where).Updates(date).Error
}

func (m *MdWorkflow) Delete(ctx context.Context, where map[string]interface{}) error {
	return db.WithContext(ctx).Model(m).Where(where).Delete(m).Error
}

func (m *MdWorkflow) FindOne(ctx context.Context, where map[string]interface{}) (*MdWorkflow, error) {
	var mdWorkflow MdWorkflow
	err := db.WithContext(ctx).Model(m).Where(where).First(&mdWorkflow).Error
	return &mdWorkflow, err
}

// 分页查询
func (m *MdWorkflow) FindList(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]MdWorkflow, int64, error) {
	var mdWorkflows []MdWorkflow
	var count int64
	db := db.WithContext(ctx).Model(m).Where(where)
	_ = db.Count(&count).Error
	err := db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&mdWorkflows).Error
	if err != nil {
		return nil, 0, err
	}
	return mdWorkflows, count, err
}

func (m *MdWorkflow) FindAll(ctx context.Context, where map[string]interface{}) ([]MdWorkflow, error) {
	var mdWorkflows []MdWorkflow
	err := db.WithContext(ctx).Model(m).Where(where).Find(&mdWorkflows).Error
	return mdWorkflows, err
}
