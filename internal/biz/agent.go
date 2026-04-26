package biz

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"ruleGoKratos/internal/conf"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

const (
	defaultMaxIterations    = 32
	defaultMaxToolCalls     = 64
	defaultToolTimeoutSecs  = 5
	defaultMaxSubAgentDepth = 2
	// LLM HTTP 流式读取超时（OpenAI SDK Client.Timeout）；规划/长推理需显著大于旧版 60s。
	defaultStreamTimeoutSecs = 600
	defaultChunkSize         = 120
	// 与 Harness / 管理端聊天共用；工具是否可用以运行时为准。
	defaultSystemPrompt = `You are the Code Assistant for this RuleGo / Flowgram deployment. Deliver accurate, actionable engineering help: runnable code when appropriate, concrete shell or API steps, real file paths, and ordered reasoning—avoid vague platitudes.

Tools: When the runtime exposes tools (official skill tool, MCP, workspace file read/write/shell), call them only through real execution; never invent tool outputs, logs, or claim success when a tool did not run. If a tool errors or is unavailable, report it briefly.

Language: Match the user's language (reply in Chinese when they write Chinese).

Facts: Do not invent repository layout, configs, or command results; when unsure, say what you infer and what you need from the user.

Vision & multimodal: User turns may include images attached via the chat multimodal API; interpret them directly with your vision abilities. Do not claim you lack image understanding or use shell/download/workspace tools solely to «view» images already supplied in this conversation—unless the user asks to persist files to workspace or analyze them offline.

Image URLs: If the user pastes HTTPS links to images in the message, the server may fetch them into the same multimodal payload you receive—treat those as viewable images, not as links you must open yourself. Still do not claim you can browse arbitrary sites beyond what is supplied in the conversation payload.

Style: Stay concise; use Markdown with fenced code blocks for code and log excerpts.

MCP registration (this deployment): MCP entries live in the admin «MCP 配置» table. transport=http: endpoint (SSE-capable MCP URL) plus optional headers JSON. transport=stdio: stdio_command, stdio_args_json (JSON array of strings), stdio_env_json (JSON object), endpoint omitted. The tool save_mcp_server_config writes or updates that table. Enabled MCP server tools are exposed directly as concrete tools named mcp_<server>_<tool>; use those concrete MCP tools when available. After saving, remind the user to enable the entry and use «测试» to verify connectivity.`
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
	// ManagedAgentID 非零时由 enrichHarnessWithManagedAgent 注入 Agent 配置（覆盖 ToolOptions / 模型对与 Skill 可见范围）。
	ManagedAgentID int64
	// SkillCatalogFilter 为 nil 时官方 skill tool 可见全部可用技能；非 nil 且 len=0 不暴露技能；非 nil 且 len>0 仅暴露这些 skill name。
	SkillCatalogFilter *[]string

	// WorkspaceSessionDir 相对于配置的 workspace 根的子路径；非空时本轮 Harness 内 read/write/shell 工具仅在该目录下操作（运行前会 MkdirAll）。
	WorkspaceSessionDir string

	// Playground 协作：将 Harness 内工具调用写入 Trace（可选）。
	PlaygroundRunID   string
	PlaygroundAgentID string
	TraceSink         HarnessTraceSink
	// SubAgentDepth 仅用于 run_sub_agent 递归保护；根请求为 0。
	SubAgentDepth int
}

// HarnessTraceSink Playground 注入，映射到 TraceEngine 的工具事件。
type HarnessTraceSink interface {
	EmitToolCall(runID, agentID, toolName, args string)
	EmitToolResult(runID, agentID, toolName, result string, success bool)
}

type harnessSubAgentCtxKey struct{}

type harnessSubAgentContext struct {
	Request HarnessRequest
	Config  HarnessConfig
}

func withHarnessSubAgentContext(ctx context.Context, req HarnessRequest, cfg HarnessConfig) context.Context {
	return context.WithValue(ctx, harnessSubAgentCtxKey{}, harnessSubAgentContext{
		Request: req,
		Config:  cfg,
	})
}

func getHarnessSubAgentContext(ctx context.Context) (harnessSubAgentContext, bool) {
	if ctx == nil {
		return harnessSubAgentContext{}, false
	}
	v := ctx.Value(harnessSubAgentCtxKey{})
	if v == nil {
		return harnessSubAgentContext{}, false
	}
	info, ok := v.(harnessSubAgentContext)
	return info, ok
}

type SkillExecutor interface {
	Execute(ctx context.Context, skillName string, payload string) (string, error)
}

type McpToolProvider interface {
	BuildMcpTools(ctx context.Context, allowlist []string) ([]*HarnessTool, error)
}

