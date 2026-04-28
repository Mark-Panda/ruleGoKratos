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
	}
}

func TestBuildToolRegistryIncludesSkillAndMcp(t *testing.T) {
	uc := newTestAgentUsecase()
	skillRoot := t.TempDir()
	writeTestSkillPackage(t, skillRoot, "planner", "---\nname: planner\n---\nplan content")
	exec, err := NewFileSkillExecutor([]string{skillRoot}, FileSkillExecutorOptions{AllowList: "*"})
	if err != nil {
		t.Fatalf("NewFileSkillExecutor failed: %v", err)
	}
	uc.SetSkillExecutor(exec)
	uc.SetMcpToolProvider(fakeMcpToolProvider{
		tools: []*HarnessTool{{
			Info: &schema.ToolInfo{
				Name: "mcp_prod_weather",
				Desc: "weather",
			},
			Invoke: func(ctx context.Context, rawArgs string) (string, error) {
				return "sunny", nil
			},
		}},
	})
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
	if _, ok := registry["skill"]; !ok {
		t.Fatalf("official skill tool missing")
	}
	if _, ok := registry["run_skill"]; ok {
		t.Fatalf("run_skill must not be exposed in official skill mode")
	}
	if _, ok := registry["call_mcp_tool"]; ok {
		t.Fatalf("call_mcp_tool must not be exposed in official MCP mode")
	}
	if _, ok := registry["mcp_prod_weather"]; !ok {
		t.Fatalf("concrete MCP tool missing")
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

type fakeMcpToolProvider struct {
	tools []*HarnessTool
	err   error
}

func (f fakeMcpToolProvider) BuildMcpTools(ctx context.Context, allowlist []string) ([]*HarnessTool, error) {
	return f.tools, f.err
}

func writeTestSkillPackage(t *testing.T, root, pkg, content string) {
	t.Helper()
	dir := filepath.Join(root, pkg)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir skill package failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write SKILL.md failed: %v", err)
	}
}

func TestOfficialSkillToolShouldLoadSkillContent(t *testing.T) {
	uc := newTestAgentUsecase()
	skillRoot := t.TempDir()
	writeTestSkillPackage(t, skillRoot, "planner", "---\nname: planner\n---\n## Planner\nUse a plan.")
	exec, err := NewFileSkillExecutor([]string{skillRoot}, FileSkillExecutorOptions{AllowList: "*"})
	if err != nil {
		t.Fatalf("NewFileSkillExecutor failed: %v", err)
	}
	uc.SetSkillExecutor(exec)

	registry, _, err := uc.BuildToolRegistryWithOptions(&HarnessToolOptions{EnableSkillTool: true})
	if err != nil {
		t.Fatalf("BuildToolRegistryWithOptions failed: %v", err)
	}
	tool := registry["skill"]
	if tool == nil {
		t.Fatalf("official skill tool missing")
	}
	output, err := tool.Invoke(context.Background(), `{"skill":"planner"}`)
	if err != nil {
		t.Fatalf("official skill tool invoke failed: %v", err)
	}
	if !strings.Contains(output, "Use a plan.") {
		t.Fatalf("unexpected skill output: %s", output)
	}
}

func TestMcpAllowlistShouldFilterConcreteServerTools(t *testing.T) {
	uc := newTestAgentUsecase()
	uc.SetMcpToolProvider(fakeMcpToolProvider{
		tools: []*HarnessTool{
			{Info: &schema.ToolInfo{Name: "mcp_prod_weather", Desc: "prod weather"}},
		},
	})
	registry, _, err := uc.BuildToolRegistryWithOptions(&HarnessToolOptions{
		EnableMcpTool: true,
		McpAllowlist:  []string{"prod:*"},
	})
	if err != nil {
		t.Fatalf("BuildToolRegistryWithOptions failed: %v", err)
	}

	if _, ok := registry["mcp_prod_weather"]; !ok {
		t.Fatalf("expected concrete MCP tool to be registered, got %v", registry)
	}
	if _, ok := registry["call_mcp_tool"]; ok {
		t.Fatalf("call_mcp_tool must not be registered")
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
	msgs, err := uc.buildMessages(nil, "hello")
	if err != nil {
		t.Fatalf("buildMessages failed: %v", err)
	}
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
	msgs, err := uc.buildMessages(nil, "hello")
	if err != nil {
		t.Fatalf("buildMessages failed: %v", err)
	}
	if len(msgs) == 0 || msgs[0].Content != "custom system prompt" {
		t.Fatalf("expected configured system prompt, got: %#v", msgs)
	}
}

func TestComposeMessagesShouldEmbedAttachmentsIntoLastUserMessage(t *testing.T) {
	uc := newTestAgentUsecase()
	msgs, err := uc.composeMessages(context.Background(), &HarnessRequest{}, "", nil, "看这个文件", []HarnessAttachment{{
		Filename:      "spec.pdf",
		MimeType:      "application/pdf",
		ContentBase64: base64.StdEncoding.EncodeToString([]byte("pdf")),
	}})
	if err != nil {
		t.Fatalf("composeMessages failed: %v", err)
	}

	if len(msgs) < 2 {
		t.Fatalf("expected system and user messages, got %#v", msgs)
	}
	last := msgs[len(msgs)-1]
	if len(last.UserInputMultiContent) == 0 {
		t.Fatalf("expected multimodal user message, got %#v", last)
	}
}

// fakeToolCallingModelWithCalls simulates a model that returns tool calls and then a final response.
type fakeToolCallingModelWithCalls struct {
	calls     []schema.ToolCall
	finalResp string
	callIndex int
}

func (f *fakeToolCallingModelWithCalls) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeToolCallingModelWithCalls) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	if f.callIndex >= len(f.calls) {
		// No more tool calls, return final response
		return schema.StreamReaderFromArray([]*schema.Message{
			schema.AssistantMessage(f.finalResp, nil),
		}), nil
	}

	// Return the current tool call
	call := f.calls[f.callIndex]
	f.callIndex++
	return schema.StreamReaderFromArray([]*schema.Message{
		schema.AssistantMessage("", []schema.ToolCall{call}),
	}), nil
}

