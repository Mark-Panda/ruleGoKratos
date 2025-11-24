package biz

import (
	"context"
	"ruleGoKratos/internal/conf"

	"github.com/go-kratos/blades"
	"github.com/go-kratos/blades/contrib/openai"
	"github.com/go-kratos/kratos/v2/log"
)

type AgentUsecase struct {
	log    *log.Helper
	config *conf.Bootstrap
}

func NewAgentUsecase(logger log.Logger, config *conf.Bootstrap) *AgentUsecase {
	return &AgentUsecase{log: log.NewHelper(logger), config: config}
}

func (uc *AgentUsecase) CreateAgent(ctx context.Context) error {
	model := openai.NewModel(uc.config.Ai.Doubao.Model, openai.Config{
		APIKey:  uc.config.Ai.Doubao.ApiKey,
		BaseURL: uc.config.Ai.Doubao.ApiBaseUrl,
	})
	_, err := blades.NewAgent(
		"RuleGo Node Generator Agent",
		blades.WithModel(model),
		blades.WithInstructions("You are a helpful assistant that provides detailed and accurate information."),
		blades.WithDescription("You are a helpful assistant that provides detailed and accurate information."),
		// blades.WithTools(tools.NewTool()),
		// blades.WithOutputSchema(&jsonschema.Schema{}),
	)
	if err != nil {
		return err
	}
	return nil
}
