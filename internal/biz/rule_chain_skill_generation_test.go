package biz

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz/entity"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestBuildRuleChainSkillGenerationPromptIncludesExecuteContract(t *testing.T) {
	prompt := BuildRuleChainSkillGenerationPrompt(RuleChainSkillPromptInput{
		RuleChainID:           "root-chain",
		RuleChainName:         "weather-agent",
		DirName:               "weather-agent",
		SkillRoot:             "/app/skills",
		MsgType:               "QUERY",
		Description:           "根据城市查询天气",
		RequestMetadataParams: `[{"name":"tenant","type":"string"}]`,
		RequestBodyParams:     `[{"name":"city","type":"string"}]`,
		ResponseBodyParams:    `[{"name":"summary","type":"string"}]`,
	})

	for _, required := range []string{
		"/api/v1/rules/{id}/execute/{msgType}",
		"/api/v1/rules/root-chain/execute/QUERY",
		`"metadata": {"tenant": "cn"}`,
		`"data": {"city": "Beijing"}`,
		`返回体中的 data`,
		"skill-creator-0.1.0",
		"YAML frontmatter",
	} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("expected prompt to contain %q, got %q", required, prompt)
		}
	}
}

func TestBuildRuleChainSkillGenerationPromptUsesCustomSkillRoot(t *testing.T) {
	prompt := BuildRuleChainSkillGenerationPrompt(RuleChainSkillPromptInput{
		RuleChainID:           "root-chain",
		RuleChainName:         "weather-agent",
		DirName:               "weather-agent",
		SkillRoot:             "/tmp/custom-skills",
		MsgType:               "QUERY",
		Description:           "根据城市查询天气",
		RequestMetadataParams: `[{"name":"tenant","type":"string"}]`,
		RequestBodyParams:     `[{"name":"city","type":"string"}]`,
		ResponseBodyParams:    `[{"name":"summary","type":"string"}]`,
	})
	if !strings.Contains(prompt, "/tmp/custom-skills/weather-agent/SKILL.md") {
		t.Fatalf("expected prompt to use custom skillRoot, got %q", prompt)
	}
}

func TestBuildRuleChainSkillPromptInputInfersMsgTypePriority(t *testing.T) {
	t.Run("flowgram entry msg type wins", func(t *testing.T) {
		rc := mustRuleChainFromEntity(t, withRuleChainWithAdditionalInfo(
			"root-chain",
			true,
			"weather-agent",
			map[string]interface{}{
				"flowgram": map[string]interface{}{
					"entry_msg_type": "fromFlowgram",
				},
			},
			map[string]interface{}{"msgType": "fromAdditional"},
		))
		if got := BuildRuleChainSkillPromptInput(rc, "weather-agent").MsgType; got != "fromFlowgram" {
			t.Fatalf("expected flowgram msgType, got %q", got)
		}
	})

	t.Run("additional info fallback", func(t *testing.T) {
		rc := mustRuleChainFromEntity(t, withRuleChainWithAdditionalInfo(
			"root-chain",
			true,
			"weather-agent",
			map[string]interface{}{},
			map[string]interface{}{"msgType": "fromAdditional"},
		))
		if got := BuildRuleChainSkillPromptInput(rc, "weather-agent").MsgType; got != "fromAdditional" {
			t.Fatalf("expected additionalInfo msgType, got %q", got)
		}
	})

	t.Run("endpoint path fallback", func(t *testing.T) {
		rc := mustRuleChainFromEntity(t, withRuleChainWithMetadataAndAdditionalInfo(
			"root-chain",
			true,
			"weather-agent",
			map[string]interface{}{},
			map[string]interface{}{
				"endpoints": []interface{}{
					map[string]interface{}{
						"type": "endpoint/rest",
						"routers": []interface{}{
							map[string]interface{}{
								"from": map[string]interface{}{"path": "/api/v1/hooks/github"},
							},
						},
					},
				},
			},
			nil,
		))
		if got := BuildRuleChainSkillPromptInput(rc, "weather-agent").MsgType; got != "github" {
			t.Fatalf("expected endpoint-derived msgType, got %q", got)
		}
	})

	t.Run("default chain", func(t *testing.T) {
		rc := mustRuleChainFromEntity(t, withRuleChain("root-chain", true, "weather-agent", map[string]interface{}{}))
		if got := BuildRuleChainSkillPromptInput(rc, "weather-agent").MsgType; got != "CHAIN" {
			t.Fatalf("expected default CHAIN msgType, got %q", got)
		}
	})
}

func TestAdditionalInfoDescriptionAffectsSkillPromptInputAndSignature(t *testing.T) {
	left := mustRuleChainFromEntity(t, withRuleChainWithAdditionalInfo(
		"root-chain",
		true,
		"weather-agent",
		map[string]interface{}{
			"flowgram": map[string]interface{}{
				"description": "flowgram description",
				"io": map[string]interface{}{
					"request_metadata_params":     []interface{}{map[string]interface{}{"name": "tenant"}},
					"request_message_body_params": []interface{}{map[string]interface{}{"name": "city"}},
					"response_message_body_params": []interface{}{
						map[string]interface{}{"name": "summary"},
					},
				},
			},
		},
		map[string]interface{}{"description": "additional description A"},
	))
	right := mustRuleChainFromEntity(t, withRuleChainWithAdditionalInfo(
		"root-chain",
		true,
		"weather-agent",
		map[string]interface{}{
			"flowgram": map[string]interface{}{
				"description": "flowgram description",
				"io": map[string]interface{}{
					"request_metadata_params":     []interface{}{map[string]interface{}{"name": "tenant"}},
					"request_message_body_params": []interface{}{map[string]interface{}{"name": "city"}},
					"response_message_body_params": []interface{}{
						map[string]interface{}{"name": "summary"},
					},
				},
			},
		},
		map[string]interface{}{"description": "additional description B"},
	))

	leftInput := BuildRuleChainSkillPromptInput(left, "weather-agent")
	rightInput := BuildRuleChainSkillPromptInput(right, "weather-agent")
	if leftInput.Description != "additional description A" {
		t.Fatalf("expected additionalInfo.description to win, got %q", leftInput.Description)
	}
	if rightInput.Description != "additional description B" {
		t.Fatalf("expected additionalInfo.description to win, got %q", rightInput.Description)
	}

	leftSig := BuildRuleChainSkillSignature(leftInput.Description, leftInput.RequestMetadataParams, leftInput.RequestBodyParams, leftInput.ResponseBodyParams)
	rightSig := BuildRuleChainSkillSignature(rightInput.Description, rightInput.RequestMetadataParams, rightInput.RequestBodyParams, rightInput.ResponseBodyParams)
	if leftSig == rightSig {
		t.Fatalf("expected signature to change with additionalInfo.description, got %q", leftSig)
	}
}