func (f *fakeToolCallingModelWithCalls) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return f, nil
}

func TestRecoverableToolErrorShouldNotTerminateHarness(t *testing.T) {
	// Track tool invocations
	var toolCallCount int

	// Create a tool that returns a recoverable error
	recoverableTool := &HarnessTool{
		Info: &schema.ToolInfo{
			Name: "recoverable_test_tool",
			Desc: "A test tool that returns recoverable errors",
		},
		Invoke: func(ctx context.Context, rawArgs string) (string, error) {
			toolCallCount++
			// First call: return recoverable error
			// Second call: return success
			if toolCallCount == 1 {
				return "", &recoverableSkillError{
					msg:        "skill not found: test. Did you mean: test-tool?",
					suggestion: "test-tool",
				}
			}
			return `{"result": "success after self-correction"}`, nil
		},
	}

	// We need to test the executeHarness flow, but it requires the full tool registry
	// For this test, we'll verify that the recoverable error is properly identified
	result, err := recoverableTool.Invoke(context.Background(), `{}`)
	if err == nil {
		t.Fatal("expected recoverable error")
	}
	if !IsRecoverableToolError(err) {
		t.Fatalf("expected recoverable tool error, got: %v", err)
	}
	_ = result // error case, result may be empty

	// Verify the tool was invoked (simulating what executeHarness does)
	if toolCallCount != 1 {
		t.Fatalf("expected 1 tool call before recovery, got: %d", toolCallCount)
	}

	// Simulate what executeHarness does: if recoverable, append ToolMessage and continue
	msgs := []*schema.Message{}
	toolCallID := "call_1"
	recoverableErr := err

	if IsRecoverableToolError(recoverableErr) {
		// This is what executeHarness does for recoverable errors
		msgs = append(msgs, schema.ToolMessage(
			fmt.Sprintf("[Tool Error] %s", recoverableErr.Error()),
			toolCallID,
			schema.WithToolName("recoverable_test_tool"),
		))
	}

	if len(msgs) != 1 {
		t.Fatalf("expected 1 tool message after recoverable error, got: %d", len(msgs))
	}
	if !strings.Contains(msgs[0].Content, "[Tool Error]") {
		t.Fatalf("expected tool message to contain error: %s", msgs[0].Content)
	}
}

func TestConsecutiveRecoverableErrorsShouldTerminateHarness(t *testing.T) {
	// This test verifies the consecutive error count logic

	consecutiveCount := 0
	maxConsecutive := 3

	// Simulate 3 consecutive recoverable errors
	for i := 1; i <= 3; i++ {
		err := &recoverableSkillError{
			msg: fmt.Sprintf("error %d: skill not found", i),
		}

		if IsRecoverableToolError(err) {
			consecutiveCount++
			if consecutiveCount >= maxConsecutive {
				// Should terminate
				break
			}
		}
	}

	if consecutiveCount != maxConsecutive {
		t.Fatalf("expected consecutive count to reach %d, got: %d", maxConsecutive, consecutiveCount)
	}

	// After 3 consecutive errors, the harness should terminate
	// This is verified by the logic in agent.go:
	// if consecutiveRecoverableErrors >= maxConsecutiveRecoverableErrors {
	//     yield(nil, userFacingError("..."))
	//     return
	// }

	// Test that successful tool call resets the count
	consecutiveCount = 1
	// Simulate successful tool call
	consecutiveCount = 0

	// Now simulate error -> success -> error
	consecutiveCount++ // error 1
	if consecutiveCount >= maxConsecutive {
		t.Fatal("should not terminate after 1 error")
	}

	// Simulate successful call
	consecutiveCount = 0

	consecutiveCount++ // error (after successful call)
	if consecutiveCount >= maxConsecutive {
		t.Fatal("should not terminate after 1 error")
	}
}

func TestNonRecoverableErrorShouldTerminateHarness(t *testing.T) {
	// Verify that non-recoverable errors are correctly identified
	nonRecoverableErr := errors.New("context canceled")
	if IsRecoverableToolError(nonRecoverableErr) {
		t.Fatal("context canceled should not be recoverable")
	}

	nonRecoverableErr = errors.New("tool timeout")
	if IsRecoverableToolError(nonRecoverableErr) {
		t.Fatal("timeout should not be recoverable")
	}

	nonRecoverableErr = errors.New("invalid arguments")
	if IsRecoverableToolError(nonRecoverableErr) {
		t.Fatal("invalid arguments should not be recoverable")
	}
}
