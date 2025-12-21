package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"ruleGoKratos/internal/biz/entity"
	"ruleGoKratos/internal/conf"
	"strings"

	"github.com/go-kratos/blades"
	"github.com/go-kratos/blades/contrib/openai"
	"github.com/go-kratos/blades/tools"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/google/uuid"
	"github.com/openai/openai-go/v3/option"
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
				RequestOptions: []option.RequestOption{
					option.WithHeader("Accept", "application/json"),
				},
			}), nil
		} else if uc.config.Ai != nil && uc.config.Ai.Openai != nil && uc.config.Ai.Openai.Model != "" {
			modelName = uc.config.Ai.Openai.Model
			return openai.NewModel(modelName, openai.Config{
				APIKey:  uc.config.Ai.Openai.ApiKey,
				BaseURL: uc.config.Ai.Openai.ApiBaseUrl,
				RequestOptions: []option.RequestOption{
					option.WithHeader("Accept", "application/json"),
				},
			}), nil
		}
		return nil, fmt.Errorf("未配置AI模型")
	}

	// 根据模型名称选择配置
	if uc.config.Ai != nil && uc.config.Ai.Doubao != nil && uc.config.Ai.Doubao.Model == modelName {
		return openai.NewModel(modelName, openai.Config{
			APIKey:  uc.config.Ai.Doubao.ApiKey,
			BaseURL: uc.config.Ai.Doubao.ApiBaseUrl,
			RequestOptions: []option.RequestOption{
				option.WithHeader("Accept", "application/json"),
			},
		}), nil
	} else if uc.config.Ai != nil && uc.config.Ai.Openai != nil {
		return openai.NewModel(modelName, openai.Config{
			APIKey:  uc.config.Ai.Openai.ApiKey,
			BaseURL: uc.config.Ai.Openai.ApiBaseUrl,
			RequestOptions: []option.RequestOption{
				option.WithHeader("Accept", "application/json"),
			},
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

func (uc *AgentUsecase) CreateRuleChainTestAgent(ctx context.Context, userMessage string) blades.Generator[*blades.Message, error] {
	model, err := uc.GetModelProvider("")
	if err != nil {
		return func(yield func(*blades.Message, error) bool) {
			yield(nil, err)
		}
	}
	uuidTool, err := uc.GenerateUUIDTool()
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
	planAgent, err := blades.NewAgent(
		"plan_agent",
		blades.WithInstruction(planPrompts),
		blades.WithDescription("任务规划agent"),
		blades.WithModel(model),
		blades.WithMiddleware(NewLogging),
		blades.WithTools(uuidTool),
		blades.WithOutputSchema(&jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"steps": {
					Type:        "array",
					Description: "任务规划步骤列表",
					Items: &jsonschema.Schema{
						Type: "object",
						Properties: map[string]*jsonschema.Schema{
							"instruction": {
								Type:        "string",
								Description: "步骤指令说明",
							},
						},
						Required: []string{"instruction"},
					},
				},
			},
			Required: []string{"steps"},
		}),
	)
	if err != nil {
		return func(yield func(*blades.Message, error) bool) {
			yield(nil, err)
		}
	}

	input := blades.UserMessage(userMessage)
	planRunner := blades.NewRunner(planAgent)
	return func(yield func(*blades.Message, error) bool) {
		stream := planRunner.RunStream(ctx, input)
		for msg, err := range stream {
			if !yield(msg, err) {
				break
			}
		}
	}
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
	executePrompts, err := getExecutePrompt(entity.ExecuteTpl{})
	if err != nil {
		return func(yield func(*blades.Message, error) bool) {
			yield(nil, err)
		}
	}
	planagent, err := blades.NewAgent(
		"execute_agent",
		blades.WithModel(model),
		blades.WithInstruction(executePrompts),
		blades.WithDescription("执行agent"),
		blades.WithTools(rulegoWorkers...),
		blades.WithMiddleware(NewLogging),
		// blades.WithOutputSchema(&jsonschema.Schema{}),
	)
	if err != nil {
		return func(yield func(*blades.Message, error) bool) {
			yield(nil, err)
		}
	}

	input := blades.UserMessage(userMessage)
	planRunner := blades.NewRunner(planagent)
	return func(yield func(*blades.Message, error) bool) {
		stream := planRunner.RunStream(ctx, input)
		for msg, err := range stream {
			if !yield(msg, err) {
				break
			}
		}
	}
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
		// blades.WithTools(uuidTool),
		// blades.WithTools(nodeConfigTool),
		// blades.WithInputSchema(&jsonschema.Schema{}),
		blades.WithOutputSchema(&jsonschema.Schema{
			Type: "array",
			Items: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id":             {Type: "string"},
					"type":           {Type: "string"},
					"name":           {Type: "string"},
					"additionalInfo": {Type: "object"},
					"configuration":  {Type: "object"},
				},
				Required: []string{"id", "type", "configuration"},
			},
		}),
	)
	if err != nil {
		uc.log.Errorf("node_agent子代理工具失败:", err)
		return nil, err
	}
	connectPrompts, err := getConnectToolPrompt(entity.ConnectUseRuleTpl{})
	if err != nil {
		return nil, err
	}
	connectAgent, err := blades.NewAgent(
		"connect_agent",
		blades.WithDescription("根据用户需求生成符合RuleGo规范的连接JSON"),
		blades.WithInstruction(connectPrompts),
		blades.WithModel(model),
		blades.WithMiddleware(NewLogging),
		// blades.WithInputSchema(&jsonschema.Schema{}),
		// blades.WithOutputSchema(&jsonschema.Schema{}),
	)
	if err != nil {
		uc.log.Errorf("connect_agent子代理工具失败:", err)
		return nil, err
	}

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
		// blades.WithInputSchema(&jsonschema.Schema{}),
		// blades.WithOutputSchema(&jsonschema.Schema{}),
	)
	if err != nil {
		uc.log.Errorf("assembly_agent子代理工具失败:", err)
		return nil, err
	}

	planPrompts, err := getPlannerPrompt(entity.PlannerTpl{})
	if err != nil {
		return nil, err
	}
	planAgent, err := blades.NewAgent(
		"plan_agent",
		blades.WithInstruction(planPrompts),
		blades.WithDescription("将业务流程文档解析并生成RuleGo的DSL"),
		blades.WithModel(model),
		blades.WithMiddleware(NewLogging),
		blades.WithTools(uuidTool),
	)
	if err != nil {
		return nil, err
	}
	return []tools.Tool{
		blades.NewAgentTool(nodeAgent),
		blades.NewAgentTool(connectAgent),
		blades.NewAgentTool(assemblyAgent),
		blades.NewAgentTool(planAgent),
		uuidTool,
		nodeConfigTool,
	}, nil
}