func TestResolveRuleChainSkillStatusRequiresCurrentSignatureAnchor(t *testing.T) {
	root := t.TempDir()
	dirName := "weather-agent"
	if err := os.MkdirAll(filepath.Join(root, dirName), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	currentSignature := BuildRuleChainSkillSignature(
		"根据城市查询天气",
		`[{"name":"tenant"}]`,
		`[{"name":"city"}]`,
		`[{"name":"summary"}]`,
	)
	oldSignature := BuildRuleChainSkillSignature(
		"旧描述",
		`[{"name":"tenant"}]`,
		`[{"name":"city"}]`,
		`[{"name":"summary"}]`,
	)
	content := strings.Join([]string{
		"---",
		"name: weather-agent",
		"description: test skill",
		"---",
		"",
		"# weather-agent",
		BuildRuleChainSkillSignatureAnchor(oldSignature),
		"rule_chain_id: root-chain",
		"execute_path: /api/v1/rules/root-chain/execute/QUERY",
		`request_body: {"metadata": {"tenant": "cn"}, "data": {"city": "Beijing"}}`,
		"metadata 和 data 必须分开整理",
	}, "\n")
	if err := os.WriteFile(filepath.Join(root, dirName, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	got, err := ResolveRuleChainSkillStatus(root, RuleChainSkillMeta{
		DirName:   dirName,
		EntryFile: "SKILL.md",
		Signature: currentSignature,
	}, currentSignature)
	if err != nil {
		t.Fatalf("ResolveRuleChainSkillStatus failed: %v", err)
	}
	if got != RuleChainSkillStatusStale {
		t.Fatalf("expected stale when current signature anchor missing, got %s", got)
	}
}

func TestGetRuleChainSkillStatusReportsReady(t *testing.T) {
	root := t.TempDir()
	dirName := "weather-agent"
	signature := BuildRuleChainSkillSignature(
		"根据城市查询天气",
		`[{"name":"tenant"}]`,
		`[{"name":"city"}]`,
		`[{"name":"summary"}]`,
	)
	if err := os.MkdirAll(filepath.Join(root, dirName), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, dirName, "SKILL.md"), []byte(strings.Join([]string{
		"---",
		"name: weather-agent",
		"description: generated skill",
		"---",
		"",
		"# generated",
		BuildRuleChainSkillSignatureAnchor(signature),
	}, "\n")), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	repo := newFakeRuleChainRepo(withRuleChain(
		"root-chain",
		true,
		"weather-agent",
		map[string]interface{}{
			"flowgram": map[string]interface{}{
				"description": "根据城市查询天气",
				"io": map[string]interface{}{
					"request_metadata_params":     []interface{}{map[string]interface{}{"name": "tenant"}},
					"request_message_body_params": []interface{}{map[string]interface{}{"name": "city"}},
					"response_message_body_params": []interface{}{
						map[string]interface{}{"name": "summary"},
					},
				},
				"skill": map[string]interface{}{
					"dir_name":                      dirName,
					"status":                        "ready",
					"signature":                     signature,
					"generated_at":                  "2026-04-24T10:30:00Z",
					"generated_by_managed_agent_id": float64(101),
					"skill_entry_file":              "SKILL.md",
				},
			},
		},
	))
	uc := newTestRuleChainUsecase(repo, root, fakeSkillAgentRunner{})

	reply, err := uc.GetRuleChainSkillStatus(context.Background(), &v1.GetRuleChainSkillStatusReq{Id: "root-chain"})
	if err != nil {
		t.Fatalf("GetRuleChainSkillStatus failed: %v", err)
	}
	if reply.GetStatus() != string(RuleChainSkillStatusReady) {
		t.Fatalf("expected ready, got %q", reply.GetStatus())
	}
	if reply.GetDirName() != dirName {
		t.Fatalf("expected dir name %q, got %q", dirName, reply.GetDirName())
	}
	if reply.GetEntryFile() != "SKILL.md" {
		t.Fatalf("expected entry file SKILL.md, got %q", reply.GetEntryFile())
	}
	if reply.GetSignature() != signature {
		t.Fatalf("expected signature %q, got %q", signature, reply.GetSignature())
	}
	if reply.GetGeneratedAt() != "2026-04-24T10:30:00Z" {
		t.Fatalf("expected generatedAt to round-trip, got %q", reply.GetGeneratedAt())
	}
	if reply.GetGeneratedByManagedAgentId() != 101 {
		t.Fatalf("expected managed agent id 101, got %d", reply.GetGeneratedByManagedAgentId())
	}
}

func TestGenerateRuleChainSkillWritesSkillAndUpdatesConfig(t *testing.T) {
	root := t.TempDir()
	expectedDir := BuildRuleChainSkillConflictDirName("weather-agent", "root-chain")
	repo := newFakeRuleChainRepo(withRuleChainWithAdditionalInfo(
		"root-chain",
		true,
		"weather-agent",
		map[string]interface{}{
			"flowgram": map[string]interface{}{
				"description":    "根据城市查询天气",
				"entry_msg_type": "QUERY",
				"io": map[string]interface{}{
					"request_metadata_params":     []interface{}{map[string]interface{}{"name": "tenant"}},
					"request_message_body_params": []interface{}{map[string]interface{}{"name": "city"}},
					"response_message_body_params": []interface{}{
						map[string]interface{}{"name": "summary"},
					},
				},
			},
		},
		map[string]interface{}{
			"description": "根据城市查询天气（additional info）",
		},
	))

	runner := fakeSkillAgentRunner{
		run: func(ctx context.Context, req HarnessRequest) (string, error) {
			if req.ManagedAgentID != 101 {
				t.Fatalf("expected managed agent id 101, got %d", req.ManagedAgentID)
			}
			if req.ToolOptions == nil || !req.ToolOptions.EnableSkillTool {
				t.Fatalf("expected skill tool to be force-enabled for generation, got %#v", req.ToolOptions)
			}
			if len(req.ToolOptions.SkillAllowlist) != 1 || req.ToolOptions.SkillAllowlist[0] != "skill-creator-0.1.0" {
				t.Fatalf("expected skill allowlist [skill-creator-0.1.0], got %#v", req.ToolOptions.SkillAllowlist)
			}
			for _, required := range []string{
				"官方 `skill` 工具",
				"skill-creator-0.1.0",
				filepath.Join(root, expectedDir, "SKILL.md"),
				"/api/v1/rules/{id}/execute/{msgType}",
				"/api/v1/rules/root-chain/execute/QUERY",
				`"metadata": {"tenant": "cn"}`,
				`"data": {"city": "Beijing"}`,
				"根据城市查询天气（additional info）",
			} {
				if !strings.Contains(req.Input, required) {
					t.Fatalf("expected prompt to contain %q, got %q", required, req.Input)
				}
			}
			targetDir := filepath.Join(root, expectedDir)
			if err := os.MkdirAll(targetDir, 0o755); err != nil {
				return "", err
			}
			if err := os.WriteFile(filepath.Join(targetDir, "SKILL.md"), []byte("# generated"), 0o644); err != nil {
				return "", err
			}
			content := strings.Join([]string{
				"---",
				"name: weather-agent",
				"description: weather skill",
				"---",
				"",
				"# weather-agent",
				BuildRuleChainSkillSignatureAnchor(BuildRuleChainSkillSignature(
					"根据城市查询天气（additional info）",
					`[{"name":"tenant"}]`,
					`[{"name":"city"}]`,
					`[{"name":"summary"}]`,
				)),
				"rule_chain_id: root-chain",
				"execute_path: /api/v1/rules/root-chain/execute/QUERY",
				"request_body: {\"metadata\": {\"tenant\": \"cn\"}, \"data\": {\"city\": \"Beijing\"}}",
				"result_explanation: successful calls return a structured object",
				"response_read: read response.data.summary",
				"metadata 和 data 必须分开整理",
			}, "\n")
			if err := os.WriteFile(filepath.Join(targetDir, "SKILL.md"), []byte(content), 0o644); err != nil {
				return "", err
			}
			return `{"ok":true}`, nil
		},
	}
	uc := newTestRuleChainUsecase(repo, root, runner)

	reply, err := uc.GenerateRuleChainSkill(context.Background(), &v1.GenerateRuleChainSkillReq{
		Id:             "root-chain",
		ManagedAgentId: 101,
	})
	if err != nil {
		t.Fatalf("GenerateRuleChainSkill failed: %v", err)
	}
	if reply.GetStatus() != string(RuleChainSkillStatusReady) {
		t.Fatalf("expected ready, got %q", reply.GetStatus())
	}
	if reply.GetDirName() != expectedDir {
		t.Fatalf("expected dir name %s, got %q", expectedDir, reply.GetDirName())
	}

	stored := repo.mustRuleChain(t, "root-chain")
	cfg := parseRuleChainConfiguration(t, stored)
	flowgram := mustJSONMap(t, cfg["flowgram"])
	skill := mustJSONMap(t, flowgram["skill"])

	if skill["dir_name"] != expectedDir {
		t.Fatalf("expected persisted dir_name %s, got %#v", expectedDir, skill["dir_name"])
	}
	if skill["status"] != string(RuleChainSkillStatusReady) {
		t.Fatalf("expected persisted status ready, got %#v", skill["status"])
	}
	if skill["skill_entry_file"] != "SKILL.md" {
		t.Fatalf("expected persisted skill entry file SKILL.md, got %#v", skill["skill_entry_file"])
	}
	if skill["last_error"] != "" {
		t.Fatalf("expected empty last_error, got %#v", skill["last_error"])
	}
	if skill["generated_at"] == "" {
		t.Fatalf("expected generated_at to be set, got %#v", skill["generated_at"])
	}
	if skill["signature"] == "" {
		t.Fatalf("expected signature to be set, got %#v", skill["signature"])
	}
	if got := asInt64(skill["generated_by_managed_agent_id"]); got != 101 {
		t.Fatalf("expected generated_by_managed_agent_id 101, got %d (%#v)", got, skill["generated_by_managed_agent_id"])
	}
}

func TestGenerateRuleChainSkillUsesCustomSkillRootForPromptAndValidation(t *testing.T) {
	root := t.TempDir()
	expectedDir := BuildRuleChainSkillConflictDirName("weather-agent", "root-chain")
	repo := newFakeRuleChainRepo(withRuleChain("root-chain", true, "weather-agent", map[string]interface{}{}))
	uc := newTestRuleChainUsecase(repo, root, fakeSkillAgentRunner{
		run: func(ctx context.Context, req HarnessRequest) (string, error) {
			if !strings.Contains(req.Input, filepath.Join(root, expectedDir, "SKILL.md")) {
				t.Fatalf("expected prompt to use custom skillRoot, got %q", req.Input)
			}
			targetDir := filepath.Join(root, expectedDir)
			if err := os.MkdirAll(targetDir, 0o755); err != nil {
				return "", err
			}
			content := strings.Join([]string{
				"---",
				"name: weather-agent",
				"description: weather skill",
				"---",
				"",
				"# weather-agent",
				BuildRuleChainSkillSignatureAnchor(BuildRuleChainSkillSignature("", "[]", "[]", "[]")),
				"rule_chain_id: root-chain",
				"execute_path: /api/v1/rules/root-chain/execute/CHAIN",
				`request_body: {"metadata": {}, "data": {}}`,
				"result_explanation: successful calls return a structured object",
				"response_read: response.data.result",
				"metadata 和 data 必须分开整理",
			}, "\n")
			return `{"ok":true}`, os.WriteFile(filepath.Join(targetDir, "SKILL.md"), []byte(content), 0o644)
		},
	})

	reply, err := uc.GenerateRuleChainSkill(context.Background(), &v1.GenerateRuleChainSkillReq{
		Id:             "root-chain",
		ManagedAgentId: 101,
	})
	if err != nil {
		t.Fatalf("GenerateRuleChainSkill failed: %v", err)
	}
	if reply.GetDirName() != expectedDir {
		t.Fatalf("expected %s, got %q", expectedDir, reply.GetDirName())
	}
}

func TestBuildRuleMsgMetadataStringifiesValues(t *testing.T) {
	md, err := buildRuleMsgMetadata(mustStructPb(t, map[string]interface{}{
		"tenant":  "cn",
		"enabled": true,
		"count":   float64(3),
		"nested": map[string]interface{}{
			"a": "b",
		},
	}))
	if err != nil {
		t.Fatalf("buildRuleMsgMetadata failed: %v", err)
	}
	if md.GetValue("tenant") != "cn" {
		t.Fatalf("expected tenant=cn, got %q", md.GetValue("tenant"))
	}
	if md.GetValue("enabled") != "true" {
		t.Fatalf("expected enabled=true, got %q", md.GetValue("enabled"))
	}
	if md.GetValue("count") != "3" {
		t.Fatalf("expected count=3, got %q", md.GetValue("count"))
	}
	if md.GetValue("nested") != `{"a":"b"}` {
		t.Fatalf("expected nested JSON string, got %q", md.GetValue("nested"))
	}
}

func TestExecuteRuleChainSyncCarriesMetadataIntoRuleMsg(t *testing.T) {
	enginePool := rulego.NewRuleGo()
	if _, err := enginePool.New("root-chain", []byte(testExecuteRuleChainSyncMetadataAwareJSON), rulego.WithConfig(rulego.NewConfig())); err != nil {
		t.Fatalf("rulego.New failed: %v", err)
	}

	metadataPb := mustStructPb(t, map[string]interface{}{"tenant": "cn", "enabled": true})
	dataPb := mustStructPb(t, map[string]interface{}{"city": "Beijing"})

	uc := &RuleChainUsecase{
		log:        log.NewHelper(log.NewStdLogger(io.Discard)),
		ruleEngine: enginePool,
	}
	reply, err := uc.ExecuteRuleChainSync(context.Background(), &v1.ExecuteRuleChainReq{
		Id:       "root-chain",
		MsgType:  "CHAIN",
		MsgId:    "msg-1",
		Metadata: metadataPb,
		Data:     dataPb,
	})
	if err != nil {
		t.Fatalf("ExecuteRuleChainSync failed: %v", err)
	}
	got := reply.GetData().AsMap()
	if got["city"] != "Beijing" {
		t.Fatalf("expected business data to remain in msg.data, got %#v", got)
	}
	if got["metadataTenant"] != "cn" {
		t.Fatalf("expected metadata tenant to be readable from RuleMsg.Metadata, got %#v", got)
	}
	if got["metadataEnabled"] != "true" {
		t.Fatalf("expected non-string metadata to be stringified, got %#v", got)
	}
}

func TestGenerateRuleChainSkillRejectsMissingResultExplanationAnchor(t *testing.T) {
	root := t.TempDir()
	expectedDir := BuildRuleChainSkillConflictDirName("weather-agent", "root-chain")
	repo := newFakeRuleChainRepo(withRuleChain("root-chain", true, "weather-agent", map[string]interface{}{}))
	uc := newTestRuleChainUsecase(repo, root, fakeSkillAgentRunner{
		run: func(ctx context.Context, req HarnessRequest) (string, error) {
			targetDir := filepath.Join(root, expectedDir)
			if err := os.MkdirAll(targetDir, 0o755); err != nil {
				return "", err
			}
			content := strings.Join([]string{
				"---",
				"name: weather-agent",
				"description: weather skill",
				"---",
				"",
				"# weather-agent",
				BuildRuleChainSkillSignatureAnchor(BuildRuleChainSkillSignature("", "[]", "[]", "[]")),
				"rule_chain_id: root-chain",
				"execute_path: /api/v1/rules/root-chain/execute/CHAIN",
				`request_body: {"metadata": {}, "data": {}}`,
				"response_read: response.data.result",
				"metadata 和 data 必须分开整理",
			}, "\n")
			return `{"ok":true}`, os.WriteFile(filepath.Join(targetDir, "SKILL.md"), []byte(content), 0o644)
		},
	})

	_, err := uc.GenerateRuleChainSkill(context.Background(), &v1.GenerateRuleChainSkillReq{
		Id:             "root-chain",
		ManagedAgentId: 101,
	})
	if err == nil {
		t.Fatal("expected missing result_explanation anchor error")
	}
	if !strings.Contains(err.Error(), "result_explanation") {
		t.Fatalf("expected result_explanation anchor error, got %v", err)
	}
}

func TestGenerateRuleChainSkillRejectsMissingResponseReadAnchor(t *testing.T) {
	root := t.TempDir()
	expectedDir := BuildRuleChainSkillConflictDirName("weather-agent", "root-chain")
	repo := newFakeRuleChainRepo(withRuleChain("root-chain", true, "weather-agent", map[string]interface{}{}))
	uc := newTestRuleChainUsecase(repo, root, fakeSkillAgentRunner{
		run: func(ctx context.Context, req HarnessRequest) (string, error) {
			targetDir := filepath.Join(root, expectedDir)
			if err := os.MkdirAll(targetDir, 0o755); err != nil {
				return "", err
			}
			content := strings.Join([]string{
				"---",
				"name: weather-agent",
				"description: weather skill",
				"---",
				"",
				"# weather-agent",
				BuildRuleChainSkillSignatureAnchor(BuildRuleChainSkillSignature("", "[]", "[]", "[]")),
				"rule_chain_id: root-chain",
				"execute_path: /api/v1/rules/root-chain/execute/CHAIN",
				`request_body: {"metadata": {}, "data": {}}`,
				"result_explanation: successful calls return a structured object",
				"metadata 和 data 必须分开整理",
			}, "\n")
			return `{"ok":true}`, os.WriteFile(filepath.Join(targetDir, "SKILL.md"), []byte(content), 0o644)
		},
	})

	_, err := uc.GenerateRuleChainSkill(context.Background(), &v1.GenerateRuleChainSkillReq{
		Id:             "root-chain",
		ManagedAgentId: 101,
	})
	if err == nil {
		t.Fatal("expected missing response_read anchor error")
	}
	if !strings.Contains(err.Error(), "response_read") {
		t.Fatalf("expected response_read anchor error, got %v", err)
	}
}

func TestGenerateRuleChainSkillRejectsMissingFrontmatter(t *testing.T) {
	root := t.TempDir()
	expectedDir := BuildRuleChainSkillConflictDirName("weather-agent", "root-chain")
	repo := newFakeRuleChainRepo(withRuleChain("root-chain", true, "weather-agent", map[string]interface{}{}))
	uc := newTestRuleChainUsecase(repo, root, fakeSkillAgentRunner{
		run: func(ctx context.Context, req HarnessRequest) (string, error) {
			targetDir := filepath.Join(root, expectedDir)
			if err := os.MkdirAll(targetDir, 0o755); err != nil {
				return "", err
			}
			content := strings.Join([]string{
				"# weather-agent",
				BuildRuleChainSkillSignatureAnchor(BuildRuleChainSkillSignature("", "[]", "[]", "[]")),
				"rule_chain_id: root-chain",
				"execute_path: /api/v1/rules/root-chain/execute/CHAIN",
				`request_body: {"metadata": {}, "data": {}}`,
				"result_explanation: successful calls return a structured object",
				"response_read: response.data.result",
				"metadata 和 data 必须分开整理",
			}, "\n")
			return `{"ok":true}`, os.WriteFile(filepath.Join(targetDir, "SKILL.md"), []byte(content), 0o644)
		},
	})

	_, err := uc.GenerateRuleChainSkill(context.Background(), &v1.GenerateRuleChainSkillReq{
		Id:             "root-chain",
		ManagedAgentId: 101,
	})
	if err == nil {
		t.Fatal("expected frontmatter validation error")
	}
	if !strings.Contains(err.Error(), "frontmatter") {
		t.Fatalf("expected frontmatter error, got %v", err)
	}
}

func TestGenerateRuleChainSkillFailureKeepsOldGeneratedSignature(t *testing.T) {
	root := t.TempDir()
	oldDescription := "旧描述"
	oldSignature := BuildRuleChainSkillSignature(
		oldDescription,
		`[{"name":"tenant"}]`,
		`[{"name":"city"}]`,
		`[{"name":"summary"}]`,
	)
	if err := os.MkdirAll(filepath.Join(root, "weather-agent"), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	oldContent := strings.Join([]string{
		"---",
		"name: weather-agent",
		"description: weather skill",
		"---",
		"",
		"# weather-agent",
		BuildRuleChainSkillSignatureAnchor(oldSignature),
		"rule_chain_id: root-chain",
		"execute_path: /api/v1/rules/root-chain/execute/CHAIN",
		"request_body: {\"metadata\": {\"tenant\": \"cn\"}, \"data\": {\"city\": \"Beijing\"}}",
		"response_read: read response.data.summary",
		"metadata 和 data 必须分开整理",
	}, "\n")
	if err := os.WriteFile(filepath.Join(root, "weather-agent", "SKILL.md"), []byte(oldContent), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	repo := newFakeRuleChainRepo(withRuleChainWithAdditionalInfo(
		"root-chain",
		true,
		"weather-agent",
		map[string]interface{}{
			"flowgram": map[string]interface{}{
				"io": map[string]interface{}{
					"request_metadata_params":     []interface{}{map[string]interface{}{"name": "tenant"}},
					"request_message_body_params": []interface{}{map[string]interface{}{"name": "city"}},
					"response_message_body_params": []interface{}{
						map[string]interface{}{"name": "summary"},
					},
				},
				"skill": map[string]interface{}{
					"dir_name":         "weather-agent",
					"skill_entry_file": "SKILL.md",
					"signature":        oldSignature,
					"status":           "ready",
				},
			},
		},
		map[string]interface{}{"description": "新描述"},
	))
	uc := newTestRuleChainUsecase(repo, root, fakeSkillAgentRunner{
		run: func(ctx context.Context, req HarnessRequest) (string, error) {
			return "", errors.New("runner failed")
		},
	})

	_, err := uc.GenerateRuleChainSkill(context.Background(), &v1.GenerateRuleChainSkillReq{
		Id:             "root-chain",
		ManagedAgentId: 101,
	})
	if err == nil {
		t.Fatal("expected generation failure")
	}

	stored := repo.mustRuleChain(t, "root-chain")
	cfg := parseRuleChainConfiguration(t, stored)
	skill := mustJSONMap(t, mustJSONMap(t, cfg["flowgram"])["skill"])
	if skill["signature"] != oldSignature {
		t.Fatalf("expected old signature to be preserved, got %#v", skill["signature"])
	}
	if skill["status"] != string(RuleChainSkillStatusStale) {
		t.Fatalf("expected stale status after failed regeneration with old artifact, got %#v", skill["status"])
	}
	if skill["last_error"] == "" {
		t.Fatalf("expected last_error to be recorded")
	}
}

func TestGenerateRuleChainSkillRejectsMissingManagedAgentID(t *testing.T) {
	repo := newFakeRuleChainRepo(withRuleChain("root-chain", true, "weather-agent", map[string]interface{}{}))
	uc := newTestRuleChainUsecase(repo, t.TempDir(), fakeSkillAgentRunner{})

	_, err := uc.GenerateRuleChainSkill(context.Background(), &v1.GenerateRuleChainSkillReq{Id: "root-chain"})
	if err == nil {
		t.Fatal("expected error for missing managedAgentId")
	}
	if !strings.Contains(err.Error(), "managedAgentId") {
		t.Fatalf("expected managedAgentId error, got %v", err)
	}
}

func TestGenerateRuleChainSkillRejectsChildRuleChain(t *testing.T) {
	repo := newFakeRuleChainRepo(withRuleChain("child-chain", false, "weather-agent", map[string]interface{}{}))
	uc := newTestRuleChainUsecase(repo, t.TempDir(), fakeSkillAgentRunner{})

	_, err := uc.GenerateRuleChainSkill(context.Background(), &v1.GenerateRuleChainSkillReq{
		Id:             "child-chain",
		ManagedAgentId: 101,
	})
	if err == nil {
		t.Fatal("expected child rule chain error")
	}
	if !strings.Contains(err.Error(), "主规则链") {
		t.Fatalf("expected root rule chain error, got %v", err)
	}
}

func TestGenerateRuleChainSkillFallsBackToHarnessOutputMarkdown(t *testing.T) {
	root := t.TempDir()
	expectedDir := BuildRuleChainSkillConflictDirName("weather-agent", "root-chain")
	repo := newFakeRuleChainRepo(withRuleChain("root-chain", true, "weather-agent", map[string]interface{}{}))
	uc := newTestRuleChainUsecase(repo, root, fakeSkillAgentRunner{
		run: func(ctx context.Context, req HarnessRequest) (string, error) {
			content := strings.Join([]string{
				"---",
				"name: weather-agent",
				"description: weather skill",
				"---",
				"",
				"# weather-agent",
				BuildRuleChainSkillSignatureAnchor(BuildRuleChainSkillSignature("", "[]", "[]", "[]")),
				"rule_chain_id: root-chain",
				"execute_path: /api/v1/rules/root-chain/execute/CHAIN",
				`request_body: {"metadata": {}, "data": {}}`,
				"result_explanation: successful calls return a structured object",
				"response_read: response.data.result",
				"metadata 和 data 必须分开整理",
			}, "\n")
			return "<generated_skill_markdown>\n" + content + "\n</generated_skill_markdown>", nil
		},
	})

	reply, err := uc.GenerateRuleChainSkill(context.Background(), &v1.GenerateRuleChainSkillReq{
		Id:             "root-chain",
		ManagedAgentId: 101,
	})
	if err != nil {
		t.Fatalf("expected fallback write from harness output, got %v", err)
	}
	if reply.GetStatus() != string(RuleChainSkillStatusReady) {
		t.Fatalf("expected ready, got %q", reply.GetStatus())
	}
	saved, err := os.ReadFile(filepath.Join(root, expectedDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("expected fallback SKILL.md written, got %v", err)
	}
	if !strings.Contains(string(saved), "rule_chain_id: root-chain") {
		t.Fatalf("expected saved fallback content to include anchors, got %q", string(saved))
	}
}

func TestGenerateRuleChainSkillFailsWhenSkillFileMissing(t *testing.T) {
	root := t.TempDir()
	repo := newFakeRuleChainRepo(withRuleChain("root-chain", true, "weather-agent", map[string]interface{}{}))
	uc := newTestRuleChainUsecase(repo, root, fakeSkillAgentRunner{
		run: func(ctx context.Context, req HarnessRequest) (string, error) {
			return `{"ok":true}`, nil
		},
	})

	_, err := uc.GenerateRuleChainSkill(context.Background(), &v1.GenerateRuleChainSkillReq{
		Id:             "root-chain",
		ManagedAgentId: 101,
	})
	if err == nil {
		t.Fatal("expected missing skill file error")
	}
	if !strings.Contains(err.Error(), "SKILL.md") {
		t.Fatalf("expected missing SKILL.md error, got %v", err)
	}
}

func TestGenerateRuleChainSkillFailsWhenGeneratedContentMissesAnchors(t *testing.T) {
	root := t.TempDir()
	expectedDir := BuildRuleChainSkillConflictDirName("weather-agent", "root-chain")
	repo := newFakeRuleChainRepo(withRuleChain("root-chain", true, "weather-agent", map[string]interface{}{}))
	uc := newTestRuleChainUsecase(repo, root, fakeSkillAgentRunner{
		run: func(ctx context.Context, req HarnessRequest) (string, error) {
			targetDir := filepath.Join(root, expectedDir)
			if err := os.MkdirAll(targetDir, 0o755); err != nil {
				return "", err
			}
			content := strings.Join([]string{
				"---",
				"name: weather-agent",
				"description: generic skill",
				"---",
				"",
				"# generic skill",
				"just do something",
			}, "\n")
			if err := os.WriteFile(filepath.Join(targetDir, "SKILL.md"), []byte(content), 0o644); err != nil {
				return "", err
			}
			return `{"ok":true}`, nil
		},
	})

	_, err := uc.GenerateRuleChainSkill(context.Background(), &v1.GenerateRuleChainSkillReq{
		Id:             "root-chain",
		ManagedAgentId: 101,
	})
	if err == nil {
		t.Fatal("expected anchor validation error")
	}
	if !strings.Contains(err.Error(), "关键锚点") {
		t.Fatalf("expected anchor validation error, got %v", err)
	}
}

func TestChooseRuleChainSkillDirNameResolvesNameConflictStably(t *testing.T) {
	repo := newFakeRuleChainRepo(
		withRuleChain(
			"chain-a",
			true,
			"Weather Agent",
			map[string]interface{}{
				"flowgram": map[string]interface{}{
					"skill": map[string]interface{}{
						"dir_name": "weather-agent",
					},
				},
			},
		),
		withRuleChain(
			"chain-b",
			true,
			"Weather Agent",
			map[string]interface{}{},
		),
	)
	uc := newTestRuleChainUsecase(repo, t.TempDir(), fakeSkillAgentRunner{})
	ruleChainDB, ruleChain, err := uc.loadRootRuleChainForSkill(context.Background(), "chain-b")
	if err != nil {
		t.Fatalf("loadRootRuleChainForSkill failed: %v", err)
	}
	dirName, err := uc.chooseRuleChainSkillDirName(context.Background(), ruleChainDB, ruleChain, RuleChainSkillMeta{})
	if err != nil {
		t.Fatalf("chooseRuleChainSkillDirName failed: %v", err)
	}
	if dirName == "weather-agent" {
		t.Fatalf("expected collision to be resolved, got %q", dirName)
	}
	if !strings.HasPrefix(dirName, "weather-agent-") {
		t.Fatalf("expected readable dir prefix, got %q", dirName)
	}
	dirNameAgain, err := uc.chooseRuleChainSkillDirName(context.Background(), ruleChainDB, ruleChain, RuleChainSkillMeta{})
	if err != nil {
		t.Fatalf("chooseRuleChainSkillDirName second run failed: %v", err)
	}
	if dirNameAgain != dirName {
		t.Fatalf("expected stable dir resolution, first=%q second=%q", dirName, dirNameAgain)
	}
}

func TestChooseRuleChainSkillDirNameAvoidsExistingFilesystemDir(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "weather-agent"), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	repo := newFakeRuleChainRepo(withRuleChain("chain-a", true, "Weather Agent", map[string]interface{}{}))
	uc := newTestRuleChainUsecase(repo, root, fakeSkillAgentRunner{})
	ruleChainDB, ruleChain, err := uc.loadRootRuleChainForSkill(context.Background(), "chain-a")
	if err != nil {
		t.Fatalf("loadRootRuleChainForSkill failed: %v", err)
	}
	dirName, err := uc.chooseRuleChainSkillDirName(context.Background(), ruleChainDB, ruleChain, RuleChainSkillMeta{})
	if err != nil {
		t.Fatalf("chooseRuleChainSkillDirName failed: %v", err)
	}
	if dirName == "weather-agent" {
		t.Fatalf("expected filesystem conflict to be resolved, got %q", dirName)
	}
}

func TestChooseRuleChainSkillDirNameReplacesConflictingExistingMetaDirName(t *testing.T) {
	repo := newFakeRuleChainRepo(
		withRuleChain(
			"chain-a",
			true,
			"Weather Agent",
			map[string]interface{}{
				"flowgram": map[string]interface{}{
					"skill": map[string]interface{}{
						"dir_name": "weather-agent",
					},
				},
			},
		),
		withRuleChain(
			"chain-b",
			true,
			"Different Name",
			map[string]interface{}{
				"flowgram": map[string]interface{}{
					"skill": map[string]interface{}{
						"dir_name": "weather-agent",
					},
				},
			},
		),
	)
	uc := newTestRuleChainUsecase(repo, t.TempDir(), fakeSkillAgentRunner{})
	ruleChainDB, ruleChain, err := uc.loadRootRuleChainForSkill(context.Background(), "chain-b")
	if err != nil {
		t.Fatalf("loadRootRuleChainForSkill failed: %v", err)
	}
	meta := ParseRuleChainSkillMeta(asJSONMap(ruleChain.RuleChain.Configuration))
	dirName, err := uc.chooseRuleChainSkillDirName(context.Background(), ruleChainDB, ruleChain, meta)
	if err != nil {
		t.Fatalf("chooseRuleChainSkillDirName failed: %v", err)
	}
	if dirName == "weather-agent" {
		t.Fatalf("expected conflicting existing meta dir to be replaced, got %q", dirName)
	}
	if !strings.HasPrefix(dirName, "different-name-") {
		t.Fatalf("expected readable replacement dir, got %q", dirName)
	}
}

func TestDeleteRuleChainRemovesSkillDirBeforeDeletingDB(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "weather-agent")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# generated"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	repo := newFakeRuleChainRepo(withRuleChain(
		"root-chain",
		true,
		"weather-agent",
		map[string]interface{}{
			"flowgram": map[string]interface{}{
				"skill": map[string]interface{}{
					"dir_name":         "weather-agent",
					"skill_entry_file": "SKILL.md",
				},
			},
		},
	))
	uc := newTestRuleChainUsecase(repo, root, fakeSkillAgentRunner{})

	_, err := uc.DeleteRuleChain(context.Background(), &v1.DeleteRuleChainReq{Id: "root-chain"})
	if err != nil {
		t.Fatalf("DeleteRuleChain failed: %v", err)
	}
	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Fatalf("expected skill dir to be deleted, stat err=%v", err)
	}
	if repo.deleteCalls != 1 {
		t.Fatalf("expected DB delete to be called once, got %d", repo.deleteCalls)
	}
	if _, exists := repo.ruleChains["root-chain"]; exists {
		t.Fatal("expected rule chain to be removed from repo")
	}
}

func TestDeleteRuleChainSucceedsWhenSkillDirMissing(t *testing.T) {
	root := t.TempDir()
	repo := newFakeRuleChainRepo(withRuleChain(
		"root-chain",
		true,
		"weather-agent",
		map[string]interface{}{
			"flowgram": map[string]interface{}{
				"skill": map[string]interface{}{
					"dir_name":         "weather-agent",
					"skill_entry_file": "SKILL.md",
				},
			},
		},
	))
	uc := newTestRuleChainUsecase(repo, root, fakeSkillAgentRunner{})

	_, err := uc.DeleteRuleChain(context.Background(), &v1.DeleteRuleChainReq{Id: "root-chain"})
	if err != nil {
		t.Fatalf("DeleteRuleChain failed: %v", err)
	}
	if repo.deleteCalls != 1 {
		t.Fatalf("expected DB delete to be called once, got %d", repo.deleteCalls)
	}
}

func TestDeleteRuleChainFailsWhenSkillDirRemovalFailsAndSkipsDBDelete(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "weather-agent")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# generated"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.Chmod(root, 0o500); err != nil {
		t.Fatalf("Chmod failed: %v", err)
	}
	defer func() {
		_ = os.Chmod(root, 0o700)
	}()
	repo := newFakeRuleChainRepo(withRuleChain(
		"root-chain",
		true,
		"weather-agent",
		map[string]interface{}{
			"flowgram": map[string]interface{}{
				"skill": map[string]interface{}{
					"dir_name":         "weather-agent",
					"skill_entry_file": "SKILL.md",
				},
			},
		},
	))
	uc := newTestRuleChainUsecase(repo, root, fakeSkillAgentRunner{})

	_, err := uc.DeleteRuleChain(context.Background(), &v1.DeleteRuleChainReq{Id: "root-chain"})
	if err == nil {
		t.Fatal("expected skill dir removal error")
	}
	if repo.deleteCalls != 0 {
		t.Fatalf("expected DB delete to be skipped, got %d calls", repo.deleteCalls)
	}
	if _, exists := repo.ruleChains["root-chain"]; !exists {
		t.Fatal("expected rule chain to remain in repo after failure")
	}
}

