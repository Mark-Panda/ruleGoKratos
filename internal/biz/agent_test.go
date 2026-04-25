package biz

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"ruleGoKratos/internal/conf"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/go-kratos/kratos/v2/log"
)

func newTestAgentUsecase() *AgentUsecase {
	helper := log.NewHelper(log.NewStdLogger(io.Discard))
	return &AgentUsecase{
		log:           helper,
		harnessLogger: NewHarnessLogger(helper),
		harnessConfig: HarnessConfig{
			MaxIterations:   defaultMaxIterations,
			MaxToolCalls:    defaultMaxToolCalls,
			ToolTimeoutSecs: defaultToolTimeoutSecs,
			ChunkSize:       defaultChunkSize,
		},
		skillExecutor: &NoopSkillExecutor{},
		mcpExecutor:   &NoopMcpExecutor{},
	}
}

func TestBuildToolRegistryIncludesSkillAndMcp(t *testing.T) {
	uc := newTestAgentUsecase()
	registry, infos, err := uc.BuildToolRegistry()
	if err != nil {
		t.Fatalf("BuildToolRegistry failed: %v", err)
	}
	if len(registry) != 7 {
		t.Fatalf("unexpected tool registry size: %d", len(registry))
	}
	if len(infos) != 7 {
		t.Fatalf("unexpected tool infos size: %d", len(infos))
	}
	if _, ok := registry["run_skill"]; !ok {
		t.Fatalf("run_skill tool missing")
	}
	if _, ok := registry["call_mcp_tool"]; !ok {
		t.Fatalf("call_mcp_tool tool missing")
	}
	for _, name := range []string{"read_workspace_file", "write_workspace_file", "run_workspace_shell"} {
		if _, ok := registry[name]; !ok {
			t.Fatalf("tool %s missing", name)
		}
	}
	if _, ok := registry["run_sub_agent"]; !ok {
		t.Fatalf("run_sub_agent tool missing")
	}
}

func TestWorkspaceWriteReadAndRejectEscape(t *testing.T) {
	root := t.TempDir()
	uc := newTestAgentUsecase()
	uc.config = &conf.Bootstrap{Agent: &conf.Agent{WorkspaceRoot: root}}

	wt, err := uc.BuildWriteWorkspaceFileTool()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Invoke(context.Background(), `{"path":"sub/x.txt","content":"hello"}`); err != nil {
		t.Fatal(err)
	}
	rt, err := uc.BuildReadWorkspaceFileTool()
	if err != nil {
		t.Fatal(err)
	}
	out, err := rt.Invoke(context.Background(), `{"path":"sub/x.txt"}`)
	if err != nil || out != "hello" {
		t.Fatalf("read back: err=%v out=%q", err, out)
	}
	if _, err := rt.Invoke(context.Background(), `{"path":"../`+filepath.Base(root)+`_leak.txt"}`); err == nil {
		t.Fatalf("expected error for escape path")
	}
}

func TestWorkspaceToolsUseSessionRootFromContext(t *testing.T) {
	root := t.TempDir()
	sess := filepath.Join(root, "playground", "run_test")
	if err := os.MkdirAll(sess, 0o755); err != nil {
		t.Fatal(err)
	}
	uc := newTestAgentUsecase()
	uc.config = &conf.Bootstrap{Agent: &conf.Agent{WorkspaceRoot: root}}

	ctx := withHarnessWorkspaceRoot(context.Background(), sess)
	wt, err := uc.BuildWriteWorkspaceFileTool()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Invoke(ctx, `{"path":"note.txt","content":"session"}`); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(sess, "note.txt"))
	if err != nil || string(b) != "session" {
		t.Fatalf("read session file: err=%v data=%q", err, string(b))
	}
	if _, err := os.Stat(filepath.Join(root, "note.txt")); err == nil {
		t.Fatal("must not write at configured workspace root without session subdirectory")
	}
}

func TestWorkspaceWriteAllowsAgentSkillAbsolutePath(t *testing.T) {
	workspaceRoot := t.TempDir()
	agentRoot := t.TempDir()
	t.Setenv("AGENT_SKILL_DIR", agentRoot)

	uc := newTestAgentUsecase()
	uc.config = &conf.Bootstrap{Agent: &conf.Agent{WorkspaceRoot: workspaceRoot}}

	wt, err := uc.BuildWriteWorkspaceFileTool()
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(agentRoot, "bug", "SKILL.md")
	if _, err := wt.Invoke(context.Background(), fmt.Sprintf(`{"path":%q,"content":"ok"}`, target)); err != nil {
		t.Fatalf("expected absolute write under agent skill root to pass, got %v", err)
	}
	b, err := os.ReadFile(target)
	if err != nil || string(b) != "ok" {
		t.Fatalf("read written agent skill: err=%v data=%q", err, string(b))
	}
}

