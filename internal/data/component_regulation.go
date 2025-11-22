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

var _ biz.ComponentRegulationRepo = &componentRegulationRepo{}

type componentRegulationRepo struct {
	data *Data
	log  *log.Helper
}

// NewComponentRegulationRepo .
func NewComponentRegulationRepo(data *Data, logger log.Logger) biz.ComponentRegulationRepo {
	return &componentRegulationRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// ComponentRegulation Methods
func (r *componentRegulationRepo) CreateComponentRegulation(ctx context.Context, componentRegulation *entity.ComponentRegulation) error {
	t := time.Now()
	info := dao.ComponentRegulation{}
	_ = copier.Copy(&info, componentRegulation)
	info.CreatedAt = &t
	info.UpdatedAt = &t
	return info.Create(ctx)
}

func (r *componentRegulationRepo) UpdateComponentRegulation(ctx context.Context, where map[string]interface{}, date map[string]interface{}) error {
	info := dao.NewComponentRegulation()
	return info.Updates(ctx, date, where)
}

func (r *componentRegulationRepo) DeleteComponentRegulation(ctx context.Context, where map[string]interface{}) error {
	info := dao.NewComponentRegulation()
	return info.Delete(ctx, where)
}

func (r *componentRegulationRepo) FindOneComponentRegulation(ctx context.Context, where map[string]interface{}) (*entity.ComponentRegulation, error) {
	info := dao.NewComponentRegulation()
	result, err := info.FindOne(ctx, where)
	if err != nil {
		return nil, err
	}
	res := entity.ComponentRegulation{}
	_ = copier.Copy(&res, result)
	return &res, err
}

func (r *componentRegulationRepo) FindListComponentRegulation(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]entity.ComponentRegulation, int64, error) {
	info := dao.NewComponentRegulation()
	results, count, err := info.FindList(ctx, where, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	resList := make([]entity.ComponentRegulation, len(results))
	for i, v := range results {
		_ = copier.Copy(&resList[i], &v)
	}
	return resList, count, err
}

func (r *componentRegulationRepo) FindAllComponentRegulation(ctx context.Context, where map[string]interface{}) ([]entity.ComponentRegulation, error) {
	info := dao.NewComponentRegulation()
	results, err := info.FindAll(ctx, where)
	if err != nil {
		return nil, err
	}
	resList := make([]entity.ComponentRegulation, len(results))
	for i, v := range results {
		_ = copier.Copy(&resList[i], &v)
	}
	return resList, err
}