func TestDeleteRuleChainRestoresSkillDirWhenDBDeleteFails(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "weather-agent")
	skillFile := filepath.Join(skillDir, "SKILL.md")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(skillFile, []byte("# generated"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	repo := newFakeRuleChainRepo(withRuleChain(
		"root-chain",
		true,
		"weather-agent",
		map[string]interface{}{
			"flowgram": map[string]interface{}{
				"skill": map[string]interface{}{
					"dir_name":         "weather-agent",
					"skill_entry_file": "SKILL.md",
				},
			},
		},
	))
	repo.deleteErr = errors.New("db delete failed")
	uc := newTestRuleChainUsecase(repo, root, fakeSkillAgentRunner{})

	_, err := uc.DeleteRuleChain(context.Background(), &v1.DeleteRuleChainReq{Id: "root-chain"})
	if err == nil {
		t.Fatal("expected DB delete failure")
	}
	if repo.deleteCalls != 1 {
		t.Fatalf("expected DB delete to be attempted once, got %d", repo.deleteCalls)
	}
	if _, err := os.Stat(skillDir); err != nil {
		t.Fatalf("expected skill dir to be restored, stat err=%v", err)
	}
	content, err := os.ReadFile(skillFile)
	if err != nil {
		t.Fatalf("expected skill file to be restored: %v", err)
	}
	if string(content) != "# generated" {
		t.Fatalf("expected skill file content restored, got %q", string(content))
	}
	if _, exists := repo.ruleChains["root-chain"]; !exists {
		t.Fatal("expected rule chain to remain in repo after DB failure")
	}
}

func TestDeleteRuleChainSucceedsWhenRecycleCleanupFailsAfterDBDelete(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "weather-agent")
	skillFile := filepath.Join(skillDir, "SKILL.md")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(skillFile, []byte("# generated"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	repo := newFakeRuleChainRepo(withRuleChain(
		"root-chain",
		true,
		"weather-agent",
		map[string]interface{}{
			"flowgram": map[string]interface{}{
				"skill": map[string]interface{}{
					"dir_name":         "weather-agent",
					"skill_entry_file": "SKILL.md",
				},
			},
		},
	))
	recycleRoot := filepath.Join(root, ".deleted-rulechain-skills")
	if !strings.HasPrefix(recycleRoot, root+string(filepath.Separator)) {
		t.Fatalf("expected recycle root inside skillRoot, recycleRoot=%q skillRoot=%q", recycleRoot, root)
	}
	repo.deleteHook = func() error {
		if err := os.Chmod(recycleRoot, 0o500); err != nil {
			return err
		}
		return nil
	}
	defer func() {
		_ = os.Chmod(recycleRoot, 0o700)
	}()
	uc := newTestRuleChainUsecase(repo, root, fakeSkillAgentRunner{})

	_, err := uc.DeleteRuleChain(context.Background(), &v1.DeleteRuleChainReq{Id: "root-chain"})
	if err != nil {
		t.Fatalf("expected delete to succeed despite recycle cleanup failure, got %v", err)
	}
	if repo.deleteCalls != 1 {
		t.Fatalf("expected DB delete to be called once, got %d", repo.deleteCalls)
	}
	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Fatalf("expected formal skill dir to stay deleted, stat err=%v", err)
	}
	entries, err := os.ReadDir(recycleRoot)
	if err != nil {
		t.Fatalf("expected recycle root to remain readable for assertion: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected recycled skill dir to remain when cleanup fails")
	}
	if _, exists := repo.ruleChains["root-chain"]; exists {
		t.Fatal("expected rule chain to be removed from repo")
	}
}

func TestExecuteRuleChainSyncParsesJSONObjectResult(t *testing.T) {
	enginePool := rulego.NewRuleGo()
	if _, err := enginePool.New("root-chain", []byte(testExecuteRuleChainSyncRuleChainJSON), rulego.WithConfig(rulego.NewConfig())); err != nil {
		t.Fatalf("rulego.New failed: %v", err)
	}

	metadataPb, err := structpb.NewStruct(map[string]interface{}{"tenant": "cn"})
	if err != nil {
		t.Fatalf("structpb.NewStruct metadata failed: %v", err)
	}
	dataPb, err := structpb.NewStruct(map[string]interface{}{"city": "Beijing"})
	if err != nil {
		t.Fatalf("structpb.NewStruct data failed: %v", err)
	}

	uc := &RuleChainUsecase{
		log:        log.NewHelper(log.NewStdLogger(io.Discard)),
		ruleEngine: enginePool,
	}
	reply, err := uc.ExecuteRuleChainSync(context.Background(), &v1.ExecuteRuleChainReq{
		Id:       "root-chain",
		MsgType:  "TEST_MSG_TYPE3",
		MsgId:    "msg-1",
		Metadata: metadataPb,
		Data:     dataPb,
	})
	if err != nil {
		t.Fatalf("ExecuteRuleChainSync failed: %v", err)
	}
	got := reply.GetData().AsMap()
	if _, exists := got["metadata"]; exists {
		t.Fatalf("expected metadata to stay in RuleMsg.Metadata instead of msg.Data, got %#v", got)
	}
	if got["city"] != "Beijing" {
		t.Fatalf("expected data to round-trip, got %#v", got)
	}
}

const testExecuteRuleChainSyncRuleChainJSON = `{
  "ruleChain": {
    "id":"chain_msg_type_switch",
    "name":"测试规则链-msgTypeSwitch",
    "root":true
  },
  "metadata": {
    "nodes": [
      {
        "id":"s1",
        "type":"msgTypeSwitch",
        "name":"消息路由"
      }
    ],
    "connections": []
  }
}`

const testExecuteRuleChainSyncMetadataAwareJSON = `{
  "ruleChain": {
    "id":"root-chain",
    "name":"metadata-aware",
    "root":true
  },
  "metadata": {
    "nodes": [
      {
        "id":"s1",
        "type":"jsTransform",
        "name":"metadata-aware-transform",
        "configuration": {
          "jsScript":"msg['metadataTenant']=metadata['tenant']; msg['metadataEnabled']=metadata['enabled']; return {'msg':msg,'metadata':metadata,'msgType':msgType};"
        }
      }
    ],
    "connections": []
  }
}`

type fakeSkillAgentRunner struct {
	run func(ctx context.Context, req HarnessRequest) (string, error)
}

func (f fakeSkillAgentRunner) ExecuteHarnessSync(ctx context.Context, req HarnessRequest) (string, error) {
	if f.run == nil {
		return "", nil
	}
	return f.run(ctx, req)
}

type fakeRuleChainRepo struct {
	ruleChains  map[string]*entity.RuleChain
	deleteCalls int
	deleteErr   error
	deleteHook  func() error
}

func newFakeRuleChainRepo(ruleChains ...*entity.RuleChain) *fakeRuleChainRepo {
	store := make(map[string]*entity.RuleChain, len(ruleChains))
	for _, rc := range ruleChains {
		store[rc.RuleChainID] = cloneRuleChainEntity(rc)
	}
	return &fakeRuleChainRepo{ruleChains: store}
}

func (f *fakeRuleChainRepo) CreateRuleChain(ctx context.Context, ruleChain *entity.RuleChain) error {
	f.ruleChains[ruleChain.RuleChainID] = cloneRuleChainEntity(ruleChain)
	return nil
}

func (f *fakeRuleChainRepo) UpdateRuleChain(ctx context.Context, where map[string]interface{}, data map[string]interface{}) error {
	id, _ := where["rule_chain_id"].(string)
	rc := f.ruleChains[id]
	if rc == nil {
		return errors.New("rule chain not found")
	}
	if v, ok := data["name"].(string); ok {
		rc.Name = v
	}
	if v, ok := data["root"].(bool); ok {
		rc.Root = v
	}
	if v, ok := data["disabled"].(bool); ok {
		rc.Disabled = v
	}
	if v, ok := data["debug_mode"].(bool); ok {
		rc.DebugMode = v
	}
	if v, ok := data["configuration"].(string); ok {
		value := v
		rc.Configuration = &value
	}
	if v, ok := data["metadata"].(string); ok {
		value := v
		rc.Metadata = &value
	}
	if v, ok := data["additional_info"].(string); ok {
		value := v
		rc.AdditionalInfo = &value
	}
	f.ruleChains[id] = rc
	return nil
}

func (f *fakeRuleChainRepo) DeleteRuleChain(ctx context.Context, where map[string]interface{}) error {
	id, _ := where["rule_chain_id"].(string)
	f.deleteCalls++
	if f.deleteHook != nil {
		if err := f.deleteHook(); err != nil {
			return err
		}
	}
	if f.deleteErr != nil {
		return f.deleteErr
	}
	delete(f.ruleChains, id)
	return nil
}

func (f *fakeRuleChainRepo) FindOneRuleChain(ctx context.Context, where map[string]interface{}) (*entity.RuleChain, error) {
	id, _ := where["rule_chain_id"].(string)
	rc := f.ruleChains[id]
	if rc == nil {
		return nil, errors.New("rule chain not found")
	}
	return cloneRuleChainEntity(rc), nil
}

func (f *fakeRuleChainRepo) FindListRuleChain(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]entity.RuleChain, int64, error) {
	return nil, 0, nil
}

