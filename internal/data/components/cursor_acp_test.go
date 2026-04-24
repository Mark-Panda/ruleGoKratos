package data

import (
	"bufio"
	"strings"
	"testing"

	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/utils/el"
)

func TestCursorAcpNew_DefaultForceEnabled(t *testing.T) {
	node := (&CursorAcpDsl{}).New()
	dsl, ok := node.(*CursorAcpDsl)
	if !ok {
		t.Fatalf("unexpected node type: %T", node)
	}
	if !dsl.Config.Force {
		t.Fatalf("expected default force=true, got false")
	}
	if len(dsl.Config.Args) == 0 || dsl.Config.Args[0] != "acp" {
		t.Fatalf("expected default args start with acp, got: %v", dsl.Config.Args)
	}
}

func TestCursorAcpInit_DefaultForceWhenMissing(t *testing.T) {
	var dsl CursorAcpDsl
	cfg := types.Configuration{
		"agentPath": "agent",
		"args":      []string{"acp"},
	}
	if err := dsl.Init(types.Config{}, cfg); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	if !dsl.Config.Force {
		t.Fatalf("expected force=true when field missing")
	}
}

func TestCursorAcpInit_ForceFalseHonored(t *testing.T) {
	var dsl CursorAcpDsl
	cfg := types.Configuration{
		"agentPath": "agent",
		"args":      []string{"acp"},
		"force":     false,
	}
	if err := dsl.Init(types.Config{}, cfg); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	if dsl.Config.Force {
		t.Fatalf("expected force=false when explicitly configured")
	}
}

func TestCursorAcpInit_InvalidArgsHead(t *testing.T) {
	var dsl CursorAcpDsl
	cfg := types.Configuration{
		"agentPath": "agent",
		"args":      []string{"status"},
	}
	err := dsl.Init(types.Config{}, cfg)
	if err == nil || !strings.Contains(err.Error(), "args 首项必须为 acp") {
		t.Fatalf("expected args head validation error, got: %v", err)
	}
}

func TestReadJSONRPCResult_SkipNotification(t *testing.T) {
	raw := strings.Join([]string{
		`{"jsonrpc":"2.0","method":"session/update","params":{"msg":"ignore"}}`,
		`{"jsonrpc":"2.0","id":2,"result":{"ok":true}}`,
	}, "\n")
	br := bufio.NewReader(strings.NewReader(raw))
	res, err := readJSONRPCResult(br, 2)
	if err != nil {
		t.Fatalf("readJSONRPCResult unexpected error: %v", err)
	}
	if !strings.Contains(string(res), `"ok":true`) {
		t.Fatalf("unexpected rpc result: %s", string(res))
	}
}

func TestReadJSONRPCResult_RPCError(t *testing.T) {
	raw := `{"jsonrpc":"2.0","id":3,"error":{"code":-1,"message":"boom"}}`
	br := bufio.NewReader(strings.NewReader(raw))
	_, err := readJSONRPCResult(br, 3)
	if err == nil || !strings.Contains(err.Error(), "rpc error") {
		t.Fatalf("expected rpc error, got: %v", err)
	}
}

func TestCursorAcpBuildAgentArgv_InjectForceWhenEnabled(t *testing.T) {
	t.Setenv("HOME", "/tmp/test-home")
	wsTpl, err := el.NewTemplate("$HOME/repo")
	if err != nil {
		t.Fatalf("new template: %v", err)
	}
	args := buildAgentArgv(map[string]interface{}{}, wsTpl, false, true, []string{"acp"})
	if !containsArg(args, "--force") {
		t.Fatalf("expected --force injected when force=true, got: %v", args)
	}
}

func TestCursorAcpBuildAgentArgv_NoForceWhenDisabled(t *testing.T) {
	t.Setenv("HOME", "/tmp/test-home")
	wsTpl, err := el.NewTemplate("$HOME/repo")
	if err != nil {
		t.Fatalf("new template: %v", err)
	}
	args := buildAgentArgv(map[string]interface{}{}, wsTpl, false, false, []string{"acp"})
	if containsArg(args, "--force") {
		t.Fatalf("expected no --force when force=false, got: %v", args)
	}
}

