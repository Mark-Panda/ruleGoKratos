# Collaboration Runtime Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 Agent Playground 的四种协作模式从四套独立 handler 迁移为统一的 Typed Execution Plan Runtime，并同步补齐运行页、恢复交互和后端 API 契约。

**Architecture:** 保留 `WorkflowService` 作为外部入口，在 `internal/biz/playground/` 下新增 `planbuilder / runtime / artifact / recovery / eventlog` 模块。四种模式先编译为 `ExecutionPlan`，再由统一 runtime 执行；失败后进入 `waiting_recovery` 并暴露恢复动作。前端保持现有三栏布局，但左栏改为真实 plan 视图，中栏加入 recovery，右栏扩展为 trace / artifacts / recovery 多视图。

**Tech Stack:** Go 1.24, Kratos HTTP service, GORM/PostgreSQL repo, existing Harness runtime, React 18, Semi UI, TypeScript, Vitest, npm scripts

---

## File Structure

- Create: `internal/biz/entity/playground_runtime.go`（`ExecutionPlan / Run / Step / Artifact / RecoveryAction` 定义）
- Create: `internal/biz/playground/planbuilder/builder.go`（PlanBuilder 接口与注册中心）
- Create: `internal/biz/playground/planbuilder/router_expert.go`
- Create: `internal/biz/playground/planbuilder/plan_exec.go`
- Create: `internal/biz/playground/planbuilder/supervision.go`
- Create: `internal/biz/playground/planbuilder/peer_handoff.go`
- Create: `internal/biz/playground/runtime/service.go`（runtime 主循环）
- Create: `internal/biz/playground/runtime/executor_route.go`
- Create: `internal/biz/playground/runtime/executor_agent.go`
- Create: `internal/biz/playground/runtime/executor_parallel.go`
- Create: `internal/biz/playground/runtime/executor_review.go`
- Create: `internal/biz/playground/runtime/executor_handoff.go`
- Create: `internal/biz/playground/runtime/executor_finalize.go`
- Create: `internal/biz/playground/runtime/repo.go`（Run / Step / Artifact / Event / Recovery 仓储接口）
- Create: `internal/biz/playground/recovery/service.go`
- Create: `internal/biz/playground/runtime/runtime_test.go`
- Create: `internal/biz/playground/planbuilder/planbuilder_test.go`
- Modify: `internal/biz/playground/workflow/service.go`（从 handler.Execute 迁移到 builder + runtime）
- Modify: `internal/biz/playground/collaboration/harness_runner.go`（作为 `AgentStepExecutor` 适配层复用）
- Modify: `internal/service/playground.go`（新增 run/steps/artifacts/recovery 接口）
- Create: `internal/data/playground/runtime_repo.go`（内存 repo，供首批开发和测试）
- Create: `internal/data/playground/runtime_db_repo.go`（数据库实现）
- Create: `flowgram/src/agent-playground/utils/runtime-view-model.ts`（前端 plan/run 显示模型）
- Create: `flowgram/src/agent-playground/utils/__tests__/runtime-view-model.spec.ts`
- Modify: `flowgram/src/services/api-playground.ts`（新增 runtime API 类型与请求）
- Modify: `flowgram/src/agent-playground/components/workflow-graph.tsx`
- Modify: `flowgram/src/agent-playground/components/run-console.tsx`
- Modify: `flowgram/src/agent-playground/components/trace-panel.tsx`
- Modify: `flowgram/src/agent-playground/pages/playground-page.tsx`

---

### Task 1: 建立运行时核心实体与仓储接口

**Files:**
- Create: `internal/biz/entity/playground_runtime.go`
- Create: `internal/biz/playground/runtime/repo.go`
- Test: `internal/biz/playground/runtime/runtime_test.go`

- [ ] **Step 1: 先写失败测试（RunState / StepState / RecoveryAction 基础语义）**

```go
package runtime_test

import (
	"testing"

	"ruleGoKratos/internal/biz/entity"
)

func TestRuntimeEntitiesExposeRecoveryState(t *testing.T) {
	run := &entity.PlaygroundRun{
		RunID:  "run-1",
		Status: entity.RunStatusWaitingRecovery,
	}
	if run.Status != entity.RunStatusWaitingRecovery {
		t.Fatalf("expected waiting_recovery, got %s", run.Status)
	}

	action := entity.RecoveryAction{
		ID:     "ra-1",
		RunID:  "run-1",
		Type:   entity.RecoveryActionRetryStep,
		StepID: "step-1",
	}
	if action.Type != entity.RecoveryActionRetryStep {
		t.Fatalf("expected retry_step, got %s", action.Type)
	}
}
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `go test ./internal/biz/playground/runtime -run TestRuntimeEntitiesExposeRecoveryState -count=1`  
Expected: FAIL，提示 `undefined: entity.PlaygroundRun` 或 `undefined: entity.RunStatusWaitingRecovery`

- [ ] **Step 3: 写最小实体与 repo 接口**

```go
// internal/biz/entity/playground_runtime.go
package entity

import "time"

type RunStatus string
type StepStatus string
type StepKind string
type RecoveryActionType string