// 生成UUID的工具函数
func (uc *AgentUsecase) GenerateUUIDTool() (tools.Tool, error) {
	return tools.NewFunc(
		"generate_uuid",
		"生成UUID",
		func(ctx context.Context, input string) (string, error) {
			uuidStr := uuid.NewString()
			uc.log.Infof("生成UUID: %s", uuidStr)
			return uuidStr, nil
		},
	)
}

// 查询节点配置信息
// func (uc *AgentUsecase) GetNodeConfig() (tools.Tool, error) {
// 	return tools.NewFunc(
// 		"get_node_config",
// 		"获取节点配置信息",
// 		func(ctx context.Context, input string) (string, error) {
// 			uc.log.Infof("节点类型: %s", input)
// 			result, err := uc.componentUseRuleRepo.FindOneComponentUseRule(ctx, map[string]interface{}{"component_name": input})
// 			if err != nil {
// 				return "", err
// 			}
// 			if result == nil {
// 				return "", nil
// 			}
// 			return result.UseRuleDesc, nil
// 		},
// 	)
// }

func (uc *AgentUsecase) GetNodeConfigHandle(ctx context.Context, input string) (string, error) {
	// input解析为 {"node_type": "string"}
	var inputMap map[string]string
	err := json.Unmarshal([]byte(input), &inputMap)
	if err != nil {
		return "", err
	}
	nodeType := inputMap["node_type"]
	result, err := uc.componentUseRuleRepo.FindOneComponentUseRule(ctx, map[string]interface{}{"component_name": nodeType})
	if err != nil {
		return "", err
	}
	if result == nil {
		return "", nil
	}
	return result.UseRuleDesc, nil
}

