package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"ruleGoKratos/internal/conf"
	"strconv"
	"strings"
	"time"

	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

const (
	defaultMaxIterations   = 32
	defaultMaxToolCalls    = 64
	defaultToolTimeoutSecs = 5
	// LLM HTTP 流式读取超时（OpenAI SDK Client.Timeout）；规划/长推理需显著大于旧版 60s。
	defaultStreamTimeoutSecs = 600
	defaultChunkSize         = 120
	// 与 Harness / 管理端聊天共用；工具是否可用以运行时为准。
	defaultSystemPrompt = `You are the Code Assistant for this RuleGo / Flowgram deployment. Deliver accurate, actionable engineering help: runnable code when appropriate, concrete shell or API steps, real file paths, and ordered reasoning—avoid vague platitudes.

Tools: When the runtime exposes tools (SKILL invocation, MCP, workspace file read/write/shell), call them only through real execution; never invent tool outputs, logs, or claim success when a tool did not run. If a tool errors or is unavailable, report it briefly.

Language: Match the user's language (reply in Chinese when they write Chinese).

Facts: Do not invent repository layout, configs, or command results; when unsure, say what you infer and what you need from the user.

Vision & multimodal: User turns may include images attached via the chat multimodal API; interpret them directly with your vision abilities. Do not claim you lack image understanding or use shell/download/workspace tools solely to «view» images already supplied in this conversation—unless the user asks to persist files to workspace or analyze them offline.

Image URLs: If the user pastes HTTPS links to images in the message, the server may fetch them into the same multimodal payload you receive—treat those as viewable images, not as links you must open yourself. Still do not claim you can browse arbitrary sites beyond what is supplied in the conversation payload.

Style: Stay concise; use Markdown with fenced code blocks for code and log excerpts.`
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
	// StreamTimeoutSecs LLM 流式请求整体读超时（秒）；单轮 Stream.Recv 受底层 HTTP Client 超时约束。
	StreamTimeoutSecs int
	ChunkSize         int
}

type HarnessAttachment struct {
	Filename      string
	MimeType      string
	Text          string
	ContentBase64 string
}

type HarnessRequest struct {
	Model          string
	History        []HistoryMessage
	Input          string
	Attachments    []HarnessAttachment // 可选；图片/音视频走 Eino UserInputMultiContent 多模态
	SystemPrompt   string
	ToolOptions    *HarnessToolOptions
	ConfigOverride *HarnessConfig
	// LlmConfigID / LlmModelEntryID 非零时从模型管理加载凭证与模型名，不再读取环境变量或 YAML ai.* 配置。
	LlmConfigID     int64
	LlmModelEntryID int64
	// ManagedAgentID 非零时由 enrichHarnessWithManagedAgent 注入 Agent 配置（覆盖 ToolOptions / 模型对与系统提示中的 SKILL 目录）。
	ManagedAgentID int64
	// SkillCatalogFilter 为 nil 时 SKILL 目录列出全部可用技能；非 nil 且 len=0 不附目录；非 nil 且 len>0 仅列出这些 skill id。
	SkillCatalogFilter *[]string

	// WorkspaceSessionDir 相对于配置的 workspace 根的子路径；非空时本轮 Harness 内 read/write/shell 工具仅在该目录下操作（运行前会 MkdirAll）。
	WorkspaceSessionDir string

	// Playground 协作：将 Harness 内工具调用写入 Trace（可选）。
	PlaygroundRunID   string
	PlaygroundAgentID string
	TraceSink         HarnessTraceSink
}

