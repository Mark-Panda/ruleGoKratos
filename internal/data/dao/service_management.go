package dao

import (
	"context"
	"time"
)

type ServiceManagement struct {
	ID               int64      `gorm:"primaryKey;column:id;comment:服务ID"`
	Name             string     `gorm:"column:name;size:255;not null;uniqueIndex:uk_name;comment:服务名称"`
	Status           int32      `gorm:"column:status;default:2;comment:服务状态 1:运行中 2:停止"`
	VolcLogServiceID string     `gorm:"column:volc_log_service_id;size:128;comment:火山云日志服务ID"`
	GitRepoURL       string     `gorm:"column:git_repo_url;size:512;comment:git仓库地址"`
	CreatedAt        time.Time  `gorm:"column:created_at;comment:创建时间"`
	UpdatedAt        time.Time  `gorm:"column:updated_at;comment:更新时间"`
	DeletedAt        *time.Time `gorm:"column:deleted_at;index;comment:删除时间"`
	Description      string     `gorm:"column:description;type:text;comment:服务描述"`
}

func (ServiceManagement) TableName() string {
	return "service_management"
}

func NewServiceManagement() *ServiceManagement {
	return &ServiceManagement{}
}

// Create 创建服务
func (s *ServiceManagement) Create(ctx context.Context) error {
	return db.WithContext(ctx).Create(s).Error
}

// GetByID 根据ID获取服务
func (s *ServiceManagement) GetByID(ctx context.Context, id int64) (*ServiceManagement, error) {
	var service ServiceManagement
	err := db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&service).Error
	if err != nil {
		return nil, err
	}
	return &service, nil
}

// GetByName 根据名称获取服务
func (s *ServiceManagement) GetByName(ctx context.Context, name string) (*ServiceManagement, error) {
	var service ServiceManagement
	err := db.WithContext(ctx).Where("name = ? AND deleted_at IS NULL", name).First(&service).Error
	if err != nil {
		return nil, err
	}
	return &service, nil
}

// List 查询服务列表
func (s *ServiceManagement) List(ctx context.Context, status int32, page, pageSize int32) ([]*ServiceManagement, int64, error) {
	var services []*ServiceManagement
	var count int64
	query := db.WithContext(ctx).Model(s).Where("deleted_at IS NULL")
	if status > 0 {
		query = query.Where("status = ?", status)
	}
	if err := query.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := query.Offset(int(offset)).Limit(int(pageSize)).Order("created_at desc").Find(&services).Error; err != nil {
		return nil, 0, err
	}
	return services, count, nil
}

// Update 更新服务
func (s *ServiceManagement) Update(ctx context.Context, id int64, data map[string]interface{}) error {
	return db.WithContext(ctx).Model(s).Where("id = ?", id).Updates(data).Error
}

// Delete 软删除服务
func (s *ServiceManagement) Delete(ctx context.Context, id int64) error {
	return db.WithContext(ctx).Model(s).Where("id = ?", id).Update("deleted_at", time.Now()).Error
}
