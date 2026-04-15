package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"ruleGoKratos/internal/conf"
	"strings"
	"time"

	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

const (
	defaultMaxIterations   = 8
	defaultMaxToolCalls    = 16
	defaultToolTimeoutSecs = 5
	defaultChunkSize       = 120
	defaultSystemPrompt    = "You are Claude Code, an AI coding assistant focused on accurate, actionable engineering help. Think step by step, follow user intent, and prefer concrete execution over vague advice. Use available tools when they improve correctness. If information is uncertain, state assumptions briefly instead of fabricating details. Keep responses concise, practical, and implementation-oriented."
)

type StreamMessage struct {
	Content string
	Done    bool
}

type StreamGenerator func(yield func(*StreamMessage, error) bool)

type HistoryMessage struct {
	Role    string
	Content string
}

type HarnessTool struct {
	Info   *schema.ToolInfo
	Invoke func(ctx context.Context, rawArgs string) (string, error)
}

type HarnessConfig struct {
	MaxIterations   int
	MaxToolCalls    int
	ToolTimeoutSecs int
	ChunkSize       int
}

type HarnessRequest struct {
	Model           string
	History         []HistoryMessage
	Input           string
	SystemPrompt    string
	ToolOptions     *HarnessToolOptions
	ConfigOverride  *HarnessConfig
}

type SkillExecutor interface {
	Execute(ctx context.Context, skillName string, payload string) (string, error)
}

type McpExecutor interface {
	Call(ctx context.Context, server string, tool string, arguments string) (string, error)
}

type NoopSkillExecutor struct{}

func (n *NoopSkillExecutor) Execute(ctx context.Context, skillName string, payload string) (string, error) {
	return "", fmt.Errorf("skill executor 未配置: %s", skillName)
}

type NoopMcpExecutor struct{}

func (n *NoopMcpExecutor) Call(ctx context.Context, server string, tool string, arguments string) (string, error) {
	return "", fmt.Errorf("mcp executor 未配置: server=%s tool=%s", server, tool)
}

type AgentUsecase struct {
	log           *log.Helper
	config        *conf.Bootstrap
	harnessLogger *HarnessLogger
	harnessConfig HarnessConfig
	skillExecutor SkillExecutor
	mcpExecutor   McpExecutor
	chatModelFunc func(ctx context.Context, modelName string) (model.ToolCallingChatModel, error)
}

func NewAgentUsecase(logger log.Logger, config *conf.Bootstrap) *AgentUsecase {
	helper := log.NewHelper(logger)
	skillDirs := defaultSkillDirs("", "")
	skillOptions := FileSkillExecutorOptions{}
	if config != nil && config.Agent != nil && config.Agent.Skill != nil {
		skillDirs = defaultSkillDirs(config.Agent.Skill.Dir, config.Agent.Skill.Dirs)
		skillOptions = FileSkillExecutorOptions{
			Namespace:         config.Agent.Skill.Namespace,
			AllowList:         config.Agent.Skill.Allowlist,
			HotReload:         config.Agent.Skill.HotReload,
			HotReloadSet:      true,
			ScanIntervalMS:    int(config.Agent.Skill.ScanIntervalMs),
			ScanIntervalMSSet: true,
		}
	}
	fileExecutor, err := NewFileSkillExecutor(skillDirs, skillOptions)
	if err != nil {
		helper.Errorf("初始化FileSkillExecutor失败: %v", err)
		fileExecutor = &FileSkillExecutor{}
	}
	return &AgentUsecase{
		log:           helper,
		config:        config,
		harnessLogger: NewHarnessLogger(helper),
		harnessConfig: HarnessConfig{
			MaxIterations:   defaultMaxIterations,
			MaxToolCalls:    defaultMaxToolCalls,
			ToolTimeoutSecs: defaultToolTimeoutSecs,
			ChunkSize:       defaultChunkSize,
		},
		skillExecutor: fileExecutor,
		mcpExecutor:   &NoopMcpExecutor{},
	}
}