type NoopSkillExecutor struct{}

func (n *NoopSkillExecutor) Execute(ctx context.Context, skillName string, payload string) (string, error) {
	return "", fmt.Errorf("skill executor 未配置: %s", skillName)
}

type NoopMcpToolProvider struct{}

func (n *NoopMcpToolProvider) BuildMcpTools(ctx context.Context, allowlist []string) ([]*HarnessTool, error) {
	return nil, nil
}

type AgentUsecase struct {
	log                *log.Helper
	config             *conf.Bootstrap
	harnessLogger      *HarnessLogger
	harnessConfig      HarnessConfig
	skillExecutor      SkillExecutor
	mcpToolProvider    McpToolProvider
	managedLLM         ManagedLLMResolver
	managedAgentLoader ManagedAgentLoader
	mcpConfigAdmin     McpConfigAdmin
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
		skillExecutor:   fileExecutor,
		mcpToolProvider: &NoopMcpToolProvider{},
	}
}

func (uc *AgentUsecase) SetSkillExecutor(executor SkillExecutor) {
	if executor != nil {
		uc.skillExecutor = executor
	}
}

func (uc *AgentUsecase) SetMcpToolProvider(provider McpToolProvider) {
	if provider != nil {
		uc.mcpToolProvider = provider
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
	return uc.composeMessages(context.Background(), req, uc.getSystemPrompt(), history, userMessage, nil)
}

func (uc *AgentUsecase) composeMessages(ctx context.Context, req *HarnessRequest, systemPrompt string, history []HistoryMessage, userText string, attachments []HarnessAttachment) []*schema.Message {
	if strings.TrimSpace(systemPrompt) == "" {
		systemPrompt = uc.getSystemPrompt()
	}
	if req == nil || req.ToolOptions == nil || req.ToolOptions.EnableSkillTool {
		includeSkillInstruction := true
		allowlist := []string(nil)
		if req != nil && req.ToolOptions != nil {
			allowlist = req.ToolOptions.SkillAllowlist
		}
		if req != nil && req.SkillCatalogFilter != nil {
			if len(*req.SkillCatalogFilter) == 0 {
				includeSkillInstruction = false
			}
			allowlist = *req.SkillCatalogFilter
		}
		if includeSkillInstruction {
			if instruction := uc.officialSkillInstruction(ctx, allowlist); strings.TrimSpace(instruction) != "" {
				systemPrompt = strings.TrimSpace(systemPrompt) + "\n\n" + instruction
			}
		}
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
	parts := buildHarnessInputPartsWithOptions(userText, attachments, uc.harnessMultimodalOptions(req))
	msgs = append(msgs, lastUserMessageFromParts(parts))
	return msgs
}

func (uc *AgentUsecase) harnessMultimodalOptions(req *HarnessRequest) HarnessMultimodalOptions {
	_ = req
	// 当前项目使用的 eino-ext OpenAI 适配层尚未消费 Eino 的通用 file_url part。
	// 因此图片/音频/视频继续走原生多模态，普通文件默认回退为可读文本附件块，避免线上请求被模型适配层拒绝。
	return HarnessMultimodalOptions{DisableGenericFilePart: true}
}

func (uc *AgentUsecase) getSystemPrompt() string {
	if uc.config != nil && uc.config.Agent != nil && strings.TrimSpace(uc.config.Agent.SystemPrompt) != "" {
		return uc.config.Agent.SystemPrompt
	}
	return defaultSystemPrompt
}

// UserFacingError 表示错误文案已审阅为不含密钥/路径等敏感信息，可直接展示给调用方。
type UserFacingError struct {
	msg string
}

func (e *UserFacingError) Error() string { return e.msg }

func userFacingError(msg string) error {
	return &UserFacingError{msg: msg}
}

const harnessErrDetailMaxRunes = 280

var (
	reBearerLike    = regexp.MustCompile(`(?i)\bBearer\s+\S+`)
	reSkOpenAIStyle = regexp.MustCompile(`\bsk-[a-zA-Z0-9]{10,}\b`)
	reUnixLikePath  = regexp.MustCompile(`(?:/Users|/home|/var|/tmp)(?:/[\w.-]+)+`)
	reWinLikePath   = regexp.MustCompile(`\b[A-Za-z]:\\(?:[\w.-]+\\)+[\w.-]+`)
	reLongOpaqueTok = regexp.MustCompile(`\b[A-Za-z0-9+/=_-]{64,}\b`)
)

func redactErrorText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	s = reBearerLike.ReplaceAllString(s, "Bearer [redacted]")
	s = reSkOpenAIStyle.ReplaceAllString(s, "sk-[redacted]")
	s = reUnixLikePath.ReplaceAllString(s, "[path]")
	s = reWinLikePath.ReplaceAllString(s, "[path]")
	s = reLongOpaqueTok.ReplaceAllString(s, "[token]")
	if utf8.RuneCountInString(s) > harnessErrDetailMaxRunes {
		r := []rune(s)
		s = string(r[:harnessErrDetailMaxRunes]) + "…"
	}
	return s
}

func harnessStageLabel(stage string) string {
	switch stage {
	case "managed_agent":
		return "托管 Agent 配置"
	case "workspace_root":
		return "工作区根路径"
	case "workspace_mkdir":
		return "工作区目录"
	case "init_model":
		return "模型初始化"
	case "build_tools":
		return "工具注册"
	case "bind_tools":
		return "工具绑定"
	case "model_stream":
		return "模型调用"
	case "model_stream_recv":
		return "模型流读取"
	case "model_stream_concat":
		return "模型输出合并"
	case "tool_limit":
		return "工具沙箱"
	case "tool_denied":
		return "工具授权"
	case "tool_invoke":
		return "工具执行"
	case "max_iterations":
		return "Agent 迭代"
	default:
		return "Agent"
	}
}

func sanitizeExternalError(stage string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return userFacingError("已取消执行")
	}
	var uf *UserFacingError
	if errors.As(err, &uf) {
		return err
	}
	label := harnessStageLabel(stage)
	detail := redactErrorText(err.Error())
	if detail == "" {
		return fmt.Errorf("%s失败，请稍后重试", label)
	}
	return fmt.Errorf("%s失败：%s", label, detail)
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
			yield(nil, sanitizeExternalError("managed_agent", err))
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
					yield(nil, sanitizeExternalError("workspace_root", err))
					return
				}
				sessionRoot := filepath.Join(baseRoot, sub)
				sessionRoot = filepath.Clean(sessionRoot)
				baseClean := filepath.Clean(baseRoot)
				rel, relErr := filepath.Rel(baseClean, sessionRoot)
				if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
					uc.harnessLogger.LogError(requestID, "workspace_session_dir", errors.New("invalid session path"))
					yield(nil, userFacingError("会话目录无效"))
					return
				}
				if err := os.MkdirAll(sessionRoot, 0o755); err != nil {
					uc.harnessLogger.LogError(requestID, "workspace_mkdir", err)
					yield(nil, sanitizeExternalError("workspace_mkdir", err))
					return
				}
				workCtx = withHarnessWorkspaceRoot(ctx, sessionRoot)
			}
		}
		workCtx = withHarnessSubAgentContext(workCtx, req, cfg)

		einoModel, err := uc.newChatModel(workCtx, req)
		if err != nil {
			uc.harnessLogger.LogError(requestID, "init_model", err)
			yield(nil, sanitizeExternalError("init_model", err))
			return
		}

		toolRegistry, toolInfos, err := uc.resolveToolRegistry(workCtx, req.ToolOptions)
		if err != nil {
			uc.harnessLogger.LogError(requestID, "build_tools", err)
			yield(nil, sanitizeExternalError("build_tools", err))
			return
		}
		modelRunner := einoModel
		if len(toolInfos) > 0 {
			modelWithTools, wErr := einoModel.WithTools(toolInfos)
			if wErr != nil {
				uc.harnessLogger.LogError(requestID, "bind_tools", wErr)
				yield(nil, sanitizeExternalError("bind_tools", wErr))
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
		msgs := uc.composeMessages(workCtx, &req, systemPrompt, req.History, req.Input, req.Attachments)
		toolCallCount := 0
		for i := 0; i < cfg.MaxIterations; i++ {
			if err := workCtx.Err(); err != nil {
				yield(nil, userFacingError("已取消执行"))
				return
			}
			modelStart := time.Now()
			// 是否调用工具由模型通过流式 chunk 中的 tool_calls 决定。
			stream, genErr := modelRunner.Stream(workCtx, msgs)
			uc.harnessLogger.LogModelRound(requestID, i+1, time.Since(modelStart), genErr)
			if genErr != nil {
				uc.harnessLogger.LogError(requestID, "model_stream", genErr)
				yield(nil, sanitizeExternalError("model_stream", genErr))
				return
			}
			// 收集流式 chunk，用于拼出完整 assistant 消息并进入下一轮状态。
			chunks := make([]*schema.Message, 0, 32)
			var streamedThisRound strings.Builder
			seenToolCall := false
			for {
				if err := workCtx.Err(); err != nil {
					stream.Close()
					yield(nil, userFacingError("已取消执行"))
					return
				}
				chunk, recvErr := stream.Recv()
				if recvErr == io.EOF {
					break
				}
				if recvErr != nil {
					stream.Close()
					uc.harnessLogger.LogError(requestID, "model_stream_recv", recvErr)
					yield(nil, sanitizeExternalError("model_stream_recv", recvErr))
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
					streamedThisRound.WriteString(chunk.Content)
					if !yield(&StreamMessage{Content: chunk.Content, Done: false}, nil) {
						stream.Close()
						return
					}
				}
			}
			stream.Close()

			if err := workCtx.Err(); err != nil {
				yield(nil, userFacingError("已取消执行"))
				return
			}

			// Eino 返回的是分片，需要先合并，才能稳定处理工具调用。
			resp, concatErr := schema.ConcatMessages(chunks)
			if concatErr != nil {
				uc.harnessLogger.LogError(requestID, "model_stream_concat", concatErr)
				yield(nil, sanitizeExternalError("model_stream_concat", concatErr))
				return
			}
			if resp == nil {
				uc.harnessLogger.LogError(requestID, "model_stream_concat", errors.New("模型返回为空"))
				yield(nil, userFacingError("模型返回为空"))
				return
			}
			if resp.Content != "" {
				assistantAcc.WriteString(resp.Content)
			}
			// 某些模型/SDK在流式阶段不稳定地产生空 chunk，正文只会在 concat 后出现。
			// 这里兜底补发本轮尚未透传给前端的增量，避免前端表现为“非流式/只在结束时更新”。
			if len(resp.ToolCalls) == 0 && resp.Content != "" {
				alreadyStreamed := streamedThisRound.String()
				remain := resp.Content
				if alreadyStreamed != "" && strings.HasPrefix(resp.Content, alreadyStreamed) {
					remain = resp.Content[len(alreadyStreamed):]
				}
				if remain != "" {
					if !yield(&StreamMessage{Content: remain, Done: false}, nil) {
						return
					}
				}
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
					yield(nil, userFacingError("工具调用次数超过沙箱限制"))
					return
				}

				if !allowedTools[call.Function.Name] {
					err = fmt.Errorf("未授权工具: %s", call.Function.Name)
					uc.harnessLogger.LogSandboxDecision(requestID, call.Function.Name, false, "tool_not_in_whitelist")
					uc.harnessLogger.LogError(requestID, "tool_denied", err)
					yield(nil, userFacingError(fmt.Sprintf("未授权工具: %s", call.Function.Name)))
					return
				}

				toolImpl, ok := toolRegistry[call.Function.Name]
				if !ok {
					err = fmt.Errorf("未授权工具: %s", call.Function.Name)
					uc.harnessLogger.LogSandboxDecision(requestID, call.Function.Name, false, "tool_not_registered")
					uc.harnessLogger.LogError(requestID, "tool_denied", err)
					yield(nil, userFacingError(fmt.Sprintf("未授权工具: %s", call.Function.Name)))
					return
				}
				uc.harnessLogger.LogSandboxDecision(requestID, call.Function.Name, true, "tool_allowed")

				if err := workCtx.Err(); err != nil {
					yield(nil, userFacingError("已取消执行"))
					return
				}

				argsStr := call.Function.Arguments
				if req.TraceSink != nil && req.PlaygroundRunID != "" && req.PlaygroundAgentID != "" {
					req.TraceSink.EmitToolCall(req.PlaygroundRunID, req.PlaygroundAgentID, call.Function.Name, argsStr)
				}

				// 工具调用默认设置独立超时，防止单个工具阻塞整轮执行。
				// 但 run_sub_agent 需要执行完整子 Agent 回合，若沿用短工具超时（默认 5s）会提前触发 context deadline exceeded。
				// 因此对子 Agent 工具放宽为继承会话上下文，具体耗时由子 Agent 自身的流超时与迭代上限控制。
				toolCtx := workCtx
				cancel := func() {}
				if call.Function.Name != "run_sub_agent" {
					toolCtx, cancel = context.WithTimeout(
						workCtx,
						time.Duration(cfg.ToolTimeoutSecs)*time.Second,
					)
				}
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
					yield(nil, sanitizeExternalError("tool_invoke", toolErr))
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
		yield(nil, userFacingError(err.Error()))
	}
}

func (uc *AgentUsecase) Run(ctx context.Context, userMessage string) StreamGenerator {
	return uc.ExecuteStream(ctx, "", nil, userMessage)
}

func (uc *AgentUsecase) BuildToolRegistry() (map[string]*HarnessTool, []*schema.ToolInfo, error) {
	return uc.BuildToolRegistryForContext(context.Background())
}

func (uc *AgentUsecase) BuildToolRegistryForContext(ctx context.Context) (map[string]*HarnessTool, []*schema.ToolInfo, error) {
	// Registry 是当前 Harness 运行期可调用工具的唯一来源。
	uuidTool, err := uc.BuildUUIDTool()
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
	subAgentTool, err := uc.BuildSubAgentTool()
	if err != nil {
		return nil, nil, err
	}
	registry := map[string]*HarnessTool{
		uuidTool.Info.Name:      uuidTool,
		readFileTool.Info.Name:  readFileTool,
		writeFileTool.Info.Name: writeFileTool,
		shellTool.Info.Name:     shellTool,
		subAgentTool.Info.Name:  subAgentTool,
	}
	infos := []*schema.ToolInfo{
		uuidTool.Info,
		readFileTool.Info, writeFileTool.Info, shellTool.Info, subAgentTool.Info,
	}
	skillTools, _, err := uc.buildOfficialSkillTools(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	for _, t := range skillTools {
		if t == nil || t.Info == nil || strings.TrimSpace(t.Info.Name) == "" {
			continue
		}
		if _, exists := registry[t.Info.Name]; exists {
			continue
		}
		registry[t.Info.Name] = t
		infos = append(infos, t.Info)
	}
	mcpTools, err := uc.buildMcpTools(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	for _, t := range mcpTools {
		if t == nil || t.Info == nil || strings.TrimSpace(t.Info.Name) == "" {
			continue
		}
		if _, exists := registry[t.Info.Name]; exists {
			continue
		}
		registry[t.Info.Name] = t
		infos = append(infos, t.Info)
	}
	if uc.mcpConfigAdmin != nil {
		saveTool, err := uc.BuildSaveMcpConfigTool()
		if err != nil {
			return nil, nil, err
		}
		registry[saveTool.Info.Name] = saveTool
		infos = append(infos, saveTool.Info)
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

func (uc *AgentUsecase) BuildSubAgentTool() (*HarnessTool, error) {
	toolInfo := &schema.ToolInfo{
		Name: "run_sub_agent",
		Desc: "拉起子 Agent 执行子任务；支持单任务或 sub_tasks_json 批量任务，并可并发聚合结果。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"task": {
				Type:     schema.String,
				Desc:     "单个子任务描述；与 sub_tasks_json 二选一或并用",
				Required: false,
			},
			"sub_tasks_json": {
				Type:     schema.String,
				Desc:     "可选；子任务数组 JSON 字符串，例如 [\"任务A\",\"任务B\"]",
				Required: false,
			},
			"system_prompt": {
				Type:     schema.String,
				Desc:     "可选；子 Agent 的系统提示词，默认继承当前 Agent",
				Required: false,
			},
			"managed_agent_id": {
				Type:     schema.Integer,
				Desc:     "可选；指定托管 Agent 配置 id",
				Required: false,
			},
			"max_iterations": {
				Type:     schema.Integer,
				Desc:     "可选；覆盖子 Agent 最大迭代轮次",
				Required: false,
			},
			"max_tool_calls": {
				Type:     schema.Integer,
				Desc:     "可选；覆盖子 Agent 最大工具调用次数",
				Required: false,
			},
			"tool_timeout_secs": {
				Type:     schema.Integer,
				Desc:     "可选；覆盖子 Agent 单次工具超时秒数",
				Required: false,
			},
			"max_concurrency": {
				Type:     schema.Integer,
				Desc:     "可选；批量任务并发度（1~8）。不传则按子任务数量自动估算",
				Required: false,
			},
		}),
	}
	type runSubAgentArgs struct {
		Task            string          `json:"task"`
		SubTasksJSON    json.RawMessage `json:"sub_tasks_json"`
		SystemPrompt    string          `json:"system_prompt"`
		ManagedAgentID  int64           `json:"managed_agent_id"`
		MaxIterations   int             `json:"max_iterations"`
		MaxToolCalls    int             `json:"max_tool_calls"`
		ToolTimeoutSecs int             `json:"tool_timeout_secs"`
		MaxConcurrency  int             `json:"max_concurrency"`
	}
	type subAgentBatchItem struct {
		Task      string   `json:"task"`
		Summary   string   `json:"summary"`
		Findings  []string `json:"findings"`
		NextSteps []string `json:"next_steps"`
		Error     string   `json:"error,omitempty"`
		Raw       string   `json:"raw,omitempty"`
	}
	type subAgentNormalizedOutput struct {
		Summary              string              `json:"summary"`
		Findings             []string            `json:"findings"`
		NextSteps            []string            `json:"next_steps"`
		SubResults           []subAgentBatchItem `json:"sub_results,omitempty"`
		TaskCount            int                 `json:"task_count,omitempty"`
		RequestedConcurrency int                 `json:"requested_concurrency,omitempty"`
		EffectiveConcurrency int                 `json:"effective_concurrency,omitempty"`
		ConcurrencyReason    string              `json:"concurrency_reason,omitempty"`
		Raw                  string              `json:"raw,omitempty"`
	}
	normalizeOutput := func(raw string) subAgentNormalizedOutput {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return subAgentNormalizedOutput{
				Summary:   "子 Agent 未返回内容",
				Findings:  []string{},
				NextSteps: []string{},
			}
		}
		var parsed subAgentNormalizedOutput
		if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
			if strings.TrimSpace(parsed.Summary) == "" {
				parsed.Summary = "子 Agent 已执行（summary 为空）"
			}
			if parsed.Findings == nil {
				parsed.Findings = []string{}
			}
			if parsed.NextSteps == nil {
				parsed.NextSteps = []string{}
			}
			return parsed
		}
		return subAgentNormalizedOutput{
			Summary:   "子 Agent 返回了非结构化内容，已自动兜底归一化",
			Findings:  []string{},
			NextSteps: []string{},
			Raw:       raw,
		}
	}
	marshalOutput := func(out subAgentNormalizedOutput) string {
		b, _ := json.Marshal(out)
		return string(b)
	}
	parseTasks := func(singleTask string, subTasksJSON string) ([]string, error) {
		tasks := make([]string, 0, 4)
		if t := strings.TrimSpace(singleTask); t != "" {
			tasks = append(tasks, t)
		}
		raw := strings.TrimSpace(subTasksJSON)
		if raw != "" {
			var arr []string
			if err := json.Unmarshal([]byte(raw), &arr); err == nil {
				for _, item := range arr {
					if t := strings.TrimSpace(item); t != "" {
						tasks = append(tasks, t)
					}
				}
			} else {
				return nil, fmt.Errorf("sub_tasks_json 不是合法 JSON 数组: %w", err)
			}
		}
		if len(tasks) == 0 {
			return nil, errors.New("task 与 sub_tasks_json 不能同时为空")
		}
		return tasks, nil
	}
	estimateConcurrency := func(taskCount int) int {
		if taskCount <= 1 {
			return 1
		}
		if taskCount <= 3 {
			return taskCount
		}
		if taskCount <= 6 {
			return 3
		}
		if taskCount <= 12 {
			return 4
		}
		return 6
	}
	buildChildReq := func(parentReq HarnessRequest, args runSubAgentArgs, task string, nextDepth int) HarnessRequest {
		childReq := parentReq
		childReq.History = nil
		childReq.Input = task + "\n\n请严格返回 JSON 对象，字段仅包含：summary(string), findings(string[]), next_steps(string[])。不要输出 markdown 代码块，不要输出额外字段。"
		childReq.Attachments = nil
		childReq.SubAgentDepth = nextDepth
		childReq.PlaygroundRunID = ""
		childReq.PlaygroundAgentID = ""
		childReq.TraceSink = nil

		if sp := strings.TrimSpace(args.SystemPrompt); sp != "" {
			childReq.SystemPrompt = sp
		}
		if args.ManagedAgentID > 0 {
			childReq.ManagedAgentID = args.ManagedAgentID
			childReq.LlmConfigID = 0
			childReq.LlmModelEntryID = 0
			childReq.Model = ""
		}
		if childReq.ToolOptions != nil {
			childReq.ToolOptions = cloneHarnessToolOptions(childReq.ToolOptions)
			childReq.ToolOptions.EnableSubAgentTool = nextDepth < defaultMaxSubAgentDepth
		}
		if args.MaxIterations > 0 || args.MaxToolCalls > 0 || args.ToolTimeoutSecs > 0 {
			override := &HarnessConfig{}
			if args.MaxIterations > 0 {
				override.MaxIterations = args.MaxIterations
			}
			if args.MaxToolCalls > 0 {
				override.MaxToolCalls = args.MaxToolCalls
			}
			if args.ToolTimeoutSecs > 0 {
				override.ToolTimeoutSecs = args.ToolTimeoutSecs
			}
			childReq.ConfigOverride = override
		}
		return childReq
	}
	return &HarnessTool{
		Info: toolInfo,
		Invoke: func(ctx context.Context, rawArgs string) (string, error) {
			var args runSubAgentArgs
			if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
				return "", err
			}
			// LLM may send sub_tasks_json as either a JSON string ("[\"a\"]") or a raw array (["a"]).
			// json.RawMessage captures the raw bytes; normalize to string for parseTasks.
			var subTasksStr string
			if len(args.SubTasksJSON) > 0 {
				trimmed := bytes.TrimSpace(args.SubTasksJSON)
				if len(trimmed) > 0 && trimmed[0] == '"' {
					// LLM sent a quoted JSON string — unquote to get the inner string value.
					var s string
					if err := json.Unmarshal(args.SubTasksJSON, &s); err == nil {
						subTasksStr = s
					} else {
						subTasksStr = string(args.SubTasksJSON)
					}
				} else {
					// LLM sent a raw JSON array or other content — pass as-is.
					subTasksStr = string(args.SubTasksJSON)
				}
			}
			tasks, err := parseTasks(args.Task, subTasksStr)
			if err != nil {
				return "", err
			}
			parentCtx, ok := getHarnessSubAgentContext(ctx)
			if !ok {
				return "", errors.New("当前上下文不支持 run_sub_agent")
			}
			parentReq := parentCtx.Request
			nextDepth := parentReq.SubAgentDepth + 1
			if nextDepth > defaultMaxSubAgentDepth {
				return "", fmt.Errorf("sub-agent 递归深度超过限制（max=%d）", defaultMaxSubAgentDepth)
			}

			executeTask := func(task string) subAgentBatchItem {
				childReq := buildChildReq(parentReq, args, task, nextDepth)
				out, execErr := uc.ExecuteHarnessSync(ctx, childReq)
				if execErr != nil {
					return subAgentBatchItem{
						Task:      task,
						Summary:   "子任务执行失败",
						Findings:  []string{execErr.Error()},
						NextSteps: []string{"检查子任务描述、工具权限或模型配置后重试"},
						Error:     execErr.Error(),
					}
				}
				norm := normalizeOutput(out)
				return subAgentBatchItem{
					Task:      task,
					Summary:   norm.Summary,
					Findings:  norm.Findings,
					NextSteps: norm.NextSteps,
					Raw:       norm.Raw,
				}
			}

			if len(tasks) == 1 {
				item := executeTask(tasks[0])
				reason := "single_task_forced_1"
				if args.MaxConcurrency > 0 {
					reason = "single_task_ignores_user_concurrency"
				}
				return marshalOutput(subAgentNormalizedOutput{
					Summary:              item.Summary,
					Findings:             item.Findings,
					NextSteps:            item.NextSteps,
					SubResults:           []subAgentBatchItem{item},
					TaskCount:            1,
					RequestedConcurrency: args.MaxConcurrency,
					EffectiveConcurrency: 1,
					ConcurrencyReason:    reason,
					Raw:                  item.Raw,
				}), nil
			}

			maxConc := args.MaxConcurrency
			reason := "user_specified"
			if maxConc <= 0 {
				maxConc = estimateConcurrency(len(tasks))
				reason = "auto_estimated_by_task_count"
			}
			if maxConc < 1 {
				maxConc = 1
				reason = "clamped_to_min_1"
			}
			if maxConc > 8 {
				maxConc = 8
				reason = "clamped_to_max_8"
			}
			type indexedResult struct {
				Index int
				Item  subAgentBatchItem
			}
			sem := make(chan struct{}, maxConc)
			outCh := make(chan indexedResult, len(tasks))
			var wg sync.WaitGroup
			for idx, task := range tasks {
				wg.Add(1)
				go func(i int, t string) {
					defer wg.Done()
					sem <- struct{}{}
					item := executeTask(t)
					<-sem
					outCh <- indexedResult{Index: i, Item: item}
				}(idx, task)
			}
			wg.Wait()
			close(outCh)

			items := make([]subAgentBatchItem, len(tasks))
			success := 0
			findings := make([]string, 0, len(tasks))
			nextSteps := make([]string, 0, len(tasks)*2)
			for res := range outCh {
				items[res.Index] = res.Item
			}
			for idx, item := range items {
				if item.Error == "" {
					success++
				}
				findings = append(findings, fmt.Sprintf("[%d] %s => %s", idx+1, item.Task, item.Summary))
				nextSteps = append(nextSteps, item.NextSteps...)
			}
			return marshalOutput(subAgentNormalizedOutput{
				Summary:              fmt.Sprintf("批量子任务完成：成功 %d / 总计 %d", success, len(tasks)),
				Findings:             findings,
				NextSteps:            nextSteps,
				SubResults:           items,
				TaskCount:            len(tasks),
				RequestedConcurrency: args.MaxConcurrency,
				EffectiveConcurrency: maxConc,
				ConcurrencyReason:    reason,
			}), nil
		},
	}, nil
}