const (
	RunStatusPending         RunStatus = "pending"
	RunStatusReady           RunStatus = "ready"
	RunStatusRunning         RunStatus = "running"
	RunStatusWaitingRecovery RunStatus = "waiting_recovery"
	RunStatusCompleted       RunStatus = "completed"
	RunStatusFailed          RunStatus = "failed"
	RunStatusCancelled       RunStatus = "cancelled"
)

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusReady     StepStatus = "ready"
	StepStatusRunning   StepStatus = "running"
	StepStatusSucceeded StepStatus = "succeeded"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
)

const (
	StepKindRoute    StepKind = "route"
	StepKindAgent    StepKind = "agent"
	StepKindParallel StepKind = "parallel"
	StepKindReview   StepKind = "review"
	StepKindHandoff  StepKind = "handoff"
	StepKindFinalize StepKind = "finalize"
)

const (
	RecoveryActionRetryStep          RecoveryActionType = "retry_step"
	RecoveryActionRerouteStep        RecoveryActionType = "reroute_step"
	RecoveryActionSkipStep           RecoveryActionType = "skip_step"
	RecoveryActionRetryFromCheckpoint RecoveryActionType = "retry_from_checkpoint"
)

type ExecutionPlan struct {
	PlanID       string         `json:"planId"`
	PlanVersion  int            `json:"planVersion"`
	SourceMode   string         `json:"sourceMode"`
	EntryStepIDs []string       `json:"entryStepIds"`
	Steps        []*PlanStep    `json:"steps"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type PlanStep struct {
	StepID       string         `json:"stepId"`
	Kind         StepKind       `json:"kind"`
	Name         string         `json:"name"`
	DependsOn    []string       `json:"dependsOn,omitempty"`
	AgentBinding string         `json:"agentBinding,omitempty"`
	InputRefs    []string       `json:"inputRefs,omitempty"`
	OutputRef    string         `json:"outputRef,omitempty"`
	Config       map[string]any `json:"config,omitempty"`
}

type PlaygroundRun struct {
	RunID             string         `json:"runId"`
	SchemeID          string         `json:"schemeId"`
	PlanID            string         `json:"planId"`
	Status            RunStatus      `json:"status"`
	InputArtifactID   string         `json:"inputArtifactId,omitempty"`
	LastCheckpointID  string         `json:"lastCheckpointId,omitempty"`
	FailureSummary    string         `json:"failureSummary,omitempty"`
	StartedAt         *time.Time     `json:"startedAt,omitempty"`
	FinishedAt        *time.Time     `json:"finishedAt,omitempty"`
}

type RuntimeStep struct {
	RunID           string         `json:"runId"`
	StepID          string         `json:"stepId"`
	Kind            StepKind       `json:"kind"`
	Name            string         `json:"name"`
	Status          StepStatus     `json:"status"`
	AgentBinding    string         `json:"agentBinding,omitempty"`
	FailureSummary  string         `json:"failureSummary,omitempty"`
	InputRefs       []string       `json:"inputRefs,omitempty"`
	OutputRef       string         `json:"outputRef,omitempty"`
	CheckpointAfter bool           `json:"checkpointAfter,omitempty"`
}

type RuntimeArtifact struct {
	ArtifactID      string         `json:"artifactId"`
	RunID           string         `json:"runId"`
	Type            string         `json:"type"`
	ProducerStepID  string         `json:"producerStepId"`
	SchemaVersion   int            `json:"schemaVersion"`
	Payload         map[string]any `json:"payload,omitempty"`
	Summary         string         `json:"summary"`
	CreatedAt       *time.Time     `json:"createdAt,omitempty"`
}

type RecoveryAction struct {
	ID        string             `json:"id"`
	RunID     string             `json:"runId"`
	StepID    string             `json:"stepId"`
	Type      RecoveryActionType `json:"type"`
	TargetRef string             `json:"targetRef,omitempty"`
	Reason    string             `json:"reason,omitempty"`
}
```

```go
// internal/biz/playground/runtime/repo.go
package runtime

import (
	"context"

	"ruleGoKratos/internal/biz/entity"
)

type Repo interface {
	SavePlan(ctx context.Context, plan *entity.ExecutionPlan) error
	SaveRun(ctx context.Context, run *entity.PlaygroundRun) error
	UpdateRun(ctx context.Context, run *entity.PlaygroundRun) error
	SaveSteps(ctx context.Context, steps []*entity.RuntimeStep) error
	UpdateStep(ctx context.Context, step *entity.RuntimeStep) error
	SaveArtifact(ctx context.Context, artifact *entity.RuntimeArtifact) error
	ListArtifacts(ctx context.Context, runID string) ([]*entity.RuntimeArtifact, error)
	SaveRecoveryActions(ctx context.Context, actions []*entity.RecoveryAction) error
	ListRecoveryActions(ctx context.Context, runID string) ([]*entity.RecoveryAction, error)
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/biz/playground/runtime -run TestRuntimeEntitiesExposeRecoveryState -count=1`  
Expected: PASS，`TestRuntimeEntitiesExposeRecoveryState` 通过

- [ ] **Step 5: 提交本任务**

```bash
git add internal/biz/entity/playground_runtime.go internal/biz/playground/runtime/repo.go internal/biz/playground/runtime/runtime_test.go
git commit -m "feat(playground): add runtime entities and repository contracts"
```

---

### Task 2: 搭建 runtime 外壳并迁移 `router_expert`

**Files:**
- Create: `internal/biz/playground/planbuilder/builder.go`
- Create: `internal/biz/playground/planbuilder/router_expert.go`
- Create: `internal/biz/playground/runtime/service.go`
- Create: `internal/biz/playground/runtime/executor_route.go`
- Create: `internal/biz/playground/runtime/executor_agent.go`
- Modify: `internal/biz/playground/workflow/service.go`
- Modify: `internal/biz/playground/collaboration/harness_runner.go`
- Test: `internal/biz/playground/planbuilder/planbuilder_test.go`
- Test: `internal/biz/playground/runtime/runtime_test.go`

- [ ] **Step 1: 先写失败测试（router_expert 编译为 `route -> agent -> finalize`）**

```go
package planbuilder_test

import (
	"testing"

	"ruleGoKratos/internal/biz/entity"
	"ruleGoKratos/internal/biz/playground/planbuilder"
)

func TestRouterExpertBuilderBuildsThreeSteps(t *testing.T) {
	builder := planbuilder.NewRouterExpertBuilder()
	scheme := &entity.CollaborationScheme{
		ID:   "scheme-1",
		Mode: entity.ModeRouterExpert,
		BindAgents: []*entity.AgentBinding{
			{AgentID: "designer", Role: "设计师"},
			{AgentID: "engineer", Role: "工程师"},
		},
	}
	plan, err := builder.Build(scheme, "做一个登录页")
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if got, want := len(plan.Steps), 3; got != want {
		t.Fatalf("expected %d steps, got %d", want, got)
	}
	if plan.Steps[0].Kind != entity.StepKindRoute || plan.Steps[1].Kind != entity.StepKindAgent || plan.Steps[2].Kind != entity.StepKindFinalize {
		t.Fatalf("unexpected step kinds: %#v", plan.Steps)
	}
}
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `go test ./internal/biz/playground/planbuilder -run TestRouterExpertBuilderBuildsThreeSteps -count=1`  
Expected: FAIL，提示 `undefined: planbuilder.NewRouterExpertBuilder`

- [ ] **Step 3: 实现 Builder、Runtime 与 WorkflowService 的最小接入**

```go
// internal/biz/playground/planbuilder/builder.go
package planbuilder

import "ruleGoKratos/internal/biz/entity"

type Builder interface {
	Mode() entity.CollaborationMode
	Build(scheme *entity.CollaborationScheme, userInput string) (*entity.ExecutionPlan, error)
}

type Registry struct {
	builders map[entity.CollaborationMode]Builder
}

func NewRegistry() *Registry {
	return &Registry{builders: map[entity.CollaborationMode]Builder{}}
}

func (r *Registry) Register(b Builder) { r.builders[b.Mode()] = b }
func (r *Registry) Get(mode entity.CollaborationMode) (Builder, bool) {
	b, ok := r.builders[mode]
	return b, ok
}
```

```go
// internal/biz/playground/planbuilder/router_expert.go
package planbuilder

import (
	"fmt"

	"github.com/google/uuid"
	"ruleGoKratos/internal/biz/entity"
)

type RouterExpertBuilder struct{}

func NewRouterExpertBuilder() *RouterExpertBuilder { return &RouterExpertBuilder{} }
func (b *RouterExpertBuilder) Mode() entity.CollaborationMode { return entity.ModeRouterExpert }

func (b *RouterExpertBuilder) Build(scheme *entity.CollaborationScheme, _ string) (*entity.ExecutionPlan, error) {
	if scheme == nil {
		return nil, fmt.Errorf("scheme is nil")
	}
	planID := uuid.NewString()
	return &entity.ExecutionPlan{
		PlanID:       planID,
		PlanVersion:  1,
		SourceMode:   string(entity.ModeRouterExpert),
		EntryStepIDs: []string{"route"},
		Steps: []*entity.PlanStep{
			{StepID: "route", Kind: entity.StepKindRoute, Name: "route", OutputRef: "route_result"},
			{StepID: "agent", Kind: entity.StepKindAgent, Name: "agent", DependsOn: []string{"route"}, InputRefs: []string{"route_result"}, OutputRef: "agent_output"},
			{StepID: "finalize", Kind: entity.StepKindFinalize, Name: "finalize", DependsOn: []string{"agent"}, InputRefs: []string{"agent_output"}, OutputRef: "final_output"},
		},
	}, nil
}
```

```go
// internal/biz/playground/workflow/service.go (核心接法)
builder, ok := s.builderRegistry.Get(scheme.Mode)
if !ok {
	return "", fmt.Errorf("builder not found for mode: %s", scheme.Mode)
}
plan, err := builder.Build(scheme, userInput)
if err != nil {
	return "", fmt.Errorf("build plan: %w", err)
}
runID, err := s.runtimeSvc.Run(execCtx, plan, scheme, pool, userInput, traceSink)
if err != nil {
	return "", fmt.Errorf("runtime run: %w", err)
}
return runID, nil
```

- [ ] **Step 4: 运行 Builder 和 Runtime 的最小测试**

Run: `go test ./internal/biz/playground/planbuilder ./internal/biz/playground/runtime -run 'TestRouterExpertBuilderBuildsThreeSteps|TestRuntimeRunRouterPlan' -count=1`  
Expected: PASS，Builder 测试和 Router 运行时最小路径测试通过

- [ ] **Step 5: 提交本任务**

```bash
git add internal/biz/playground/planbuilder internal/biz/playground/runtime internal/biz/playground/workflow/service.go internal/biz/playground/collaboration/harness_runner.go
git commit -m "refactor(playground): route router expert through plan builder and runtime"
```

---

### Task 3: 增加 `plan_exec / supervision / peer_handoff` Step 执行能力

**Files:**
- Create: `internal/biz/playground/planbuilder/plan_exec.go`
- Create: `internal/biz/playground/planbuilder/supervision.go`
- Create: `internal/biz/playground/planbuilder/peer_handoff.go`
- Create: `internal/biz/playground/runtime/executor_parallel.go`
- Create: `internal/biz/playground/runtime/executor_review.go`
- Create: `internal/biz/playground/runtime/executor_handoff.go`
- Create: `internal/biz/playground/runtime/executor_finalize.go`
- Test: `internal/biz/playground/planbuilder/planbuilder_test.go`
- Test: `internal/biz/playground/runtime/runtime_test.go`

- [ ] **Step 1: 先写失败测试（plan_exec / supervision / peer_handoff 的 plan 形状）**

```go
func TestPlanExecBuilderUsesSequentialAgents(t *testing.T) {
	builder := planbuilder.NewPlanExecBuilder()
	scheme := &entity.CollaborationScheme{
		Mode: entity.ModePlanExec,
		BindAgents: []*entity.AgentBinding{
			{AgentID: "planner", Role: "规划师"},
			{AgentID: "designer", Role: "设计师"},
			{AgentID: "engineer", Role: "工程师"},
		},
	}
	plan, err := builder.Build(scheme, "做一个搜索页")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Steps[0].Kind != entity.StepKindAgent || plan.Steps[len(plan.Steps)-1].Kind != entity.StepKindFinalize {
		t.Fatalf("unexpected step sequence")
	}
}

func TestSupervisionBuilderUsesParallelAndReview(t *testing.T) {
	builder := planbuilder.NewSupervisionBuilder()
	plan, err := builder.Build(&entity.CollaborationScheme{
		Mode: entity.ModeSupervision,
		BindAgents: []*entity.AgentBinding{
			{AgentID: "supervisor", Role: "监督者"},
			{AgentID: "designer", Role: "设计师"},
			{AgentID: "engineer", Role: "工程师"},
		},
	}, "并发分析")
	if err != nil {
		t.Fatal(err)
	}
	kinds := []entity.StepKind{plan.Steps[0].Kind, plan.Steps[1].Kind, plan.Steps[2].Kind, plan.Steps[3].Kind}
	want := []entity.StepKind{entity.StepKindReview, entity.StepKindParallel, entity.StepKindReview, entity.StepKindFinalize}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("step %d kind = %s, want %s", i, kinds[i], want[i])
		}
	}
}