func (uc *AgentUsecase) SetSkillExecutor(executor SkillExecutor) {
	if executor != nil {
		uc.skillExecutor = executor
	}
}

func (uc *AgentUsecase) SetMcpExecutor(executor McpExecutor) {
	if executor != nil {
		uc.mcpExecutor = executor
	}
}

func (uc *AgentUsecase) SetHarnessConfig(cfg HarnessConfig) {
	uc.harnessConfig = cfg
}

type resolvedModel struct {
	name    string
	apiKey  string
	baseURL string
}

func (uc *AgentUsecase) resolveModel(modelName string) (*resolvedModel, error) {
	if uc.config == nil || uc.config.Ai == nil {
		return nil, errors.New("未配置AI模型")
	}

	if modelName == "" {
		if uc.config.Ai.Doubao != nil && uc.config.Ai.Doubao.Model != "" {
			return &resolvedModel{
				name:    uc.config.Ai.Doubao.Model,
				apiKey:  uc.config.Ai.Doubao.ApiKey,
				baseURL: uc.config.Ai.Doubao.ApiBaseUrl,
			}, nil
		}
		if uc.config.Ai.Openai != nil && uc.config.Ai.Openai.Model != "" {
			return &resolvedModel{
				name:    uc.config.Ai.Openai.Model,
				apiKey:  uc.config.Ai.Openai.ApiKey,
				baseURL: uc.config.Ai.Openai.ApiBaseUrl,
			}, nil
		}
		return nil, errors.New("未配置AI模型")
	}

	if uc.config.Ai.Doubao != nil && uc.config.Ai.Doubao.Model == modelName {
		return &resolvedModel{
			name:    modelName,
			apiKey:  uc.config.Ai.Doubao.ApiKey,
			baseURL: uc.config.Ai.Doubao.ApiBaseUrl,
		}, nil
	}
	if uc.config.Ai.Openai != nil {
		return &resolvedModel{
			name:    modelName,
			apiKey:  uc.config.Ai.Openai.ApiKey,
			baseURL: uc.config.Ai.Openai.ApiBaseUrl,
		}, nil
	}
	return nil, fmt.Errorf("未找到模型配置: %s", modelName)
}

func (uc *AgentUsecase) newChatModel(ctx context.Context, modelName string) (model.ToolCallingChatModel, error) {
	if uc.chatModelFunc != nil {
		return uc.chatModelFunc(ctx, modelName)
	}
	resolved, err := uc.resolveModel(modelName)
	if err != nil {
		return nil, err
	}
	return einoopenai.NewChatModel(ctx, &einoopenai.ChatModelConfig{
		APIKey:  resolved.apiKey,
		BaseURL: resolved.baseURL,
		Model:   resolved.name,
		Timeout: 60 * time.Second,
	})
}

func (uc *AgentUsecase) buildMessages(history []HistoryMessage, userMessage string) []*schema.Message {
	return uc.composeMessages(uc.getSystemPrompt(), history, userMessage)
}

func (uc *AgentUsecase) composeMessages(systemPrompt string, history []HistoryMessage, userMessage string) []*schema.Message {
	if strings.TrimSpace(systemPrompt) == "" {
		systemPrompt = uc.getSystemPrompt()
	}
	msgs := make([]*schema.Message, 0, len(history)+2)
	msgs = append(msgs, schema.SystemMessage(systemPrompt))
	for _, item := range history {
		switch item.Role {
		case "assistant":
			msgs = append(msgs, schema.AssistantMessage(item.Content, nil))
		default:
			msgs = append(msgs, schema.UserMessage(item.Content))
		}
	}
	msgs = append(msgs, schema.UserMessage(userMessage))
	return msgs
}

func (uc *AgentUsecase) getSystemPrompt() string {
	if uc.config != nil && uc.config.Agent != nil && strings.TrimSpace(uc.config.Agent.SystemPrompt) != "" {
		return uc.config.Agent.SystemPrompt
	}
	return defaultSystemPrompt
}

func sanitizeExternalError(err error) error {
	if err == nil {
		return nil
	}
	return errors.New("Agent执行失败，请稍后重试")
}

