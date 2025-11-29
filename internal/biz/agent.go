package biz

import (
	"context"
	"fmt"
	"ruleGoKratos/internal/conf"

	"github.com/go-kratos/blades"
	"github.com/go-kratos/blades/contrib/openai"
	"github.com/go-kratos/blades/tools"
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
		blades.WithInstruction(nodeSystemPrompt),
		blades.WithDescription("You are a helpful assistant that provides detailed and accurate information."),
		// blades.WithTools(tools.NewTool()),
		// blades.WithOutputSchema(&jsonschema.Schema{}),
	)
}

// GetModelProvider 获取模型提供者
func (uc *AgentUsecase) GetModelProvider(modelName string) (blades.ModelProvider, error) {
	if modelName == "" {
		// 默认使用配置中的模型
		if uc.config.Ai != nil && uc.config.Ai.Doubao != nil && uc.config.Ai.Doubao.Model != "" {
			modelName = uc.config.Ai.Doubao.Model
			return openai.NewModel(modelName, openai.Config{
				APIKey:  uc.config.Ai.Doubao.ApiKey,
				BaseURL: uc.config.Ai.Doubao.ApiBaseUrl,
			}), nil
		} else if uc.config.Ai != nil && uc.config.Ai.Openai != nil && uc.config.Ai.Openai.Model != "" {
			modelName = uc.config.Ai.Openai.Model
			return openai.NewModel(modelName, openai.Config{
				APIKey:  uc.config.Ai.Openai.ApiKey,
				BaseURL: uc.config.Ai.Openai.ApiBaseUrl,
			}), nil
		}
		return nil, fmt.Errorf("未配置AI模型")
	}

	// 根据模型名称选择配置
	if uc.config.Ai != nil && uc.config.Ai.Doubao != nil && uc.config.Ai.Doubao.Model == modelName {
		return openai.NewModel(modelName, openai.Config{
			APIKey:  uc.config.Ai.Doubao.ApiKey,
			BaseURL: uc.config.Ai.Doubao.ApiBaseUrl,
		}), nil
	} else if uc.config.Ai != nil && uc.config.Ai.Openai != nil {
		return openai.NewModel(modelName, openai.Config{
			APIKey:  uc.config.Ai.Openai.ApiKey,
			BaseURL: uc.config.Ai.Openai.ApiBaseUrl,
		}), nil
	}

	return nil, fmt.Errorf("未找到模型配置: %s", modelName)
}

// ChatStream 流式对话
func (uc *AgentUsecase) ChatStream(ctx context.Context, modelName string, history []*blades.Message, userMessage string) blades.Generator[*blades.Message, error] {
	model, err := uc.GetModelProvider(modelName)
	if err != nil {
		// 返回一个生成器，立即返回错误
		return func(yield func(*blades.Message, error) bool) {
			yield(nil, err)
		}
	}
	// 读取文件内容
	// content, err := os.ReadFile("proto.md")
	// if err != nil {
	// 	return func(yield func(*blades.Message, error) bool) {
	// 		yield(nil, err)
	// 	}
	// }
	// systemPrompt := string(content)

	agent, err := blades.NewAgent(
		"Chat Assistant",
		blades.WithModel(model),
		// blades.WithInstruction(systemPrompt),
		blades.WithInstruction("你是一个有用的AI助手，能够回答用户的问题。"),
	)
	if err != nil {
		return func(yield func(*blades.Message, error) bool) {
			yield(nil, err)
		}
	}

	// 构建用户消息
	msg := blades.UserMessage(userMessage)

	// 构建Invocation，包含历史消息
	invocation := &blades.Invocation{
		ID:         blades.NewInvocationID(),
		Message:    msg,
		History:    history,
		Streamable: true,
	}

	// 使用Agent的Run方法进行流式对话
	return agent.Run(ctx, invocation)
}

