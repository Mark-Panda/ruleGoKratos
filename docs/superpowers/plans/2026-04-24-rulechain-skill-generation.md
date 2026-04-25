# Rulechain Skill Generation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为每个主规则链增加可同步创建/更新的 Agent-oriented Skill 生成能力，并在规则链删除、Skill 丢失、描述/入参/出参变化时保持状态一致。

**Architecture:** 以后端规则链服务为单一事实源：`rules.proto` 扩展 Skill 状态与生成接口，`RuleChainUsecase` 新增签名、状态判定、Skill 生成与删除同步编排，前端列表页与详情页只展示状态和调用接口。Skill 由托管 Agent 同步执行 `run_skill("skill-creator-0.1.0")` 生成到 `/app/skills/<dir>/SKILL.md`，并把状态元数据回写到 `configuration.flowgram.skill`。

**Tech Stack:** Go / Kratos / protobuf / `make api` / `make validate` / TypeScript / React / Vitest

---

### Task 1: 建立规则链 Skill 元数据与状态判定内核

**Files:**
- Create: `internal/biz/rule_chain_skill.go`
- Create: `internal/biz/rule_chain_skill_test.go`
- Modify: `internal/biz/rule_chain.go`

- [ ] **Step 1: Write the failing test**

```go
func TestResolveRuleChainSkillStatus(t *testing.T) {
	root := t.TempDir()
	meta := RuleChainSkillMeta{
		DirName:       "weather-agent",
		EntryFile:     "SKILL.md",
		Signature:     "sig-current",
		LastGenerated: "sig-current",
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
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# ok"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	status, err = ResolveRuleChainSkillStatus(root, meta, "sig-current")
	if err != nil {
		t.Fatalf("ResolveRuleChainSkillStatus returned error: %v", err)
	}
	if status != RuleChainSkillStatusReady {
		t.Fatalf("expected ready, got %s", status)
	}

	status, err = ResolveRuleChainSkillStatus(root, meta, "sig-next")
	if err != nil {
		t.Fatalf("ResolveRuleChainSkillStatus returned error: %v", err)
	}
	if status != RuleChainSkillStatusStale {
		t.Fatalf("expected stale, got %s", status)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/biz -run TestResolveRuleChainSkillStatus -count=1`

Expected: FAIL，因为 `RuleChainSkillMeta`、`ResolveRuleChainSkillStatus` 与状态常量尚不存在。

- [ ] **Step 3: Write minimal implementation**

```go
type RuleChainSkillStatus string

const (
	RuleChainSkillStatusMissing RuleChainSkillStatus = "missing"
	RuleChainSkillStatusStale   RuleChainSkillStatus = "stale"
	RuleChainSkillStatusReady   RuleChainSkillStatus = "ready"
)

type RuleChainSkillMeta struct {
	DirName       string
	EntryFile     string
	Signature     string
	LastGenerated string
}

func ResolveRuleChainSkillStatus(skillRoot string, meta RuleChainSkillMeta, currentSignature string) (RuleChainSkillStatus, error) {
	if strings.TrimSpace(meta.DirName) == "" {
		return RuleChainSkillStatusMissing, nil
	}
	entry := strings.TrimSpace(meta.EntryFile)
	if entry == "" {
		entry = "SKILL.md"
	}
	abs := filepath.Join(skillRoot, meta.DirName, entry)
	if _, err := os.Stat(abs); err != nil {
		if os.IsNotExist(err) {
			return RuleChainSkillStatusMissing, nil
		}
		return "", err
	}
	if strings.TrimSpace(meta.LastGenerated) != strings.TrimSpace(currentSignature) {
		return RuleChainSkillStatusStale, nil
	}
	return RuleChainSkillStatusReady, nil
}
```

- [ ] **Step 4: Expand the helper coverage**

Add two more tests in `internal/biz/rule_chain_skill_test.go`:

