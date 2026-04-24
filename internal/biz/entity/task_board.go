package entity

import (
	"time"
)

// TaskBoard 任务看板实体
type TaskBoard struct {
	ID             int64      `gorm:"primaryKey;column:id;comment:任务ID"`
	Name           string     `gorm:"column:name;size:255;not null;comment:任务名称"`
	Priority       int32      `gorm:"column:priority;default:99;comment:任务优先级 0-99"`
	Status         int32      `gorm:"column:status;default:1;comment:任务状态 1:待处理 2:处理中 3:已完成 4:处理失败"`
	Type           int32      `gorm:"column:type;default:4;comment:任务类型 1:缺陷 2:需求 3:功能 4:其他"`
	CreatedAt      time.Time  `gorm:"column:created_at;comment:创建时间"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;comment:更新时间"`
	DeletedAt      *time.Time `gorm:"column:deleted_at;index;comment:删除时间"`
	HandlerUserID  string     `gorm:"column:handler_user_id;size:64;comment:处理用户ID"`
	Description    string     `gorm:"column:description;type:text;comment:任务描述"`
}

// TableName 表名
func (TaskBoard) TableName() string {
	return "task_board"
}

// 任务状态常量
const (
	TaskStatusPending    = 1 // 待处理
	TaskStatusProcessing = 2 // 处理中
	TaskStatusCompleted  = 3 // 已完成
	TaskStatusFailed     = 4 // 处理失败
)

// 任务类型常量
const (
	TaskTypeBug     = 1 // 缺陷
	TaskTypeRequire = 2 // 需求
	TaskTypeFeature = 3 // 功能
	TaskTypeOther   = 4 // 其他
)