func (uc *AgentUsecase) toolLogging() tools.Middleware {
	return func(next tools.Handler) tools.Handler {
		return tools.HandleFunc(func(ctx context.Context, req string) (string, error) {
			uc.log.Infof("Request received: %s", req)
			return next.Handle(ctx, req)
		})
	}
}
func (uc *AgentUsecase) GetNodeConfig() (tools.Tool, error) {
	aa := tools.NewTool(
		"get_node_config",
		"获取节点配置信息",
		tools.HandleFunc(uc.GetNodeConfigHandle),
		tools.WithInputSchema(&jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"node_type": {
					Type:        "string",
					Description: "节点类型",
				},
			},
		}),
		tools.WithMiddleware(uc.toolLogging()),
	)
	return aa, nil
}

func (uc *AgentUsecase) RuleChainTestAgent(ctx context.Context, userMessage string) (*blades.Message, error) {
	model, err := uc.GetModelProvider("")
	if err != nil {
		return nil, err
	}
	uuidTool, err := uc.GenerateUUIDTool()
	if err != nil {
		return nil, err
	}
	nodeConfigTool, err := uc.GetNodeConfig()
	if err != nil {
		return nil, err
	}
	planPrompts, err := getPlannerPrompt(entity.PlannerTpl{})
	if err != nil {
		return nil, err
	}
	planAgent, err := blades.NewAgent(
		"plan_agent",
		blades.WithInstruction(planPrompts),
		blades.WithDescription("任务规划agent"),
		blades.WithModel(model),
		blades.WithMiddleware(NewLogging),
		blades.WithTools(uuidTool),
	)
	if err != nil {
		return nil, err
	}

	nodePrompts, err := getNodeToolPrompt(entity.NodeToolTpl{})
	if err != nil {
		return nil, err
	}
	nodeAgent, err := blades.NewAgent(
		"node_agent",
		blades.WithDescription("节点agent"),
		blades.WithModel(model),
		blades.WithMiddleware(NewLogging),
		blades.WithInstruction(nodePrompts),
		blades.WithTools(nodeConfigTool),
	)
	if err != nil {
		return nil, err
	}
	connectPrompts, err := getConnectToolPrompt(entity.ConnectUseRuleTpl{})
	if err != nil {
		return nil, err
	}
	connectAgent, err := blades.NewAgent(
		"connect_agent",
		blades.WithDescription("连接agent"),
		blades.WithModel(model),
		blades.WithInstruction(connectPrompts),
		blades.WithMiddleware(NewLogging),
	)
	if err != nil {
		return nil, err
	}
	input := blades.UserMessage(userMessage)
	planRunner := blades.NewRunner(planAgent)
	stream := planRunner.RunStream(ctx, input)
	var lastMsg *blades.Message
	for msg, err := range stream {
		if err != nil {
			return nil, err
		}
		if msg != nil {
			lastMsg = msg
			// 判断是否是最后一个消息
			if msg.Status == blades.StatusCompleted {
				break
			}
		}
	}
	planMsgStr := lastMsg.Text()
	fmt.Println("planMsgStr:", planMsgStr)
	planResult, err := FinalJSONParser(planMsgStr)
	if err != nil {
		fmt.Println("SafeUnmarshalPlan error:", err)
	} else {
		fmt.Println("planResult:", planResult)
	}
	var lastNodeMsg, lastConnectMsg *blades.Message
	for i, step := range planResult.Steps {
		if i == len(planResult.Steps)-3 {
			nodeInput := blades.UserMessage(step.Instruction)
			nodeRunner := blades.NewRunner(nodeAgent)
			nodeStream := nodeRunner.RunStream(ctx, nodeInput)
			for nodeMsg, err := range nodeStream {
				if err != nil {
					break
				}
				if nodeMsg != nil {
					// 判断是否是最后一个消息
					if nodeMsg.Status == blades.StatusCompleted {
						lastNodeMsg = nodeMsg
						break
					}
				}
			}
		}
		if i == len(planResult.Steps)-2 {
			connectInput := blades.UserMessage(step.Instruction)
			connectRunner := blades.NewRunner(connectAgent)
			connectStream := connectRunner.RunStream(ctx, connectInput)
			for connectMsg, err := range connectStream {
				if err != nil {
					break
				}
				if connectMsg != nil {
					// 判断是否是最后一个消息
					if connectMsg.Status == blades.StatusCompleted {
						lastConnectMsg = connectMsg
						break
					}
				}
			}
		}
	}
	nodeMsgStr := lastNodeMsg.Text()
	fmt.Println("nodeMsgStr:", nodeMsgStr)
	connectMsgStr := lastConnectMsg.Text()
	fmt.Println("connectMsgStr:", connectMsgStr)
	return lastMsg, nil
}

