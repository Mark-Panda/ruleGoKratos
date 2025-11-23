package service

import (
	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz"
)

// RuleGoService is a rulego service.
type MdWorkflowService struct {
	v1.UnimplementedMdWorkflowServer
	rlu *biz.RunLogUsecase
}

// NewMdWorkflowService new a mdworkflow service.
func NewMdWorkflowService(rlu *biz.RunLogUsecase) *MdWorkflowService {
	return &MdWorkflowService{rlu: rlu}
}
