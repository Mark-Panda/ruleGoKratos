package biz

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileSkillExecutorLoadAndExecute(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "planner.md"), []byte("plan: {{payload}}"), 0o644)
	if err != nil {
		t.Fatalf("write skill file failed: %v", err)
	}

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

func TestFileSkillExecutorNotFound(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("content"), 0o644)
	if err != nil {
		t.Fatalf("write skill file failed: %v", err)
	}

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
	skillPath := filepath.Join(dir, "planner.md")
	err := os.WriteFile(skillPath, []byte("version1"), 0o644)
	if err != nil {
		t.Fatalf("write skill file failed: %v", err)
	}
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
	skillPath := filepath.Join(dir, "planner.md")
	err := os.WriteFile(skillPath, []byte("version1"), 0o644)
	if err != nil {
		t.Fatalf("write skill file failed: %v", err)
	}
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
	err := os.WriteFile(filepath.Join(dir, "planner.md"), []byte("planner-content"), 0o644)
	if err != nil {
		t.Fatalf("write planner file failed: %v", err)
	}
	err = os.WriteFile(filepath.Join(dir, "other.md"), []byte("other-content"), 0o644)
	if err != nil {
		t.Fatalf("write other file failed: %v", err)
	}
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

func TestFileSkillExecutorProvidesPackageAliasForSkillMarkdown(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "skill-creator-0.1.0"), 0o755); err != nil {
		t.Fatalf("mkdir skill package failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skill-creator-0.1.0", "SKILL.md"), []byte("creator: {{payload}}"), 0o644); err != nil {
		t.Fatalf("write SKILL.md failed: %v", err)
	}

	exec, err := NewFileSkillExecutor([]string{dir}, FileSkillExecutorOptions{HotReload: false, HotReloadSet: true})
	if err != nil {
		t.Fatalf("NewFileSkillExecutor failed: %v", err)
	}

	byAlias, err := exec.Execute(context.Background(), "skill-creator-0.1.0", "demo")
	if err != nil {
		t.Fatalf("expected package alias executable, got err=%v", err)
	}
	if byAlias != "creator: demo" {
		t.Fatalf("unexpected alias output: %q", byAlias)
	}

	byFileName, err := exec.Execute(context.Background(), "skill-creator-0.1.0/SKILL", "demo")
	if err != nil {
		t.Fatalf("expected file key executable, got err=%v", err)
	}
	if byFileName != "creator: demo" {
		t.Fatalf("unexpected file key output: %q", byFileName)
	}
}