// 创建任务规划agent
func (uc *AgentUsecase) CreateTaskPlanningAgent(ctx context.Context, history []*blades.Message, userMessage string) blades.Generator[*blades.Message, error] {
	model, err := uc.GetModelProvider("")
	if err != nil {
		return func(yield func(*blades.Message, error) bool) {
			yield(nil, err)
		}
	}
	translatorWorkers, err := uc.CreateTranslatorWorkers(model)
	if err != nil {
		uc.log.Errorf("创建子代理工具失败:", err)
		return func(yield func(*blades.Message, error) bool) {
			yield(nil, err)
		}
	}
	orchestratorAgent, err := blades.NewAgent(
		"orchestrator_agent",
		blades.WithInstruction(`你是一个翻译代理。你使用提供给你的工具进行翻译。如果要求提供多个翻译，你需要按顺序调用相关工具。你永远不要独自翻译，你总是要使用提供的工具。`),
		blades.WithModel(model),
		blades.WithTools(translatorWorkers...),
		blades.WithMiddleware(NewLogging),
	)
	if err != nil {
		uc.log.Errorf("任务规划失败:", err)
		return func(yield func(*blades.Message, error) bool) {
			yield(nil, err)
		}
	}
	synthesizerAgent, err := blades.NewAgent(
		"synthesizer_agent",
		blades.WithInstruction("你检查翻译内容，如有需要则进行修正，并生成最终的连贯回复。"),
		blades.WithModel(model),
		blades.WithMiddleware(NewLogging),
	)
	if err != nil {
		uc.log.Errorf("任务规划失败1111:", err)
		return func(yield func(*blades.Message, error) bool) {
			yield(nil, err)
		}
	}
	input := blades.UserMessage(userMessage)
	orchestratorRunner := blades.NewRunner(orchestratorAgent)
	// 构建Invocation，包含历史消息
	// invocation := &blades.Invocation{
	// 	ID:         blades.NewInvocationID(),
	// 	Message:    input,
	// 	History:    history,
	// 	Streamable: true,
	// }

	// 运行任务规划
	stream := orchestratorRunner.RunStream(ctx, input)
	var message *blades.Message
	for message, err = range stream {
		if err != nil {
			uc.log.Errorf("任务规划失败2222:", err)
			return func(yield func(*blades.Message, error) bool) {
				yield(nil, err)
			}
		}
	}
	// 运行任务修正
	synthesizerRunner := blades.NewRunner(synthesizerAgent)
	output, err := synthesizerRunner.Run(ctx, blades.UserMessage(message.Text()))
	if err != nil {
		uc.log.Errorf("任务规划失败3333:", err)
		return func(yield func(*blades.Message, error) bool) {
			yield(nil, err)
		}
	}
	return func(yield func(*blades.Message, error) bool) {
		yield(output, nil)
	}
}

// 子代理工具
func (uc *AgentUsecase) CreateTranslatorWorkers(model blades.ModelProvider) ([]tools.Tool, error) {
	spanishAgent, err := blades.NewAgent(
		"spanish_agent",
		blades.WithDescription("An English to Spanish translator"),
		blades.WithInstruction("You translate the user's message to Spanish"),
		blades.WithModel(model),
		blades.WithMiddleware(NewLogging),
	)
	if err != nil {
		uc.log.Errorf("子代理工具失败:", err)
		return nil, err
	}
	frenchAgent, err := blades.NewAgent(
		"french_agent",
		blades.WithDescription("An English to French translator"),
		blades.WithInstruction("You translate the user's message to French"),
		blades.WithModel(model),
		blades.WithMiddleware(NewLogging),
	)
	if err != nil {
		uc.log.Errorf("子代理工具失败1111:", err)
		return nil, err
	}
	italianAgent, err := blades.NewAgent(
		"italian_agent",
		blades.WithDescription("An English to Italian translator"),
		blades.WithInstruction("You translate the user's message to Italian"),
		blades.WithModel(model),
		blades.WithMiddleware(NewLogging),
	)
	if err != nil {
		uc.log.Errorf("子代理工具失败2222:", err)
		return nil, err
	}
	return []tools.Tool{
		blades.NewAgentTool(spanishAgent),
		blades.NewAgentTool(frenchAgent),
		blades.NewAgentTool(italianAgent),
	}, nil
}
