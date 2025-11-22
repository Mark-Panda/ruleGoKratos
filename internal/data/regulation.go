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

var _ biz.RegulationRepo = &regulationRepo{}

type regulationRepo struct {
	data *Data
	log  *log.Helper
}

// NewRegulationRepo .
func NewRegulationRepo(data *Data, logger log.Logger) biz.RegulationRepo {
	return &regulationRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *regulationRepo) CreateRegulation(ctx context.Context, regulation *entity.Regulation) error {
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

func (r *regulationRepo) UpdateRegulation(ctx context.Context, where map[string]interface{}, date map[string]interface{}) error {
	info := dao.NewRegulation()
	return info.Updates(ctx, date, where)
}

func (r *regulationRepo) DeleteRegulation(ctx context.Context, where map[string]interface{}) error {
	info := dao.NewRegulation()
	return info.Delete(ctx, where)
}

func (r *regulationRepo) FindOneRegulation(ctx context.Context, where map[string]interface{}) (*entity.Regulation, error) {
	info := dao.NewRegulation()
	regulation, err := info.FindOne(ctx, where)
	if err != nil {
		return nil, err
	}

	regulationInfo := entity.Regulation{}
	_ = copier.Copy(&regulationInfo, regulation)
	return &regulationInfo, err
}

func (r *regulationRepo) FindListRegulation(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]entity.Regulation, int64, error) {
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

func (r *regulationRepo) FindAllRegulation(ctx context.Context, where map[string]interface{}) ([]entity.Regulation, error) {
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