func (uc *AgentUsecase) sanitizeConfig() HarnessConfig {
	// 兜底默认值：即使外部只注入了部分配置，也能保证运行时稳定。
	cfg := uc.harnessConfig
	if cfg.MaxIterations <= 0 {
		cfg.MaxIterations = defaultMaxIterations
	}
	if cfg.MaxToolCalls <= 0 {
		cfg.MaxToolCalls = defaultMaxToolCalls
	}
	if cfg.ToolTimeoutSecs <= 0 {
		cfg.ToolTimeoutSecs = defaultToolTimeoutSecs
	}
	if cfg.ChunkSize <= 0 {
		cfg.ChunkSize = defaultChunkSize
	}
	return cfg
}

func (uc *AgentUsecase) ExecuteStream(ctx context.Context, modelName string, history []HistoryMessage, userMessage string) StreamGenerator {
	// 对上层暴露的统一流式入口。
	req := HarnessRequest{
		Model:   modelName,
		History: history,
		Input:   userMessage,
	}
	return uc.executeHarness(req, ctx)
}

func (uc *AgentUsecase) executeHarness(req HarnessRequest, ctx context.Context) StreamGenerator {
	return func(yield func(*StreamMessage, error) bool) {
		requestID := uuid.NewString()
		start := time.Now()
		uc.harnessLogger.LogRunStart(requestID, req.Model, len(req.History), req.Input)
		defer func() {
			uc.harnessLogger.LogRunFinish(requestID, time.Since(start))
		}()
		cfg := uc.effectiveHarnessConfig(req.ConfigOverride)

		einoModel, err := uc.newChatModel(ctx, req.Model)
		if err != nil {
			uc.harnessLogger.LogError(requestID, "init_model", err)
			yield(nil, sanitizeExternalError(err))
			return
		}

		toolRegistry, toolInfos, err := uc.resolveToolRegistry(req.ToolOptions)
		if err != nil {
			uc.harnessLogger.LogError(requestID, "build_tools", err)
			yield(nil, sanitizeExternalError(err))
			return
		}
		modelRunner := einoModel
		if len(toolInfos) > 0 {
			modelWithTools, wErr := einoModel.WithTools(toolInfos)
			if wErr != nil {
				uc.harnessLogger.LogError(requestID, "bind_tools", wErr)
				yield(nil, sanitizeExternalError(wErr))
				return
			}
			modelRunner = modelWithTools
		}
		allowedTools := make(map[string]bool, len(toolRegistry))
		for name := range toolRegistry {
			allowedTools[name] = true
		}

		systemPrompt := strings.TrimSpace(req.SystemPrompt)
		if systemPrompt == "" {
			systemPrompt = uc.getSystemPrompt()
		}
		msgs := uc.composeMessages(systemPrompt, req.History, req.Input)
		toolCallCount := 0
		for i := 0; i < cfg.MaxIterations; i++ {
			modelStart := time.Now()
			// 是否调用工具由模型通过流式 chunk 中的 tool_calls 决定。
			stream, genErr := modelRunner.Stream(ctx, msgs)
			uc.harnessLogger.LogModelRound(requestID, i+1, time.Since(modelStart), genErr)
			if genErr != nil {
				uc.harnessLogger.LogError(requestID, "model_stream", genErr)
				yield(nil, sanitizeExternalError(genErr))
				return
			}
			// 收集流式 chunk，用于拼出完整 assistant 消息并进入下一轮状态。
			chunks := make([]*schema.Message, 0, 32)
			seenToolCall := false
			for {
				chunk, recvErr := stream.Recv()
				if recvErr == io.EOF {
					break
				}
				if recvErr != nil {
					stream.Close()
					uc.harnessLogger.LogError(requestID, "model_stream_recv", recvErr)
					yield(nil, sanitizeExternalError(recvErr))
					return
				}
				if chunk == nil {
					continue
				}
				chunks = append(chunks, chunk)
				if len(chunk.ToolCalls) > 0 {
					seenToolCall = true
				}
				// 在出现工具调用前，只向调用方透传纯文本 chunk。
				if !seenToolCall && chunk.Content != "" {
					if !yield(&StreamMessage{Content: chunk.Content, Done: false}, nil) {
						stream.Close()
						return
					}
				}
			}
			stream.Close()

			// Eino 返回的是分片，需要先合并，才能稳定处理工具调用。
			resp, concatErr := schema.ConcatMessages(chunks)
			if concatErr != nil {
				uc.harnessLogger.LogError(requestID, "model_stream_concat", concatErr)
				yield(nil, sanitizeExternalError(concatErr))
				return
			}
			if resp == nil {
				uc.harnessLogger.LogError(requestID, "model_stream_concat", errors.New("模型返回为空"))
				yield(nil, sanitizeExternalError(errors.New("模型返回为空")))
				return
			}
			msgs = append(msgs, resp)

			if len(resp.ToolCalls) == 0 {
				if !yield(&StreamMessage{Done: true}, nil) {
					return
				}
				return
			}

			// Harness 只负责策略校验，具体调用哪个工具由模型决策。
			for _, call := range resp.ToolCalls {
				toolCallCount++
				if toolCallCount > cfg.MaxToolCalls {
					err = errors.New("工具调用次数超过沙箱限制")
					uc.harnessLogger.LogSandboxDecision(requestID, call.Function.Name, false, "tool_call_limit_exceeded")
					uc.harnessLogger.LogError(requestID, "tool_limit", err)
					yield(nil, sanitizeExternalError(err))
					return
				}

				if !allowedTools[call.Function.Name] {
					err = fmt.Errorf("未授权工具: %s", call.Function.Name)
					uc.harnessLogger.LogSandboxDecision(requestID, call.Function.Name, false, "tool_not_in_whitelist")
					uc.harnessLogger.LogError(requestID, "tool_denied", err)
					yield(nil, sanitizeExternalError(err))
					return
				}

				toolImpl, ok := toolRegistry[call.Function.Name]
				if !ok {
					err = fmt.Errorf("未授权工具: %s", call.Function.Name)
					uc.harnessLogger.LogSandboxDecision(requestID, call.Function.Name, false, "tool_not_registered")
					uc.harnessLogger.LogError(requestID, "tool_denied", err)
					yield(nil, sanitizeExternalError(err))
					return
				}
				uc.harnessLogger.LogSandboxDecision(requestID, call.Function.Name, true, "tool_allowed")

				// 工具调用设置独立超时，防止单个工具阻塞整轮执行。
				toolCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.ToolTimeoutSecs)*time.Second)
				toolStart := time.Now()
				toolOutput, toolErr := toolImpl.Invoke(toolCtx, call.Function.Arguments)
				cancel()
				uc.harnessLogger.LogToolCall(requestID, call.Function.Name, time.Since(toolStart), toolErr)
				if toolErr != nil {
					uc.harnessLogger.LogError(requestID, "tool_invoke", toolErr)
					yield(nil, sanitizeExternalError(toolErr))
					return
				}

				toolCallID := call.ID
				if toolCallID == "" {
					toolCallID = uuid.NewString()
				}
				// 将工具输出回灌为 ToolMessage，供模型继续推理。
				msgs = append(msgs, schema.ToolMessage(toolOutput, toolCallID, schema.WithToolName(call.Function.Name)))
			}
		}

		err = errors.New("超过最大迭代次数，Agent提前终止")
		uc.harnessLogger.LogError(requestID, "max_iterations", err)
		yield(nil, sanitizeExternalError(err))
	}
}

