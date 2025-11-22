package service

import (
	"context"

	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz"
)

// RuleGoService is a rulego service.
type RuleGoService struct {
	v1.UnimplementedRuleGoServer

	uc *biz.RuleGoUsecase
}

// NewRuleGoService new a rulego service.
func NewRuleGoService(uc *biz.RuleGoUsecase) *RuleGoService {
	return &RuleGoService{uc: uc}
}

// SayHello implements helloworld.GreeterServer.
func (s *RuleGoService) SayHello(ctx context.Context, in *v1.HelloRequest) (*v1.HelloReply, error) {
	g, err := s.uc.CreateRuleGo(ctx, &biz.RuleGo{Hello: in.Name})
	if err != nil {
		return nil, err
	}
	return &v1.HelloReply{Message: "Hello " + g.Hello}, nil
}
