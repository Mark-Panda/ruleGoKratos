package biz

import (
	"context"
	"fmt"
	"ruleGoKratos/internal/biz/entity"
	"ruleGoKratos/internal/conf"

	"github.com/go-kratos/blades"
	"github.com/go-kratos/blades/contrib/openai"
	"github.com/go-kratos/blades/tools"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

type AgentUsecase struct {
	log                  *log.Helper
	config               *conf.Bootstrap
	componentUseRuleRepo ComponentUseRuleRepo
}

func NewAgentUsecase(logger log.Logger, config *conf.Bootstrap, componentUseRuleRepo ComponentUseRuleRepo) *AgentUsecase {
	return &AgentUsecase{log: log.NewHelper(logger), config: config, componentUseRuleRepo: componentUseRuleRepo}
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
	agent, err := blades.NewAgent(
		"RuleGo规则链架构师",
		blades.WithModel(model),
		blades.WithInstruction(RuleChainPlannerPrompt),
		blades.WithDescription("将Markdown格式的业务流程文档转化为符合官方规范的RuleGo规则链JSON"),
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

// 创建RuleGo任务规划agent
func (uc *AgentUsecase) CreateRuleChainPlannerAgent(ctx context.Context, userMessage string) blades.Generator[*blades.Message, error] {
	model, err := uc.GetModelProvider("")
	if err != nil {
		return func(yield func(*blades.Message, error) bool) {
			yield(nil, err)
		}
	}
	// 创建子代理工具
	rulegoWorkers, err := uc.CreateRuleChainWorker(model)
	if err != nil {
		return func(yield func(*blades.Message, error) bool) {
			yield(nil, err)
		}
	}
	planPrompts, err := getPlannerPrompt(entity.PlannerTpl{})
	if err != nil {
		return func(yield func(*blades.Message, error) bool) {
			yield(nil, err)
		}
	}
	planagent, err := blades.NewAgent(
		"plan_agent",
		blades.WithModel(model),
		blades.WithInstruction(planPrompts),
		blades.WithDescription("将业务流程文档解析并生成RuleGo的DSL"),
		blades.WithTools(rulegoWorkers...),
		blades.WithMiddleware(NewLogging),
	)
	if err != nil {
		return func(yield func(*blades.Message, error) bool) {
			yield(nil, err)
		}
	}

	input := blades.UserMessage(userMessage)
	planRunner := blades.NewRunner(planagent)
	// output, err := planRunner.Run(ctx, input)
	// if err != nil {
	// 	return func(yield func(*blades.Message, error) bool) {
	// 		yield(nil, err)
	// 	}
	// }
	// 运行任务规划
	stream := planRunner.RunStream(ctx, input)
	var message *blades.Message
	for message, err = range stream {
		if err != nil {
			uc.log.Errorf("任务规划失败2222:", err)
			return func(yield func(*blades.Message, error) bool) {
				yield(nil, err)
			}
		}
		return func(yield func(*blades.Message, error) bool) {
			yield(message, nil)
		}
	}
	return func(yield func(*blades.Message, error) bool) {
		yield(message, nil)
	}
	// // 最终组装输出规则链的agent
	// assemblyPrompts, err := getAssemblyPrompt(entity.AssemblyTpl{})
	// if err != nil {
	// 	return func(yield func(*blades.Message, error) bool) {
	// 		yield(nil, err)
	// 	}
	// }
	// assemblyAgent, err := blades.NewAgent(
	// 	"assembly_agent",
	// 	blades.WithInstruction(assemblyPrompts),
	// 	blades.WithDescription("将节点配置和连接关系组装成符合RuleGo规范的规则链JSON"),
	// 	blades.WithModel(model),
	// 	blades.WithMiddleware(NewLogging),
	// )
	// assemblyRunner := blades.NewRunner(assemblyAgent)
	// output, err := assemblyRunner.Run(ctx, blades.UserMessage(message.Text()))
	// if err != nil {
	// 	uc.log.Errorf("任务规划失败3333:", err)
	// 	return func(yield func(*blades.Message, error) bool) {
	// 		yield(nil, err)
	// 	}
	// }
	// return func(yield func(*blades.Message, error) bool) {
	// 	yield(output, nil)
	// }
}

// 创建RuleGo子代理
func (uc *AgentUsecase) CreateRuleChainWorker(model blades.ModelProvider) ([]tools.Tool, error) {
	uuidTool, err := uc.GenerateUUIDTool()
	if err != nil {
		return nil, err
	}
	nodeConfigTool, err := uc.GetNodeConfig()
	if err != nil {
		return nil, err
	}
	nodePrompts, err := getNodeToolPrompt(entity.NodeToolTpl{})
	if err != nil {
		return nil, err
	}
	nodeAgent, err := blades.NewAgent(
		"node_agent",
		blades.WithDescription("根据用户需求生成符合RuleGo规范的节点JSON"),
		blades.WithInstruction(nodePrompts),
		blades.WithModel(model),
		blades.WithMiddleware(NewLogging),
		blades.WithTools(uuidTool),
		blades.WithTools(nodeConfigTool),
	)
	if err != nil {
		uc.log.Errorf("子代理工具失败:", err)
		return nil, err
	}
	// connectPrompts, err := getConnectToolPrompt(entity.ConnectUseRuleTpl{})
	// if err != nil {
	// 	return nil, err
	// }
	// connectAgent, err := blades.NewAgent(
	// 	"connect_agent",
	// 	blades.WithDescription("根据用户需求生成符合RuleGo规范的连接JSON"),
	// 	blades.WithInstruction(connectPrompts),
	// 	blades.WithModel(model),
	// 	blades.WithMiddleware(NewLogging),
	// 	blades.WithTools(),
	// )
	// if err != nil {
	// 	uc.log.Errorf("子代理工具失败:", err)
	// 	return nil, err
	// }

	// 最终组装输出规则链的agent
	assemblyPrompts, err := getAssemblyPrompt(entity.AssemblyTpl{})
	if err != nil {
		return nil, err
	}
	assemblyAgent, err := blades.NewAgent(
		"assembly_agent",
		blades.WithInstruction(assemblyPrompts),
		blades.WithDescription("将节点配置和连接关系组装成符合RuleGo规范的规则链JSON"),
		blades.WithModel(model),
		blades.WithMiddleware(NewLogging),
	)
	return []tools.Tool{
		blades.NewAgentTool(nodeAgent),
		blades.NewAgentTool(assemblyAgent),
	}, nil
}

// 生成UUID的工具函数
func (uc *AgentUsecase) GenerateUUIDTool() (tools.Tool, error) {
	return tools.NewFunc(
		"generate_uuid",
		"生成UUID",
		func(ctx context.Context, input string) (string, error) {
			return uuid.NewString(), nil
		},
	)
}

// 查询节点配置信息
func (uc *AgentUsecase) GetNodeConfig() (tools.Tool, error) {
	return tools.NewFunc(
		"get_node_config",
		"获取节点配置信息",
		func(ctx context.Context, input string) (string, error) {
			result, err := uc.componentUseRuleRepo.FindOneComponentUseRule(ctx, map[string]interface{}{"component_name": input})
			if err != nil {
				return "", err
			}
			if result == nil {
				return "", nil
			}
			return result.UseRuleDesc, nil
		},
	)
}
