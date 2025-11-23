package service

import (
	"context"
	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz"
	"ruleGoKratos/internal/biz/entity"
	"time"
)

// MdWorkflowService is a mdworkflow service.
type MdWorkflowService struct {
	v1.UnimplementedMdWorkflowServer
	uc *biz.MdWorkflowUsecase
}

// NewMdWorkflowService new a mdworkflow service.
func NewMdWorkflowService(uc *biz.MdWorkflowUsecase) *MdWorkflowService {
	return &MdWorkflowService{uc: uc}
}

func (s *MdWorkflowService) List(ctx context.Context, req *v1.ListMdRequest) (*v1.ListMdReply, error) {
	list, total, err := s.uc.List(ctx, req.Page, req.Size)
	if err != nil {
		return nil, err
	}
	items := make([]*v1.MdItem, 0, len(list))
	for _, x := range list {
		item := &v1.MdItem{
			Id:           x.ID,
			Title:        x.Title,
			Content:      x.Content,
			Desc:         x.Desc,
			ChainId:      x.ChainID,
			ChainName:    x.ChainName,
			ChainVersion: int32(x.ChainVersion),
		}
		if x.CreatedAt != nil {
			item.CreatedAt = x.CreatedAt.Format(time.RFC3339Nano)
		}
		if x.UpdatedAt != nil {
			item.UpdatedAt = x.UpdatedAt.Format(time.RFC3339Nano)
		}
		items = append(items, item)
	}
	return &v1.ListMdReply{
		Items: items,
		Page:  req.Page,
		Size:  req.Size,
		Total: total,
	}, nil
}

func (s *MdWorkflowService) Update(ctx context.Context, req *v1.UpdateMdRequest) (*v1.MdItem, error) {
	md := &entity.MdWorkflow{
		ID:      req.Id,
		Title:   req.Title,
		Content: req.Content,
		Desc:    req.Desc,
		ChainID: req.ChainId,
	}
	updated, err := s.uc.Update(ctx, md)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, nil
	}

	item := &v1.MdItem{
		Id:           updated.ID,
		Title:        updated.Title,
		Content:      updated.Content,
		Desc:         updated.Desc,
		ChainId:      updated.ChainID,
		ChainName:    updated.ChainName,
		ChainVersion: int32(updated.ChainVersion),
	}
	if updated.CreatedAt != nil {
		item.CreatedAt = updated.CreatedAt.Format(time.RFC3339Nano)
	}
	if updated.UpdatedAt != nil {
		item.UpdatedAt = updated.UpdatedAt.Format(time.RFC3339Nano)
	}
	return item, nil
}

func (s *MdWorkflowService) Create(ctx context.Context, req *v1.CreateMdRequest) (*v1.MdItem, error) {
	md := &entity.MdWorkflow{
		Title:   req.Title,
		Content: req.Content,
		Desc:    req.Desc,
		ChainID: req.ChainId,
	}
	created, err := s.uc.Create(ctx, md)
	if err != nil {
		return nil, err
	}

	item := &v1.MdItem{
		Id:           created.ID,
		Title:        created.Title,
		Content:      created.Content,
		Desc:         created.Desc,
		ChainId:      created.ChainID,
		ChainName:    created.ChainName,
		ChainVersion: int32(created.ChainVersion),
	}
	if created.CreatedAt != nil {
		item.CreatedAt = created.CreatedAt.Format(time.RFC3339Nano)
	}
	if created.UpdatedAt != nil {
		item.UpdatedAt = created.UpdatedAt.Format(time.RFC3339Nano)
	}
	return item, nil
}