```go
func TestBuildRuleChainSkillSignatureStable(t *testing.T) {
	left := BuildRuleChainSkillSignature("desc", `[{"name":"city"}]`, `[{"name":"query"}]`, `[{"name":"answer"}]`)
	right := BuildRuleChainSkillSignature("desc", `[{"name":"city"}]`, `[{"name":"query"}]`, `[{"name":"answer"}]`)
	if left != right {
		t.Fatalf("expected stable signature, left=%q right=%q", left, right)
	}
}

func TestSanitizeRuleChainSkillDirName(t *testing.T) {
	got := SanitizeRuleChainSkillDirName("Weather Agent / Beijing")
	if got != "weather-agent-beijing" {
		t.Fatalf("expected normalized dir name, got %q", got)
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/biz -run 'Test(ResolveRuleChainSkillStatus|BuildRuleChainSkillSignatureStable|SanitizeRuleChainSkillDirName)' -count=1`

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/biz/rule_chain.go internal/biz/rule_chain_skill.go internal/biz/rule_chain_skill_test.go
git commit -m "feat: add rulechain skill state helpers"
```

### Task 2: 扩规则链同步执行契约与 Skill 管理接口

**Files:**
- Modify: `api/rulego/v1/rules.proto`
- Modify: `api/rulego/v1/rules.pb.go`
- Modify: `api/rulego/v1/rules_grpc.pb.go`
- Modify: `api/rulego/v1/rules_http.pb.go`
- Modify: `api/rulego/v1/rules.pb.validate.go`
- Modify: `internal/service/rules.go`
- Modify: `internal/biz/rule_chain.go`
- Test: `internal/biz/rule_chain_skill_test.go`

- [ ] **Step 1: Write the failing test**

Add a contract-focused test for the request mapper:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/biz -run TestBuildRuleChainSyncExecutePayloadIncludesMetadata -count=1`

Expected: FAIL，因为 metadata-aware 执行载体与 helper 尚不存在。

- [ ] **Step 3: Update proto contract**

Modify `api/rulego/v1/rules.proto`:

```proto
rpc GetRuleChainSkillStatus(GetRuleChainSkillStatusReq) returns (GetRuleChainSkillStatusReply) {
  option (google.api.http) = {
    get: "/api/v1/rules/{id}/skill/status"
  };
}

rpc GenerateRuleChainSkill(GenerateRuleChainSkillReq) returns (GenerateRuleChainSkillReply) {
  option (google.api.http) = {
    post: "/api/v1/rules/{id}/skill/generate"
    body: "*"
  };
}

message ExecuteRuleChainReq {
  string id = 1;
  string msgType = 2;
  bool debugMode = 3;
  string msgId = 4;
  google.protobuf.Struct metadata = 5;
  google.protobuf.Struct data = 6;
}
```

Also add:

```proto
message GetRuleChainSkillStatusReq { string id = 1; }
message GetRuleChainSkillStatusReply {
  string status = 1;
  string dir_name = 2 [json_name = "dirName"];
  string entry_file = 3 [json_name = "entryFile"];
  string signature = 4;
  string generated_at = 5 [json_name = "generatedAt"];
  int64 generated_by_managed_agent_id = 6 [json_name = "generatedByManagedAgentId"];
  string last_error = 7 [json_name = "lastError"];
}
message GenerateRuleChainSkillReq {
  string id = 1;
  int64 managed_agent_id = 2 [json_name = "managedAgentId"];
}
message GenerateRuleChainSkillReply {
  string status = 1;
  string dir_name = 2 [json_name = "dirName"];
}
```

- [ ] **Step 4: Regenerate protobuf artifacts**

Run:

```bash
make api && make validate
```

Expected: `api/rulego/v1/rules*.go` regenerated with new RPCs and `ExecuteRuleChainReq.metadata`.

- [ ] **Step 5: Write minimal implementation**

Implement the request mapper and service forwarding:

```go
func BuildRuleChainSyncExecutePayload(metadata, data map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"metadata": cloneJSONMap(metadata),
		"data":     cloneJSONMap(data),
	}, nil
}

func (s *RuleGoService) GetRuleChainSkillStatus(ctx context.Context, in *v1.GetRuleChainSkillStatusReq) (*v1.GetRuleChainSkillStatusReply, error) {
	return s.rc.GetRuleChainSkillStatus(ctx, in)
}

func (s *RuleGoService) GenerateRuleChainSkill(ctx context.Context, in *v1.GenerateRuleChainSkillReq) (*v1.GenerateRuleChainSkillReply, error) {
	return s.rc.GenerateRuleChainSkill(ctx, in)
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run:

```bash
go test ./internal/biz -run TestBuildRuleChainSyncExecutePayloadIncludesMetadata -count=1
go test ./api/rulego/v1/... ./internal/service/... -count=1
```

Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add api/rulego/v1/rules.proto api/rulego/v1/rules.pb.go api/rulego/v1/rules_grpc.pb.go api/rulego/v1/rules_http.pb.go api/rulego/v1/rules.pb.validate.go internal/service/rules.go internal/biz/rule_chain.go internal/biz/rule_chain_skill_test.go
git commit -m "feat: add rulechain skill service contract"
```