// FinalJSONParser 核心解析函数
func FinalJSONParser(raw string) (*PlanResult, error) {
	// 1. 物理隔离：只截取最外层大括号内部的内容，彻底无视 Markdown 反引号
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")

	if start == -1 || end == -1 || end < start {
		return nil, fmt.Errorf("未能在字符串中找到有效的 JSON 结构")
	}

	content := raw[start : end+1]

	// 2. 预处理：处理 instruction 内部的非法嵌套引号
	content = fixInternalQuotesSimple(content)

	// 3. 处理换行符：将真实的换行符替换为空格，防止 Unmarshal 崩溃
	content = strings.ReplaceAll(content, "\n", " ")
	content = strings.ReplaceAll(content, "\r", " ")

	// 4. 解析为结构体
	var result PlanResult
	err := json.Unmarshal([]byte(content), &result)
	if err != nil {
		return nil, fmt.Errorf("解析失败: %v\n截取内容预览: %s", err, truncate(content, 100))
	}

	return &result, nil
}

// fixInternalQuotesSimple 修复 "instruction": "..." 内部未转义的引号
func fixInternalQuotesSimple(input string) string {
	sb := strings.Builder{}
	// 按关键字切割，这样我们可以精确处理每个 instruction 的值
	parts := strings.Split(input, `"instruction"`)

	sb.WriteString(parts[0])
	for i := 1; i < len(parts); i++ {
		sb.WriteString(`"instruction"`)

		// 找到该 instruction 值的起始双引号
		firstQuote := strings.Index(parts[i], `"`)
		if firstQuote == -1 {
			sb.WriteString(parts[i])
			continue
		}

		// 从后往前找该 instruction 值的结束双引号
		// 逻辑：结束引号后通常紧跟 ',' 或 '}'
		lastQuote := -1
		for j := len(parts[i]) - 1; j > firstQuote; j-- {
			if parts[i][j] == '"' {
				remain := strings.TrimSpace(parts[i][j+1:])
				if strings.HasPrefix(remain, ",") || strings.HasPrefix(remain, "}") || remain == "" {
					lastQuote = j
					break
				}
			}
		}

		if lastQuote != -1 {
			// prefix 包含冒号和起始引号，如 `: "`
			prefix := parts[i][:firstQuote+1]
			// body 是 instruction 的文字内容
			body := parts[i][firstQuote+1 : lastQuote]
			// suffix 包含结束引号和后续结构，如 `" ,`
			suffix := parts[i][lastQuote:]

			// 【核心修复】：将 body 内部所有双引号转义
			// 先把已有的 \" 还原成 "，再统一转义成 \"，防止重复转义
			body = strings.ReplaceAll(body, `\"`, `"`)
			body = strings.ReplaceAll(body, `"`, `\"`)

			sb.WriteString(prefix)
			sb.WriteString(body)
			sb.WriteString(suffix)
		} else {
			sb.WriteString(parts[i])
		}
	}
	return sb.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

type PlanResult struct {
	Steps []Step `json:"steps"`
}

type Step struct {
	Instruction string `json:"instruction"`
}

func (uc *AgentUsecase) RuleChainTestNodeAgent(ctx context.Context, userMessage string) (*blades.Message, error) {
	model, err := uc.GetModelProvider("")
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
		blades.WithDescription("节点agent"),
		blades.WithModel(model),
		blades.WithMiddleware(NewLogging),
		blades.WithInstruction(nodePrompts),
		blades.WithTools(nodeConfigTool),
		blades.WithOutputSchema(&jsonschema.Schema{
			Type: "array",
			Items: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id":             {Type: "string"},
					"type":           {Type: "string"},
					"name":           {Type: "string"},
					"additionalInfo": {Type: "object"},
					"configuration":  {Type: "object"},
				},
				Required: []string{"id", "type", "configuration"},
			},
		}),
	)
	if err != nil {
		return nil, err
	}

	input := blades.UserMessage(userMessage)
	nodeRunner := blades.NewRunner(nodeAgent)
	stream := nodeRunner.RunStream(ctx, input)
	var lastMsg *blades.Message
	for msg, err := range stream {
		if err != nil {
			return nil, err
		}
		if msg != nil {
			lastMsg = msg
			// 判断是否是最后一个消息
			if msg.Status == blades.StatusCompleted {
				break
			}
		}
	}
	nodeMsgStr := lastMsg.Text()
	fmt.Println("planMsgStr:", nodeMsgStr)
	return lastMsg, nil
}
