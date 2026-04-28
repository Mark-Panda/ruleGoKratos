package biz

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rulego/rulego/api/types"
)

func TestResolveRuleChainSkillStatus(t *testing.T) {
	root := t.TempDir()
	meta := RuleChainSkillMeta{
		DirName:       "weather-agent",
		EntryFile:     "SKILL.md",
		Signature:     "sig-current",
		LastGenerated: "2026-04-24T10:30:00Z",
	}

	status, err := ResolveRuleChainSkillStatus(root, meta, "sig-current")
	if err != nil {
		t.Fatalf("ResolveRuleChainSkillStatus returned error: %v", err)
	}
	if status != RuleChainSkillStatusMissing {
		t.Fatalf("expected missing, got %s", status)
	}

	skillDir := filepath.Join(root, "weather-agent")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# ok\n"+BuildRuleChainSkillSignatureAnchor("sig-current")), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	status, err = ResolveRuleChainSkillStatus(root, meta, "sig-current")
	if err != nil {
		t.Fatalf("ResolveRuleChainSkillStatus returned error: %v", err)
	}
	if status != RuleChainSkillStatusReady {
		t.Fatalf("expected ready, got %s", status)
	}

	status, err = ResolveRuleChainSkillStatus(root, RuleChainSkillMeta{
		DirName:       "weather-agent",
		EntryFile:     "SKILL.md",
		Signature:     "sig-next",
		LastGenerated: "2026-04-24T10:31:00Z",
	}, "sig-current")
	if err != nil {
		t.Fatalf("ResolveRuleChainSkillStatus returned error: %v", err)
	}
	if status != RuleChainSkillStatusStale {
		t.Fatalf("expected stale, got %s", status)
	}
}

func TestResolveRuleChainSkillStatusTrimsMetaFields(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "weather-agent")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# ok\n"+BuildRuleChainSkillSignatureAnchor("sig-current")), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	status, err := ResolveRuleChainSkillStatus(root, RuleChainSkillMeta{
		DirName:       " weather-agent ",
		EntryFile:     "  SKILL.md  ",
		Signature:     "sig-current",
		LastGenerated: "2026-04-24T10:30:00Z",
	}, "sig-current")
	if err != nil {
		t.Fatalf("ResolveRuleChainSkillStatus returned error: %v", err)
	}
	if status != RuleChainSkillStatusReady {
		t.Fatalf("expected ready after trimming fields, got %s", status)
	}
}

func TestResolveRuleChainSkillStatusAlwaysAnchorsOnSkillMarkdown(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "weather-agent")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "README.md"), []byte("# readme"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	status, err := ResolveRuleChainSkillStatus(root, RuleChainSkillMeta{
		DirName:   "weather-agent",
		EntryFile: "README.md",
		Signature: "sig-current",
	}, "sig-current")
	if err != nil {
		t.Fatalf("ResolveRuleChainSkillStatus returned error: %v", err)
	}
	if status != RuleChainSkillStatusMissing {
		t.Fatalf("expected missing when only non-SKILL.md exists, got %s", status)
	}

	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# skill\n"+BuildRuleChainSkillSignatureAnchor("sig-current")), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	status, err = ResolveRuleChainSkillStatus(root, RuleChainSkillMeta{
		DirName:   "weather-agent",
		EntryFile: "README.md",
		Signature: "sig-current",
	}, "sig-current")
	if err != nil {
		t.Fatalf("ResolveRuleChainSkillStatus returned error: %v", err)
	}
	if status != RuleChainSkillStatusReady {
		t.Fatalf("expected ready to still anchor on SKILL.md, got %s", status)
	}
}

func TestResolveRuleChainSkillStatusRejectsPathTraversal(t *testing.T) {
	root := t.TempDir()
	outsideDir := filepath.Join(filepath.Dir(root), "outside-skill")
	if err := os.MkdirAll(outsideDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outsideDir, "SKILL.md"), []byte("# outside"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	for _, dirName := range []string{"../outside-skill", outsideDir} {
		status, err := ResolveRuleChainSkillStatus(root, RuleChainSkillMeta{
			DirName:   dirName,
			EntryFile: "SKILL.md",
			Signature: "sig-current",
		}, "sig-current")
		if err != nil {
			t.Fatalf("ResolveRuleChainSkillStatus returned error: %v", err)
		}
		if status != RuleChainSkillStatusMissing {
			t.Fatalf("expected missing for unsafe path %q, got %s", dirName, status)
		}
	}
}

func TestResolveRuleChainSkillStatusDoesNotTreatEmptyFileAsReady(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "weather-agent")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	status, err := ResolveRuleChainSkillStatus(root, RuleChainSkillMeta{
		DirName:   "weather-agent",
		EntryFile: "SKILL.md",
		Signature: "sig-current",
	}, "sig-current")
	if err != nil {
		t.Fatalf("ResolveRuleChainSkillStatus returned error: %v", err)
	}
	if status != RuleChainSkillStatusMissing {
		t.Fatalf("expected empty skill file to be missing, got %s", status)
	}
}

