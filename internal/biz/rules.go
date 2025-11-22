package biz

import (
	"context"

	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz/entity"

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
	CreateRegulation(ctx context.Context, regulation *entity.Regulation) error
	UpdateRegulation(ctx context.Context, where map[string]interface{}, date map[string]interface{}) error
	DeleteRegulation(ctx context.Context, where map[string]interface{}) error
	FindOneRegulation(ctx context.Context, where map[string]interface{}) (*entity.Regulation, error)
	FindListRegulation(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]entity.Regulation, int64, error)
	FindAllRegulation(ctx context.Context, where map[string]interface{}) ([]entity.Regulation, error)

	CreateRunLog(ctx context.Context, runLog *entity.RunLog) error
	UpdateRunLog(ctx context.Context, where map[string]interface{}, date map[string]interface{}) error
	DeleteRunLog(ctx context.Context, where map[string]interface{}) error
	FindOneRunLog(ctx context.Context, where map[string]interface{}) (*entity.RunLog, error)
	FindListRunLog(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]entity.RunLog, int64, error)
	FindAllRunLog(ctx context.Context, where map[string]interface{}) ([]entity.RunLog, error)

	CreateMdWorkflow(ctx context.Context, mdWorkflow *entity.MdWorkflow) error
	UpdateMdWorkflow(ctx context.Context, where map[string]interface{}, date map[string]interface{}) error
	DeleteMdWorkflow(ctx context.Context, where map[string]interface{}) error
	FindOneMdWorkflow(ctx context.Context, where map[string]interface{}) (*entity.MdWorkflow, error)
	FindListMdWorkflow(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]entity.MdWorkflow, int64, error)
	FindAllMdWorkflow(ctx context.Context, where map[string]interface{}) ([]entity.MdWorkflow, error)

	CreateComponentUseRule(ctx context.Context, componentUseRule *entity.ComponentUseRule) error
	UpdateComponentUseRule(ctx context.Context, where map[string]interface{}, date map[string]interface{}) error
	DeleteComponentUseRule(ctx context.Context, where map[string]interface{}) error
	FindOneComponentUseRule(ctx context.Context, where map[string]interface{}) (*entity.ComponentUseRule, error)
	FindListComponentUseRule(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]entity.ComponentUseRule, int64, error)
	FindAllComponentUseRule(ctx context.Context, where map[string]interface{}) ([]entity.ComponentUseRule, error)

	CreateComponentRegulation(ctx context.Context, componentRegulation *entity.ComponentRegulation) error
	UpdateComponentRegulation(ctx context.Context, where map[string]interface{}, date map[string]interface{}) error
	DeleteComponentRegulation(ctx context.Context, where map[string]interface{}) error
	FindOneComponentRegulation(ctx context.Context, where map[string]interface{}) (*entity.ComponentRegulation, error)
	FindListComponentRegulation(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]entity.ComponentRegulation, int64, error)
	FindAllComponentRegulation(ctx context.Context, where map[string]interface{}) ([]entity.ComponentRegulation, error)
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
	return nil, nil
}