func TestWorkspaceWriteRejectsWorkflowSkillAbsolutePath(t *testing.T) {
	workspaceRoot := t.TempDir()
	workflowRoot := t.TempDir()
	t.Setenv("WORKFLOW_SKILL_DIR", workflowRoot)

	uc := newTestAgentUsecase()
	uc.config = &conf.Bootstrap{Agent: &conf.Agent{WorkspaceRoot: workspaceRoot}}

	wt, err := uc.BuildWriteWorkspaceFileTool()
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(workflowRoot, "bug", "SKILL.md")
	if _, err := wt.Invoke(context.Background(), fmt.Sprintf(`{"path":%q,"content":"nope"}`, target)); err == nil {
		t.Fatal("expected absolute write under workflow skill root to fail")
	}
}

func TestWorkspaceWriteRejectsAbsolutePathOutsideAllowedRoots(t *testing.T) {
	workspaceRoot := t.TempDir()
	workflowRoot := t.TempDir()
	outsideRoot := t.TempDir()
	t.Setenv("WORKFLOW_SKILL_DIR", workflowRoot)

	uc := newTestAgentUsecase()
	uc.config = &conf.Bootstrap{Agent: &conf.Agent{WorkspaceRoot: workspaceRoot}}

	wt, err := uc.BuildWriteWorkspaceFileTool()
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(outsideRoot, "escape.txt")
	if _, err := wt.Invoke(context.Background(), fmt.Sprintf(`{"path":%q,"content":"nope"}`, target)); err == nil {
		t.Fatal("expected absolute write outside allowed roots to fail")
	}
}

func TestRunSkillToolValidateArgs(t *testing.T) {
	uc := newTestAgentUsecase()
	tool, err := uc.BuildSkillTool()
	if err != nil {
		t.Fatalf("BuildSkillTool failed: %v", err)
	}
	_, err = tool.Invoke(context.Background(), `{"payload":"abc"}`)
	if err == nil || !strings.Contains(err.Error(), "skill_name") {
		t.Fatalf("expected skill_name validation error, got: %v", err)
	}
}

func TestCallMcpToolValidateArgs(t *testing.T) {
	uc := newTestAgentUsecase()
	tool, err := uc.BuildMCPTool()
	if err != nil {
		t.Fatalf("BuildMCPTool failed: %v", err)
	}
	_, err = tool.Invoke(context.Background(), `{"server":"","tool":"x"}`)
	if err == nil || !strings.Contains(err.Error(), "server") {
		t.Fatalf("expected server/tool validation error, got: %v", err)
	}
}

func TestCallMcpToolShouldRejectSkillIDServer(t *testing.T) {
	helper := log.NewHelper(log.NewStdLogger(io.Discard))
	skillDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(skillDir, "agent-browser-clawdbot-0.1.0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "agent-browser-clawdbot-0.1.0", "SKILL.md"), []byte("# browser skill"), 0o644); err != nil {
		t.Fatal(err)
	}
	fe, err := NewFileSkillExecutor([]string{skillDir}, FileSkillExecutorOptions{HotReload: false, HotReloadSet: true})
	if err != nil {
		t.Fatalf("NewFileSkillExecutor failed: %v", err)
	}
	uc := &AgentUsecase{
		log:           helper,
		harnessLogger: NewHarnessLogger(helper),
		skillExecutor: fe,
		mcpExecutor:   &fakeMcpExecutor{},
	}
	tool, err := uc.BuildMCPTool()
	if err != nil {
		t.Fatalf("BuildMCPTool failed: %v", err)
	}

	_, err = tool.Invoke(context.Background(), `{"server":"agent-browser-clawdbot-0.1.0","tool":"search"}`)
	if err == nil || !strings.Contains(err.Error(), "run_skill") {
		t.Fatalf("expected skill/mcp misuse error, got: %v", err)
	}
}

type fakeSkillExecutor struct {
	called bool
	name   string
	data   string
}

func (f *fakeSkillExecutor) Execute(ctx context.Context, skillName string, payload string) (string, error) {
	f.called = true
	f.name = skillName
	f.data = payload
	return fmt.Sprintf("skill:%s:%s", skillName, payload), nil
}

type fakeMcpExecutor struct {
	called    bool
	server    string
	tool      string
	arguments string
}

func (f *fakeMcpExecutor) Call(ctx context.Context, server string, tool string, arguments string) (string, error) {
	f.called = true
	f.server = server
	f.tool = tool
	f.arguments = arguments
	return fmt.Sprintf("mcp:%s:%s:%s", server, tool, arguments), nil
}

