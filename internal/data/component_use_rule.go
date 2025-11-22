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

type componentUseRuleRepo struct {
	data *Data
	log  *log.Helper
}

// NewComponentUseRuleRepo .
func NewComponentUseRuleRepo(data *Data, logger log.Logger) biz.ComponentUseRuleRepo {
	return &componentUseRuleRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// ComponentUseRule Methods
func (r *componentUseRuleRepo) CreateComponentUseRule(ctx context.Context, componentUseRule *entity.ComponentUseRule) error {
	t := time.Now()
	info := dao.ComponentUseRule{}
	_ = copier.Copy(&info, componentUseRule)
	info.CreatedAt = &t
	info.UpdatedAt = &t
	return info.Create(ctx)
}

func (r *componentUseRuleRepo) UpdateComponentUseRule(ctx context.Context, where map[string]interface{}, date map[string]interface{}) error {
	info := dao.NewComponentUseRule()
	return info.Updates(ctx, date, where)
}

func (r *componentUseRuleRepo) DeleteComponentUseRule(ctx context.Context, where map[string]interface{}) error {
	info := dao.NewComponentUseRule()
	return info.Delete(ctx, where)
}

func (r *componentUseRuleRepo) FindOneComponentUseRule(ctx context.Context, where map[string]interface{}) (*entity.ComponentUseRule, error) {
	info := dao.NewComponentUseRule()
	result, err := info.FindOne(ctx, where)
	if err != nil {
		return nil, err
	}
	res := entity.ComponentUseRule{}
	_ = copier.Copy(&res, result)
	return &res, err
}

func (r *componentUseRuleRepo) FindListComponentUseRule(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]entity.ComponentUseRule, int64, error) {
	info := dao.NewComponentUseRule()
	results, count, err := info.FindList(ctx, where, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	resList := make([]entity.ComponentUseRule, len(results))
	for i, v := range results {
		_ = copier.Copy(&resList[i], &v)
	}
	return resList, count, err
}

func (r *componentUseRuleRepo) FindAllComponentUseRule(ctx context.Context, where map[string]interface{}) ([]entity.ComponentUseRule, error) {
	info := dao.NewComponentUseRule()
	results, err := info.FindAll(ctx, where)
	if err != nil {
		return nil, err
	}
	resList := make([]entity.ComponentUseRule, len(results))
	for i, v := range results {
		_ = copier.Copy(&resList[i], &v)
	}
	return resList, err
}
