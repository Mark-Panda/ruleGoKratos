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
func (s *RuleGoService) GetComponents(ctx context.Context, in *v1.GetComponentsReq) (*v1.GetComponentsReply, error) {
	return s.ru.GetComponents(ctx)
}

func (s *RuleGoService) GetRegulationsList(ctx context.Context, in *v1.GetRegulationsListReq) (*v1.GetRegulationsListReply, error) {
	return s.ru.GetRegulationsList(ctx, in)
}

func (s *RuleGoService) ExecuteRuleChain(ctx context.Context, in *v1.ExecuteRuleChainReq) (*v1.ExecuteRuleChainReply, error) {
	return s.ru.ExecuteRuleChain(ctx, in)
}

func (s *RuleGoService) ExecuteRuleChainSync(ctx context.Context, in *v1.ExecuteRuleChainReq) (*v1.ExecuteRuleChainSyncReply, error) {
	return s.ru.ExecuteRuleChainSync(ctx, in)
}