func (f *fakeRuleChainRepo) FindAllRuleChain(ctx context.Context, where map[string]interface{}) ([]entity.RuleChain, error) {
	out := make([]entity.RuleChain, 0, len(f.ruleChains))
	for _, rc := range f.ruleChains {
		out = append(out, *cloneRuleChainEntity(rc))
	}
	return out, nil
}

func (f *fakeRuleChainRepo) mustRuleChain(t *testing.T, id string) *entity.RuleChain {
	t.Helper()
	rc := f.ruleChains[id]
	if rc == nil {
		t.Fatalf("rule chain %q not found", id)
	}
	return cloneRuleChainEntity(rc)
}

func newTestRuleChainUsecase(repo RuleChainRepo, skillRoot string, runner fakeSkillAgentRunner) *RuleChainUsecase {
	return &RuleChainUsecase{
		ruleChainRepo: repo,
		log:           log.NewHelper(log.NewStdLogger(io.Discard)),
		ruleEngine:    rulego.NewRuleGo(),
		skillAgent:    runner,
		skillRoot:     skillRoot,
	}
}

func withRuleChain(id string, root bool, name string, configuration map[string]interface{}) *entity.RuleChain {
	return withRuleChainWithAdditionalInfo(id, root, name, configuration, nil)
}

func withRuleChainWithAdditionalInfo(id string, root bool, name string, configuration map[string]interface{}, additionalInfo map[string]interface{}) *entity.RuleChain {
	return withRuleChainWithMetadataAndAdditionalInfo(id, root, name, configuration, nil, additionalInfo)
}

