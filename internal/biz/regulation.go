package biz

import (
	"context"

	"ruleGoKratos/internal/biz/entity"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/rulego/rulego"
)

type RegulationRepo interface {
	CreateRegulation(ctx context.Context, regulation *entity.Regulation) error
	UpdateRegulation(ctx context.Context, where map[string]interface{}, date map[string]interface{}) error
	DeleteRegulation(ctx context.Context, where map[string]interface{}) error
	FindOneRegulation(ctx context.Context, where map[string]interface{}) (*entity.Regulation, error)
	FindListRegulation(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]entity.Regulation, int64, error)
	FindAllRegulation(ctx context.Context, where map[string]interface{}) ([]entity.Regulation, error)
}

// RegulationUsecase is a Regulation usecase.
type RegulationUsecase struct {
	repo       RegulationRepo
	log        *log.Helper
	ruleEngine *rulego.RuleGo
}

// NewRegulationUsecase new a Regulation usecase.
func NewRegulationUsecase(repo RegulationRepo, logger log.Logger, ruleEngine *rulego.RuleGo) *RegulationUsecase {
	return &RegulationUsecase{repo: repo, log: log.NewHelper(logger), ruleEngine: ruleEngine}
}
