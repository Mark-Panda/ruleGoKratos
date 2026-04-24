package data

import (
	"testing"

	"github.com/rulego/rulego/utils/el"
)

func TestCursorAgentStatusLooksAuthed(t *testing.T) {
	cases := []struct {
		raw  string
		want bool
	}{
		{"", false},
		{"   ", false},
		{"Not logged in", false},
		{"Error: not logged in\n", false},
		{"✓ Login successful!\nLogged in (unable to fetch user details)", true},
		{"Login successful", true},
		{"Logged in as user@example.com", true},
		{"Logged in (unable to fetch user details)", true},
		{"something else", false},
	}
	for _, tc := range cases {
		if got := cursorAgentStatusLooksAuthed(tc.raw); got != tc.want {
			t.Errorf("cursorAgentStatusLooksAuthed(%q) = %v, want %v", tc.raw, got, tc.want)
		}
	}
}

func TestBuildAgentArgv_DefaultInjectTrustAndWorkspace(t *testing.T) {
	t.Setenv("HOME", "/tmp/test-home")
	wsTpl, err := el.NewTemplate("")
	if err != nil {
		t.Fatalf("new template: %v", err)
	}
	got := buildAgentArgv(map[string]interface{}{}, wsTpl, false, true, []string{"-p", "hello"})

	if len(got) < 3 {
		t.Fatalf("unexpected argv length: %v", got)
	}
	if got[0] != "--workspace" || got[1] != "/tmp/test-home" {
		t.Fatalf("workspace arg mismatch: %v", got)
	}
	if !containsArg(got, "--trust") {
		t.Fatalf("expected default --trust in argv: %v", got)
	}
	if !containsArg(got, "--force") {
		t.Fatalf("expected default --force in argv: %v", got)
	}
}

func TestBuildAgentArgv_DoNotDuplicateExplicitTrust(t *testing.T) {
	t.Setenv("HOME", "/tmp/test-home")
	wsTpl, err := el.NewTemplate("$HOME/repo")
	if err != nil {
		t.Fatalf("new template: %v", err)
	}
	got := buildAgentArgv(map[string]interface{}{}, wsTpl, false, true, []string{"--trust", "-p", "hello"})

	if countArg(got, "--trust") != 1 {
		t.Fatalf("expected exactly one --trust, got argv: %v", got)
	}
}

func TestBuildAgentArgv_DoNotDuplicateExplicitForce(t *testing.T) {
	t.Setenv("HOME", "/tmp/test-home")
	wsTpl, err := el.NewTemplate("$HOME/repo")
	if err != nil {
		t.Fatalf("new template: %v", err)
	}
	got := buildAgentArgv(map[string]interface{}{}, wsTpl, false, true, []string{"--force", "-p", "hello"})
	if countArg(got, "--force") != 1 {
		t.Fatalf("expected exactly one --force, got argv: %v", got)
	}
	gotYolo := buildAgentArgv(map[string]interface{}{}, wsTpl, false, true, []string{"--yolo", "-p", "hello"})
	if countArg(gotYolo, "--force") != 0 {
		t.Fatalf("expected no injected --force when --yolo is present, got argv: %v", gotYolo)
	}
}

func TestBuildAgentArgv_ForceDisabled(t *testing.T) {
	t.Setenv("HOME", "/tmp/test-home")
	wsTpl, err := el.NewTemplate("$HOME/repo")
	if err != nil {
		t.Fatalf("new template: %v", err)
	}
	got := buildAgentArgv(map[string]interface{}{}, wsTpl, false, false, []string{"-p", "hello"})
	if containsArg(got, "--force") {
		t.Fatalf("expected no --force when force disabled, got argv: %v", got)
	}
}

func TestBuildAgentArgv_WorktreeAndTrustOrderStable(t *testing.T) {
	t.Setenv("HOME", "/tmp/test-home")
	wsTpl, err := el.NewTemplate("$HOME/repo")
	if err != nil {
		t.Fatalf("new template: %v", err)
	}
	userArgs := []string{"-p", "hello", "--model", "gpt-5.4"}
	got := buildAgentArgv(map[string]interface{}{}, wsTpl, true, true, userArgs)

	wantPrefix := []string{"--workspace", "/tmp/test-home/repo", "--worktree", "--trust", "--force"}
	if len(got) < len(wantPrefix) {
		t.Fatalf("argv too short: %v", got)
	}
	for i, want := range wantPrefix {
		if got[i] != want {
			t.Fatalf("argv order mismatch at %d: got=%q want=%q full=%v", i, got[i], want, got)
		}
	}
	if got[5] != "-p" || got[6] != "hello" {
		t.Fatalf("expected user args preserved after prefix, got: %v", got)
	}
}

func TestResolveWorkspacePath(t *testing.T) {
	t.Setenv("HOME", "/tmp/test-home")
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty_fallback_home", in: "", want: "/tmp/test-home"},
		{name: "env_expand", in: "$HOME/repo", want: "/tmp/test-home/repo"},
		{name: "tilde", in: "~", want: "/tmp/test-home"},
		{name: "tilde_subdir", in: "~/repo", want: "/tmp/test-home/repo"},
		{name: "literal_path", in: "/data/repo", want: "/data/repo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveWorkspacePath(tc.in); got != tc.want {
				t.Fatalf("resolveWorkspacePath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func containsArg(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}

func countArg(args []string, want string) int {
	n := 0
	for _, a := range args {
		if a == want {
			n++
		}
	}
	return n
}
