package data

import (
	"context"
	"time"

	"ruleGoKratos/internal/biz"
	"ruleGoKratos/internal/biz/entity"
	"ruleGoKratos/internal/data/dao"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/jinzhu/copier"
	"gorm.io/gorm"
)

var _ biz.RuleChainRepo = &ruleChainRepo{}

type ruleChainRepo struct {
	data *Data
	log  *log.Helper
}

// NewRuleChainRepo .
func NewRuleChainRepo(data *Data, logger log.Logger) biz.RuleChainRepo {
	return &ruleChainRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *ruleChainRepo) CreateRuleChain(ctx context.Context, ruleChain *entity.RuleChain) error {
	t := time.Now()
	info := dao.RuleChain{
		UserName:       ruleChain.UserName,
		Root:           ruleChain.Root,
		Disabled:       ruleChain.Disabled,
		Name:           ruleChain.Name,
		RuleChainID:    ruleChain.RuleChainID,
		RuleVersion:    ruleChain.RuleVersion,
		Configuration:  ruleChain.Configuration,
		Metadata:       ruleChain.Metadata,
		AdditionalInfo: ruleChain.AdditionalInfo,
		CreatedAt:      &t,
		UpdatedAt:      &t,
	}
	err := info.Create(ctx)
	return err
}

func (r *ruleChainRepo) UpdateRuleChain(ctx context.Context, where map[string]interface{}, data map[string]interface{}) error {
	info := dao.NewRuleChain()
	return info.Updates(ctx, data, where)
}

func (r *ruleChainRepo) DeleteRuleChain(ctx context.Context, where map[string]interface{}) error {
	info := dao.NewRuleChain()
	return info.Delete(ctx, where)
}

func (r *ruleChainRepo) FindOneRuleChain(ctx context.Context, where map[string]interface{}) (*entity.RuleChain, error) {
	info := dao.NewRuleChain()
	ruleChain, err := info.FindOne(ctx, where)
	if err != nil {
		if gorm.ErrRecordNotFound != err {
			return nil, err
		}
		return nil, nil
	}

	ruleChainInfo := entity.RuleChain{}
	_ = copier.Copy(&ruleChainInfo, ruleChain)
	return &ruleChainInfo, nil
}

func (r *ruleChainRepo) FindListRuleChain(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]entity.RuleChain, int64, error) {
	info := dao.NewRuleChain()
	ruleChains, count, err := info.FindList(ctx, where, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	ruleChainList := make([]entity.RuleChain, len(ruleChains))
	for i, item := range ruleChains {
		_ = copier.Copy(&ruleChainList[i], &item)
	}
	return ruleChainList, count, err
}

func (r *ruleChainRepo) FindAllRuleChain(ctx context.Context, where map[string]interface{}) ([]entity.RuleChain, error) {
	info := dao.NewRuleChain()
	ruleChains, err := info.FindAll(ctx, where)
	if err != nil {
		return nil, err
	}
	ruleChainList := make([]entity.RuleChain, len(ruleChains))
	for i, item := range ruleChains {
		_ = copier.Copy(&ruleChainList[i], &item)
	}
	return ruleChainList, err
}