func withRuleChainWithMetadataAndAdditionalInfo(id string, root bool, name string, configuration map[string]interface{}, metadata map[string]interface{}, additionalInfo map[string]interface{}) *entity.RuleChain {
	cfgBytes, _ := json.Marshal(configuration)
	cfg := string(cfgBytes)
	var metadataPtr *string
	if metadata != nil {
		metadataBytes, _ := json.Marshal(metadata)
		metadataStr := string(metadataBytes)
		metadataPtr = &metadataStr
	}
	var additionalInfoPtr *string
	if additionalInfo != nil {
		additionalInfoBytes, _ := json.Marshal(additionalInfo)
		additionalInfoStr := string(additionalInfoBytes)
		additionalInfoPtr = &additionalInfoStr
	}
	return &entity.RuleChain{
		RuleChainID:    id,
		Root:           root,
		Name:           name,
		Configuration:  &cfg,
		Metadata:       metadataPtr,
		AdditionalInfo: additionalInfoPtr,
	}
}

func mustRuleChainFromEntity(t *testing.T, rc *entity.RuleChain) *types.RuleChain {
	t.Helper()
	uc := newTestRuleChainUsecase(newFakeRuleChainRepo(), t.TempDir(), fakeSkillAgentRunner{})
	out, err := uc.RuleChainDBToRuleChain(rc)
	if err != nil {
		t.Fatalf("RuleChainDBToRuleChain failed: %v", err)
	}
	return out
}

