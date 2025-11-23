package service

import (
	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz"
)

// RuleGoService is a rulego service.
type RunLogService struct {
	v1.UnimplementedRunLogServer
	rlu *biz.RunLogUsecase
}

// NewRunLogService new a runlog service.
func NewRunLogService(rlu *biz.RunLogUsecase) *RunLogService {
	return &RunLogService{rlu: rlu}
}