func TestResolveRuleChainSkillStatusRejectsSymlinkedSkillFileOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outsideDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "weather-agent"), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	outsideFile := filepath.Join(outsideDir, "SKILL.md")
	if err := os.WriteFile(outsideFile, []byte("# outside\n"+BuildRuleChainSkillSignatureAnchor("sig-current")), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.Symlink(outsideFile, filepath.Join(root, "weather-agent", "SKILL.md")); err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}

	status, err := ResolveRuleChainSkillStatus(root, RuleChainSkillMeta{
		DirName:   "weather-agent",
		EntryFile: "SKILL.md",
		Signature: "sig-current",
	}, "sig-current")
	if err != nil {
		t.Fatalf("ResolveRuleChainSkillStatus returned error: %v", err)
	}
	if status != RuleChainSkillStatusMissing {
		t.Fatalf("expected missing for symlink escape, got %s", status)
	}
}

func TestReadRuleChainSkillFileRejectsSymlinkedSkillDirOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "SKILL.md")
	if err := os.WriteFile(outsideFile, []byte("# outside"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.Symlink(outsideDir, filepath.Join(root, "weather-agent")); err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}

	_, err := ReadRuleChainSkillFile(root, "weather-agent", "SKILL.md")
	if err == nil {
		t.Fatal("expected symlink escape error")
	}
	if !strings.Contains(err.Error(), "越界") {
		t.Fatalf("expected 越界 error, got %v", err)
	}
}

func TestBuildRuleChainSkillSignatureStable(t *testing.T) {
	left := BuildRuleChainSkillSignature("desc", `[{"name":"city"}]`, `[{"name":"query"}]`, `[{"name":"answer"}]`)
	right := BuildRuleChainSkillSignature("desc", `[{"name":"city"}]`, `[{"name":"query"}]`, `[{"name":"answer"}]`)
	if left != right {
		t.Fatalf("expected stable signature, left=%q right=%q", left, right)
	}
}

func TestBuildRuleChainSkillSignatureAvoidsDelimiterCollision(t *testing.T) {
	left := BuildRuleChainSkillSignature("a", "b\n---\nc", "d", "e")
	right := BuildRuleChainSkillSignature("a\n---\nb", "c", "d", "e")
	if left == right {
		t.Fatalf("expected structured signature serialization to avoid collisions, got %q", left)
	}
}

func TestBuildRuleChainSkillSignatureCanonicalizesJSONParams(t *testing.T) {
	left := BuildRuleChainSkillSignature(
		"desc",
		`[{"type":"string","name":"city"}]`,
		"[ \n  { \"name\": \"query\", \"type\": \"string\" }\n]",
		`{"fields":[{"type":"string","name":"answer"}]}`,
	)
	right := BuildRuleChainSkillSignature(
		"desc",
		`[ { "name" : "city", "type" : "string" } ]`,
		`[{"type":"string","name":"query"}]`,
		`{
		  "fields": [
		    {
		      "name": "answer",
		      "type": "string"
		    }
		  ]
		}`,
	)
	if left != right {
		t.Fatalf("expected JSON formatting differences to produce same signature, left=%q right=%q", left, right)
	}
}

func TestSanitizeRuleChainSkillDirName(t *testing.T) {
	got := SanitizeRuleChainSkillDirName("Weather Agent / Beijing")
	if got != "weather-agent-beijing" {
		t.Fatalf("expected normalized dir name, got %q", got)
	}
}

func TestSanitizeRuleChainSkillDirNameLimitsLength(t *testing.T) {
	got := SanitizeRuleChainSkillDirName(strings.Repeat("Weather Agent ", 10))
	if len(got) > 64 {
		t.Fatalf("expected sanitized dir name length <= 64, got %d (%q)", len(got), got)
	}
}

