package biz

import (
	"context"
	"ruleGoKratos/internal/biz/entity"

	"github.com/go-kratos/kratos/v2/log"
)

// RuleGoRepo is a RuleGo repo.
type ComponentUseRuleRepo interface {
	CreateComponentUseRule(ctx context.Context, componentUseRule *entity.ComponentUseRule) error
	UpdateComponentUseRule(ctx context.Context, where map[string]interface{}, date map[string]interface{}) error
	DeleteComponentUseRule(ctx context.Context, where map[string]interface{}) error
	FindOneComponentUseRule(ctx context.Context, where map[string]interface{}) (*entity.ComponentUseRule, error)
	FindListComponentUseRule(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]entity.ComponentUseRule, int64, error)
	FindAllComponentUseRule(ctx context.Context, where map[string]interface{}) ([]entity.ComponentUseRule, error)
}

// ComponentUseRuleUsecase is a ComponentUseRule usecase.
type ComponentUseRuleUsecase struct {
	repo ComponentUseRuleRepo
	log  *log.Helper
}

// NewComponentUseRuleUsecase new a ComponentUseRule usecase.
func NewComponentUseRuleUsecase(repo ComponentUseRuleRepo, logger log.Logger) *ComponentUseRuleUsecase {
	return &ComponentUseRuleUsecase{repo: repo, log: log.NewHelper(logger)}
}

// CreateComponentUseRule creates a ComponentUseRule, and returns the new ComponentUseRule.
func (uc *ComponentUseRuleUsecase) CreateComponentUseRule(ctx context.Context) error {
	return nil
}
