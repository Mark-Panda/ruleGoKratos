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

func (uc *MdWorkflowUsecase) List(ctx context.Context, page, size int32) ([]entity.MdWorkflow, int64, error) {
	return uc.repo.FindListMdWorkflow(ctx, nil, int(page), int(size))
}

func (uc *MdWorkflowUsecase) Update(ctx context.Context, md *entity.MdWorkflow) (*entity.MdWorkflow, error) {
	if md.ID > 0 {
		err := uc.repo.UpdateMdWorkflow(ctx, map[string]interface{}{"id": md.ID}, map[string]interface{}{
			"title":    md.Title,
			"content":  md.Content,
			"desc":     md.Desc,
			"chain_id": md.ChainID,
		})
		if err != nil {
			return nil, err
		}
		return uc.repo.FindOneMdWorkflow(ctx, map[string]interface{}{"id": md.ID})
	}
	return nil, nil
}

// Create creates a MdWorkflow, and returns the new MdWorkflow.
func (uc *MdWorkflowUsecase) Create(ctx context.Context, md *entity.MdWorkflow) (*entity.MdWorkflow, error) {
	err := uc.repo.CreateMdWorkflow(ctx, md)
	if err != nil {
		return nil, err
	}
	return md, nil
}
