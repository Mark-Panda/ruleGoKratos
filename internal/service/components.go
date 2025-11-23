package service

import (
	"context"
	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz"
	"ruleGoKratos/internal/biz/entity"
	"time"
)

// RuleGoService is a rulego service.
type ComponentService struct {
	v1.UnimplementedComponentServer
	rlu *biz.RunLogUsecase
	cur *biz.ComponentUseRuleUsecase
}

// NewComponentService new a component service.
func NewComponentService(rlu *biz.RunLogUsecase, cur *biz.ComponentUseRuleUsecase) *ComponentService {
	return &ComponentService{rlu: rlu, cur: cur}
}

func (s *ComponentService) ListComponentUseRule(ctx context.Context, req *v1.ListComponentUseRuleRequest) (*v1.ListComponentUseRuleReply, error) {
	list, total, err := s.cur.ListComponentUseRule(ctx, int(req.Page), int(req.Size))
	if err != nil {
		return nil, err
	}
	items := make([]*v1.ComponentUseRuleItem, 0, len(list))
	for _, x := range list {
		createdAt := ""
		if x.CreatedAt != nil {
			createdAt = x.CreatedAt.Format(time.RFC3339Nano)
		}
		updatedAt := ""
		if x.UpdatedAt != nil {
			updatedAt = x.UpdatedAt.Format(time.RFC3339Nano)
		}
		items = append(items, &v1.ComponentUseRuleItem{
			Id:            x.ID,
			ComponentName: x.ComponentName,
			ComponentType: x.ComponentType,
			Disabled:      x.Disabled,
			UseDesc:       x.UseDesc,
			UseRuleDesc:   x.UseRuleDesc,
			CreatedAt:     createdAt,
			UpdatedAt:     updatedAt,
		})
	}
	return &v1.ListComponentUseRuleReply{
		List:  items,
		Total: total,
	}, nil
}

func (s *ComponentService) UpdateComponentUseRule(ctx context.Context, req *v1.UpdateComponentUseRuleRequest) (*v1.UpdateComponentUseRuleReply, error) {
	err := s.cur.UpdateComponentUseRule(ctx, entity.ComponentUseRule{
		ID:            req.Id,
		ComponentName: req.ComponentName,
		ComponentType: req.ComponentType,
		Disabled:      req.Disabled,
		UseDesc:       req.UseDesc,
		UseRuleDesc:   req.UseRuleDesc,
	})
	return &v1.UpdateComponentUseRuleReply{}, err
}

func (s *ComponentService) CreateComponentUseRule(ctx context.Context, req *v1.CreateComponentUseRuleRequest) (*v1.CreateComponentUseRuleReply, error) {
	err := s.cur.CreateComponentUseRule(ctx, entity.ComponentUseRule{
		ComponentName: req.ComponentName,
		ComponentType: req.ComponentType,
		Disabled:      req.Disabled,
		UseDesc:       req.UseDesc,
		UseRuleDesc:   req.UseRuleDesc,
	})
	return &v1.CreateComponentUseRuleReply{}, err
}
