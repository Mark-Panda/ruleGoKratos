package service

import (
	"context"

	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz"
)

// RuleGoService is a rulego service.
type RuleGoService struct {
	v1.UnimplementedRuleGoServer

	ru  *biz.RegulationUsecase
	cru *biz.ComponentRegulationUsecase
	cur *biz.ComponentUseRuleUsecase
	mwu *biz.MdWorkflowUsecase
	rlu *biz.RunLogUsecase
}

// NewRuleGoService new a rulego service.
func NewRuleGoService(ru *biz.RegulationUsecase, cru *biz.ComponentRegulationUsecase, cur *biz.ComponentUseRuleUsecase, mwu *biz.MdWorkflowUsecase, rlu *biz.RunLogUsecase) *RuleGoService {
	return &RuleGoService{ru: ru, cru: cru, cur: cur, mwu: mwu, rlu: rlu}
}

// SayHello implements helloworld.GreeterServer.
func (s *RuleGoService) SayHello(ctx context.Context, in *v1.HelloRequest) (*v1.HelloReply, error) {
	return nil, nil
}
