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
func (uc *ComponentUseRuleUsecase) CreateComponentUseRule(ctx context.Context, info entity.ComponentUseRule) error {
	return uc.repo.CreateComponentUseRule(ctx, &info)
}

// ListComponentUseRule list ComponentUseRule
func (uc *ComponentUseRuleUsecase) ListComponentUseRule(ctx context.Context, page, size int) ([]entity.ComponentUseRule, int64, error) {
	return uc.repo.FindListComponentUseRule(ctx, nil, page, size)
}

// UpdateComponentUseRule update ComponentUseRule
func (uc *ComponentUseRuleUsecase) UpdateComponentUseRule(ctx context.Context, info entity.ComponentUseRule) error {
	if info.ID == 0 {
		return uc.repo.CreateComponentUseRule(ctx, &info)
	}
	return uc.repo.UpdateComponentUseRule(ctx, map[string]interface{}{"id": info.ID}, map[string]interface{}{
		"component_name": info.ComponentName,
		"component_type": info.ComponentType,
		"disabled":       info.Disabled,
		"use_desc":       info.UseDesc,
		"use_rule_desc":  info.UseRuleDesc,
	})
}
