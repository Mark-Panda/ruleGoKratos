package data

import (
	"context"

	"ruleGoKratos/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
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

func (r *ruleGoRepo) Save(ctx context.Context, g *biz.RuleGo) (*biz.RuleGo, error) {
	return g, nil
}

func (r *ruleGoRepo) Update(ctx context.Context, g *biz.RuleGo) (*biz.RuleGo, error) {
	return g, nil
}

func (r *ruleGoRepo) FindByID(context.Context, int64) (*biz.RuleGo, error) {
	return nil, nil
}

func (r *ruleGoRepo) ListByHello(context.Context, string) ([]*biz.RuleGo, error) {
	return nil, nil
}

func (r *ruleGoRepo) ListAll(context.Context) ([]*biz.RuleGo, error) {
	return nil, nil
}
