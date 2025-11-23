package service

import (
	"context"

	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz"
)

// RuleGoService is a rulego service.
type RuleGoService struct {
	v1.UnimplementedRuleGoServer
	rc  *biz.RuleChainUsecase
	cru *biz.ComponentRegulationUsecase
	cur *biz.ComponentUseRuleUsecase
	mwu *biz.MdWorkflowUsecase
	rlu *biz.RunLogUsecase
}

// NewRuleGoService new a rulego service.
func NewRuleGoService(rc *biz.RuleChainUsecase, cru *biz.ComponentRegulationUsecase, cur *biz.ComponentUseRuleUsecase, mwu *biz.MdWorkflowUsecase, rlu *biz.RunLogUsecase) *RuleGoService {
	return &RuleGoService{rc: rc, cru: cru, cur: cur, mwu: mwu, rlu: rlu}
}

// SayHello implements helloworld.GreeterServer.
func (s *RuleGoService) GetComponents(ctx context.Context, in *v1.GetComponentsReq) (*v1.GetComponentsReply, error) {
	return s.rc.GetComponents(ctx)
}

func (s *RuleGoService) GetRegulationsList(ctx context.Context, in *v1.GetRegulationsListReq) (*v1.GetRegulationsListReply, error) {
	return s.rc.GetRegulationsList(ctx, in)
}

func (s *RuleGoService) GetRuleChain(ctx context.Context, in *v1.GetRuleChainReq) (*v1.GetRuleChainReply, error) {
	return s.rc.GetRuleChain(ctx, in)
}

func (s *RuleGoService) ExecuteRuleChain(ctx context.Context, in *v1.ExecuteRuleChainReq) (*v1.ExecuteRuleChainReply, error) {
	return s.rc.ExecuteRuleChain(ctx, in)
}

func (s *RuleGoService) ExecuteRuleChainSync(ctx context.Context, in *v1.ExecuteRuleChainReq) (*v1.ExecuteRuleChainSyncReply, error) {
	return s.rc.ExecuteRuleChainSync(ctx, in)
}

func (s *RuleGoService) DeployRuleChain(ctx context.Context, in *v1.DeployRuleChainReq) (*v1.DeployRuleChainReply, error) {
	return s.rc.DeployRuleChain(ctx, in)
}

func (s *RuleGoService) UpdateRuleChainBaseInfo(ctx context.Context, in *v1.UpdateRuleChainBaseInfoReq) (*v1.UpdateRuleChainBaseInfoReply, error) {
	return s.rc.UpdateRuleChainBaseInfo(ctx, in)
}

func (s *RuleGoService) UpsertRuleChain(ctx context.Context, in *v1.UpsertRuleChainReq) (*v1.UpsertRuleChainReply, error) {
	return s.rc.UpsertRuleChain(ctx, in)
}

func (s *RuleGoService) DeleteRuleChain(ctx context.Context, in *v1.DeleteRuleChainReq) (*v1.DeleteRuleChainReply, error) {
	return s.rc.DeleteRuleChain(ctx, in)
}