### Task 3: 实现 Skill 状态查询与同步生成编排

**Files:**
- Create: `internal/biz/rule_chain_skill_prompt.go`
- Create: `internal/biz/rule_chain_skill_generation_test.go`
- Modify: `internal/biz/rule_chain.go`
- Modify: `internal/biz/biz.go`
- Modify: `internal/service/rules.go`
- Modify: `cmd/ruleGoKratos/wire_gen.go`

- [ ] **Step 1: Write the failing test**

Create a focused generation orchestration test with temp dir + fake agent runner:

```go
func TestGenerateRuleChainSkillWritesSkillAndUpdatesConfig(t *testing.T) {
	root := t.TempDir()
	repo := newFakeRuleChainRepo(withRuleChain(
		"root-chain",
		true,
		"天气查询",
		map[string]interface{}{
			"flowgram": map[string]interface{}{
				"description": "根据城市查询天气",
				"io": map[string]interface{}{
					"request_metadata_params": []interface{}{map[string]interface{}{"name": "tenant"}},
					"request_message_body_params": []interface{}{map[string]interface{}{"name": "city"}},
					"response_message_body_params": []interface{}{map[string]interface{}{"name": "summary"}},
				},
			},
		},
	))
	uc := newTestRuleChainUsecase(repo, root, fakeSkillAgentRunner{
		run: func(targetDir string) error {
			return os.WriteFile(filepath.Join(root, targetDir, "SKILL.md"), []byte("# generated"), 0o644)
		},
	})

	reply, err := uc.GenerateRuleChainSkill(context.Background(), &v1.GenerateRuleChainSkillReq{
		Id:             "root-chain",
		ManagedAgentId: 101,
	})
	if err != nil {
		t.Fatalf("GenerateRuleChainSkill failed: %v", err)
	}
	if reply.GetStatus() != "ready" {
		t.Fatalf("expected ready, got %q", reply.GetStatus())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/biz -run TestGenerateRuleChainSkillWritesSkillAndUpdatesConfig -count=1`

Expected: FAIL，因为生成编排、fake agent 接口与配置回写尚不存在。

- [ ] **Step 3: Add narrow dependencies for the usecase**

Refactor constructor inputs so `RuleChainUsecase` receives:

```go
type RuleChainSkillAgentRunner interface {
	ExecuteHarnessSync(ctx context.Context, req HarnessRequest) (string, error)
}
```

and config-backed skill root resolution:

```go
func resolveRuleChainSkillRoot(config *conf.Bootstrap) string
```

Update `NewRuleChainUsecase(...)` and `internal/biz/biz.go` / `wire` wiring accordingly.

- [ ] **Step 4: Write minimal implementation**

Implement:

```go
func (s *RuleChainUsecase) GetRuleChainSkillStatus(ctx context.Context, in *v1.GetRuleChainSkillStatusReq) (*v1.GetRuleChainSkillStatusReply, error)
func (s *RuleChainUsecase) GenerateRuleChainSkill(ctx context.Context, in *v1.GenerateRuleChainSkillReq) (*v1.GenerateRuleChainSkillReply, error)
```

Generation should:

```go
dirName := chooseExistingOrGeneratedDirName(...)
prompt := BuildRuleChainSkillGenerationPrompt(ruleChain, dirName)
_, err := s.skillAgent.ExecuteHarnessSync(ctx, HarnessRequest{
	ManagedAgentID: in.GetManagedAgentId(),
	Input:          prompt,
})
status, err := ResolveRuleChainSkillStatus(s.skillRoot, meta, currentSignature)
```

Prompt builder must explicitly require:

```go
必须调用 run_skill
skill_name 固定为 skill-creator-0.1.0
输出文件固定为 /app/skills/<dir>/SKILL.md
Skill 面向 Agent，不写面向终端用户的操作说明
必须写清 metadata/data 整理规则、同步执行接口、结果解释、失败兜底
```

- [ ] **Step 5: Run tests to verify they pass**

Run:

```bash
go test ./internal/biz -run 'Test(GenerateRuleChainSkillWritesSkillAndUpdatesConfig|ResolveRuleChainSkillStatus|BuildRuleChainSkillSignatureStable)' -count=1
go test ./internal/biz/... -count=1
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/biz/rule_chain.go internal/biz/rule_chain_skill.go internal/biz/rule_chain_skill_prompt.go internal/biz/rule_chain_skill_test.go internal/biz/rule_chain_skill_generation_test.go internal/biz/biz.go internal/service/rules.go cmd/ruleGoKratos/wire_gen.go
git commit -m "feat: generate rulechain skills with managed agents"
```

