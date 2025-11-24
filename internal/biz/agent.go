package biz

import (
	"context"
	"fmt"
	"ruleGoKratos/internal/conf"

	"github.com/go-kratos/blades"
	"github.com/go-kratos/kratos/v2/log"
)

type AgentUsecase struct {
	log                  *log.Helper
	config               *conf.Bootstrap
	componentUseRuleRepo ComponentUseRuleRepo
}

func NewAgentUsecase(logger log.Logger, config *conf.Bootstrap, componentUseRuleRepo ComponentUseRuleRepo) *AgentUsecase {
	return &AgentUsecase{log: log.NewHelper(logger), config: config, componentUseRuleRepo: componentUseRuleRepo}
}

func (uc *AgentUsecase) CreateAgent(ctx context.Context) error {
	// model := openai.NewModel(uc.config.Ai.Doubao.Model, openai.Config{
	// 	APIKey:  uc.config.Ai.Doubao.ApiKey,
	// 	BaseURL: uc.config.Ai.Doubao.ApiBaseUrl,
	// })

	return nil
}

// 节点agent
func (uc *AgentUsecase) CreateNodeAgent(ctx context.Context, model blades.ModelProvider, nodeType string) (blades.Agent, error) {
	componentUseRule, err := uc.componentUseRuleRepo.FindOneComponentUseRule(ctx, map[string]interface{}{"component_name": nodeType})
	if err != nil {
		return nil, err
	}
	nodeSystemPrompt := fmt.Sprintf(`
	你是一个RuleGo %s节点构造师,你擅长根据用户给的需求构造出符合需求的RuleGo节点
	`, componentUseRule.ComponentName)
	return blades.NewAgent(
		"RuleGo Node Generator Agent",
		blades.WithModel(model),
		blades.WithInstructions(nodeSystemPrompt),
		blades.WithDescription("You are a helpful assistant that provides detailed and accurate information."),
		// blades.WithTools(tools.NewTool()),
		// blades.WithOutputSchema(&jsonschema.Schema{}),
	)
}
