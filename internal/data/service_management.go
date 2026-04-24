package data

import (
	"context"

	"ruleGoKratos/internal/biz"
	"ruleGoKratos/internal/biz/entity"
	"ruleGoKratos/internal/data/dao"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/jinzhu/copier"
	"gorm.io/gorm"
)

var _ biz.ServiceManagementRepo = &serviceManagementRepo{}

type serviceManagementRepo struct {
	data *Data
	log  *log.Helper
}

// NewServiceManagementRepo 创建服务管理仓库实例
func NewServiceManagementRepo(data *Data, logger log.Logger) biz.ServiceManagementRepo {
	return &serviceManagementRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// Create 创建服务
func (r *serviceManagementRepo) Create(ctx context.Context, service *entity.ServiceManagement) error {
	s := dao.ServiceManagement{}
	_ = copier.Copy(&s, service)
	return s.Create(ctx)
}

// GetByID 根据ID获取服务
func (r *serviceManagementRepo) GetByID(ctx context.Context, id int64) (*entity.ServiceManagement, error) {
	s := dao.NewServiceManagement()
	service, err := s.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	res := entity.ServiceManagement{}
	_ = copier.Copy(&res, service)
	return &res, nil
}

// List 查询服务列表
func (r *serviceManagementRepo) List(ctx context.Context, status int32, page, pageSize int32) ([]*entity.ServiceManagement, int64, error) {
	s := dao.NewServiceManagement()
	services, count, err := s.List(ctx, status, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	res := make([]*entity.ServiceManagement, len(services))
	for i, item := range services {
		res[i] = &entity.ServiceManagement{}
		_ = copier.Copy(res[i], item)
	}
	return res, count, nil
}

// Update 更新服务
func (r *serviceManagementRepo) Update(ctx context.Context, service *entity.ServiceManagement) error {
	data := map[string]interface{}{
		"name":               service.Name,
		"status":             service.Status,
		"volc_log_service_id": service.VolcLogServiceID,
		"git_repo_url":       service.GitRepoURL,
		"description":        service.Description,
		"updated_at":         service.UpdatedAt,
	}
	s := dao.NewServiceManagement()
	return s.Update(ctx, service.ID, data)
}

// Delete 删除服务
func (r *serviceManagementRepo) Delete(ctx context.Context, id int64) error {
	s := dao.NewServiceManagement()
	return s.Delete(ctx, id)
}