### Task 4: 删除规则链时同步删除 Skill 目录

**Files:**
- Modify: `internal/biz/rule_chain.go`
- Modify: `internal/biz/rule_chain_skill.go`
- Modify: `internal/biz/rule_chain_skill_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestDeleteRuleChainRemovesSkillDirectory(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "weather-agent")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# ok"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	repo := newFakeRuleChainRepo(withRuleChain(
		"root-chain", true, "天气查询",
		map[string]interface{}{"flowgram": map[string]interface{}{"skill": map[string]interface{}{"dir_name": "weather-agent"}}},
	))
	uc := newTestRuleChainUsecase(repo, root, fakeSkillAgentRunner{})

	if _, err := uc.DeleteRuleChain(context.Background(), &v1.DeleteRuleChainReq{Id: "root-chain"}); err != nil {
		t.Fatalf("DeleteRuleChain failed: %v", err)
	}
	if _, err := os.Stat(skillDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected skill dir removed, stat err=%v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/biz -run TestDeleteRuleChainRemovesSkillDirectory -count=1`

Expected: FAIL，因为当前 `DeleteRuleChain` 只删引擎与数据库，不删 Skill 目录。

- [ ] **Step 3: Write minimal implementation**

```go
func (s *RuleChainUsecase) deleteRuleChainSkillDir(ruleChain *types.RuleChain) error {
	dirName := ExtractRuleChainSkillMeta(ruleChain.RuleChain.Configuration).DirName
	if strings.TrimSpace(dirName) == "" {
		return nil
	}
	abs := filepath.Join(s.skillRoot, dirName)
	if _, err := os.Stat(abs); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.RemoveAll(abs)
}
```

Call it before repo deletion inside `DeleteRuleChain`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/biz -run TestDeleteRuleChainRemovesSkillDirectory -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/biz/rule_chain.go internal/biz/rule_chain_skill.go internal/biz/rule_chain_skill_test.go
git commit -m "feat: remove rulechain skill directories on delete"
```

### Task 5: 前端接入默认托管 Agent、状态展示与生成按钮

**Files:**
- Create: `flowgram/src/utils/managed-agent-storage.ts`
- Create: `flowgram/src/utils/__tests__/managed-agent-storage.spec.ts`
- Modify: `flowgram/src/management/sections/OverviewChatSection.tsx`
- Modify: `flowgram/src/services/api-rules.ts`
- Modify: `flowgram/src/management/sections/WorkflowSection.tsx`
- Modify: `flowgram/src/management/rule-detail.tsx`

- [ ] **Step 1: Write the failing test**

Create a small storage helper test:

```ts
import { describe, expect, it, vi } from 'vitest';
import { loadStoredManagedAgentId, saveStoredManagedAgentId } from '../managed-agent-storage';

