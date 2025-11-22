package biz

import (
	"context"
	"ruleGoKratos/internal/biz/entity"

	"github.com/go-kratos/kratos/v2/log"
)

// RuleGoRepo is a RuleGo repo.
type RunLogRepo interface {
	CreateRunLog(ctx context.Context, runLog *entity.RunLog) error
	UpdateRunLog(ctx context.Context, where map[string]interface{}, date map[string]interface{}) error
	DeleteRunLog(ctx context.Context, where map[string]interface{}) error
	FindOneRunLog(ctx context.Context, where map[string]interface{}) (*entity.RunLog, error)
	FindListRunLog(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]entity.RunLog, int64, error)
	FindAllRunLog(ctx context.Context, where map[string]interface{}) ([]entity.RunLog, error)
}

// RunLogUsecase is a RunLog usecase.
type RunLogUsecase struct {
	repo RunLogRepo
	log  *log.Helper
}

// NewRunLogUsecase new a RunLog usecase.
func NewRunLogUsecase(repo RunLogRepo, logger log.Logger) *RunLogUsecase {
	return &RunLogUsecase{repo: repo, log: log.NewHelper(logger)}
}

// CreateRunLog creates a RunLog, and returns the new RunLog.
func (uc *RunLogUsecase) CreateRunLog(ctx context.Context) error {
	return nil
}