func TestBuildRuleChainSyncExecutePayloadIncludesMetadata(t *testing.T) {
	payload, err := BuildRuleChainSyncExecutePayload(
		map[string]interface{}{"tenant": "cn"},
		map[string]interface{}{"question": "天气"},
	)
	if err != nil {
		t.Fatalf("BuildRuleChainSyncExecutePayload failed: %v", err)
	}

	meta, _ := payload["metadata"].(map[string]interface{})
	data, _ := payload["data"].(map[string]interface{})
	if meta["tenant"] != "cn" || data["question"] != "天气" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestInjectIdentityMetadataFromContextFillsMissingValues(t *testing.T) {
	ctx := context.WithValue(context.Background(), userIDContextKey, "u-123")
	ctx = context.WithValue(ctx, projectPathContextKey, "/workspace/demo")

	got := injectIdentityMetadataFromContext(ctx, nil)
	if got == nil {
		t.Fatal("expected metadata created")
	}
	if got.GetValue(userIDContextKey) != "u-123" {
		t.Fatalf("expected x-user-id injected, got %q", got.GetValue(userIDContextKey))
	}
	if got.GetValue(projectPathContextKey) != "/workspace/demo" {
		t.Fatalf("expected x-project-path injected, got %q", got.GetValue(projectPathContextKey))
	}
}

func TestInjectIdentityMetadataFromContextKeepsExplicitMetadata(t *testing.T) {
	ctx := context.WithValue(context.Background(), userIDContextKey, "ctx-user")
	ctx = context.WithValue(ctx, projectPathContextKey, "/ctx/project")

	metadata := types.NewMetadata()
	metadata.PutValue(userIDContextKey, "request-user")
	metadata.PutValue(projectPathContextKey, "/request/project")

	got := injectIdentityMetadataFromContext(ctx, metadata)
	if got.GetValue(userIDContextKey) != "request-user" {
		t.Fatalf("expected request x-user-id to win, got %q", got.GetValue(userIDContextKey))
	}
	if got.GetValue(projectPathContextKey) != "/request/project" {
		t.Fatalf("expected request x-project-path to win, got %q", got.GetValue(projectPathContextKey))
	}
}

func TestEnsureIdentityMetadataDefaultsGeneratesWhenMissing(t *testing.T) {
	got := ensureIdentityMetadataDefaults(nil)
	if got == nil {
		t.Fatal("expected metadata created")
	}
	if got.GetValue(userIDContextKey) == "" {
		t.Fatalf("expected fallback %s injected", userIDContextKey)
	}
	if got.GetValue(projectPathContextKey) == "" {
		t.Fatalf("expected fallback %s injected", projectPathContextKey)
	}
	if got.GetValue(sessionIDContextKey) == "" {
		t.Fatalf("expected fallback %s injected", sessionIDContextKey)
	}
}

func TestEnsureIdentityMetadataDefaultsKeepsExplicitValues(t *testing.T) {
	metadata := types.NewMetadata()
	metadata.PutValue(userIDContextKey, "request-user")
	metadata.PutValue(projectPathContextKey, "/request/project")

	got := ensureIdentityMetadataDefaults(metadata)
	if got.GetValue(userIDContextKey) != "request-user" {
		t.Fatalf("expected request x-user-id to win, got %q", got.GetValue(userIDContextKey))
	}
	if got.GetValue(projectPathContextKey) != "/request/project" {
		t.Fatalf("expected request x-project-path to win, got %q", got.GetValue(projectPathContextKey))
	}
	if got.GetValue(sessionIDContextKey) == "" {
		t.Fatalf("expected x-session-id backfilled")
	}
}

func TestBuildRuleChainSkillRequestBodyExampleNoParamsUsesEmptyObjects(t *testing.T) {
	got := BuildRuleChainSkillRequestBodyExample(RuleChainSkillPromptInput{
		RequestMetadataParams: "[]",
		RequestBodyParams:     "[]",
	})
	if got != `{"metadata": {}, "data": {}}` {
		t.Fatalf("expected empty request body example, got %q", got)
	}
}

func TestBuildRuleChainSkillRequestBodyExampleUsesDeclaredParamsOnly(t *testing.T) {
	got := BuildRuleChainSkillRequestBodyExample(RuleChainSkillPromptInput{
		RequestMetadataParams: `[{"name":"tenant"}]`,
		RequestBodyParams:     `[{"name":"city"}]`,
	})
	want := `{"metadata": {"tenant": "cn"}, "data": {"city": "Beijing"}}`
	if got != want {
		t.Fatalf("expected request body example %q, got %q", want, got)
	}
}

func TestExtractRuleChainSkillMarkdownFromHarnessOutput(t *testing.T) {
	raw := "ok\n<generated_skill_markdown>\n---\nname: demo\ndescription: demo\n---\n# body\n</generated_skill_markdown>\nend"
	got := ExtractRuleChainSkillMarkdownFromHarnessOutput(raw)
	if !strings.Contains(got, "name: demo") {
		t.Fatalf("expected markdown block extracted, got %q", got)
	}
}

func TestNormalizeGeneratedRuleChainSkillContentBackfillsColonAnchors(t *testing.T) {
	content := strings.Join([]string{
		"---",
		"name: demo",
		"description: demo",
		"---",
		"### result_explanation",
		"ok",
		"### response_read",
		"read response.data.result",
	}, "\n")
	normalized := NormalizeGeneratedRuleChainSkillContent(content, RuleChainSkillPromptInput{})
	if !strings.Contains(normalized, "result_explanation:") {
		t.Fatalf("expected result_explanation colon anchor, got %q", normalized)
	}
	if !strings.Contains(normalized, "response_read:") {
		t.Fatalf("expected response_read colon anchor, got %q", normalized)
	}
}
