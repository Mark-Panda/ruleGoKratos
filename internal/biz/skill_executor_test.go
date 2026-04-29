package biz

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSkillPackage(t *testing.T, root, pkg, content string) string {
	t.Helper()
	dir := filepath.Join(root, pkg)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir skill package failed: %v", err)
	}
	path := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write SKILL.md failed: %v", err)
	}
	return path
}

func TestFileSkillExecutorLoadAndExecute(t *testing.T) {
	dir := t.TempDir()
	writeSkillPackage(t, dir, "planner", "plan: {{payload}}")

	exec, err := NewFileSkillExecutor([]string{dir}, FileSkillExecutorOptions{HotReload: false, HotReloadSet: true})
	if err != nil {
		t.Fatalf("NewFileSkillExecutor failed: %v", err)
	}

	output, err := exec.Execute(context.Background(), "planner", "demo-task")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if output != "plan: demo-task" {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestFileSkillExecutorKeepsFirstDuplicateSkillName(t *testing.T) {
	appDir := t.TempDir()
	agentDir := t.TempDir()
	workflowDir := t.TempDir()
	for _, item := range []struct {
		dir     string
		content string
	}{
		{workflowDir, "workflow"},
		{agentDir, "agent"},
		{appDir, "app"},
	} {
		writeSkillPackage(t, item.dir, "shared", item.content)
	}

	exec, err := NewFileSkillExecutor([]string{appDir, agentDir, workflowDir}, FileSkillExecutorOptions{HotReload: false, HotReloadSet: true})
	if err != nil {
		t.Fatalf("NewFileSkillExecutor failed: %v", err)
	}
	output, err := exec.Execute(context.Background(), "shared", "")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if output != "app" {
		t.Fatalf("expected first directory duplicate to win, got %q", output)
	}
}

func TestDefaultSkillDirsUsesServiceRootsInPriorityOrder(t *testing.T) {
	t.Setenv("APP_SKILL_DIR", "/custom/app")
	t.Setenv("WORKFLOW_SKILL_DIR", "/custom/workflow")

	got := defaultSkillDirs("/custom/agent", "/ignored/extra")
	want := []string{
		"/custom/app",
		"/custom/agent",
		"/custom/workflow",
		"/root/.agents/skills",
		"/root/.claude/skills",
		"/root/.cursor/skills",
	}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestFileSkillExecutorNotFound(t *testing.T) {
	dir := t.TempDir()
	writeSkillPackage(t, dir, "a", "content")

	exec, err := NewFileSkillExecutor([]string{dir}, FileSkillExecutorOptions{HotReload: false, HotReloadSet: true})
	if err != nil {
		t.Fatalf("NewFileSkillExecutor failed: %v", err)
	}

	_, err = exec.Execute(context.Background(), "missing", "")
	if err == nil || !strings.Contains(err.Error(), "skill不存在") {
		t.Fatalf("expected missing skill error, got: %v", err)
	}
}

func TestFileSkillExecutorAllowEmptyDirectories(t *testing.T) {
	exec, err := NewFileSkillExecutor([]string{filepath.Join(t.TempDir(), "not-exist")}, FileSkillExecutorOptions{HotReload: false, HotReloadSet: true})
	if err != nil {
		t.Fatalf("NewFileSkillExecutor should not fail on empty dirs: %v", err)
	}
	_, err = exec.Execute(context.Background(), "planner", "")
	if err == nil || !strings.Contains(err.Error(), "暂无可用技能") {
		t.Fatalf("expected empty skill directory error, got: %v", err)
	}
}

func TestFileSkillExecutorHotReload(t *testing.T) {
	dir := t.TempDir()
	skillPath := writeSkillPackage(t, dir, "planner", "version1")
	exec, err := NewFileSkillExecutor([]string{dir}, FileSkillExecutorOptions{
		HotReload:         true,
		HotReloadSet:      true,
		ScanIntervalMS:    0,
		ScanIntervalMSSet: true,
	})
	if err != nil {
		t.Fatalf("NewFileSkillExecutor failed: %v", err)
	}
	first, err := exec.Execute(context.Background(), "planner", "")
	if err != nil || first != "version1" {
		t.Fatalf("unexpected first execute result: %q, err=%v", first, err)
	}
	err = os.WriteFile(skillPath, []byte("version2"), 0o644)
	if err != nil {
		t.Fatalf("rewrite skill file failed: %v", err)
	}
	second, err := exec.Execute(context.Background(), "planner", "")
	if err != nil || second != "version2" {
		t.Fatalf("unexpected second execute result: %q, err=%v", second, err)
	}
}

func TestFileSkillExecutorHotReloadToEmpty(t *testing.T) {
	dir := t.TempDir()
	skillPath := writeSkillPackage(t, dir, "planner", "version1")
	exec, err := NewFileSkillExecutor([]string{dir}, FileSkillExecutorOptions{
		HotReload:         true,
		HotReloadSet:      true,
		ScanIntervalMS:    0,
		ScanIntervalMSSet: true,
	})
	if err != nil {
		t.Fatalf("NewFileSkillExecutor failed: %v", err)
	}
	first, err := exec.Execute(context.Background(), "planner", "")
	if err != nil || first != "version1" {
		t.Fatalf("unexpected first execute result: %q, err=%v", first, err)
	}
	if err := os.Remove(skillPath); err != nil {
		t.Fatalf("remove skill file failed: %v", err)
	}
	_, err = exec.Execute(context.Background(), "planner", "")
	if err == nil || !strings.Contains(err.Error(), "暂无可用技能") {
		t.Fatalf("expected empty skill directory error after reload, got: %v", err)
	}
}

func TestFileSkillExecutorNamespaceAndAllowList(t *testing.T) {
	dir := t.TempDir()
	writeSkillPackage(t, dir, "planner", "planner-content")
	writeSkillPackage(t, dir, "other", "other-content")
	exec, err := NewFileSkillExecutor([]string{dir}, FileSkillExecutorOptions{
		Namespace:    "teamA",
		AllowList:    "planner",
		HotReload:    false,
		HotReloadSet: true,
	})
	if err != nil {
		t.Fatalf("NewFileSkillExecutor failed: %v", err)
	}
	okOutput, err := exec.Execute(context.Background(), "planner", "")
	if err != nil || okOutput != "planner-content" {
		t.Fatalf("expected allowed planner skill, output=%q, err=%v", okOutput, err)
	}
	_, err = exec.Execute(context.Background(), "other", "")
	if err == nil || !strings.Contains(err.Error(), "无权限") {
		t.Fatalf("expected allowlist denial, got: %v", err)
	}
}

func TestFileSkillExecutorUsesFrontMatterName(t *testing.T) {
	dir := t.TempDir()
	writeSkillPackage(t, dir, "package-dir", "---\nname: canonical-skill\n---\ncreator: {{payload}}")

	exec, err := NewFileSkillExecutor([]string{dir}, FileSkillExecutorOptions{HotReload: false, HotReloadSet: true})
	if err != nil {
		t.Fatalf("NewFileSkillExecutor failed: %v", err)
	}

	output, err := exec.Execute(context.Background(), "canonical-skill", "demo")
	if err != nil {
		t.Fatalf("expected frontmatter skill name executable, got err=%v", err)
	}
	if output != "---\nname: canonical-skill\n---\ncreator: demo" {
		t.Fatalf("unexpected output: %q", output)
	}

	_, err = exec.Execute(context.Background(), "package-dir", "demo")
	if err == nil || !strings.Contains(err.Error(), "skill不存在") {
		t.Fatalf("expected package dir not to be exposed when frontmatter name exists, got: %v", err)
	}
}

func TestFileSkillExecutorIgnoresLooseFilesAndReferenceFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "loose.md"), []byte("loose"), 0o644); err != nil {
		t.Fatalf("write loose file failed: %v", err)
	}
	writeSkillPackage(t, dir, "pkg", "package")
	refDir := filepath.Join(dir, "pkg", "reference")
	if err := os.MkdirAll(refDir, 0o755); err != nil {
		t.Fatalf("mkdir reference dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(refDir, "guide.md"), []byte("guide"), 0o644); err != nil {
		t.Fatalf("write reference file failed: %v", err)
	}

	exec, err := NewFileSkillExecutor([]string{dir}, FileSkillExecutorOptions{HotReload: false, HotReloadSet: true})
	if err != nil {
		t.Fatalf("NewFileSkillExecutor failed: %v", err)
	}
	names := exec.ListAvailableSkillNames()
	if len(names) != 1 || names[0] != "pkg" {
		t.Fatalf("expected only package skill, got %v", names)
	}
	for _, name := range []string{"loose", "pkg/SKILL", "pkg/reference/guide"} {
		if _, err := exec.Execute(context.Background(), name, ""); err == nil {
			t.Fatalf("expected %s not to be exposed", name)
		}
	}
}