func TestRunSkillToolShouldInvokeExecutor(t *testing.T) {
	uc := newTestAgentUsecase()
	fake := &fakeSkillExecutor{}
	uc.SetSkillExecutor(fake)
	tool, err := uc.BuildSkillTool()
	if err != nil {
		t.Fatalf("BuildSkillTool failed: %v", err)
	}
	output, err := tool.Invoke(context.Background(), `{"skill_name":"planner","payload":"demo"}`)
	if err != nil {
		t.Fatalf("tool invoke failed: %v", err)
	}
	if !fake.called || fake.name != "planner" || fake.data != "demo" {
		t.Fatalf("skill executor was not called correctly")
	}
	if output != "skill:planner:demo" {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestCallMcpToolShouldInvokeExecutor(t *testing.T) {
	uc := newTestAgentUsecase()
	fake := &fakeMcpExecutor{}
	uc.SetMcpExecutor(fake)
	tool, err := uc.BuildMCPTool()
	if err != nil {
		t.Fatalf("BuildMCPTool failed: %v", err)
	}
	output, err := tool.Invoke(context.Background(), `{"server":"cursor","tool":"browser_tabs","arguments":"{\"action\":\"list\"}"}`)
	if err != nil {
		t.Fatalf("tool invoke failed: %v", err)
	}
	if !fake.called || fake.server != "cursor" || fake.tool != "browser_tabs" {
		t.Fatalf("mcp executor was not called correctly")
	}
	if output == "" {
		t.Fatalf("unexpected empty output")
	}
}

func TestMcpAllowlistWildcardShouldAllowServerTools(t *testing.T) {
	uc := newTestAgentUsecase()
	fake := &fakeMcpExecutor{}
	uc.SetMcpExecutor(fake)
	registry, _, err := uc.BuildToolRegistryWithOptions(&HarnessToolOptions{
		EnableMcpTool: true,
		McpAllowlist:  []string{"prod:*"},
	})
	if err != nil {
		t.Fatalf("BuildToolRegistryWithOptions failed: %v", err)
	}

	output, err := registry["call_mcp_tool"].Invoke(context.Background(), `{"server":"prod","tool":"weather","arguments":"{}"}`)
	if err != nil {
		t.Fatalf("expected wildcard allowlist to pass, got: %v", err)
	}
	if !fake.called || fake.server != "prod" || fake.tool != "weather" {
		t.Fatalf("mcp executor was not called correctly")
	}
	if output == "" {
		t.Fatalf("unexpected empty output")
	}
}

type fakeToolCallingModel struct {
	streamErr error
}

func (f *fakeToolCallingModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeToolCallingModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	if f.streamErr != nil {
		return nil, f.streamErr
	}
	return schema.StreamReaderFromArray([]*schema.Message{}), nil
}

func (f *fakeToolCallingModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return f, nil
}

type fakeStaticToolCallingModel struct {
	output string
}

func (f *fakeStaticToolCallingModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeStaticToolCallingModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return schema.StreamReaderFromArray([]*schema.Message{
		schema.AssistantMessage(f.output, nil),
	}), nil
}

func (f *fakeStaticToolCallingModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return f, nil
}

func TestExecuteHarnessShouldYieldErrorWhenModelStreamFailed(t *testing.T) {
	uc := newTestAgentUsecase()
	uc.chatModelFunc = func(ctx context.Context, req HarnessRequest) (model.ToolCallingChatModel, error) {
		return &fakeToolCallingModel{streamErr: errors.New("stream init failed")}, nil
	}
	gen := uc.ExecuteStream(context.Background(), "", nil, "hello")
	var gotErr error
	gen(func(msg *StreamMessage, err error) bool {
		if err != nil {
			gotErr = err
			return false
		}
		return true
	})
	if gotErr == nil || !strings.Contains(gotErr.Error(), "模型调用失败") {
		t.Fatalf("expected staged model error, got: %v", gotErr)
	}
	if !strings.Contains(gotErr.Error(), "stream init failed") {
		t.Fatalf("expected redacted detail to retain safe message, got: %v", gotErr)
	}
}

func TestRunSubAgentToolBatchShouldAggregate(t *testing.T) {
	uc := newTestAgentUsecase()
	uc.chatModelFunc = func(ctx context.Context, req HarnessRequest) (model.ToolCallingChatModel, error) {
		return &fakeStaticToolCallingModel{
			output: `{"summary":"子任务已完成","findings":["完成执行"],"next_steps":["继续下一个子任务"]}`,
		}, nil
	}
	tool, err := uc.BuildSubAgentTool()
	if err != nil {
		t.Fatalf("BuildSubAgentTool failed: %v", err)
	}

	parentReq := HarnessRequest{
		Model:           "test-model",
		LlmConfigID:     1,
		LlmModelEntryID: 1,
		ToolOptions: &HarnessToolOptions{
			EnableSubAgentTool: true,
		},
	}
	ctx := withHarnessSubAgentContext(context.Background(), parentReq, HarnessConfig{})
	out, err := tool.Invoke(ctx, `{"sub_tasks_json":"[\"任务A\",\"任务B\"]","max_concurrency":2}`)
	if err != nil {
		t.Fatalf("run_sub_agent batch invoke failed: %v", err)
	}
	var got struct {
		Summary              string   `json:"summary"`
		Findings             []string `json:"findings"`
		TaskCount            int      `json:"task_count"`
		RequestedConcurrency int      `json:"requested_concurrency"`
		EffectiveConcurrency int      `json:"effective_concurrency"`
		ConcurrencyReason    string   `json:"concurrency_reason"`
		SubResults           []struct {
			Task    string `json:"task"`
			Summary string `json:"summary"`
			Error   string `json:"error,omitempty"`
		} `json:"sub_results"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("batch output must be valid json, got err=%v out=%s", err, out)
	}
	if !strings.Contains(got.Summary, "成功 2 / 总计 2") {
		t.Fatalf("unexpected summary: %s", got.Summary)
	}
	if len(got.Findings) != 2 || len(got.SubResults) != 2 {
		t.Fatalf("unexpected aggregate size findings=%d sub_results=%d", len(got.Findings), len(got.SubResults))
	}
	if got.RequestedConcurrency != 2 || got.EffectiveConcurrency != 2 {
		t.Fatalf("unexpected concurrency requested=%d effective=%d", got.RequestedConcurrency, got.EffectiveConcurrency)
	}
	if got.TaskCount != 2 {
		t.Fatalf("unexpected task count: %d", got.TaskCount)
	}
	if got.ConcurrencyReason != "user_specified" {
		t.Fatalf("unexpected concurrency reason: %s", got.ConcurrencyReason)
	}
}

func TestBuildMessagesShouldUseDefaultSystemPrompt(t *testing.T) {
	uc := newTestAgentUsecase()
	msgs := uc.buildMessages(nil, "hello")
	if len(msgs) == 0 || msgs[0].Content != defaultSystemPrompt {
		t.Fatalf("expected default system prompt, got: %#v", msgs)
	}
}

func TestRedactErrorTextShouldStripPathsAndSecrets(t *testing.T) {
	in := "dial tcp: Bearer sk-abcdefghijklmnopqrstuvwxyz0123456789 at /Users/x/proj/foo.go"
	out := redactErrorText(in)
	if strings.Contains(out, "/Users/") || strings.Contains(out, "Bearer sk-") {
		t.Fatalf("expected path/Bearer stripped, got: %q", out)
	}
	if !strings.Contains(out, "[path]") || !strings.Contains(out, "Bearer [redacted]") {
		t.Fatalf("expected redaction markers, got: %q", out)
	}
}

func TestSanitizeExternalErrorShouldCombineStageAndDetail(t *testing.T) {
	err := sanitizeExternalError("model_stream", errors.New("timeout"))
	if err == nil || !strings.Contains(err.Error(), "模型调用失败") || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestSanitizeExternalErrorShouldPassThroughUserFacing(t *testing.T) {
	err := sanitizeExternalError("model_stream", userFacingError("自定义说明"))
	if err == nil || err.Error() != "自定义说明" {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestBuildMessagesShouldUseConfiguredSystemPrompt(t *testing.T) {
	uc := newTestAgentUsecase()
	uc.config = &conf.Bootstrap{
		Agent: &conf.Agent{
			SystemPrompt: "custom system prompt",
		},
	}
	msgs := uc.buildMessages(nil, "hello")
	if len(msgs) == 0 || msgs[0].Content != "custom system prompt" {
		t.Fatalf("expected configured system prompt, got: %#v", msgs)
	}
}

func TestComposeMessagesShouldEmbedAttachmentsIntoLastUserMessage(t *testing.T) {
	uc := newTestAgentUsecase()
	msgs := uc.composeMessages(&HarnessRequest{}, "", nil, "看这个文件", []HarnessAttachment{{
		Filename:      "spec.pdf",
		MimeType:      "application/pdf",
		ContentBase64: base64.StdEncoding.EncodeToString([]byte("pdf")),
	}})

	if len(msgs) < 2 {
		t.Fatalf("expected system and user messages, got %#v", msgs)
	}
	last := msgs[len(msgs)-1]
	if len(last.UserInputMultiContent) == 0 {
		t.Fatalf("expected multimodal user message, got %#v", last)
	}
}
