package service

import (
	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz"
)

// RuleGoService is a rulego service.
type ComponentService struct {
	v1.UnimplementedComponentServer
	rlu *biz.RunLogUsecase
}

// NewComponentService new a component service.
func NewComponentService(rlu *biz.RunLogUsecase) *ComponentService {
	return &ComponentService{rlu: rlu}
}