describe('managed-agent-storage', () => {
  it('loads and saves overview chat managed agent id', () => {
    const store = new Map<string, string>();
    vi.stubGlobal('localStorage', {
      getItem: (k: string) => store.get(k) ?? null,
      setItem: (k: string, v: string) => void store.set(k, v),
      removeItem: (k: string) => void store.delete(k),
    });

    saveStoredManagedAgentId(23);
    expect(loadStoredManagedAgentId()).toBe(23);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pnpm --dir flowgram exec vitest run src/utils/__tests__/managed-agent-storage.spec.ts`

Expected: FAIL，因为共享 helper 文件还不存在。

- [ ] **Step 3: Write minimal implementation**

Extract the storage key from `OverviewChatSection.tsx` into a shared helper:

```ts
export const STORAGE_MANAGED_AGENT_KEY = 'flowgram-overview-chat-managed-agent-v1';

export function loadStoredManagedAgentId(): number { /* existing logic */ }
export function saveStoredManagedAgentId(id: number): void { /* existing logic */ }
```

Extend `api-rules.ts`:

```ts
export interface RuleChainSkillStatusReply {
  status: 'missing' | 'stale' | 'ready';
  dirName?: string;
  entryFile?: string;
}

export const getRuleChainSkillStatus = (id: string) =>
  requestJSON<RuleChainSkillStatusReply>(`/rules/${encodeURIComponent(id)}/skill/status`);

export const generateRuleChainSkill = (id: string, managedAgentId: number) =>
  requestJSON(`/rules/${encodeURIComponent(id)}/skill/generate`, {
    method: 'POST',
    body: { id, managedAgentId },
  });
```

Wire UI:

```tsx
const storedManagedAgentId = loadStoredManagedAgentId();
const label = skillStatus === 'missing' ? '创建技能' : '更新技能';
```

When no stored agent exists, open a simple `Select + Modal` fed by `listManagedAgents()`.

- [ ] **Step 4: Run tests and type-check**

Run:

```bash
pnpm --dir flowgram exec vitest run src/utils/__tests__/managed-agent-storage.spec.ts
pnpm --dir flowgram run ts-check
```

Expected: PASS

- [ ] **Step 5: Manually verify the UI flow**

Check:

```text
1. 主流程列表显示技能按钮，子流程不显示
2. 详情页显示相同按钮
3. localStorage 里已有 managed agent 时可直接发起生成
4. 没有 managed agent 时弹选择框
5. 生成成功后按钮状态刷新
```

- [ ] **Step 6: Commit**

```bash
git add flowgram/src/utils/managed-agent-storage.ts flowgram/src/utils/__tests__/managed-agent-storage.spec.ts flowgram/src/management/sections/OverviewChatSection.tsx flowgram/src/services/api-rules.ts flowgram/src/management/sections/WorkflowSection.tsx flowgram/src/management/rule-detail.tsx
git commit -m "feat: add rulechain skill controls to management UI"
```

### Task 6: 全量验证与回归检查

**Files:**
- Modify: `api/rulego/v1/rules.proto`
- Modify: `internal/biz/rule_chain.go`
- Modify: `internal/biz/rule_chain_skill.go`
- Modify: `internal/service/rules.go`
- Modify: `flowgram/src/services/api-rules.ts`
- Modify: `flowgram/src/management/sections/WorkflowSection.tsx`
- Modify: `flowgram/src/management/rule-detail.tsx`

- [ ] **Step 1: Run focused backend tests**

```bash
go test ./internal/biz -run 'Test(ResolveRuleChainSkillStatus|BuildRuleChainSkillSignatureStable|BuildRuleChainSyncExecutePayloadIncludesMetadata|GenerateRuleChainSkillWritesSkillAndUpdatesConfig|DeleteRuleChainRemovesSkillDirectory)' -count=1
```

- [ ] **Step 2: Run backend package tests**

```bash
go test ./internal/biz/... ./internal/service/... -count=1
```

- [ ] **Step 3: Run protobuf and wire integrity checks**

```bash
make api && make validate && make wire
```

- [ ] **Step 4: Run frontend test and type check**

```bash
pnpm --dir flowgram exec vitest run src/utils/__tests__/managed-agent-storage.spec.ts
pnpm --dir flowgram run ts-check
```

- [ ] **Step 5: Manual end-to-end verification**

```text
1. 新建一个主流程，填写描述 + metadata/data 入参 + 出参
2. 点击“创建技能”，选择托管 Agent，确认 /app/skills/<dir>/SKILL.md 被生成
3. 修改描述，再次进入页面，按钮状态变为“更新技能”或显示“待更新”
4. 手工删除 /app/skills/<dir>/SKILL.md，刷新后状态变为“创建技能”
5. 删除规则链，确认 /app/skills/<dir>/ 被同步删除
6. 检查生成的 SKILL.md 包含规则链 ID、同步执行接口、metadata/data 整理规则、结果解释
```

- [ ] **Step 6: Commit**

```bash
git add api/rulego/v1/rules.proto api/rulego/v1/rules.pb.go api/rulego/v1/rules_grpc.pb.go api/rulego/v1/rules_http.pb.go api/rulego/v1/rules.pb.validate.go internal/biz/rule_chain.go internal/biz/rule_chain_skill.go internal/biz/rule_chain_skill_prompt.go internal/biz/rule_chain_skill_test.go internal/biz/rule_chain_skill_generation_test.go internal/service/rules.go flowgram/src/utils/managed-agent-storage.ts flowgram/src/utils/__tests__/managed-agent-storage.spec.ts flowgram/src/management/sections/OverviewChatSection.tsx flowgram/src/services/api-rules.ts flowgram/src/management/sections/WorkflowSection.tsx flowgram/src/management/rule-detail.tsx
git commit -m "feat: add managed skill generation for root rulechains"
```
