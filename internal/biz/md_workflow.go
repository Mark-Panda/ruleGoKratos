package biz

import (
	"context"
	"ruleGoKratos/internal/biz/entity"

	"github.com/go-kratos/kratos/v2/log"
)

// MdWorkflowRepo is a MdWorkflow repo.
type MdWorkflowRepo interface {
	CreateMdWorkflow(ctx context.Context, mdWorkflow *entity.MdWorkflow) error
	UpdateMdWorkflow(ctx context.Context, where map[string]interface{}, date map[string]interface{}) error
	DeleteMdWorkflow(ctx context.Context, where map[string]interface{}) error
	FindOneMdWorkflow(ctx context.Context, where map[string]interface{}) (*entity.MdWorkflow, error)
	FindListMdWorkflow(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]entity.MdWorkflow, int64, error)
	FindAllMdWorkflow(ctx context.Context, where map[string]interface{}) ([]entity.MdWorkflow, error)
}

// MdWorkflowUsecase is a MdWorkflow usecase.
type MdWorkflowUsecase struct {
	repo MdWorkflowRepo
	log  *log.Helper
}

// NewMdWorkflowUsecase new a MdWorkflow usecase.
func NewMdWorkflowUsecase(repo MdWorkflowRepo, logger log.Logger) *MdWorkflowUsecase {
	return &MdWorkflowUsecase{repo: repo, log: log.NewHelper(logger)}
}

// CreateMdWorkflow creates a MdWorkflow, and returns the new MdWorkflow.
func (uc *MdWorkflowUsecase) CreateMdWorkflow(ctx context.Context) error {
	return nil
}