// HarnessTraceSink Playground 注入，映射到 TraceEngine 的工具事件。
type HarnessTraceSink interface {
	EmitToolCall(runID, agentID, toolName, args string)
	EmitToolResult(runID, agentID, toolName, result string, success bool)
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
	log                *log.Helper
	config             *conf.Bootstrap
	harnessLogger      *HarnessLogger
	harnessConfig      HarnessConfig
	skillExecutor      SkillExecutor
	mcpExecutor        McpExecutor
	managedLLM         ManagedLLMResolver
	managedAgentLoader ManagedAgentLoader
	chatModelFunc      func(ctx context.Context, req HarnessRequest) (model.ToolCallingChatModel, error)
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

func (uc *AgentUsecase) SetManagedLLMResolver(r ManagedLLMResolver) {
	uc.managedLLM = r
}

func (uc *AgentUsecase) ResolveManagedLLM(ctx context.Context, configID int64, entryID int64) (modelName string, apiKey string, baseURL string, err error) {
	if uc.managedLLM == nil {
		return "", "", "", errors.New("LLM 管理服务未就绪")
	}
	return uc.managedLLM.ResolveManagedLLM(ctx, configID, entryID)
}

func defaultOpenAIBaseURL(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return "https://api.openai.com/v1"
	}
	return base
}

func (uc *AgentUsecase) newChatModel(ctx context.Context, req HarnessRequest) (model.ToolCallingChatModel, error) {
	if uc.chatModelFunc != nil {
		return uc.chatModelFunc(ctx, req)
	}
	if req.LlmConfigID > 0 && req.LlmModelEntryID > 0 {
		name, key, base, err := uc.ResolveManagedLLM(ctx, req.LlmConfigID, req.LlmModelEntryID)
		if err != nil {
			return nil, err
		}
		cfgEff := uc.effectiveHarnessConfig(req.ConfigOverride)
		streamTimeout := time.Duration(cfgEff.StreamTimeoutSecs) * time.Second
		if streamTimeout <= 0 {
			streamTimeout = defaultStreamTimeoutSecs * time.Second
		}
		return einoopenai.NewChatModel(ctx, &einoopenai.ChatModelConfig{
			APIKey:  key,
			BaseURL: defaultOpenAIBaseURL(base),
			Model:   name,
			Timeout: streamTimeout,
		})
	}
	return nil, errors.New("请在流程节点中选择模型管理中的 LLM 配置与模型（已不再使用环境变量 AI 密钥）")
}

func (uc *AgentUsecase) buildMessages(history []HistoryMessage, userMessage string) []*schema.Message {
	req := &HarnessRequest{History: history}
	return uc.composeMessages(req, uc.getSystemPrompt(), history, userMessage, nil)
}

func (uc *AgentUsecase) composeMessages(req *HarnessRequest, systemPrompt string, history []HistoryMessage, userText string, attachments []HarnessAttachment) []*schema.Message {
	if strings.TrimSpace(systemPrompt) == "" {
		systemPrompt = uc.getSystemPrompt()
	}
	systemPrompt = uc.appendSkillCatalogToSystemWithFilter(systemPrompt, req.SkillCatalogFilter)
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
	parts := buildHarnessInputParts(userText, attachments)
	msgs = append(msgs, lastUserMessageFromParts(parts))
	return msgs
}

func (uc *AgentUsecase) getSystemPrompt() string {
	if uc.config != nil && uc.config.Agent != nil && strings.TrimSpace(uc.config.Agent.SystemPrompt) != "" {
		return uc.config.Agent.SystemPrompt
	}
	return defaultSystemPrompt
}

// 将技能 id 目录附在系统提示后，使模型知悉可调用 run_skill 的精确名称（与 FileSkillExecutor 扫描结果一致）。
const skillCatalogMaxBytes = 32000