// BuildSaveMcpConfigTool 将一条 MCP 登记到本机「MCP 配置」表（与 Code 助手 / 管理后台同源）；需已注入 McpConfigAdmin。
func (uc *AgentUsecase) BuildSaveMcpConfigTool() (*HarnessTool, error) {
	if uc.mcpConfigAdmin == nil {
		return nil, errors.New("mcp config admin 未配置")
	}
	toolInfo := &schema.ToolInfo{
		Name: "save_mcp_server_config",
		Desc: "将一条 MCP 服务写入本系统的「MCP 配置」数据库（与管理后台相同）。transport=http 时填写 endpoint（SSE URL）与可选 headers_json；transport=stdio 时填写 stdio_command、stdio_args_json（JSON 字符串数组）、stdio_env_json（JSON 对象），endpoint 可省略。启用后该 server 下的 MCP tools 会以 mcp_<server>_<tool> 形式直接暴露。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"id": {
				Type:     schema.String,
				Desc:     "已有记录 id，非零则更新；省略或 0 为新建",
				Required: false,
			},
			"name": {
				Type:     schema.String,
				Desc:     "展示名称",
				Required: true,
			},
			"server": {
				Type:     schema.String,
				Desc:     "逻辑服务名，用作 MCP tool 名称前缀",
				Required: true,
			},
			"transport": {
				Type:     schema.String,
				Desc:     "http 或 stdio；可省略则：有 stdio_command 视为 stdio，否则 http",
				Required: false,
			},
			"endpoint": {
				Type:     schema.String,
				Desc:     "http 模式：MCP HTTP(SSE) 入口 URL；stdio 模式可空",
				Required: false,
			},
			"headers_json": {
				Type:     schema.String,
				Desc:     "HTTP 请求头 JSON 对象字符串，可为 {}",
				Required: false,
			},
			"stdio_command": {
				Type:     schema.String,
				Desc:     "stdio 模式：可执行文件，如 uv、npx、python3",
				Required: false,
			},
			"stdio_args_json": {
				Type:     schema.String,
				Desc:     "stdio 模式：参数 JSON 数组字符串，如 [\"-y\",\"@pkg\"]",
				Required: false,
			},
			"stdio_env_json": {
				Type:     schema.String,
				Desc:     "stdio 模式：环境变量 JSON 对象字符串",
				Required: false,
			},
			"enabled": {
				Type:     schema.String,
				Desc:     "是否启用：true 或 false",
				Required: false,
			},
			"description": {
				Type:     schema.String,
				Desc:     "说明，可选",
				Required: false,
			},
		}),
	}
	type saveMcpArgs struct {
		ID            interface{} `json:"id"`
		Name          string      `json:"name"`
		Server        string      `json:"server"`
		Transport     string      `json:"transport"`
		Endpoint      string      `json:"endpoint"`
		HeadersJSON   string      `json:"headers_json"`
		StdioCommand  string      `json:"stdio_command"`
		StdioArgsJSON string      `json:"stdio_args_json"`
		StdioEnvJSON  string      `json:"stdio_env_json"`
		Enabled       *bool       `json:"enabled"`
		Description   string      `json:"description"`
	}
	parseIDAny := func(v interface{}) int64 {
		if v == nil {
			return 0
		}
		switch x := v.(type) {
		case float64:
			return int64(x)
		case string:
			x = strings.TrimSpace(x)
			if x == "" {
				return 0
			}
			n, err := strconv.ParseInt(x, 10, 64)
			if err != nil {
				return 0
			}
			return n
		case json.Number:
			n, err := x.Int64()
			if err != nil {
				return 0
			}
			return n
		default:
			return 0
		}
	}
	return &HarnessTool{
		Info: toolInfo,
		Invoke: func(ctx context.Context, rawArgs string) (string, error) {
			var args saveMcpArgs
			if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
				return "", err
			}
			id := parseIDAny(args.ID)
			enabled := true
			if args.Enabled != nil {
				enabled = *args.Enabled
			}
			transport := strings.ToLower(strings.TrimSpace(args.Transport))
			if transport == "" {
				if strings.TrimSpace(args.StdioCommand) != "" {
					transport = "stdio"
				} else {
					transport = "http"
				}
			}
			outID, action, err := uc.mcpConfigAdmin.UpsertMcpConfig(ctx, McpConfigUpsertArgs{
				ID:            id,
				Name:          args.Name,
				Server:        args.Server,
				Transport:     transport,
				Endpoint:      args.Endpoint,
				HeadersJSON:   args.HeadersJSON,
				StdioCommand:  args.StdioCommand,
				StdioArgsJSON: args.StdioArgsJSON,
				StdioEnvJSON:  args.StdioEnvJSON,
				Enabled:       enabled,
				Description:   args.Description,
			})
			if err != nil {
				return "", err
			}
			return fmt.Sprintf(`{"ok":true,"id":%d,"action":%q}`, outID, action), nil
		},
	}, nil
}
