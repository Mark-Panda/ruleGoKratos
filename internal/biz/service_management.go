package biz

import (
	"context"
	"time"

	"ruleGoKratos/internal/biz/entity"
)

// ServiceManagementRepo 服务管理仓库接口
type ServiceManagementRepo interface {
	Create(ctx context.Context, service *entity.ServiceManagement) error
	GetByID(ctx context.Context, id int64) (*entity.ServiceManagement, error)
	List(ctx context.Context, status int32, page, pageSize int32) ([]*entity.ServiceManagement, int64, error)
	Update(ctx context.Context, service *entity.ServiceManagement) error
	Delete(ctx context.Context, id int64) error
}

// ServiceManagementUsecase 服务管理业务逻辑
type ServiceManagementUsecase struct {
	repo ServiceManagementRepo
}

// NewServiceManagementUsecase 创建服务管理业务逻辑实例
func NewServiceManagementUsecase(repo ServiceManagementRepo) *ServiceManagementUsecase {
	return &ServiceManagementUsecase{repo: repo}
}

// CreateService 创建服务
func (uc *ServiceManagementUsecase) CreateService(ctx context.Context, name string, status int32, volcLogServiceID, gitRepoURL, description string) (*entity.ServiceManagement, error) {
	service := &entity.ServiceManagement{
		Name:             name,
		Status:           status,
		VolcLogServiceID: volcLogServiceID,
		GitRepoURL:       gitRepoURL,
		Description:      description,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	if status == 0 {
		service.Status = entity.ServiceStatusStopped
	}
	if err := uc.repo.Create(ctx, service); err != nil {
		return nil, err
	}
	return service, nil
}

// GetService 获取服务详情
func (uc *ServiceManagementUsecase) GetService(ctx context.Context, id int64) (*entity.ServiceManagement, error) {
	return uc.repo.GetByID(ctx, id)
}

// ListServices 查询服务列表
func (uc *ServiceManagementUsecase) ListServices(ctx context.Context, status int32, page, pageSize int32) ([]*entity.ServiceManagement, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	return uc.repo.List(ctx, status, page, pageSize)
}

// UpdateService 更新服务
func (uc *ServiceManagementUsecase) UpdateService(ctx context.Context, id int64, name *string, status *int32, volcLogServiceID *string, gitRepoURL *string, description *string) (*entity.ServiceManagement, error) {
	service, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if name != nil {
		service.Name = *name
	}
	if status != nil {
		service.Status = *status
	}
	if volcLogServiceID != nil {
		service.VolcLogServiceID = *volcLogServiceID
	}
	if gitRepoURL != nil {
		service.GitRepoURL = *gitRepoURL
	}
	if description != nil {
		service.Description = *description
	}
	service.UpdatedAt = time.Now()
	if err := uc.repo.Update(ctx, service); err != nil {
		return nil, err
	}
	return service, nil
}

// DeleteService 删除服务
func (uc *ServiceManagementUsecase) DeleteService(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}
