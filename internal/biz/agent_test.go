package biz

import (
	"context"
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
	if len(registry) != 6 {
		t.Fatalf("unexpected tool registry size: %d", len(registry))
	}
	if len(infos) != 6 {
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