func (uc *AgentUsecase) Run(ctx context.Context, userMessage string) StreamGenerator {
	return uc.ExecuteStream(ctx, "", nil, userMessage)
}

func (uc *AgentUsecase) BuildToolRegistry() (map[string]*HarnessTool, []*schema.ToolInfo, error) {
	// Registry 是当前 Harness 运行期可调用工具的唯一来源。
	uuidTool, err := uc.BuildUUIDTool()
	if err != nil {
		return nil, nil, err
	}
	skillTool, err := uc.BuildSkillTool()
	if err != nil {
		return nil, nil, err
	}
	mcpTool, err := uc.BuildMCPTool()
	if err != nil {
		return nil, nil, err
	}
	readFileTool, err := uc.BuildReadWorkspaceFileTool()
	if err != nil {
		return nil, nil, err
	}
	writeFileTool, err := uc.BuildWriteWorkspaceFileTool()
	if err != nil {
		return nil, nil, err
	}
	shellTool, err := uc.BuildRunWorkspaceShellTool()
	if err != nil {
		return nil, nil, err
	}
	registry := map[string]*HarnessTool{
		uuidTool.Info.Name:         uuidTool,
		skillTool.Info.Name:        skillTool,
		mcpTool.Info.Name:          mcpTool,
		readFileTool.Info.Name:     readFileTool,
		writeFileTool.Info.Name:    writeFileTool,
		shellTool.Info.Name:        shellTool,
	}
	infos := []*schema.ToolInfo{
		uuidTool.Info, skillTool.Info, mcpTool.Info,
		readFileTool.Info, writeFileTool.Info, shellTool.Info,
	}
	return registry, infos, nil
}

