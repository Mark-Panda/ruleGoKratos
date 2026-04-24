package entity

import (
	"time"
)

// ServiceManagement 服务管理实体
type ServiceManagement struct {
	ID               int64      `gorm:"primaryKey;column:id;comment:服务ID"`
	Name             string     `gorm:"column:name;size:255;not null;comment:服务名称"`
	Status           int32      `gorm:"column:status;default:2;comment:服务状态 1:运行中 2:停止"`
	VolcLogServiceID string     `gorm:"column:volc_log_service_id;size:128;comment:火山云日志服务ID"`
	GitRepoURL       string     `gorm:"column:git_repo_url;size:512;comment:git仓库地址"`
	CreatedAt        time.Time  `gorm:"column:created_at;comment:创建时间"`
	UpdatedAt        time.Time  `gorm:"column:updated_at;comment:更新时间"`
	DeletedAt        *time.Time `gorm:"column:deleted_at;index;comment:删除时间"`
	Description      string     `gorm:"column:description;type:text;comment:服务描述"`
}

// TableName 表名
func (ServiceManagement) TableName() string {
	return "service_management"
}

// 服务状态常量
const (
	ServiceStatusRunning = 1 // 运行中
	ServiceStatusStopped = 2 // 停止
)