func TestPeerHandoffBuilderIncludesHandoffStep(t *testing.T) {
	builder := planbuilder.NewPeerHandoffBuilder()
	plan, err := builder.Build(&entity.CollaborationScheme{
		Mode: entity.ModePeerHandoff,
		BindAgents: []*entity.AgentBinding{
			{AgentID: "designer", Role: "设计师"},
			{AgentID: "pm", Role: "产品经理"},
			{AgentID: "engineer", Role: "工程师"},
		},
	}, "开始接力")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, step := range plan.Steps {
		if step.Kind == entity.StepKindHandoff {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected handoff step")
	}
}
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `go test ./internal/biz/playground/planbuilder -run 'TestPlanExecBuilderUsesSequentialAgents|TestSupervisionBuilderUsesParallelAndReview|TestPeerHandoffBuilderIncludesHandoffStep' -count=1`  
Expected: FAIL，缺少对应 Builder 或 StepKind 断言不成立

- [ ] **Step 3: 实现三个 Builder 与并发/评审/交接执行器**

```go
// supervision builder 的关键 plan 结构
steps := []*entity.PlanStep{
	{StepID: "supervisor_assign", Kind: entity.StepKindReview, Name: "supervisor_assign", OutputRef: "assignment"},
	{
		StepID:    "workers",
		Kind:      entity.StepKindParallel,
		Name:      "workers",
		DependsOn: []string{"supervisor_assign"},
		InputRefs: []string{"assignment"},
		OutputRef: "worker_results",
		Config: map[string]any{
			"workers": []string{"designer", "engineer"},
		},
	},
	{StepID: "supervisor_review", Kind: entity.StepKindReview, Name: "supervisor_review", DependsOn: []string{"workers"}, InputRefs: []string{"worker_results"}, OutputRef: "review_result"},
	{StepID: "finalize", Kind: entity.StepKindFinalize, Name: "finalize", DependsOn: []string{"supervisor_review"}, InputRefs: []string{"review_result"}, OutputRef: "final_output"},
}
```

```go
// peer handoff executor 的关键输出结构
type HandoffDecision struct {
	NextAgent      string `json:"nextAgent"`
	HandoffReason  string `json:"handoffReason"`
	PayloadSummary string `json:"payloadSummary"`
	StopOrContinue string `json:"stopOrContinue"`
}

func (e *HandoffStepExecutor) Execute(ctx context.Context, step *entity.RuntimeStep, rtCtx *ExecutionContext) error {
	decision := HandoffDecision{
		NextAgent:      "engineer",
		HandoffReason:  "设计完成后交给工程实现",
		PayloadSummary: "页面布局与交互要求已经明确",
		StopOrContinue: "continue",
	}
	return rtCtx.Artifacts.WriteJSON(ctx, step.RunID, step.StepID, "handoff_payload", decision)
}
```

- [ ] **Step 4: 运行 planbuilder 与 runtime 测试确认通过**

Run: `go test ./internal/biz/playground/planbuilder ./internal/biz/playground/runtime -run 'TestPlanExecBuilderUsesSequentialAgents|TestSupervisionBuilderUsesParallelAndReview|TestPeerHandoffBuilderIncludesHandoffStep|TestRuntimeRecoveryActionsOnFailure' -count=1`  
Expected: PASS，三个 Builder 测试和失败后生成 recovery actions 的 runtime 测试通过

- [ ] **Step 5: 提交本任务**

```bash
git add internal/biz/playground/planbuilder internal/biz/playground/runtime
git commit -m "feat(playground): add sequential parallel and handoff runtime steps"
```

---

### Task 4: 提供 run/steps/artifacts/recovery 后端契约与持久化

**Files:**
- Create: `internal/data/playground/runtime_repo.go`
- Create: `internal/data/playground/runtime_db_repo.go`
- Modify: `internal/service/playground.go`
- Modify: `internal/biz/playground/workflow/service.go`
- Test: `internal/biz/playground/runtime/runtime_test.go`
- Test: `internal/service/playground_runtime_test.go`

- [ ] **Step 1: 先写失败测试（运行详情映射出 step / artifact / recovery 数据）**

```go
func TestBuildRunDetailRespIncludesRecoveryActions(t *testing.T) {
	svc := &PlaygroundService{}
	run := &entity.PlaygroundRun{RunID: "run-1", Status: entity.RunStatusWaitingRecovery}
	steps := []*entity.RuntimeStep{{RunID: "run-1", StepID: "agent", Kind: entity.StepKindAgent, Status: entity.StepStatusFailed, FailureSummary: "agent timeout"}}
	artifacts := []*entity.RuntimeArtifact{{ArtifactID: "a1", RunID: "run-1", Type: "route_result", ProducerStepID: "route", Summary: "选择 engineer"}}
	actions := []*entity.RecoveryAction{{ID: "ra-1", RunID: "run-1", StepID: "agent", Type: entity.RecoveryActionRetryStep, Reason: "重新执行当前步骤"}}

	resp := svc.buildRunDetailResp(run, steps, artifacts, actions)
	if len(resp.RecoveryActions) != 1 {
		t.Fatalf("expected 1 recovery action, got %d", len(resp.RecoveryActions))
	}
	if resp.Steps[0].FailureSummary != "agent timeout" {
		t.Fatalf("expected failed step summary, got %#v", resp.Steps[0])
	}
}
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `go test ./internal/service -run TestBuildRunDetailRespIncludesRecoveryActions -count=1`  
Expected: FAIL，提示 `buildRunDetailResp` 未定义或响应结构缺失 `RecoveryActions`

- [ ] **Step 3: 增加查询与恢复执行接口**

```go
// internal/service/playground.go
type runtimeStepResp struct {
	StepID         string            `json:"stepId"`
	Kind           string            `json:"kind"`
	Name           string            `json:"name"`
	Status         string            `json:"status"`
	AgentBinding   string            `json:"agentBinding"`
	FailureSummary string            `json:"failureSummary,omitempty"`
	InputRefs      []string          `json:"inputRefs,omitempty"`
	OutputRef      string            `json:"outputRef,omitempty"`
}

type runtimeArtifactResp struct {
	ArtifactID      string `json:"artifactId"`
	Type            string `json:"type"`
	ProducerStepID  string `json:"producerStepId"`
	Summary         string `json:"summary"`
}

type recoveryActionResp struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	StepID string `json:"stepId"`
	Reason string `json:"reason"`
}

