package biz

import (
	"context"
	"ruleGoKratos/internal/biz/entity"

	"github.com/go-kratos/kratos/v2/log"
)

// ComponentRegulationRepo is a ComponentRegulation repo.
type ComponentRegulationRepo interface {
	CreateComponentRegulation(ctx context.Context, componentRegulation *entity.ComponentRegulation) error
	UpdateComponentRegulation(ctx context.Context, where map[string]interface{}, date map[string]interface{}) error
	DeleteComponentRegulation(ctx context.Context, where map[string]interface{}) error
	FindOneComponentRegulation(ctx context.Context, where map[string]interface{}) (*entity.ComponentRegulation, error)
	FindListComponentRegulation(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]entity.ComponentRegulation, int64, error)
	FindAllComponentRegulation(ctx context.Context, where map[string]interface{}) ([]entity.ComponentRegulation, error)
}

// ComponentRegulationUsecase is a ComponentRegulation usecase.
type ComponentRegulationUsecase struct {
	repo ComponentRegulationRepo
	log  *log.Helper
}

// NewComponentRegulationUsecase new a ComponentRegulation usecase.
func NewComponentRegulationUsecase(repo ComponentRegulationRepo, logger log.Logger) *ComponentRegulationUsecase {
	return &ComponentRegulationUsecase{repo: repo, log: log.NewHelper(logger)}
}

// CreateComponentRegulation creates a ComponentRegulation, and returns the new ComponentRegulation.
func (uc *ComponentRegulationUsecase) CreateComponentRegulation(ctx context.Context) error {
	return nil
}
