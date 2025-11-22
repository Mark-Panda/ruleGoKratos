package biz

import (
	"context"

	v1 "ruleGoKratos/api/rulego/v1"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/rulego/rulego"
)

var (
	// ErrUserNotFound is user not found.
	ErrRuleGoNotFound = errors.NotFound(v1.ErrorReason_RULEGO_NOT_FOUND.String(), "user not found")
)

// RuleGo is a RuleGo model.
type RuleGo struct {
	Hello string
}

// RuleGoRepo is a RuleGo repo.
type RuleGoRepo interface {
	Save(context.Context, *RuleGo) (*RuleGo, error)
	Update(context.Context, *RuleGo) (*RuleGo, error)
	FindByID(context.Context, int64) (*RuleGo, error)
	ListByHello(context.Context, string) ([]*RuleGo, error)
	ListAll(context.Context) ([]*RuleGo, error)
}

// RuleGoUsecase is a RuleGo usecase.
type RuleGoUsecase struct {
	repo       RuleGoRepo
	log        *log.Helper
	ruleEngine *rulego.RuleGo
}

// NewRuleGoUsecase new a RuleGo usecase.
func NewRuleGoUsecase(repo RuleGoRepo, logger log.Logger, ruleEngine *rulego.RuleGo) *RuleGoUsecase {
	return &RuleGoUsecase{repo: repo, log: log.NewHelper(logger), ruleEngine: ruleEngine}
}

// CreateRuleGo creates a RuleGo, and returns the new RuleGo.
func (uc *RuleGoUsecase) CreateRuleGo(ctx context.Context, g *RuleGo) (*RuleGo, error) {
	uc.log.WithContext(ctx).Infof("CreateGreeter: %v", g.Hello)
	return uc.repo.Save(ctx, g)
}
