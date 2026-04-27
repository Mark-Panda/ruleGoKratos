package dao

import (
	"context"
	"time"
)

type RunLog struct {
	ID             int64      `gorm:"column:id" json:"id"`
	RunID          string     `gorm:"column:run_id" json:"runId"`
	ChainID        string     `gorm:"column:chain_id" json:"chainId"`
	ChainName      string     `gorm:"column:chain_name" json:"chainName"`
	NodeLog        string     `gorm:"column:node_log" json:"nodeLog"`
	AdditionalInfo string     `gorm:"column:additional_info" json:"additionalInfo"`
	RuleChainInfo  string     `gorm:"column:rule_chain_info" json:"ruleChainInfo"`
	Metadata       string     `gorm:"column:metadata" json:"metadata"`
	StartTs        int64      `gorm:"column:start_ts" json:"startTs"`
	EndTs          int64      `gorm:"column:end_ts" json:"endTs"`
	CreatedAt      *time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt      *time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

func (RunLog) TableName() string {
	return "run_log"
}

func NewRunLog() *RunLog {
	return &RunLog{}
}

func (r *RunLog) Create(ctx context.Context) error {
	return db.WithContext(ctx).Create(r).Error
}

func (r *RunLog) Updates(ctx context.Context, date map[string]interface{}, where map[string]interface{}) error {
	return db.WithContext(ctx).Model(r).Where(where).Updates(date).Error
}

func (r *RunLog) Delete(ctx context.Context, where map[string]interface{}) error {
	return db.WithContext(ctx).Model(r).Where(where).Delete(r).Error
}

func (r *RunLog) FindOne(ctx context.Context, where map[string]interface{}) (*RunLog, error) {
	var runLog RunLog
	err := db.WithContext(ctx).Model(r).Where(where).First(&runLog).Error
	return &runLog, err
}

// 分页查询
func (r *RunLog) FindList(ctx context.Context, where string, page int, pageSize int) ([]RunLog, int64, error) {
	var runLogs []RunLog
	var count int64
	db := db.WithContext(ctx).Model(r).Where(where)
	err := db.Count(&count).Error
	err = db.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&runLogs).Error
	if err != nil {
		return nil, 0, err
	}
	return runLogs, count, err
}

func (r *RunLog) FindAll(ctx context.Context, where map[string]interface{}) ([]RunLog, error) {
	var runLogs []RunLog
	err := db.WithContext(ctx).Model(r).Where(where).Find(&runLogs).Error
	return runLogs, err
}