func (uc *AgentUsecase) BuildUUIDTool() (*HarnessTool, error) {
	toolInfo := &schema.ToolInfo{
		Name: "generate_uuid",
		Desc: "生成UUID",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"reason": {
				Type:     schema.String,
				Desc:     "用途说明，可选",
				Required: false,
			},
		}),
	}
	return &HarnessTool{
		Info: toolInfo,
		Invoke: func(ctx context.Context, rawArgs string) (string, error) {
			return uuid.NewString(), nil
		},
	}, nil
}

func (uc *AgentUsecase) BuildSkillTool() (*HarnessTool, error) {
	toolInfo := &schema.ToolInfo{
		Name: "run_skill",
		Desc: "执行技能，输入技能名与负载内容",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"skill_name": {
				Type:     schema.String,
				Desc:     "技能名称",
				Required: true,
			},
			"payload": {
				Type:     schema.String,
				Desc:     "技能输入",
				Required: false,
			},
		}),
	}
	type runSkillArgs struct {
		SkillName string `json:"skill_name"`
		Payload   string `json:"payload"`
	}
	return &HarnessTool{
		Info: toolInfo,
		Invoke: func(ctx context.Context, rawArgs string) (string, error) {
			// 保持严格入参契约，确保下游 skill 执行器拿到确定性 payload。
			var args runSkillArgs
			if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
				return "", err
			}
			if strings.TrimSpace(args.SkillName) == "" {
				return "", errors.New("skill_name 不能为空")
			}
			return uc.skillExecutor.Execute(ctx, args.SkillName, args.Payload)
		},
	}, nil
}

func (uc *AgentUsecase) BuildMCPTool() (*HarnessTool, error) {
	toolInfo := &schema.ToolInfo{
		Name: "call_mcp_tool",
		Desc: "调用MCP工具，输入服务名、工具名和参数",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"server": {
				Type:     schema.String,
				Desc:     "MCP服务名",
				Required: true,
			},
			"tool": {
				Type:     schema.String,
				Desc:     "MCP工具名",
				Required: true,
			},
			"arguments": {
				Type:     schema.String,
				Desc:     "JSON字符串参数",
				Required: false,
			},
		}),
	}
	type callMcpArgs struct {
		Server    string `json:"server"`
		Tool      string `json:"tool"`
		Arguments string `json:"arguments"`
	}
	return &HarnessTool{
		Info: toolInfo,
		Invoke: func(ctx context.Context, rawArgs string) (string, error) {
			// MCP 路由由 mcpExecutor 实现；Harness 只做最小参数契约校验。
			var args callMcpArgs
			if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
				return "", err
			}
			if strings.TrimSpace(args.Server) == "" || strings.TrimSpace(args.Tool) == "" {
				return "", errors.New("server 和 tool 不能为空")
			}
			return uc.mcpExecutor.Call(ctx, args.Server, args.Tool, args.Arguments)
		},
	}, nil
}
