package biz

import (
	"context"
	"os"
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
	model := openai.NewModel("gpt-5", openai.Config{
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
	})
	_, err := blades.NewAgent(
		"Blades Agent",
		blades.WithModel(model),
		blades.WithInstructions("You are a helpful assistant that provides detailed and accurate information."),
	)
	if err != nil {
		return err
	}
	return nil
}
