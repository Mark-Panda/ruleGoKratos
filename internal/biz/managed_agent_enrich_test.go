package biz

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
)

type fakeManagedAgentLoader struct {
	profile    *ManagedAgentProfile
	allMcpList []string
}

func (f *fakeManagedAgentLoader) Load(ctx context.Context, id int64) (*ManagedAgentProfile, error) {
	return f.profile, nil
}

func (f *fakeManagedAgentLoader) ResolveModelEntryForHarness(ctx context.Context, p *ManagedAgentProfile) (configID, entryID int64, err error) {
	return 11, 22, nil
}

func (f *fakeManagedAgentLoader) EnabledMcpAllowlistStrings(ctx context.Context) ([]string, error) {
	return append([]string(nil), f.allMcpList...), nil
}

func newManagedEnrichTestUsecase(t *testing.T) *AgentUsecase {
	t.Helper()
	helper := log.NewHelper(log.NewStdLogger(io.Discard))
	skillDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(skillDir, "pkg-a"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "pkg-a", "SKILL.md"), []byte("# demo skill"), 0o644); err != nil {
		t.Fatal(err)
	}
	fe, err := NewFileSkillExecutor([]string{skillDir}, FileSkillExecutorOptions{HotReload: false, HotReloadSet: true})
	if err != nil {
		t.Fatalf("NewFileSkillExecutor failed: %v", err)
	}
	return &AgentUsecase{
		log:           helper,
		harnessLogger: NewHarnessLogger(helper),
		skillExecutor: fe,
	}
}

func TestEnrichHarnessWithManagedAgentShouldMergeManagedToolOptions(t *testing.T) {
	uc := newManagedEnrichTestUsecase(t)
	uc.SetManagedAgentLoader(&fakeManagedAgentLoader{
		profile: &ManagedAgentProfile{
			Enabled:         true,
			SystemPrompt:    "managed prompt",
			SkillPackageIDs: []string{"pkg-a"},
		},
		allMcpList: []string{"prod:*", "search:*"},
	})

	parentOptions := &HarnessToolOptions{
		EnableUUIDTool:       true,
		EnableSkillTool:      true,
		EnableMcpTool:        true,
		EnableWorkspaceTools: true,
		EnableSubAgentTool:   true,
		SkillAllowlist:       []string{"custom/skill"},
		McpAllowlist:         []string{"server\x00tool"},
	}
	filter := []string{"custom/skill"}
	req := HarnessRequest{
		ManagedAgentID:     99,
		ToolOptions:        cloneHarnessToolOptions(parentOptions),
		SkillCatalogFilter: &filter,
	}

	out, err := uc.enrichHarnessWithManagedAgent(context.Background(), req)
	if err != nil {
		t.Fatalf("enrichHarnessWithManagedAgent failed: %v", err)
	}
	if out.ToolOptions == nil {
		t.Fatal("expected merged tool options")
	}
	if !out.ToolOptions.EnableSkillTool || !out.ToolOptions.EnableMcpTool {
		t.Fatalf("expected managed tool switches enabled after merge, got %#v", out.ToolOptions)
	}
	if len(out.ToolOptions.SkillAllowlist) != 0 {
		t.Fatalf("expected managed agent to clear skill allowlist restrictions, got %v", out.ToolOptions.SkillAllowlist)
	}
	expectedMcpAllow := []string{"server\x00tool", "prod:*", "search:*"}
	if !reflect.DeepEqual(out.ToolOptions.McpAllowlist, expectedMcpAllow) {
		t.Fatalf("expected merged mcp allowlist %v, got %v", expectedMcpAllow, out.ToolOptions.McpAllowlist)
	}
	if out.SkillCatalogFilter != nil {
		t.Fatalf("expected managed agent to expose full skill catalog, got %#v", out.SkillCatalogFilter)
	}
	if out.LlmConfigID != 11 || out.LlmModelEntryID != 22 {
		t.Fatalf("expected managed llm injected, got config=%d entry=%d", out.LlmConfigID, out.LlmModelEntryID)
	}
}

func TestEnrichHarnessWithManagedAgentShouldInjectToolsWhenMissing(t *testing.T) {
	uc := newManagedEnrichTestUsecase(t)
	uc.SetManagedAgentLoader(&fakeManagedAgentLoader{
		profile: &ManagedAgentProfile{
			Enabled:         true,
			SystemPrompt:    "managed prompt",
			SkillPackageIDs: []string{"pkg-a"},
		},
		allMcpList: []string{"prod:*"},
	})

	out, err := uc.enrichHarnessWithManagedAgent(context.Background(), HarnessRequest{ManagedAgentID: 100})
	if err != nil {
		t.Fatalf("enrichHarnessWithManagedAgent failed: %v", err)
	}
	if out.ToolOptions == nil {
		t.Fatal("expected tool options injected for managed agent")
	}
	if !out.ToolOptions.EnableSkillTool || !out.ToolOptions.EnableMcpTool {
		t.Fatalf("expected skill/mcp tool enabled, got %#v", out.ToolOptions)
	}
	if out.SkillCatalogFilter != nil {
		t.Fatalf("expected managed agent to use full skill catalog, got %#v", out.SkillCatalogFilter)
	}
}

func TestEnrichHarnessWithManagedAgentShouldMergeSkillCreatorAllowlist(t *testing.T) {
	helper := log.NewHelper(log.NewStdLogger(io.Discard))
	skillDir := t.TempDir()
	// 模拟 run_skill 直接使用的 skill_name: skill-creator-0.1.0
	if err := os.WriteFile(filepath.Join(skillDir, "skill-creator-0.1.0.md"), []byte("# skill creator"), 0o644); err != nil {
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
	}
	uc.SetManagedAgentLoader(&fakeManagedAgentLoader{
		profile: &ManagedAgentProfile{
			Enabled:         true,
			SystemPrompt:    "managed prompt",
			SkillPackageIDs: []string{"skill-creator-0.1.0"},
		},
	})

	req := HarnessRequest{
		ManagedAgentID: 200,
		ToolOptions: &HarnessToolOptions{
			EnableUUIDTool:       true,
			EnableSkillTool:      false,
			EnableMcpTool:        false,
			EnableWorkspaceTools: true,
			EnableSubAgentTool:   true,
			SkillAllowlist:       []string{"custom/skill"},
		},
	}
	out, err := uc.enrichHarnessWithManagedAgent(context.Background(), req)
	if err != nil {
		t.Fatalf("enrichHarnessWithManagedAgent failed: %v", err)
	}
	if out.ToolOptions == nil {
		t.Fatal("expected merged tool options")
	}
	if !out.ToolOptions.EnableSkillTool {
		t.Fatalf("expected EnableSkillTool to be true after managed merge, got %#v", out.ToolOptions)
	}
	if len(out.ToolOptions.SkillAllowlist) != 0 {
		t.Fatalf("expected managed agent to clear skill allowlist restrictions, got %v", out.ToolOptions.SkillAllowlist)
	}
}
