package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type TaskBoard struct {
	ID            int64      `gorm:"primaryKey;column:id;comment:任务ID"`
	Name          string     `gorm:"column:name;size:255;not null;comment:任务名称"`
	Priority      int32      `gorm:"column:priority;default:99;comment:任务优先级 0-99"`
	Status        int32      `gorm:"column:status;default:1;comment:任务状态 1:待处理 2:处理中 3:已完成 4:处理失败"`
	Type          int32      `gorm:"column:type;default:4;comment:任务类型 1:缺陷 2:需求 3:功能 4:其他"`
	CreatedAt     time.Time  `gorm:"column:created_at;comment:创建时间"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;comment:更新时间"`
	DeletedAt     *time.Time `gorm:"column:deleted_at;index;comment:删除时间"`
	HandlerUserID string     `gorm:"column:handler_user_id;size:64;comment:处理用户ID"`
	Description   string     `gorm:"column:description;type:text;comment:任务描述"`
	RuleChainID   string     `gorm:"column:rule_chain_id;size:64;comment:关联的规则链ID"`
	ParentID      *int64     `gorm:"column:parent_id;comment:父任务ID"`
	LastRunID     string     `gorm:"column:last_run_id;size:64;comment:最近一次规则链执行的记录ID"`
}

func (TaskBoard) TableName() string {
	return "task_board"
}

func NewTaskBoard() *TaskBoard {
	return &TaskBoard{}
}

// Create 创建任务
func (t *TaskBoard) Create(ctx context.Context) error {
	return db.WithContext(ctx).Create(t).Error
}

// GetByID 根据ID获取任务
func (t *TaskBoard) GetByID(ctx context.Context, id int64) (*TaskBoard, error) {
	var task TaskBoard
	err := db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// List 查询任务列表
func (t *TaskBoard) List(ctx context.Context, status, typ int32, handlerUserID string, page, pageSize int32) ([]*TaskBoard, int64, error) {
	var tasks []*TaskBoard
	var count int64
	query := db.WithContext(ctx).Model(t).Where("deleted_at IS NULL")
	if status > 0 {
		query = query.Where("status = ?", status)
	}
	if typ > 0 {
		query = query.Where("type = ?", typ)
	}
	if handlerUserID != "" {
		query = query.Where("handler_user_id = ?", handlerUserID)
	}
	if err := query.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := query.Offset(int(offset)).Limit(int(pageSize)).Order("created_at desc").Find(&tasks).Error; err != nil {
		return nil, 0, err
	}
	return tasks, count, nil
}

// Update 更新任务
func (t *TaskBoard) Update(ctx context.Context, id int64, data map[string]interface{}) error {
	return db.WithContext(ctx).Model(t).Where("id = ?", id).Updates(data).Error
}

// Delete 软删除任务，同时清除子任务的父关联（事务保证一致性）
func (t *TaskBoard) Delete(ctx context.Context, id int64) error {
	now := time.Now()
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(t).Where("id = ?", id).Update("deleted_at", now).Error; err != nil {
			return err
		}
		if err := tx.Model(t).Where("parent_id = ?", id).Update("parent_id", nil).Error; err != nil {
			return err
		}
		return nil
	})
}

// ListByParentID 根据父任务ID查询子任务列表
func (t *TaskBoard) ListByParentID(ctx context.Context, parentID int64, page, pageSize int32) ([]*TaskBoard, int64, error) {
	var tasks []*TaskBoard
	var count int64
	query := db.WithContext(ctx).Model(t).Where("parent_id = ? AND deleted_at IS NULL", parentID)
	if err := query.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := query.Offset(int(offset)).Limit(int(pageSize)).Order("created_at desc").Find(&tasks).Error; err != nil {
		return nil, 0, err
	}
	return tasks, count, nil
}
