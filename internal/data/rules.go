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

type ruleGoRepo struct {
	data *Data
	log  *log.Helper
}

// NewRuleGoRepo .
func NewRuleGoRepo(data *Data, logger log.Logger) biz.RuleGoRepo {
	return &ruleGoRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *ruleGoRepo) CreateRegulation(ctx context.Context, regulation *entity.Regulation) error {
	t := time.Now()
	info := dao.Regulation{
		UserName:    regulation.UserName,
		Root:        regulation.Root,
		Disabled:    regulation.Disabled,
		Name:        regulation.Name,
		RuleChainID: regulation.RuleChainID,
		RuleVersion: regulation.RuleVersion,
		RuleConfig:  regulation.RuleConfig,
		CreatedAt:   &t,
		UpdatedAt:   &t,
	}
	err := info.Create(ctx)
	return err
}

func (r *ruleGoRepo) UpdateRegulation(ctx context.Context, where map[string]interface{}, date map[string]interface{}) error {
	info := dao.NewRegulation()
	return info.Updates(ctx, date, where)
}

func (r *ruleGoRepo) DeleteRegulation(ctx context.Context, where map[string]interface{}) error {
	info := dao.NewRegulation()
	return info.Delete(ctx, where)
}

func (r *ruleGoRepo) FindOneRegulation(ctx context.Context, where map[string]interface{}) (*entity.Regulation, error) {
	info := dao.NewRegulation()
	regulation, err := info.FindOne(ctx, where)
	if err != nil {
		return nil, err
	}

	regulationInfo := entity.Regulation{}
	_ = copier.Copy(&regulationInfo, regulation)
	return &regulationInfo, err
}

func (r *ruleGoRepo) FindListRegulation(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]entity.Regulation, int64, error) {
	info := dao.NewRegulation()
	regulations, count, err := info.FindList(ctx, where, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	regulationList := make([]entity.Regulation, len(regulations))
	for i, regulation := range regulations {
		_ = copier.Copy(&regulationList[i], &regulation)
	}
	return regulationList, count, err
}

func (r *ruleGoRepo) FindAllRegulation(ctx context.Context, where map[string]interface{}) ([]entity.Regulation, error) {
	info := dao.NewRegulation()
	regulations, err := info.FindAll(ctx, where)
	if err != nil {
		return nil, err
	}
	regulationList := make([]entity.Regulation, len(regulations))
	for i, regulation := range regulations {
		_ = copier.Copy(&regulationList[i], &regulation)
	}
	return regulationList, err
}
