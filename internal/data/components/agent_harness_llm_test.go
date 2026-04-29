package data

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"ruleGoKratos/internal/biz"
)

func TestExtractHarnessAttachmentsShouldReadArrayFromMessageData(t *testing.T) {
	raw := `{"attachments":[{"filename":"demo.png","mimeType":"image/png","contentBase64":"YWJj"}]}`

	got := extractHarnessAttachments(raw, nil)
	want := []biz.HarnessAttachment{{
		Filename:      "demo.png",
		MimeType:      "image/png",
		ContentBase64: "YWJj",
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestExtractHarnessAttachmentsShouldFallbackToMetadata(t *testing.T) {
	env := map[string]interface{}{
		"metadata": map[string]interface{}{
			"attachments": []interface{}{
				map[string]interface{}{
					"filename": "clip.mp4",
					"mimeType": "video/mp4",
				},
			},
		},
	}

	got := extractHarnessAttachments("", env)
	want := []biz.HarnessAttachment{{
		Filename: "clip.mp4",
		MimeType: "video/mp4",
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestAgentHarnessShouldNotExposeMcpNodeConfigFields(t *testing.T) {
	cfgType := reflect.TypeOf(AgentHarnessLLMConfig{})
	if _, ok := cfgType.FieldByName("EnableMcpTool"); ok {
		t.Fatal("AgentHarnessLLMConfig must not expose EnableMcpTool")
	}
	nodeType := reflect.TypeOf(AgentHarnessLLM{})
	if _, ok := nodeType.FieldByName("mcpAllow"); ok {
		t.Fatal("AgentHarnessLLM must not keep node-level MCP allowlist state")
	}
}

func TestResolveWorkspaceRootForComponentShouldReadWorkspaceRootDir(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})
	workspaceID := "ws-test"
	root := filepath.Join(tmp, "actual-root")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "code_workspace"), 0o755); err != nil {
		t.Fatalf("mkdir code_workspace failed: %v", err)
	}
	content := `{"ruleGoWorkspace":{"name":"demo","rootDir":"` + root + `","repositories":[]}}`
	if err := os.WriteFile(filepath.Join(tmp, "code_workspace", workspaceID+".code-workspace"), []byte(content), 0o644); err != nil {
		t.Fatalf("write workspace config failed: %v", err)
	}
	got := resolveWorkspaceRootForComponent(workspaceID)
	if filepath.Clean(got) != filepath.Clean(root) {
		t.Fatalf("workspace root mismatch: got=%s want=%s", got, root)
	}
}