type runDetailResp struct {
	Run             *traceRunResp         `json:"run"`
	Steps           []*runtimeStepResp    `json:"steps"`
	Artifacts       []*runtimeArtifactResp `json:"artifacts"`
	RecoveryActions []*recoveryActionResp `json:"recoveryActions"`
}

func (s *PlaygroundService) buildRunDetailResp(run *entity.PlaygroundRun, steps []*entity.RuntimeStep, artifacts []*entity.RuntimeArtifact, actions []*entity.RecoveryAction) *runDetailResp {
	stepResp := make([]*runtimeStepResp, 0, len(steps))
	for _, step := range steps {
		stepResp = append(stepResp, &runtimeStepResp{
			StepID:         step.StepID,
			Kind:           string(step.Kind),
			Name:           step.Name,
			Status:         string(step.Status),
			AgentBinding:   step.AgentBinding,
			FailureSummary: step.FailureSummary,
			InputRefs:      step.InputRefs,
			OutputRef:      step.OutputRef,
		})
	}
	artifactResp := make([]*runtimeArtifactResp, 0, len(artifacts))
	for _, artifact := range artifacts {
		artifactResp = append(artifactResp, &runtimeArtifactResp{
			ArtifactID:     artifact.ArtifactID,
			Type:           artifact.Type,
			ProducerStepID: artifact.ProducerStepID,
			Summary:        artifact.Summary,
		})
	}
	actionResp := make([]*recoveryActionResp, 0, len(actions))
	for _, action := range actions {
		actionResp = append(actionResp, &recoveryActionResp{
			ID:     action.ID,
			Type:   string(action.Type),
			StepID: action.StepID,
			Reason: action.Reason,
		})
	}
	return &runDetailResp{Run: &traceRunResp{RunID: run.RunID, Status: string(run.Status)}, Steps: stepResp, Artifacts: artifactResp, RecoveryActions: actionResp}
}