func cloneRuleChainEntity(in *entity.RuleChain) *entity.RuleChain {
	if in == nil {
		return nil
	}
	out := *in
	if in.Configuration != nil {
		cfg := *in.Configuration
		out.Configuration = &cfg
	}
	if in.Metadata != nil {
		meta := *in.Metadata
		out.Metadata = &meta
	}
	if in.AdditionalInfo != nil {
		info := *in.AdditionalInfo
		out.AdditionalInfo = &info
	}
	return &out
}

func parseRuleChainConfiguration(t *testing.T, rc *entity.RuleChain) map[string]interface{} {
	t.Helper()
	if rc.Configuration == nil {
		return map[string]interface{}{}
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(*rc.Configuration), &out); err != nil {
		t.Fatalf("Unmarshal configuration failed: %v", err)
	}
	return out
}

func mustJSONMap(t *testing.T, v interface{}) map[string]interface{} {
	t.Helper()
	m, ok := v.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %#v", v)
	}
	return m
}

func asInt64(v interface{}) int64 {
	switch typed := v.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	default:
		return 0
	}
}

func mustStructPb(t *testing.T, v map[string]interface{}) *structpb.Struct {
	t.Helper()
	out, err := structpb.NewStruct(v)
	if err != nil {
		t.Fatalf("structpb.NewStruct failed: %v", err)
	}
	return out
}
