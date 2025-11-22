package data

import (
	"context"
	"time"

	"ruleGoKratos/internal/biz"
	"ruleGoKratos/internal/biz/entity"
	"ruleGoKratos/internal/data/dao"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/jinzhu/copier"
)

var _ biz.MdWorkflowRepo = &mdWorkflowRepo{}

type mdWorkflowRepo struct {
	data *Data
	log  *log.Helper
}

// NewMdWorkflowRepo .
func NewMdWorkflowRepo(data *Data, logger log.Logger) biz.MdWorkflowRepo {
	return &mdWorkflowRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// MdWorkflow Methods
func (r *mdWorkflowRepo) CreateMdWorkflow(ctx context.Context, mdWorkflow *entity.MdWorkflow) error {
	t := time.Now()
	info := dao.MdWorkflow{}
	_ = copier.Copy(&info, mdWorkflow)
	info.CreatedAt = &t
	info.UpdatedAt = &t
	return info.Create(ctx)
}

func (r *mdWorkflowRepo) UpdateMdWorkflow(ctx context.Context, where map[string]interface{}, date map[string]interface{}) error {
	info := dao.NewMdWorkflow()
	return info.Updates(ctx, date, where)
}

func (r *mdWorkflowRepo) DeleteMdWorkflow(ctx context.Context, where map[string]interface{}) error {
	info := dao.NewMdWorkflow()
	return info.Delete(ctx, where)
}

func (r *mdWorkflowRepo) FindOneMdWorkflow(ctx context.Context, where map[string]interface{}) (*entity.MdWorkflow, error) {
	info := dao.NewMdWorkflow()
	result, err := info.FindOne(ctx, where)
	if err != nil {
		return nil, err
	}
	res := entity.MdWorkflow{}
	_ = copier.Copy(&res, result)
	return &res, err
}

func (r *mdWorkflowRepo) FindListMdWorkflow(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]entity.MdWorkflow, int64, error) {
	info := dao.NewMdWorkflow()
	results, count, err := info.FindList(ctx, where, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	resList := make([]entity.MdWorkflow, len(results))
	for i, v := range results {
		_ = copier.Copy(&resList[i], &v)
	}
	return resList, count, err
}

func (r *mdWorkflowRepo) FindAllMdWorkflow(ctx context.Context, where map[string]interface{}) ([]entity.MdWorkflow, error) {
	info := dao.NewMdWorkflow()
	results, err := info.FindAll(ctx, where)
	if err != nil {
		return nil, err
	}
	resList := make([]entity.MdWorkflow, len(results))
	for i, v := range results {
		_ = copier.Copy(&resList[i], &v)
	}
	return resList, err
}