func (s *PlaygroundService) listRecoveryActions(ctx khttp.Context) error {
	var req struct{ RunID string `json:"runId"` }
	if err := ctx.BindVars(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	actions, err := s.workflowSvc.ListRecoveryActions(ctx, req.RunID)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	resp := make([]*recoveryActionResp, 0, len(actions))
	for _, action := range actions {
		resp = append(resp, &recoveryActionResp{ID: action.ID, Type: string(action.Type), StepID: action.StepID, Reason: action.Reason})
	}
	return ctx.JSON(http.StatusOK, map[string]any{"actions": resp})
}
```

- [ ] **Step 4: 运行服务层和 runtime 测试确认通过**

Run: `go test ./internal/service ./internal/biz/playground/runtime -run 'TestBuildRunDetailRespIncludesRecoveryActions|TestApplyRecoveryAction|TestRuntimePersistsArtifacts' -count=1`  
Expected: PASS，服务层接口测试和 runtime 持久化测试通过

- [ ] **Step 5: 提交本任务**

```bash
git add internal/data/playground/runtime_repo.go internal/data/playground/runtime_db_repo.go internal/service/playground.go internal/biz/playground/workflow/service.go
git commit -m "feat(playground): expose runtime steps artifacts and recovery APIs"
```

---

### Task 5: 重构前端运行页为 Plan / Run / Recovery 视图

**Files:**
- Create: `flowgram/src/agent-playground/utils/runtime-view-model.ts`
- Create: `flowgram/src/agent-playground/utils/__tests__/runtime-view-model.spec.ts`
- Modify: `flowgram/src/services/api-playground.ts`
- Modify: `flowgram/src/agent-playground/components/workflow-graph.tsx`
- Modify: `flowgram/src/agent-playground/components/run-console.tsx`
- Modify: `flowgram/src/agent-playground/components/trace-panel.tsx`
- Modify: `flowgram/src/agent-playground/pages/playground-page.tsx`

- [ ] **Step 1: 先写失败测试（前端显示模型可以把 Step/Artifact/Recovery 合并成三栏数据）**

```ts
import { describe, expect, it } from 'vitest';
import { buildRuntimeViewModel } from '../runtime-view-model';

describe('buildRuntimeViewModel', () => {
  it('marks failed step and exposes recovery actions', () => {
    const vm = buildRuntimeViewModel({
      run: { runId: 'run-1', status: 'waiting_recovery', failureSummary: 'agent timeout' } as any,
      steps: [
        { stepId: 'route', kind: 'route', status: 'succeeded' },
        { stepId: 'agent', kind: 'agent', status: 'failed', failureSummary: 'agent timeout' },
      ] as any,
      artifacts: [{ artifactId: 'a1', type: 'route_result', producerStepId: 'route', summary: '选择 engineer' }] as any,
      recoveryActions: [{ id: 'ra-1', type: 'retry_step', stepId: 'agent', reason: '重新执行当前步骤' }] as any,
    });

    expect(vm.failedStep?.stepId).toBe('agent');
    expect(vm.recoveryActions).toHaveLength(1);
    expect(vm.planNodes.map(n => n.status)).toEqual(['succeeded', 'failed']);
  });
});
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `cd flowgram && npm run test:unit -- runtime-view-model`  
Expected: FAIL，提示 `Cannot find module '../runtime-view-model'`

- [ ] **Step 3: 增加前端 runtime API 类型与三栏视图重构**

```ts
// flowgram/src/services/api-playground.ts
export interface RuntimeStep {
  stepId: string;
  kind: 'route' | 'agent' | 'parallel' | 'review' | 'handoff' | 'finalize';
  name: string;
  status: 'pending' | 'ready' | 'running' | 'succeeded' | 'failed' | 'skipped';
  agentBinding?: string;
  failureSummary?: string;
  inputRefs?: string[];
  outputRef?: string;
}

export interface RuntimeArtifact {
  artifactId: string;
  type: string;
  producerStepId: string;
  summary: string;
}

export interface RecoveryAction {
  id: string;
  type: 'retry_step' | 'reroute_step' | 'skip_step' | 'retry_from_checkpoint';
  stepId: string;
  reason: string;
}

export const getRunSteps = async (runId: string) => requestJSON<{ steps: RuntimeStep[] }>(`/playground/run/${encodeURIComponent(runId)}/steps`);
export const getRunArtifacts = async (runId: string) => requestJSON<{ artifacts: RuntimeArtifact[] }>(`/playground/run/${encodeURIComponent(runId)}/artifacts`);
export const getRecoveryActions = async (runId: string) => requestJSON<{ actions: RecoveryAction[] }>(`/playground/run/${encodeURIComponent(runId)}/recovery-actions`);
export const applyRecoveryAction = async (runId: string, actionId: string, payload?: Record<string, unknown>) =>
  requestJSON<{ ok: boolean }>(`/playground/run/${encodeURIComponent(runId)}/recovery-actions/${encodeURIComponent(actionId)}`, { method: 'POST', body: payload ?? {} });
```

```ts
// flowgram/src/agent-playground/utils/runtime-view-model.ts
export function buildRuntimeViewModel(input: {
  run?: any;
  steps: any[];
  artifacts: any[];
  recoveryActions: any[];
}) {
  const failedStep = input.steps.find(s => s.status === 'failed');
  return {
    failedStep,
    recoveryActions: input.recoveryActions,
    planNodes: input.steps.map(step => ({
      id: step.stepId,
      label: step.name,
      kind: step.kind,
      status: step.status,
      artifacts: input.artifacts.filter(a => a.producerStepId === step.stepId),
    })),
  };
}
```

- [ ] **Step 4: 运行前端单测、类型检查与构建**

Run: `cd flowgram && npm run test:unit -- runtime-view-model && npm run ts-check && npm run build:prod`  
Expected: PASS，Vitest、TS 检查和生产构建均通过

- [ ] **Step 5: 提交本任务**

```bash
git add flowgram/src/services/api-playground.ts flowgram/src/agent-playground/utils/runtime-view-model.ts flowgram/src/agent-playground/utils/__tests__/runtime-view-model.spec.ts flowgram/src/agent-playground/components/workflow-graph.tsx flowgram/src/agent-playground/components/run-console.tsx flowgram/src/agent-playground/components/trace-panel.tsx flowgram/src/agent-playground/pages/playground-page.tsx
git commit -m "feat(flowgram): add runtime plan recovery and artifact views"
```

---

### Task 6: 为方案编辑页补齐模式专属配置

**Files:**
- Modify: `flowgram/src/services/api-playground.ts`
- Modify: `flowgram/src/agent-playground/pages/playground-page.tsx`
- Modify: `internal/service/playground.go`
- Test: `flowgram/src/agent-playground/utils/__tests__/runtime-view-model.spec.ts`
- Test: `internal/service/playground_runtime_test.go`

- [ ] **Step 1: 先写失败测试（方案响应包含模式专属配置）**

```go
func TestSchemeToRespIncludesModeConfig(t *testing.T) {
	svc := &PlaygroundService{}
	scheme := &entity.CollaborationScheme{
		ID:   "scheme-1",
		Name: "监督方案",
		Mode: entity.ModeSupervision,
		Config: &entity.SchemeConfig{
			MaxIterations:  32,
			MaxToolCalls:   64,
			TimeoutSeconds: 300,
			ModeConfig: &entity.ModeConfig{
				SupervisionConfig: &entity.SupervisionConfig{
					SupervisorAgent: "supervisor",
					WorkerAgents:    []string{"engineer"},
				},
			},
		},
	}
	resp := svc.schemeToResp(scheme)
	if resp.Config == nil || resp.Config.ModeConfig == nil || resp.Config.ModeConfig.SupervisionConfig == nil {
		t.Fatalf("expected supervision mode config, got %#v", resp.Config)
	}
}
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `go test ./internal/service -run TestSchemeToRespIncludesModeConfig -count=1`  
Expected: FAIL，响应不包含 `modeConfig`

- [ ] **Step 3: 扩展前后端表单与响应模型**

```ts
// flowgram/src/services/api-playground.ts
export interface SchemeModeConfig {
  routerConfig?: { fallbackAgent?: string; routingPrompt?: string };
  planExecConfig?: { plannerAgent?: string; executionOrder?: string[] };
  supervisionConfig?: { supervisorAgent?: string; workerAgents?: string[] };
  peerHandoffConfig?: { entryAgent?: string; meshAgents?: string[]; handoffRules?: string };
}

export interface SchemeConfig {
  maxIterations: number;
  maxToolCalls: number;
  timeoutSeconds: number;
  finalizerPrompt?: string;
  modeConfig?: SchemeModeConfig;
}
```

```go
// internal/service/playground.go
type routerConfigResp struct {
	FallbackAgent string `json:"fallbackAgent"`
	RoutingPrompt string `json:"routingPrompt"`
}

type planExecConfigResp struct {
	PlannerAgent   string   `json:"plannerAgent"`
	ExecutionOrder []string `json:"executionOrder"`
}

type supervisionConfigResp struct {
	SupervisorAgent string   `json:"supervisorAgent"`
	WorkerAgents    []string `json:"workerAgents"`
}

type peerHandoffConfigResp struct {
	EntryAgent   string   `json:"entryAgent"`
	MeshAgents   []string `json:"meshAgents"`
	HandoffRules string   `json:"handoffRules"`
}

type schemeModeConfigResp struct {
	RouterConfig      *routerConfigResp      `json:"routerConfig,omitempty"`
	PlanExecConfig    *planExecConfigResp    `json:"planExecConfig,omitempty"`
	SupervisionConfig *supervisionConfigResp `json:"supervisionConfig,omitempty"`
	PeerHandoffConfig *peerHandoffConfigResp `json:"peerHandoffConfig,omitempty"`
}

type schemeConfigResp struct {
	MaxIterations   int                 `json:"maxIterations"`
	MaxToolCalls    int                 `json:"maxToolCalls"`
	TimeoutSeconds  int                 `json:"timeoutSeconds"`
	FinalizerPrompt string              `json:"finalizerPrompt"`
	ModeConfig      *schemeModeConfigResp `json:"modeConfig,omitempty"`
}
```

- [ ] **Step 4: 运行前后端验证**

Run: `go test ./internal/service -run TestSchemeToRespIncludesModeConfig -count=1 && cd flowgram && npm run ts-check && npm run build:prod`  
Expected: PASS，后端模式配置测试通过，前端类型检查和构建通过

- [ ] **Step 5: 提交本任务**

```bash
git add internal/service/playground.go flowgram/src/services/api-playground.ts flowgram/src/agent-playground/pages/playground-page.tsx
git commit -m "feat(playground): support mode specific scheme configuration"
```

---

### Task 7: 全链路验证与旧 handler 收口

**Files:**
- Modify: `internal/biz/playground/workflow/service.go`
- Modify: `internal/biz/playground/collaboration/router.go`
- Modify: `internal/biz/playground/collaboration/plan_exec.go`
- Modify: `internal/biz/playground/collaboration/supervision.go`
- Modify: `internal/biz/playground/collaboration/peer_handoff.go`
- Test: `internal/biz/playground/workflow/service_test.go`
- Test: `flowgram/src/agent-playground/utils/__tests__/runtime-view-model.spec.ts`

- [ ] **Step 1: 写失败测试（WorkflowService 不再直接调用 handler.Execute）**

```go
type fakeRuntimeService struct{ called bool }

func (f *fakeRuntimeService) Run(ctx context.Context, plan *entity.ExecutionPlan, scheme *entity.CollaborationScheme, pool *entity.AgentPool, userInput string, trace collaboration.TraceEmitter) (string, error) {
	f.called = true
	return "run-1", nil
}

func TestExecutePlanUsesRuntimeService(t *testing.T) {
	fakeRuntime := &fakeRuntimeService{}
	svc := &WorkflowService{runtimeSvc: fakeRuntime}
	runID, err := svc.executePlanWithRuntime(
		context.Background(),
		&entity.ExecutionPlan{PlanID: "plan-1"},
		&entity.CollaborationScheme{ID: "scheme-router"},
		nil,
		"设计一个表单",
		nil,
	)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if runID == "" {
		t.Fatal("expected run id")
	}
	if !fakeRuntime.called {
		t.Fatal("expected runtime path instead of legacy handler execute")
	}
}
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `go test ./internal/biz/playground/workflow -run TestExecutePlanUsesRuntimeService -count=1`  
Expected: FAIL，说明仍走 legacy handler 路径

- [ ] **Step 3: 收口 legacy handler，只保留兼容翻译职责**

```go
// internal/biz/playground/collaboration/router.go
// 迁移后仅保留兼容层，避免新代码继续直接扩写 Execute 逻辑。
func NewRouterExpertHandler() *RouterExpertHandler {
	return &RouterExpertHandler{}
}

func (h *RouterExpertHandler) Execute(ctx context.Context, runID string, input string, trace TraceEmitter) (*entity.AgentInstance, error) {
	return nil, fmt.Errorf("legacy router handler is deprecated; use runtime plan execution")
}
```

```go
// internal/biz/playground/workflow/service.go
func (s *WorkflowService) executePlanWithRuntime(
	ctx context.Context,
	plan *entity.ExecutionPlan,
	scheme *entity.CollaborationScheme,
	pool *entity.AgentPool,
	userInput string,
	traceSink collaboration.TraceEmitter,
) (string, error) {
	if s.runtimeSvc == nil {
		return "", fmt.Errorf("runtime service is nil")
	}
	return s.runtimeSvc.Run(ctx, plan, scheme, pool, userInput, traceSink)
}
```

- [ ] **Step 4: 运行全链路验证**

Run: `go test ./internal/biz/playground/... ./internal/service/... -count=1 && cd flowgram && npm run test:unit -- runtime-view-model && npm run ts-check && npm run build:prod`  
Expected: PASS，后端 playground 包测试通过，前端单测 / 类型检查 / 生产构建通过

- [ ] **Step 5: 提交本任务**

```bash
git add internal/biz/playground internal/service flowgram/src/agent-playground flowgram/src/services/api-playground.ts
git commit -m "refactor(playground): finalize runtime migration and remove legacy execution path"
```
