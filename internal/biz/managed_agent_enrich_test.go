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
	profile *ManagedAgentProfile
	mcpList []string
}

func (f *fakeManagedAgentLoader) Load(ctx context.Context, id int64) (*ManagedAgentProfile, error) {
	return f.profile, nil
}

func (f *fakeManagedAgentLoader) ResolveModelEntryForHarness(ctx context.Context, p *ManagedAgentProfile) (configID, entryID int64, err error) {
	return 11, 22, nil
}

func (f *fakeManagedAgentLoader) McpAllowlistStrings(ctx context.Context, mcpIDs []int64) ([]string, error) {
	return append([]string(nil), f.mcpList...), nil
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

func TestEnrichHarnessWithManagedAgentShouldKeepParentToolOptions(t *testing.T) {
	uc := newManagedEnrichTestUsecase(t)
	uc.SetManagedAgentLoader(&fakeManagedAgentLoader{
		profile: &ManagedAgentProfile{
			Enabled:         true,
			SystemPrompt:    "managed prompt",
			SkillPackageIDs: []string{"pkg-a"},
			McpIDs:          []int64{1},
		},
		mcpList: []string{"prod:query"},
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
		ManagedAgentID:    99,
		ToolOptions:       cloneHarnessToolOptions(parentOptions),
		SkillCatalogFilter: &filter,
	}

	out, err := uc.enrichHarnessWithManagedAgent(context.Background(), req)
	if err != nil {
		t.Fatalf("enrichHarnessWithManagedAgent failed: %v", err)
	}
	if !reflect.DeepEqual(out.ToolOptions, req.ToolOptions) {
		t.Fatalf("expected tool options inherited from parent, got %#v", out.ToolOptions)
	}
	if out.SkillCatalogFilter == nil || !reflect.DeepEqual(*out.SkillCatalogFilter, filter) {
		t.Fatalf("expected skill catalog filter inherited from parent, got %#v", out.SkillCatalogFilter)
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
			McpIDs:          []int64{1},
		},
		mcpList: []string{"prod:query"},
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
	if out.SkillCatalogFilter == nil || len(*out.SkillCatalogFilter) == 0 {
		t.Fatalf("expected managed skill catalog filter injected, got %#v", out.SkillCatalogFilter)
	}
}