func (uc *AgentUsecase) appendSkillCatalogToSystemWithFilter(systemPrompt string, filter *[]string) string {
	if filter != nil && len(*filter) == 0 {
		return systemPrompt
	}
	var names []string
	if filter == nil {
		fe, ok := uc.skillExecutor.(*FileSkillExecutor)
		if !ok {
			return systemPrompt
		}
		names = fe.ListAvailableSkillNames()
	} else {
		names = *filter
	}
	if len(names) == 0 {
		return systemPrompt
	}
	var b strings.Builder
	b.Grow(len(systemPrompt) + 256 + min(len(names)*32, skillCatalogMaxBytes))
	b.WriteString(systemPrompt)
	b.WriteString("\n\n---\n## SKILL 目录（工具 run_skill 的 skill_name 须与下列 id 完全一致，共 ")
	b.WriteString(strconv.Itoa(len(names)))
	b.WriteString(" 项；按逗号分隔）\n")
	joined := strings.Join(names, ", ")
	if len(joined) > skillCatalogMaxBytes {
		b.WriteString(joined[:skillCatalogMaxBytes])
		b.WriteString("\n…（目录过长已截断；完整 id 仍可通过「技能文件路径」推断，或调用 run_skill 使用完整 skill_name。）")
	} else {
		b.WriteString(joined)
	}
	return b.String()
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
	if cfg.StreamTimeoutSecs <= 0 {
		cfg.StreamTimeoutSecs = defaultStreamTimeoutSecs
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

// StreamHarness 供 Chat 网关等场景使用：全量工具（nil ToolOptions）与完整 HarnessRequest（含托管模型 ID）。
func (uc *AgentUsecase) StreamHarness(ctx context.Context, req HarnessRequest) StreamGenerator {
	return uc.executeHarness(req, ctx)
}

func (uc *AgentUsecase) executeHarness(req HarnessRequest, ctx context.Context) StreamGenerator {
	return func(yield func(*StreamMessage, error) bool) {
		requestID := uuid.NewString()
		enriched, err := uc.enrichHarnessWithManagedAgent(ctx, req)
		if err != nil {
			uc.harnessLogger.LogError(requestID, "managed_agent", err)
			yield(nil, sanitizeExternalError(err))
			return
		}
		req = enriched
		start := time.Now()
		var assistantAcc strings.Builder
		logModel := req.Model
		if req.LlmConfigID > 0 && req.LlmModelEntryID > 0 {
			if n, _, _, err := uc.ResolveManagedLLM(ctx, req.LlmConfigID, req.LlmModelEntryID); err == nil && n != "" {
				logModel = n
			} else {
				logModel = fmt.Sprintf("managed:%d:%d", req.LlmConfigID, req.LlmModelEntryID)
			}
		}
		if req.ManagedAgentID > 0 {
			logModel = fmt.Sprintf("profile:%d:%s", req.ManagedAgentID, logModel)
		}
		uc.harnessLogger.LogRunStart(requestID, logModel, len(req.History), req.Input)
		defer func() {
			uc.harnessLogger.LogRunFinish(requestID, time.Since(start))
		}()
		cfg := uc.effectiveHarnessConfig(req.ConfigOverride)

		workCtx := ctx
		if sub := strings.TrimSpace(req.WorkspaceSessionDir); sub != "" {
			sub = sanitizePlaygroundWorkspaceSessionDir(sub)
			if sub != "" {
				baseRoot, err := uc.resolveAgentWorkspaceRoot()
				if err != nil {
					uc.harnessLogger.LogError(requestID, "workspace_root", err)
					yield(nil, sanitizeExternalError(err))
					return
				}
				sessionRoot := filepath.Join(baseRoot, sub)
				sessionRoot = filepath.Clean(sessionRoot)
				baseClean := filepath.Clean(baseRoot)
				rel, relErr := filepath.Rel(baseClean, sessionRoot)
				if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
					uc.harnessLogger.LogError(requestID, "workspace_session_dir", errors.New("invalid session path"))
					yield(nil, sanitizeExternalError(errors.New("会话目录无效")))
					return
				}
				if err := os.MkdirAll(sessionRoot, 0o755); err != nil {
					uc.harnessLogger.LogError(requestID, "workspace_mkdir", err)
					yield(nil, sanitizeExternalError(err))
					return
				}
				workCtx = withHarnessWorkspaceRoot(ctx, sessionRoot)
			}
		}

		einoModel, err := uc.newChatModel(workCtx, req)
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
			// 已选托管 Agent 时，系统提示以主站配置为准；留空则使用轻量默认，避免退回到服务级「Code Assistant」大段说明
			if req.ManagedAgentID > 0 {
				systemPrompt = "你是一个由管理员配置的 Agent；请遵循用户任务，并在可用时使用工具。"
			} else {
				systemPrompt = uc.getSystemPrompt()
			}
		}
		msgs := uc.composeMessages(&req, systemPrompt, req.History, req.Input, req.Attachments)
		toolCallCount := 0
		for i := 0; i < cfg.MaxIterations; i++ {
			modelStart := time.Now()
			// 是否调用工具由模型通过流式 chunk 中的 tool_calls 决定。
			stream, genErr := modelRunner.Stream(workCtx, msgs)
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
			if resp.Content != "" {
				assistantAcc.WriteString(resp.Content)
			}
			msgs = append(msgs, resp)

			if len(resp.ToolCalls) == 0 {
				uc.harnessLogger.LogHarnessOutput(requestID, assistantAcc.String())
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

				argsStr := call.Function.Arguments
				if req.TraceSink != nil && req.PlaygroundRunID != "" && req.PlaygroundAgentID != "" {
					req.TraceSink.EmitToolCall(req.PlaygroundRunID, req.PlaygroundAgentID, call.Function.Name, argsStr)
				}

				// 工具调用设置独立超时，防止单个工具阻塞整轮执行。
				toolCtx, cancel := context.WithTimeout(workCtx, time.Duration(cfg.ToolTimeoutSecs)*time.Second)
				toolStart := time.Now()
				toolOutput, toolErr := toolImpl.Invoke(toolCtx, argsStr)
				cancel()
				if req.TraceSink != nil && req.PlaygroundRunID != "" && req.PlaygroundAgentID != "" {
					out := toolOutput
					if len(out) > 24000 {
						out = out[:24000] + "...(truncated)"
					}
					if toolErr != nil {
						out = out + "\nerror: " + toolErr.Error()
					}
					req.TraceSink.EmitToolResult(req.PlaygroundRunID, req.PlaygroundAgentID, call.Function.Name, out, toolErr == nil)
				}
				uc.harnessLogger.LogToolCallIO(requestID, call.Function.Name, time.Since(toolStart), argsStr, toolOutput, toolErr)
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

		err = fmt.Errorf("超过最大迭代次数（maxIterations=%d），Agent 提前终止；多轮工具调用场景请在协作方案 config 中提高 maxIterations/maxToolCalls", cfg.MaxIterations)
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
		uuidTool.Info.Name:      uuidTool,
		skillTool.Info.Name:     skillTool,
		mcpTool.Info.Name:       mcpTool,
		readFileTool.Info.Name:  readFileTool,
		writeFileTool.Info.Name: writeFileTool,
		shellTool.Info.Name:     shellTool,
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
	desc := "执行 SKILL：读取磁盘上的技能文件（Markdown/YAML 等）。skill_name 必须与系统提示中「SKILL 目录」里的某一 id 完全一致。"
	if fe, ok := uc.skillExecutor.(*FileSkillExecutor); ok {
		names := fe.ListAvailableSkillNames()
		if n := len(names); n > 0 {
			desc += fmt.Sprintf(" 当前可用 %d 个。", n)
			head := 48
			if len(names) < head {
				head = len(names)
			}
			desc += " 示例：" + strings.Join(names[:head], ", ")
			if len(names) > head {
				desc += fmt.Sprintf(" …（另有 %d 项见系统提示 SKILL 目录）", len(names)-head)
			}
		}
	}
	toolInfo := &schema.ToolInfo{
		Name: "run_skill",
		Desc: desc,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"skill_name": {
				Type:     schema.String,
				Desc:     "技能 id，与 SKILL 目录中某项完全一致",
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
