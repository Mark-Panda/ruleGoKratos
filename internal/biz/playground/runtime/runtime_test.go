package runtime_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"ruleGoKratos/internal/biz/entity"
	"ruleGoKratos/internal/biz/playground/collaboration"
	"ruleGoKratos/internal/biz/playground/planbuilder"
	"ruleGoKratos/internal/biz/playground/runtime"
	playgrounddata "ruleGoKratos/internal/data/playground"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRuntimeEntitiesExposeCoreStatuses(t *testing.T) {
	runStatuses := []entity.RunStatus{
		entity.RunStatusPending,
		entity.RunStatusReady,
		entity.RunStatusRunning,
		entity.RunStatusWaitingRecovery,
		entity.RunStatusCompleted,
		entity.RunStatusFailed,
		entity.RunStatusCancelled,
	}
	expectedRunStatuses := []string{
		"pending",
		"ready",
		"running",
		"waiting_recovery",
		"completed",
		"failed",
		"cancelled",
	}
	assertStringValues(t, "run status", runStatuses, expectedRunStatuses)

	stepStatuses := []entity.StepStatus{
		entity.StepStatusPending,
		entity.StepStatusReady,
		entity.StepStatusRunning,
		entity.StepStatusSucceeded,
		entity.StepStatusFailed,
		entity.StepStatusSkipped,
	}
	expectedStepStatuses := []string{
		"pending",
		"ready",
		"running",
		"succeeded",
		"failed",
		"skipped",
	}
	assertStringValues(t, "step status", stepStatuses, expectedStepStatuses)

	stepKinds := []entity.StepKind{
		entity.StepKindRoute,
		entity.StepKindAgent,
		entity.StepKindParallel,
		entity.StepKindReview,
		entity.StepKindHandoff,
		entity.StepKindFinalize,
	}
	expectedStepKinds := []string{
		"route",
		"agent",
		"parallel",
		"review",
		"handoff",
		"finalize",
	}
	assertStringValues(t, "step kind", stepKinds, expectedStepKinds)

	recoveryTypes := []entity.RecoveryActionType{
		entity.RecoveryActionRetryStep,
		entity.RecoveryActionRerouteStep,
		entity.RecoveryActionSkipStep,
		entity.RecoveryActionRetryFromCheckpoint,
	}
	expectedRecoveryTypes := []string{
		"retry_step",
		"reroute_step",
		"skip_step",
		"retry_from_checkpoint",
	}
	assertStringValues(t, "recovery action type", recoveryTypes, expectedRecoveryTypes)
}

func TestRuntimeEntitiesComposeCoreModels(t *testing.T) {
	plan := &entity.ExecutionPlan{
		PlanID:       "plan-1",
		PlanVersion:  1,
		SourceMode:   "router_expert",
		EntryStepIDs: []string{"route-1"},
		Steps: []*entity.PlanStep{
			{
				StepID:    "route-1",
				Kind:      entity.StepKindRoute,
				Name:      "Route request",
				OutputRef: "route-selection",
			},
		},
	}
	if plan.PlanID != "plan-1" || len(plan.Steps) != 1 {
		t.Fatalf("unexpected plan: %+v", plan)
	}

	run := &entity.PlaygroundRun{
		RunID:            "run-1",
		SchemeID:         "scheme-1",
		PlanID:           plan.PlanID,
		Status:           entity.RunStatusWaitingRecovery,
		InputArtifactID:  "artifact-input",
		LastCheckpointID: "checkpoint-1",
		CurrentStepIDs:   []string{"agent-1"},
		FailureSummary:   "agent step failed",
		Metadata: map[string]any{
			"mode": "router_expert",
		},
	}
	if run.Status != entity.RunStatusWaitingRecovery {
		t.Fatalf("expected waiting_recovery, got %s", run.Status)
	}

	step := &entity.RuntimeStep{
		RunID:           run.RunID,
		StepID:          "agent-1",
		Kind:            entity.StepKindAgent,
		Name:            "Solve task",
		Status:          entity.StepStatusFailed,
		AgentBinding:    "writer",
		InputRefs:       []string{"route-selection"},
		OutputRef:       "draft-answer",
		CheckpointAfter: true,
	}
	if step.Status != entity.StepStatusFailed {
		t.Fatalf("expected failed step, got %s", step.Status)
	}

	artifact := &entity.RuntimeArtifact{
		ArtifactID:     "artifact-1",
		RunID:          run.RunID,
		Type:           "worker_result",
		ProducerStepID: step.StepID,
		SchemaVersion:  1,
		Payload: map[string]any{
			"text": "draft",
		},
		Summary: "worker draft",
	}
	if artifact.ProducerStepID != step.StepID {
		t.Fatalf("artifact should reference producer step %q, got %q", step.StepID, artifact.ProducerStepID)
	}

	action := &entity.RecoveryAction{
		ID:        "action-1",
		RunID:     run.RunID,
		StepID:    step.StepID,
		Type:      entity.RecoveryActionRetryStep,
		TargetRef: step.StepID,
		Reason:    "retry failed worker",
	}
	if action.Type != entity.RecoveryActionRetryStep {
		t.Fatalf("expected retry_step, got %s", action.Type)
	}
}

func TestRepoExpressesRuntimePersistenceContracts(t *testing.T) {
	ctx := context.Background()
	repo := &stubRepo{}

	plan := &entity.ExecutionPlan{PlanID: "plan-1", PlanVersion: 1}
	run := &entity.PlaygroundRun{RunID: "run-1", PlanID: plan.PlanID}
	step := &entity.RuntimeStep{RunID: run.RunID, StepID: "step-1", Name: "step one"}
	artifact := &entity.RuntimeArtifact{ArtifactID: "artifact-1", RunID: run.RunID, Summary: "artifact one"}
	action := &entity.RecoveryAction{ID: "action-1", RunID: run.RunID, Reason: "manual retry"}

	var runtimeRepo runtime.Repo = repo
	if err := runtimeRepo.SavePlan(ctx, plan); err != nil {
		t.Fatalf("SavePlan failed: %v", err)
	}
	if err := runtimeRepo.UpdatePlan(ctx, plan); err != nil {
		t.Fatalf("UpdatePlan failed: %v", err)
	}
	gotPlan, err := runtimeRepo.GetPlan(ctx, plan.PlanID)
	if err != nil {
		t.Fatalf("GetPlan failed: %v", err)
	}
	if gotPlan.PlanID != plan.PlanID {
		t.Fatalf("unexpected plan: %+v", gotPlan)
	}
	if err := runtimeRepo.SaveRun(ctx, run); err != nil {
		t.Fatalf("SaveRun failed: %v", err)
	}
	if err := runtimeRepo.UpdateRun(ctx, run); err != nil {
		t.Fatalf("UpdateRun failed: %v", err)
	}
	gotRun, err := runtimeRepo.GetRun(ctx, run.RunID)
	if err != nil {
		t.Fatalf("GetRun failed: %v", err)
	}
	if gotRun.RunID != run.RunID {
		t.Fatalf("unexpected run: %+v", gotRun)
	}
	if err := runtimeRepo.SaveSteps(ctx, []*entity.RuntimeStep{step}); err != nil {
		t.Fatalf("SaveSteps failed: %v", err)
	}
	if err := runtimeRepo.UpdateStep(ctx, step); err != nil {
		t.Fatalf("UpdateStep failed: %v", err)
	}
	steps, err := runtimeRepo.ListSteps(ctx, run.RunID)
	if err != nil {
		t.Fatalf("ListSteps failed: %v", err)
	}
	if len(steps) != 1 || steps[0].StepID != step.StepID {
		t.Fatalf("unexpected steps: %+v", steps)
	}
	if err := runtimeRepo.SaveArtifact(ctx, artifact); err != nil {
		t.Fatalf("SaveArtifact failed: %v", err)
	}
	if err := runtimeRepo.UpdateArtifact(ctx, artifact); err != nil {
		t.Fatalf("UpdateArtifact failed: %v", err)
	}
	artifacts, err := runtimeRepo.ListArtifacts(ctx, run.RunID)
	if err != nil {
		t.Fatalf("ListArtifacts failed: %v", err)
	}
	if len(artifacts) != 1 || artifacts[0].ArtifactID != artifact.ArtifactID {
		t.Fatalf("unexpected artifacts: %+v", artifacts)
	}
	if err := runtimeRepo.SaveRecoveryActions(ctx, []*entity.RecoveryAction{action}); err != nil {
		t.Fatalf("SaveRecoveryActions failed: %v", err)
	}
	if err := runtimeRepo.UpdateRecoveryAction(ctx, action); err != nil {
		t.Fatalf("UpdateRecoveryAction failed: %v", err)
	}
	actions, err := runtimeRepo.ListRecoveryActions(ctx, run.RunID)
	if err != nil {
		t.Fatalf("ListRecoveryActions failed: %v", err)
	}
	if len(actions) != 1 || actions[0].ID != action.ID {
		t.Fatalf("unexpected recovery actions: %+v", actions)
	}
}

func TestRepoUpdateRequiresExistingObjects(t *testing.T) {
	ctx := context.Background()
	var runtimeRepo runtime.Repo = &stubRepo{}

	if err := runtimeRepo.UpdatePlan(ctx, &entity.ExecutionPlan{PlanID: "plan-missing"}); !errors.Is(err, errNotFound) {
		t.Fatalf("UpdatePlan should fail with errNotFound, got %v", err)
	}
	if err := runtimeRepo.UpdateRun(ctx, &entity.PlaygroundRun{RunID: "run-missing"}); !errors.Is(err, errNotFound) {
		t.Fatalf("UpdateRun should fail with errNotFound, got %v", err)
	}
	if err := runtimeRepo.UpdateStep(ctx, &entity.RuntimeStep{RunID: "run-missing", StepID: "step-missing"}); !errors.Is(err, errNotFound) {
		t.Fatalf("UpdateStep should fail with errNotFound, got %v", err)
	}
	if err := runtimeRepo.UpdateArtifact(ctx, &entity.RuntimeArtifact{ArtifactID: "artifact-missing"}); !errors.Is(err, errNotFound) {
		t.Fatalf("UpdateArtifact should fail with errNotFound, got %v", err)
	}
	if err := runtimeRepo.UpdateRecoveryAction(ctx, &entity.RecoveryAction{ID: "action-missing"}); !errors.Is(err, errNotFound) {
		t.Fatalf("UpdateRecoveryAction should fail with errNotFound, got %v", err)
	}
}

func TestRepoSaveRejectsDuplicateObjects(t *testing.T) {
	ctx := context.Background()
	var runtimeRepo runtime.Repo = &stubRepo{}

	plan := &entity.ExecutionPlan{PlanID: "plan-1"}
	if err := runtimeRepo.SavePlan(ctx, plan); err != nil {
		t.Fatalf("SavePlan failed: %v", err)
	}
	if err := runtimeRepo.SavePlan(ctx, plan); !errors.Is(err, errAlreadyExists) {
		t.Fatalf("duplicate SavePlan should fail with errAlreadyExists, got %v", err)
	}

	run := &entity.PlaygroundRun{RunID: "run-1"}
	if err := runtimeRepo.SaveRun(ctx, run); err != nil {
		t.Fatalf("SaveRun failed: %v", err)
	}
	if err := runtimeRepo.SaveRun(ctx, run); !errors.Is(err, errAlreadyExists) {
		t.Fatalf("duplicate SaveRun should fail with errAlreadyExists, got %v", err)
	}

	step := &entity.RuntimeStep{RunID: "run-1", StepID: "step-1"}
	if err := runtimeRepo.SaveSteps(ctx, []*entity.RuntimeStep{step}); err != nil {
		t.Fatalf("SaveSteps failed: %v", err)
	}
	if err := runtimeRepo.SaveSteps(ctx, []*entity.RuntimeStep{step}); !errors.Is(err, errAlreadyExists) {
		t.Fatalf("duplicate SaveSteps should fail with errAlreadyExists, got %v", err)
	}

	artifact := &entity.RuntimeArtifact{ArtifactID: "artifact-1", RunID: "run-1"}
	if err := runtimeRepo.SaveArtifact(ctx, artifact); err != nil {
		t.Fatalf("SaveArtifact failed: %v", err)
	}
	if err := runtimeRepo.SaveArtifact(ctx, artifact); !errors.Is(err, errAlreadyExists) {
		t.Fatalf("duplicate SaveArtifact should fail with errAlreadyExists, got %v", err)
	}

	action := &entity.RecoveryAction{ID: "action-1", RunID: "run-1"}
	if err := runtimeRepo.SaveRecoveryActions(ctx, []*entity.RecoveryAction{action}); err != nil {
		t.Fatalf("SaveRecoveryActions failed: %v", err)
	}
	if err := runtimeRepo.SaveRecoveryActions(ctx, []*entity.RecoveryAction{action}); !errors.Is(err, errAlreadyExists) {
		t.Fatalf("duplicate SaveRecoveryActions should fail with errAlreadyExists, got %v", err)
	}
}

func TestDataRuntimeRepoListsArtifactsAndRecoveryActions(t *testing.T) {
	ctx := context.Background()
	repo := playgrounddata.NewMemoryRuntimeRepo()
	run := &entity.PlaygroundRun{
		RunID:    "run-data-1",
		PlanID:   "plan-data-1",
		Status:   entity.RunStatusWaitingRecovery,
		SchemeID: "scheme-data-1",
	}
	steps := []*entity.RuntimeStep{
		{
			RunID:          run.RunID,
			StepID:         "agent",
			Kind:           entity.StepKindAgent,
			Name:           "agent",
			Status:         entity.StepStatusFailed,
			FailureSummary: "agent timeout",
			OutputRef:      "agent_output",
		},
	}
	artifact := &entity.RuntimeArtifact{
		ArtifactID:     "artifact-1",
		RunID:          run.RunID,
		Type:           "worker_result",
		ProducerStepID: "agent",
		SchemaVersion:  1,
		Summary:        "failed worker output",
	}
	action := &entity.RecoveryAction{
		ID:        "action-1",
		RunID:     run.RunID,
		StepID:    "agent",
		Type:      entity.RecoveryActionRetryStep,
		TargetRef: "agent",
		Reason:    "retry failed step",
	}

	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatalf("SaveRun failed: %v", err)
	}
	if err := repo.SaveSteps(ctx, steps); err != nil {
		t.Fatalf("SaveSteps failed: %v", err)
	}
	if err := repo.SaveArtifact(ctx, artifact); err != nil {
		t.Fatalf("SaveArtifact failed: %v", err)
	}
	if err := repo.SaveRecoveryActions(ctx, []*entity.RecoveryAction{action}); err != nil {
		t.Fatalf("SaveRecoveryActions failed: %v", err)
	}

	artifacts, err := repo.ListArtifacts(ctx, run.RunID)
	if err != nil {
		t.Fatalf("ListArtifacts failed: %v", err)
	}
	if len(artifacts) != 1 || artifacts[0].ArtifactID != artifact.ArtifactID {
		t.Fatalf("unexpected artifacts: %+v", artifacts)
	}

	actions, err := repo.ListRecoveryActions(ctx, run.RunID)
	if err != nil {
		t.Fatalf("ListRecoveryActions failed: %v", err)
	}
	if len(actions) != 1 || actions[0].ID != action.ID {
		t.Fatalf("unexpected actions: %+v", actions)
	}
}

func TestRuntimeRunRouterPlan(t *testing.T) {
	ctx := context.Background()
	repo := &stubRepo{}
	var executedAgent string
	svc := runtime.NewServiceWithAgentRunner(repo, func(
		_ context.Context,
		_ string,
		def *entity.AgentDefinition,
		userInput string,
		_ collaboration.TraceEmitter,
		_ *entity.SchemeConfig,
	) (string, error) {
		if def == nil {
			t.Fatal("expected agent definition")
		}
		executedAgent = def.ID
		return "[" + def.ID + "] " + userInput, nil
	})

	plan := &entity.ExecutionPlan{
		PlanID:       "plan-router",
		PlanVersion:  1,
		SourceMode:   string(entity.ModeRouterExpert),
		EntryStepIDs: []string{"route"},
		Steps: []*entity.PlanStep{
			{
				StepID:    "route",
				Kind:      entity.StepKindRoute,
				Name:      "route",
				OutputRef: "route_result",
				Config: map[string]any{
					"candidateAgents": []string{"designer", "engineer"},
					"fallbackAgent":   "designer",
				},
			},
			{
				StepID:    "agent",
				Kind:      entity.StepKindAgent,
				Name:      "agent",
				DependsOn: []string{"route"},
				InputRefs: []string{"route_result"},
				OutputRef: "agent_output",
			},
			{
				StepID:    "finalize",
				Kind:      entity.StepKindFinalize,
				Name:      "finalize",
				DependsOn: []string{"agent"},
				InputRefs: []string{"agent_output"},
				OutputRef: "final_output",
			},
		},
	}
	scheme := &entity.CollaborationScheme{
		ID:   "scheme-1",
		Mode: entity.ModeRouterExpert,
		BindAgents: []*entity.AgentBinding{
			{AgentID: "designer"},
			{AgentID: "engineer"},
		},
	}
	pool := &entity.AgentPool{
		ID: "default",
		Agents: []*entity.AgentDefinition{
			{ID: "designer", Name: "设计师", Enabled: true},
			{ID: "engineer", Name: "工程师", Enabled: true},
		},
	}

	output, err := svc.Execute(ctx, "run-router", plan, scheme, pool, "请实现登录接口", noopTraceEmitter{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if executedAgent != "engineer" {
		t.Fatalf("expected engineer to be selected, got %q", executedAgent)
	}
	if output != "[engineer] 请实现登录接口" {
		t.Fatalf("unexpected output: %q", output)
	}

	run, err := repo.GetRun(ctx, "run-router")
	if err != nil {
		t.Fatalf("GetRun failed: %v", err)
	}
	if run.Status != entity.RunStatusCompleted {
		t.Fatalf("expected completed run, got %s", run.Status)
	}

	steps, err := repo.ListSteps(ctx, "run-router")
	if err != nil {
		t.Fatalf("ListSteps failed: %v", err)
	}
	if got, want := len(steps), 3; got != want {
		t.Fatalf("expected %d runtime steps, got %d", want, got)
	}
	for _, step := range steps {
		if step.Status != entity.StepStatusSucceeded {
			t.Fatalf("expected step %s to succeed, got %s", step.StepID, step.Status)
		}
	}

	artifacts, err := repo.ListArtifacts(ctx, "run-router")
	if err != nil {
		t.Fatalf("ListArtifacts failed: %v", err)
	}
	if got, want := len(artifacts), 4; got != want {
		t.Fatalf("expected %d artifacts, got %d", want, got)
	}
	routeArtifact := findArtifactByType(artifacts, "route_result")
	if routeArtifact == nil {
		t.Fatal("expected route_result artifact")
	}
	if got := routeArtifact.Payload["selected_agent"]; got != "engineer" {
		t.Fatalf("expected selected_agent engineer, got %#v", got)
	}
	finalArtifact := findArtifactByType(artifacts, "final_answer")
	if finalArtifact == nil {
		t.Fatal("expected final_answer artifact")
	}
	if finalArtifact.Summary != output {
		t.Fatalf("expected final artifact summary %q, got %q", output, finalArtifact.Summary)
	}
}

func TestRuntimeFailureTransitionsToWaitingRecovery(t *testing.T) {
	ctx := context.Background()
	repo := &stubRepo{}
	svc := runtime.NewServiceWithAgentRunner(repo, func(
		_ context.Context,
		_ string,
		_ *entity.AgentDefinition,
		_ string,
		_ collaboration.TraceEmitter,
		_ *entity.SchemeConfig,
	) (string, error) {
		return "", errors.New("agent step failed")
	})

	plan := &entity.ExecutionPlan{
		PlanID:       "plan-router-fail",
		PlanVersion:  1,
		SourceMode:   string(entity.ModeRouterExpert),
		EntryStepIDs: []string{"route"},
		Steps: []*entity.PlanStep{
			{
				StepID:    "route",
				Kind:      entity.StepKindRoute,
				Name:      "route",
				OutputRef: "route_result",
				Config: map[string]any{
					"candidateAgents": []string{"engineer"},
					"fallbackAgent":   "engineer",
				},
			},
			{
				StepID:    "agent",
				Kind:      entity.StepKindAgent,
				Name:      "agent",
				DependsOn: []string{"route"},
				InputRefs: []string{"route_result"},
				OutputRef: "agent_output",
			},
		},
	}
	scheme := &entity.CollaborationScheme{
		ID:   "scheme-1",
		Mode: entity.ModeRouterExpert,
	}
	pool := &entity.AgentPool{
		ID: "default",
		Agents: []*entity.AgentDefinition{
			{ID: "engineer", Name: "工程师", Enabled: true},
		},
	}

	_, err := svc.Execute(ctx, "run-router-fail", plan, scheme, pool, "请实现接口", noopTraceEmitter{})
	if err == nil {
		t.Fatal("expected runtime failure")
	}

	run, err := repo.GetRun(ctx, "run-router-fail")
	if err != nil {
		t.Fatalf("GetRun failed: %v", err)
	}
	if run.Status != entity.RunStatusWaitingRecovery {
		t.Fatalf("expected waiting_recovery, got %s", run.Status)
	}
	if run.FailureSummary != "agent step failed" {
		t.Fatalf("unexpected failure summary: %q", run.FailureSummary)
	}
	if got, want := run.CurrentStepIDs, []string{"agent"}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("expected current failed step to remain addressable, got %#v", got)
	}

	steps, err := repo.ListSteps(ctx, "run-router-fail")
	if err != nil {
		t.Fatalf("ListSteps failed: %v", err)
	}
	agentStep := findRuntimeStep(steps, "agent")
	if agentStep == nil {
		t.Fatal("expected failed agent step")
	}
	if agentStep.Status != entity.StepStatusFailed {
		t.Fatalf("expected failed agent step status, got %s", agentStep.Status)
	}

	actions, err := repo.ListRecoveryActions(ctx, "run-router-fail")
	if err != nil {
		t.Fatalf("ListRecoveryActions failed: %v", err)
	}
	retryAC := findRecoveryActionByType(actions, entity.RecoveryActionRetryStep)
	if retryAC == nil {
		t.Fatalf("expected retry_step among recovery actions, got %d actions", len(actions))
	}
}

func TestRuntimeRunPlanExecPlanPassesSequentialContext(t *testing.T) {
	ctx := context.Background()
	repo := &stubRepo{}
	var (
		mu     sync.Mutex
		inputs = make(map[string]string)
	)
	svc := runtime.NewServiceWithAgentRunner(repo, func(
		_ context.Context,
		_ string,
		def *entity.AgentDefinition,
		userInput string,
		_ collaboration.TraceEmitter,
		_ *entity.SchemeConfig,
	) (string, error) {
		mu.Lock()
		inputs[def.ID] = userInput
		mu.Unlock()
		return "[" + def.ID + "] " + userInput, nil
	})

	builder := planbuilder.NewPlanExecBuilder()
	scheme := &entity.CollaborationScheme{
		ID:   "scheme-plan-exec",
		Mode: entity.ModePlanExec,
		BindAgents: []*entity.AgentBinding{
			{AgentID: "planner", Role: "规划师"},
			{AgentID: "designer", Role: "设计师"},
			{AgentID: "engineer", Role: "工程师"},
		},
		Config: &entity.SchemeConfig{
			ModeConfig: &entity.ModeConfig{
				PlanExecConfig: &entity.PlanExecConfig{
					PlannerAgent:   "planner",
					ExecutionOrder: []string{"designer", "engineer"},
				},
			},
		},
	}
	plan, err := builder.Build(scheme, "做一个搜索页")
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	pool := &entity.AgentPool{
		ID: "default",
		Agents: []*entity.AgentDefinition{
			{ID: "planner", Name: "规划师", Enabled: true},
			{ID: "designer", Name: "设计师", Enabled: true},
			{ID: "engineer", Name: "工程师", Enabled: true},
		},
	}

	output, err := svc.Execute(ctx, "run-plan-exec", plan, scheme, pool, "做一个搜索页", noopTraceEmitter{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if got := inputs["planner"]; got != "做一个搜索页" {
		t.Fatalf("expected planner to receive original input, got %q", got)
	}
	if got := inputs["designer"]; !strings.Contains(got, "[planner] 做一个搜索页") {
		t.Fatalf("expected designer input to include planner output, got %q", got)
	}
	if got := inputs["engineer"]; !strings.Contains(got, "[designer]") {
		t.Fatalf("expected engineer input to include designer output, got %q", got)
	}
	if !strings.Contains(output, "[engineer]") {
		t.Fatalf("expected final output to come from engineer, got %q", output)
	}
}

func TestRuntimeReviewStepUsesAgentRunnerAndStoresOutput(t *testing.T) {
	ctx := context.Background()
	repo := &stubRepo{}
	var calledAgents []string
	svc := runtime.NewServiceWithAgentRunner(repo, func(
		_ context.Context,
		_ string,
		def *entity.AgentDefinition,
		userInput string,
		_ collaboration.TraceEmitter,
		_ *entity.SchemeConfig,
	) (string, error) {
		calledAgents = append(calledAgents, def.ID)
		switch def.ID {
		case "writer":
			return "[writer] 初稿方案", nil
		case "reviewer":
			if !strings.Contains(userInput, "[worker_result] [writer] 初稿方案") {
				t.Fatalf("expected reviewer input to include upstream worker output, got %q", userInput)
			}
			return "reviewer: 建议按当前初稿进入下一步", nil
		default:
			return "", fmt.Errorf("unexpected agent %q", def.ID)
		}
	})

	plan := &entity.ExecutionPlan{
		PlanID:       "plan-review-runner",
		PlanVersion:  1,
		SourceMode:   string(entity.ModePlanExec),
		EntryStepIDs: []string{"draft"},
		Steps: []*entity.PlanStep{
			{
				StepID:       "draft",
				Kind:         entity.StepKindAgent,
				Name:         "draft",
				AgentBinding: "writer",
				OutputRef:    "draft_output",
			},
			{
				StepID:       "review",
				Kind:         entity.StepKindReview,
				Name:         "review",
				AgentBinding: "reviewer",
				DependsOn:    []string{"draft"},
				InputRefs:    []string{"draft_output"},
				OutputRef:    "review_output",
			},
			{
				StepID:    "finalize",
				Kind:      entity.StepKindFinalize,
				Name:      "finalize",
				DependsOn: []string{"review"},
				InputRefs: []string{"review_output"},
				OutputRef: "final_output",
			},
		},
	}
	pool := &entity.AgentPool{
		ID: "default",
		Agents: []*entity.AgentDefinition{
			{ID: "writer", Name: "作者", Enabled: true},
			{ID: "reviewer", Name: "评审者", Enabled: true},
		},
	}

	output, err := svc.Execute(ctx, "run-review-runner", plan, &entity.CollaborationScheme{ID: "scheme-review"}, pool, "请产出初稿", noopTraceEmitter{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if output != "reviewer: 建议按当前初稿进入下一步" {
		t.Fatalf("unexpected final output: %q", output)
	}
	if got, want := strings.Join(calledAgents, ","), "writer,reviewer"; got != want {
		t.Fatalf("expected runner to be called by writer and reviewer, got %q", got)
	}

	artifacts, err := repo.ListArtifacts(ctx, "run-review-runner")
	if err != nil {
		t.Fatalf("ListArtifacts failed: %v", err)
	}
	reviewArtifact := findArtifactByType(artifacts, "review_result")
	if reviewArtifact == nil {
		t.Fatal("expected review_result artifact")
	}
	if reviewArtifact.Summary != "reviewer: 建议按当前初稿进入下一步" {
		t.Fatalf("expected review artifact summary to use runner output, got %q", reviewArtifact.Summary)
	}
	if got := reviewArtifact.Payload["review_summary"]; got != "reviewer: 建议按当前初稿进入下一步" {
		t.Fatalf("expected review_summary to use runner output, got %#v", got)
	}
	if got := reviewArtifact.Payload["reviewer_agent"]; got != "reviewer" {
		t.Fatalf("expected reviewer_agent reviewer, got %#v", got)
	}
}

func TestRuntimeRunSupervisionPlanAggregatesParallelResults(t *testing.T) {
	ctx := context.Background()
	repo := &stubRepo{}
	callCount := make(map[string]int)
	svc := runtime.NewServiceWithAgentRunner(repo, func(
		_ context.Context,
		_ string,
		def *entity.AgentDefinition,
		userInput string,
		_ collaboration.TraceEmitter,
		_ *entity.SchemeConfig,
	) (string, error) {
		callCount[def.ID]++
		if def.ID == "supervisor" {
			if callCount[def.ID] == 1 {
				if !strings.Contains(userInput, "designer, engineer") {
					t.Fatalf("expected supervisor assignment input to include worker list, got %q", userInput)
				}
				return "supervisor-assignment: designer 与 engineer 分工明确", nil
			}
			if !strings.Contains(userInput, "[parallel_result]") {
				t.Fatalf("expected supervisor review input to include parallel_result, got %q", userInput)
			}
			return "supervisor-review: 设计与实现结果已汇总", nil
		}
		return "[" + def.ID + "] " + userInput, nil
	})

	builder := planbuilder.NewSupervisionBuilder()
	scheme := &entity.CollaborationScheme{
		ID:   "scheme-supervision",
		Mode: entity.ModeSupervision,
		BindAgents: []*entity.AgentBinding{
			{AgentID: "supervisor", Role: "监督者"},
			{AgentID: "designer", Role: "设计师"},
			{AgentID: "engineer", Role: "工程师"},
		},
	}
	plan, err := builder.Build(scheme, "并发分析")
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	pool := &entity.AgentPool{
		ID: "default",
		Agents: []*entity.AgentDefinition{
			{ID: "supervisor", Name: "监督者", Enabled: true},
			{ID: "designer", Name: "设计师", Enabled: true},
			{ID: "engineer", Name: "工程师", Enabled: true},
		},
	}

	output, err := svc.Execute(ctx, "run-supervision", plan, scheme, pool, "并发分析", noopTraceEmitter{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if output != "supervisor-review: 设计与实现结果已汇总" {
		t.Fatalf("expected final output to use supervisor runner result, got %q", output)
	}
	if callCount["supervisor"] == 0 {
		t.Fatal("expected supervisor review step to invoke runner")
	}

	artifacts, err := repo.ListArtifacts(ctx, "run-supervision")
	if err != nil {
		t.Fatalf("ListArtifacts failed: %v", err)
	}
	parallelArtifact := findArtifactByType(artifacts, "parallel_result")
	if parallelArtifact == nil {
		t.Fatal("expected parallel_result artifact")
	}
	results, ok := parallelArtifact.Payload["results"].([]map[string]any)
	if !ok || len(results) != 2 {
		t.Fatalf("expected 2 structured parallel results, got %#v", parallelArtifact.Payload["results"])
	}
	reviewArtifact := findArtifactByType(artifacts, "review_result")
	if reviewArtifact == nil {
		t.Fatal("expected review_result artifact")
	}
	if got := reviewArtifact.Payload["review_summary"]; got != "supervisor-review: 设计与实现结果已汇总" {
		t.Fatalf("expected supervision review to use runner output, got %#v", got)
	}
}

func TestRuntimeParallelFailureKeepsPartialSuccessArtifacts(t *testing.T) {
	ctx := context.Background()
	repo := &stubRepo{}
	svc := runtime.NewServiceWithAgentRunner(repo, func(
		_ context.Context,
		_ string,
		def *entity.AgentDefinition,
		userInput string,
		_ collaboration.TraceEmitter,
		_ *entity.SchemeConfig,
	) (string, error) {
		if def.ID == "engineer" {
			return "", errors.New("engineer failed")
		}
		return "[" + def.ID + "] " + userInput, nil
	})

	builder := planbuilder.NewSupervisionBuilder()
	scheme := &entity.CollaborationScheme{
		ID:   "scheme-supervision-partial-fail",
		Mode: entity.ModeSupervision,
		BindAgents: []*entity.AgentBinding{
			{AgentID: "supervisor", Role: "监督者"},
			{AgentID: "designer", Role: "设计师"},
			{AgentID: "engineer", Role: "工程师"},
		},
	}
	plan, err := builder.Build(scheme, "并发分析")
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	pool := &entity.AgentPool{
		ID: "default",
		Agents: []*entity.AgentDefinition{
			{ID: "supervisor", Name: "监督者", Enabled: true},
			{ID: "designer", Name: "设计师", Enabled: true},
			{ID: "engineer", Name: "工程师", Enabled: true},
		},
	}

	_, err = svc.Execute(ctx, "run-supervision-partial-fail", plan, scheme, pool, "并发分析", noopTraceEmitter{})
	if err == nil {
		t.Fatal("expected runtime failure")
	}

	artifacts, err := repo.ListArtifacts(ctx, "run-supervision-partial-fail")
	if err != nil {
		t.Fatalf("ListArtifacts failed: %v", err)
	}
	partial := findArtifactByType(artifacts, "parallel_partial_result")
	if partial == nil {
		t.Fatal("expected parallel_partial_result artifact")
	}
	results, ok := partial.Payload["results"].([]map[string]any)
	if !ok || len(results) != 1 {
		t.Fatalf("expected one successful partial result, got %#v", partial.Payload["results"])
	}
	if got := results[0]["agent_id"]; got != "designer" {
		t.Fatalf("expected designer partial result to be preserved, got %#v", got)
	}
	failures, ok := partial.Payload["failures"].([]map[string]any)
	if !ok || len(failures) != 1 {
		t.Fatalf("expected one failure summary, got %#v", partial.Payload["failures"])
	}
	if got := failures[0]["agent_id"]; got != "engineer" {
		t.Fatalf("expected engineer failure summary, got %#v", got)
	}
}

func TestRuntimeParallelPanicKeepsPartialSuccessAndWaitingRecovery(t *testing.T) {
	ctx := context.Background()
	repo := &stubRepo{}
	svc := runtime.NewServiceWithAgentRunner(repo, func(
		_ context.Context,
		_ string,
		def *entity.AgentDefinition,
		userInput string,
		_ collaboration.TraceEmitter,
		_ *entity.SchemeConfig,
	) (string, error) {
		if def.ID == "engineer" {
			panic("engineer panic")
		}
		return "[" + def.ID + "] " + userInput, nil
	})

	builder := planbuilder.NewSupervisionBuilder()
	scheme := &entity.CollaborationScheme{
		ID:   "scheme-supervision-panic",
		Mode: entity.ModeSupervision,
		BindAgents: []*entity.AgentBinding{
			{AgentID: "supervisor", Role: "监督者"},
			{AgentID: "designer", Role: "设计师"},
			{AgentID: "engineer", Role: "工程师"},
		},
	}
	plan, err := builder.Build(scheme, "并发分析")
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	pool := &entity.AgentPool{
		ID: "default",
		Agents: []*entity.AgentDefinition{
			{ID: "supervisor", Name: "监督者", Enabled: true},
			{ID: "designer", Name: "设计师", Enabled: true},
			{ID: "engineer", Name: "工程师", Enabled: true},
		},
	}

	_, err = svc.Execute(ctx, "run-supervision-panic", plan, scheme, pool, "并发分析", noopTraceEmitter{})
	if err == nil {
		t.Fatal("expected runtime failure")
	}

	run, err := repo.GetRun(ctx, "run-supervision-panic")
	if err != nil {
		t.Fatalf("GetRun failed: %v", err)
	}
	if run.Status != entity.RunStatusWaitingRecovery {
		t.Fatalf("expected waiting_recovery, got %s", run.Status)
	}

	artifacts, err := repo.ListArtifacts(ctx, "run-supervision-panic")
	if err != nil {
		t.Fatalf("ListArtifacts failed: %v", err)
	}
	partial := findArtifactByType(artifacts, "parallel_partial_result")
	if partial == nil {
		t.Fatal("expected parallel_partial_result artifact")
	}
	results, ok := partial.Payload["results"].([]map[string]any)
	if !ok || len(results) != 1 {
		t.Fatalf("expected one successful partial result, got %#v", partial.Payload["results"])
	}
	if got := results[0]["agent_id"]; got != "designer" {
		t.Fatalf("expected designer partial result to be preserved, got %#v", got)
	}
	failures, ok := partial.Payload["failures"].([]map[string]any)
	if !ok || len(failures) != 1 {
		t.Fatalf("expected one failure summary, got %#v", partial.Payload["failures"])
	}
	if got := failures[0]["agent_id"]; got != "engineer" {
		t.Fatalf("expected engineer failure summary, got %#v", got)
	}
	if got := failures[0]["error"]; !strings.Contains(fmt.Sprint(got), "panic") {
		t.Fatalf("expected panic to be converted to structured error, got %#v", got)
	}
}

func TestRuntimeRunPeerHandoffPlanProducesStructuredDecision(t *testing.T) {
	ctx := context.Background()
	repo := &stubRepo{}
	callSeq := 0
	decisionSeq := 0
	svc := runtime.NewServiceWithAgentRunner(repo, func(
		_ context.Context,
		_ string,
		def *entity.AgentDefinition,
		userInput string,
		_ collaboration.TraceEmitter,
		_ *entity.SchemeConfig,
	) (string, error) {
		callSeq++
		if callSeq%2 == 0 {
			if !strings.Contains(userInput, "[worker_result]") {
				t.Fatalf("expected handoff decision input to include worker_result, got %q", userInput)
			}
			decisionSeq++
			switch decisionSeq {
			case 1:
				return `{"next_agent":"pm","handoff_reason":"runner-decision-1","payload_summary":"runner-summary-1","stop_or_continue":"continue"}`, nil
			case 2:
				return `{"next_agent":"engineer","handoff_reason":"runner-decision-2","payload_summary":"runner-summary-2","stop_or_continue":"continue"}`, nil
			default:
				return fmt.Sprintf(`{"next_agent":"","handoff_reason":"runner-decision-%d","payload_summary":"runner-summary-%d","stop_or_continue":"stop"}`, decisionSeq, decisionSeq), nil
			}
		}
		return "[" + def.ID + "] " + userInput, nil
	})

	builder := planbuilder.NewPeerHandoffBuilder()
	scheme := &entity.CollaborationScheme{
		ID:   "scheme-peer-handoff",
		Mode: entity.ModePeerHandoff,
		BindAgents: []*entity.AgentBinding{
			{AgentID: "designer", Role: "设计师"},
			{AgentID: "pm", Role: "产品经理"},
			{AgentID: "engineer", Role: "工程师"},
		},
		Config: &entity.SchemeConfig{
			ModeConfig: &entity.ModeConfig{
				PeerHandoffConfig: &entity.PeerHandoffConfig{
					EntryAgent: "designer",
					MeshAgents: []string{"designer", "pm", "engineer"},
				},
			},
		},
	}
	plan, err := builder.Build(scheme, "开始接力")
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	pool := &entity.AgentPool{
		ID: "default",
		Agents: []*entity.AgentDefinition{
			{ID: "designer", Name: "设计师", Enabled: true},
			{ID: "pm", Name: "产品经理", Enabled: true},
			{ID: "engineer", Name: "工程师", Enabled: true},
		},
	}

	output, err := svc.Execute(ctx, "run-peer-handoff", plan, scheme, pool, "开始接力", noopTraceEmitter{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if strings.TrimSpace(output) == "" {
		t.Fatal("expected non-empty final output")
	}

	artifacts, err := repo.ListArtifacts(ctx, "run-peer-handoff")
	if err != nil {
		t.Fatalf("ListArtifacts failed: %v", err)
	}
	handoffArtifacts := findArtifactsByType(artifacts, "handoff_payload")
	if len(handoffArtifacts) == 0 {
		t.Fatal("expected handoff_payload artifacts")
	}
	first := handoffArtifacts[0]
	if got := first.Payload["handoff_reason"]; !strings.Contains(fmt.Sprint(got), "runner-decision-") {
		t.Fatalf("expected first handoff artifact to use runner decision, got %#v", got)
	}
	if got := first.Payload["payload_summary"]; !strings.Contains(fmt.Sprint(got), "runner-summary-") {
		t.Fatalf("expected first handoff payload summary to use runner output, got %#v", got)
	}
	if callSeq <= len(pool.Agents) {
		t.Fatalf("expected peer handoff runtime to trigger extra runner calls for handoff decisions, got %d calls", callSeq)
	}
	last := handoffArtifacts[len(handoffArtifacts)-1]
	if got := last.Payload["handoff_reason"]; !strings.Contains(fmt.Sprint(got), "runner-decision-") {
		t.Fatalf("expected last handoff artifact to use runner decision, got %#v", got)
	}
}

func TestRuntimeHandoffStepUsesAgentRunnerAndProducesStructuredDecision(t *testing.T) {
	ctx := context.Background()
	repo := &stubRepo{}
	designerCalls := 0
	svc := runtime.NewServiceWithAgentRunner(repo, func(
		_ context.Context,
		_ string,
		def *entity.AgentDefinition,
		userInput string,
		_ collaboration.TraceEmitter,
		_ *entity.SchemeConfig,
	) (string, error) {
		if def.ID != "designer" {
			return "", fmt.Errorf("unexpected agent %q", def.ID)
		}
		designerCalls++
		if designerCalls == 1 {
			return "[designer] 页面草图与交互建议", nil
		}
		if !strings.Contains(userInput, "[worker_result] [designer] 页面草图与交互建议") {
			t.Fatalf("expected handoff runner input to include upstream worker output, got %q", userInput)
		}
		return `{"next_agent":"engineer","handoff_reason":"需要工程师落地页面细节","payload_summary":"已整理页面结构与交互重点","stop_or_continue":"continue"}`, nil
	})

	plan := &entity.ExecutionPlan{
		PlanID:       "plan-handoff-runner",
		PlanVersion:  1,
		SourceMode:   string(entity.ModePeerHandoff),
		EntryStepIDs: []string{"design"},
		Steps: []*entity.PlanStep{
			{
				StepID:       "design",
				Kind:         entity.StepKindAgent,
				Name:         "design",
				AgentBinding: "designer",
				OutputRef:    "design_output",
			},
			{
				StepID:    "handoff",
				Kind:      entity.StepKindHandoff,
				Name:      "handoff",
				DependsOn: []string{"design"},
				InputRefs: []string{"design_output"},
				OutputRef: "handoff_output",
				Config: map[string]any{
					"current_agent": "designer",
				},
			},
			{
				StepID:    "finalize",
				Kind:      entity.StepKindFinalize,
				Name:      "finalize",
				DependsOn: []string{"handoff"},
				InputRefs: []string{"handoff_output"},
				OutputRef: "final_output",
			},
		},
	}
	pool := &entity.AgentPool{
		ID: "default",
		Agents: []*entity.AgentDefinition{
			{ID: "designer", Name: "设计师", Enabled: true},
		},
	}

	_, err := svc.Execute(ctx, "run-handoff-runner", plan, &entity.CollaborationScheme{ID: "scheme-handoff"}, pool, "请先给出设计方案再决定交接", noopTraceEmitter{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if designerCalls != 2 {
		t.Fatalf("expected handoff to invoke runner with current_agent, got %d calls", designerCalls)
	}

	artifacts, err := repo.ListArtifacts(ctx, "run-handoff-runner")
	if err != nil {
		t.Fatalf("ListArtifacts failed: %v", err)
	}
	handoffArtifact := findArtifactByType(artifacts, "handoff_payload")
	if handoffArtifact == nil {
		t.Fatal("expected handoff_payload artifact")
	}
	if got := handoffArtifact.Payload["next_agent"]; got != "engineer" {
		t.Fatalf("expected next_agent engineer, got %#v", got)
	}
	if got := handoffArtifact.Payload["handoff_reason"]; got != "需要工程师落地页面细节" {
		t.Fatalf("expected handoff_reason from runner output, got %#v", got)
	}
	if got := handoffArtifact.Payload["payload_summary"]; got != "已整理页面结构与交互重点" {
		t.Fatalf("expected payload_summary from runner output, got %#v", got)
	}
	if got := handoffArtifact.Payload["stop_or_continue"]; got != "continue" {
		t.Fatalf("expected stop_or_continue continue, got %#v", got)
	}
}

func TestRuntimeHandoffNextAgentOverridesFollowingAgentBinding(t *testing.T) {
	ctx := context.Background()
	repo := &stubRepo{}
	var calledAgents []string
	svc := runtime.NewServiceWithAgentRunner(repo, func(
		_ context.Context,
		_ string,
		def *entity.AgentDefinition,
		userInput string,
		_ collaboration.TraceEmitter,
		_ *entity.SchemeConfig,
	) (string, error) {
		calledAgents = append(calledAgents, def.ID)
		switch len(calledAgents) {
		case 1:
			if def.ID != "designer" {
				t.Fatalf("expected first step to run designer, got %q", def.ID)
			}
			return "[designer] 已完成方案设计", nil
		case 2:
			if def.ID != "designer" {
				t.Fatalf("expected handoff decision to run on designer, got %q", def.ID)
			}
			return `{"next_agent":"engineer","handoff_reason":"转给工程师继续实现","payload_summary":"设计结论已整理","stop_or_continue":"continue"}`, nil
		case 3:
			if def.ID != "engineer" {
				t.Fatalf("expected next_agent to override static binding, got %q", def.ID)
			}
			if !strings.Contains(userInput, "[handoff_payload] 设计结论已整理") {
				t.Fatalf("expected downstream agent input to include handoff payload, got %q", userInput)
			}
			return "[engineer] 已接手实现", nil
		default:
			return "", fmt.Errorf("unexpected call sequence: %#v", calledAgents)
		}
	})

	plan := &entity.ExecutionPlan{
		PlanID:       "plan-handoff-next-agent",
		PlanVersion:  1,
		SourceMode:   string(entity.ModePeerHandoff),
		EntryStepIDs: []string{"design"},
		Steps: []*entity.PlanStep{
			{
				StepID:       "design",
				Kind:         entity.StepKindAgent,
				Name:         "design",
				AgentBinding: "designer",
				OutputRef:    "design_output",
			},
			{
				StepID:       "handoff",
				Kind:         entity.StepKindHandoff,
				Name:         "handoff",
				DependsOn:    []string{"design"},
				AgentBinding: "designer",
				InputRefs:    []string{"design_output"},
				OutputRef:    "handoff_output",
				Config: map[string]any{
					"current_agent": "designer",
				},
			},
			{
				StepID:       "implement",
				Kind:         entity.StepKindAgent,
				Name:         "implement",
				DependsOn:    []string{"handoff"},
				AgentBinding: "pm",
				InputRefs:    []string{"handoff_output"},
				OutputRef:    "implement_output",
			},
			{
				StepID:    "finalize",
				Kind:      entity.StepKindFinalize,
				Name:      "finalize",
				DependsOn: []string{"implement"},
				InputRefs: []string{"implement_output"},
				OutputRef: "final_output",
			},
		},
	}
	pool := &entity.AgentPool{
		ID: "default",
		Agents: []*entity.AgentDefinition{
			{ID: "designer", Name: "设计师", Enabled: true},
			{ID: "pm", Name: "产品经理", Enabled: true},
			{ID: "engineer", Name: "工程师", Enabled: true},
		},
	}

	output, err := svc.Execute(ctx, "run-handoff-next-agent", plan, &entity.CollaborationScheme{ID: "scheme-next-agent"}, pool, "开始接力实现", noopTraceEmitter{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if output != "[engineer] 已接手实现" {
		t.Fatalf("expected final output from engineer, got %q", output)
	}
	if got, want := strings.Join(calledAgents, ","), "designer,designer,engineer"; got != want {
		t.Fatalf("unexpected runner call order: got %q want %q", got, want)
	}
}

func TestRuntimeHandoffStopSkipsFollowingAgentAndStillFinalizes(t *testing.T) {
	ctx := context.Background()
	repo := &stubRepo{}
	var calledAgents []string
	svc := runtime.NewServiceWithAgentRunner(repo, func(
		_ context.Context,
		_ string,
		def *entity.AgentDefinition,
		_ string,
		_ collaboration.TraceEmitter,
		_ *entity.SchemeConfig,
	) (string, error) {
		calledAgents = append(calledAgents, def.ID)
		switch len(calledAgents) {
		case 1:
			return "[designer] 已完成本轮结论", nil
		case 2:
			return `{"next_agent":"","handoff_reason":"当前结果足够，直接收口","payload_summary":"最终结论已具备","stop_or_continue":"stop"}`, nil
		default:
			return "", fmt.Errorf("unexpected extra agent call: %q", def.ID)
		}
	})

	plan := &entity.ExecutionPlan{
		PlanID:       "plan-handoff-stop",
		PlanVersion:  1,
		SourceMode:   string(entity.ModePeerHandoff),
		EntryStepIDs: []string{"design"},
		Steps: []*entity.PlanStep{
			{
				StepID:       "design",
				Kind:         entity.StepKindAgent,
				Name:         "design",
				AgentBinding: "designer",
				OutputRef:    "design_output",
			},
			{
				StepID:       "handoff",
				Kind:         entity.StepKindHandoff,
				Name:         "handoff",
				DependsOn:    []string{"design"},
				AgentBinding: "designer",
				InputRefs:    []string{"design_output"},
				OutputRef:    "handoff_output",
				Config: map[string]any{
					"current_agent": "designer",
				},
			},
			{
				StepID:       "implement",
				Kind:         entity.StepKindAgent,
				Name:         "implement",
				DependsOn:    []string{"handoff"},
				AgentBinding: "engineer",
				InputRefs:    []string{"handoff_output"},
				OutputRef:    "implement_output",
			},
			{
				StepID:    "finalize",
				Kind:      entity.StepKindFinalize,
				Name:      "finalize",
				DependsOn: []string{"implement"},
				InputRefs: []string{"implement_output"},
				OutputRef: "final_output",
			},
		},
	}
	pool := &entity.AgentPool{
		ID: "default",
		Agents: []*entity.AgentDefinition{
			{ID: "designer", Name: "设计师", Enabled: true},
			{ID: "engineer", Name: "工程师", Enabled: true},
		},
	}

	output, err := svc.Execute(ctx, "run-handoff-stop", plan, &entity.CollaborationScheme{ID: "scheme-stop"}, pool, "只做当前阶段即可", noopTraceEmitter{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if output != "最终结论已具备" {
		t.Fatalf("expected finalize to use handoff payload summary, got %q", output)
	}
	if got, want := strings.Join(calledAgents, ","), "designer,designer"; got != want {
		t.Fatalf("expected downstream agent to be skipped, got calls %q", got)
	}

	steps, err := repo.ListSteps(ctx, "run-handoff-stop")
	if err != nil {
		t.Fatalf("ListSteps failed: %v", err)
	}
	implementStep := findRuntimeStep(steps, "implement")
	if implementStep == nil {
		t.Fatal("expected implement runtime step")
	}
	if implementStep.Status != entity.StepStatusSkipped {
		t.Fatalf("expected implement step skipped, got %s", implementStep.Status)
	}
	finalizeStep := findRuntimeStep(steps, "finalize")
	if finalizeStep == nil || finalizeStep.Status != entity.StepStatusSucceeded {
		t.Fatalf("expected finalize step succeeded, got %+v", finalizeStep)
	}
}

func TestRuntimeHandoffStopPersistsSkippedStepArtifactForRepoReload(t *testing.T) {
	ctx := context.Background()
	db := openRuntimeGormTestDB(t)
	repo := playgrounddata.NewGormRuntimeRepo(db)
	var calledAgents []string
	svc := runtime.NewServiceWithAgentRunner(repo, func(
		_ context.Context,
		_ string,
		def *entity.AgentDefinition,
		_ string,
		_ collaboration.TraceEmitter,
		_ *entity.SchemeConfig,
	) (string, error) {
		calledAgents = append(calledAgents, def.ID)
		switch len(calledAgents) {
		case 1:
			return "[designer] 已完成本轮结论", nil
		case 2:
			return `{"next_agent":"","handoff_reason":"当前结果足够，直接收口","payload_summary":"最终结论已具备","stop_or_continue":"stop"}`, nil
		default:
			return "", fmt.Errorf("unexpected extra agent call: %q", def.ID)
		}
	})

	plan := &entity.ExecutionPlan{
		PlanID:       "plan-handoff-stop-persisted",
		PlanVersion:  1,
		SourceMode:   string(entity.ModePeerHandoff),
		EntryStepIDs: []string{"design"},
		Steps: []*entity.PlanStep{
			{
				StepID:       "design",
				Kind:         entity.StepKindAgent,
				Name:         "design",
				AgentBinding: "designer",
				OutputRef:    "design_output",
			},
			{
				StepID:       "handoff",
				Kind:         entity.StepKindHandoff,
				Name:         "handoff",
				DependsOn:    []string{"design"},
				AgentBinding: "designer",
				InputRefs:    []string{"design_output"},
				OutputRef:    "handoff_output",
				Config: map[string]any{
					"current_agent": "designer",
				},
			},
			{
				StepID:       "implement",
				Kind:         entity.StepKindAgent,
				Name:         "implement",
				DependsOn:    []string{"handoff"},
				AgentBinding: "engineer",
				InputRefs:    []string{"handoff_output"},
				OutputRef:    "implement_output",
			},
			{
				StepID:    "finalize",
				Kind:      entity.StepKindFinalize,
				Name:      "finalize",
				DependsOn: []string{"implement"},
				InputRefs: []string{"implement_output"},
				OutputRef: "final_output",
			},
		},
	}
	pool := &entity.AgentPool{
		ID: "default",
		Agents: []*entity.AgentDefinition{
			{ID: "designer", Name: "设计师", Enabled: true},
			{ID: "engineer", Name: "工程师", Enabled: true},
		},
	}

	output, err := svc.Execute(ctx, "run-handoff-stop-persisted", plan, &entity.CollaborationScheme{ID: "scheme-stop-persisted"}, pool, "只做当前阶段即可", noopTraceEmitter{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if output != "最终结论已具备" {
		t.Fatalf("expected finalize to use handoff payload summary, got %q", output)
	}

	reloadedRepo := playgrounddata.NewGormRuntimeRepo(db)
	reloadedPlan, err := reloadedRepo.GetPlan(ctx, plan.PlanID)
	if err != nil {
		t.Fatalf("GetPlan failed: %v", err)
	}
	reloadedArtifacts, err := reloadedRepo.ListArtifacts(ctx, "run-handoff-stop-persisted")
	if err != nil {
		t.Fatalf("ListArtifacts failed: %v", err)
	}
	implementArtifact := findArtifactByOutputRefFromPlan(reloadedPlan, reloadedArtifacts, "implement_output")
	if implementArtifact == nil {
		t.Fatal("expected skipped implement step output to be reloadable by output ref")
	}
	if implementArtifact.ProducerStepID != "implement" {
		t.Fatalf("expected skipped artifact producer step implement, got %q", implementArtifact.ProducerStepID)
	}
	if implementArtifact.Type != "handoff_payload" {
		t.Fatalf("expected skipped artifact type handoff_payload, got %q", implementArtifact.Type)
	}
	if implementArtifact.Summary != "最终结论已具备" {
		t.Fatalf("expected skipped artifact summary to be aliased from handoff payload, got %q", implementArtifact.Summary)
	}
}

func TestRuntimePromotesDependentStepToReadyBeforeRunning(t *testing.T) {
	ctx := context.Background()
	repo := &recordingRepo{stubRepo: &stubRepo{}}
	svc := runtime.NewServiceWithAgentRunner(repo, func(
		_ context.Context,
		_ string,
		def *entity.AgentDefinition,
		userInput string,
		_ collaboration.TraceEmitter,
		_ *entity.SchemeConfig,
	) (string, error) {
		return "[" + def.ID + "] " + userInput, nil
	})

	plan := &entity.ExecutionPlan{
		PlanID:       "plan-ready",
		PlanVersion:  1,
		SourceMode:   string(entity.ModeRouterExpert),
		EntryStepIDs: []string{"route"},
		Steps: []*entity.PlanStep{
			{
				StepID:    "route",
				Kind:      entity.StepKindRoute,
				Name:      "route",
				OutputRef: "route_result",
				Config: map[string]any{
					"candidateAgents": []string{"engineer"},
					"fallbackAgent":   "engineer",
				},
			},
			{
				StepID:    "agent",
				Kind:      entity.StepKindAgent,
				Name:      "agent",
				DependsOn: []string{"route"},
				InputRefs: []string{"route_result"},
				OutputRef: "agent_output",
			},
			{
				StepID:    "finalize",
				Kind:      entity.StepKindFinalize,
				Name:      "finalize",
				DependsOn: []string{"agent"},
				InputRefs: []string{"agent_output"},
				OutputRef: "final_output",
			},
		},
	}
	scheme := &entity.CollaborationScheme{ID: "scheme-1", Mode: entity.ModeRouterExpert}
	pool := &entity.AgentPool{
		ID: "default",
		Agents: []*entity.AgentDefinition{
			{ID: "engineer", Name: "工程师", Enabled: true},
		},
	}

	_, err := svc.Execute(ctx, "run-ready", plan, scheme, pool, "请实现接口", noopTraceEmitter{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !repo.hasStepStatusSequence("agent", entity.StepStatusReady, entity.StepStatusRunning, entity.StepStatusSucceeded) {
		t.Fatalf("expected agent step to transition through ready -> running -> succeeded, got %#v", repo.stepStatusLog["agent"])
	}
	if !repo.hasStepStatusSequence("finalize", entity.StepStatusReady, entity.StepStatusRunning, entity.StepStatusSucceeded) {
		t.Fatalf("expected finalize step to transition through ready -> running -> succeeded, got %#v", repo.stepStatusLog["finalize"])
	}
}

func TestRuntimeApplyRecoveryActionRetriesFailedStepAndCompletesRun(t *testing.T) {
	ctx := context.Background()
	repo := &stubRepo{}
	agentCalls := 0
	svc := runtime.NewServiceWithAgentRunner(repo, func(
		_ context.Context,
		_ string,
		def *entity.AgentDefinition,
		userInput string,
		_ collaboration.TraceEmitter,
		_ *entity.SchemeConfig,
	) (string, error) {
		if def.ID != "engineer" {
			return "", fmt.Errorf("unexpected agent %q", def.ID)
		}
		agentCalls++
		if agentCalls == 1 {
			return "", fmt.Errorf("temporary failure")
		}
		return "[engineer] recovered: " + userInput, nil
	})

	plan := &entity.ExecutionPlan{
		PlanID:       "plan-recovery-retry",
		PlanVersion:  1,
		SourceMode:   string(entity.ModeRouterExpert),
		EntryStepIDs: []string{"agent"},
		Steps: []*entity.PlanStep{
			{
				StepID:       "agent",
				Kind:         entity.StepKindAgent,
				Name:         "agent",
				AgentBinding: "engineer",
				InputRefs:    []string{"user_input"},
				OutputRef:    "agent_output",
			},
			{
				StepID:    "finalize",
				Kind:      entity.StepKindFinalize,
				Name:      "finalize",
				DependsOn: []string{"agent"},
				InputRefs: []string{"agent_output"},
				OutputRef: "final_output",
			},
		},
	}
	scheme := &entity.CollaborationScheme{ID: "scheme-recovery", Mode: entity.ModeRouterExpert}
	pool := &entity.AgentPool{
		ID: "default",
		Agents: []*entity.AgentDefinition{
			{ID: "engineer", Name: "工程师", Enabled: true},
		},
	}

	_, err := svc.Execute(ctx, "run-recovery-retry", plan, scheme, pool, "请重试", noopTraceEmitter{})
	if err == nil {
		t.Fatal("expected initial execution to fail")
	}

	run, err := repo.GetRun(ctx, "run-recovery-retry")
	if err != nil {
		t.Fatalf("GetRun failed: %v", err)
	}
	if run.Status != entity.RunStatusWaitingRecovery {
		t.Fatalf("expected waiting_recovery after first failure, got %s", run.Status)
	}

	actions, err := repo.ListRecoveryActions(ctx, "run-recovery-retry")
	if err != nil {
		t.Fatalf("ListRecoveryActions failed: %v", err)
	}
	retryAC := findRecoveryActionByType(actions, entity.RecoveryActionRetryStep)
	if retryAC == nil {
		t.Fatalf("expected retry_step recovery action, got %d actions", len(actions))
	}

	output, err := svc.ApplyRecoveryAction(ctx, "run-recovery-retry", retryAC.ID, scheme, pool, "请重试", noopTraceEmitter{}, "")
	if err != nil {
		t.Fatalf("ApplyRecoveryAction failed: %v", err)
	}
	if !strings.Contains(output, "recovered") {
		t.Fatalf("expected recovered final output, got %q", output)
	}

	run, err = repo.GetRun(ctx, "run-recovery-retry")
	if err != nil {
		t.Fatalf("GetRun after recovery failed: %v", err)
	}
	if run.Status != entity.RunStatusCompleted {
		t.Fatalf("expected completed after recovery, got %s", run.Status)
	}
	if run.FailureSummary != "" {
		t.Fatalf("expected failure summary cleared after recovery, got %q", run.FailureSummary)
	}

	steps, err := repo.ListSteps(ctx, "run-recovery-retry")
	if err != nil {
		t.Fatalf("ListSteps failed: %v", err)
	}
	if got := findRuntimeStep(steps, "agent"); got == nil || got.Status != entity.StepStatusSucceeded {
		t.Fatalf("expected agent step succeeded after retry, got %#v", got)
	}
	if got := findRuntimeStep(steps, "finalize"); got == nil || got.Status != entity.StepStatusSucceeded {
		t.Fatalf("expected finalize step succeeded after retry, got %#v", got)
	}

	actions, err = repo.ListRecoveryActions(ctx, "run-recovery-retry")
	if err != nil {
		t.Fatalf("ListRecoveryActions after recovery failed: %v", err)
	}
	applied := findAppliedRecoveryAction(actions)
	if applied == nil {
		t.Fatalf("expected applied recovery action, got %#v", actions)
	}
	if state := applied.Metadata["state"]; state != "applied" {
		t.Fatalf("expected consumed recovery action metadata to be applied, got %#v", state)
	}
	if agentCalls != 2 {
		t.Fatalf("expected agent to run twice, got %d", agentCalls)
	}
}

func findRecoveryActionByType(actions []*entity.RecoveryAction, typ entity.RecoveryActionType) *entity.RecoveryAction {
	for _, a := range actions {
		if a.Type == typ {
			return a
		}
	}
	return nil
}

func findAppliedRecoveryAction(actions []*entity.RecoveryAction) *entity.RecoveryAction {
	for _, a := range actions {
		if a.Metadata != nil && a.Metadata["state"] == "applied" {
			return a
		}
	}
	return nil
}

func findRuntimeStep(steps []*entity.RuntimeStep, stepID string) *entity.RuntimeStep {
	for _, step := range steps {
		if step.StepID == stepID {
			return step
		}
	}
	return nil
}

func findArtifactByType(artifacts []*entity.RuntimeArtifact, artifactType string) *entity.RuntimeArtifact {
	for _, artifact := range artifacts {
		if artifact.Type == artifactType {
			return artifact
		}
	}
	return nil
}

func findArtifactsByType(artifacts []*entity.RuntimeArtifact, artifactType string) []*entity.RuntimeArtifact {
	var matches []*entity.RuntimeArtifact
	for _, artifact := range artifacts {
		if artifact.Type == artifactType {
			matches = append(matches, artifact)
		}
	}
	return matches
}

func findArtifactByOutputRefFromPlan(plan *entity.ExecutionPlan, artifacts []*entity.RuntimeArtifact, outputRef string) *entity.RuntimeArtifact {
	if plan == nil || outputRef == "" {
		return nil
	}
	stepID := ""
	for _, step := range plan.Steps {
		if step != nil && step.OutputRef == outputRef {
			stepID = step.StepID
			break
		}
	}
	if stepID == "" {
		return nil
	}
	for i := len(artifacts) - 1; i >= 0; i-- {
		artifact := artifacts[i]
		if artifact != nil && artifact.ProducerStepID == stepID {
			return artifact
		}
	}
	return nil
}

func openRuntimeGormTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	return db
}

type noopTraceEmitter struct{}

func (noopTraceEmitter) TaskAssigned(string, string, string)             {}
func (noopTraceEmitter) AgentEnterWorker(string, string, string)         {}
func (noopTraceEmitter) AgentExitWorker(string, string, string, string)  {}
func (noopTraceEmitter) WorkerDelegated(string, string, string, string)  {}
func (noopTraceEmitter) Thinking(string, string, string)                 {}
func (noopTraceEmitter) ToolCall(string, string, string, string)         {}
func (noopTraceEmitter) ToolResult(string, string, string, string, bool) {}
func (noopTraceEmitter) Handoff(string, string, string, string)          {}
func (noopTraceEmitter) Error(string, string, string)                    {}
func (noopTraceEmitter) EmitEvent(context.Context, *entity.TraceEvent)   {}

func assertStringValues[T ~string](t *testing.T, name string, got []T, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("unexpected %s count: got %d want %d", name, len(got), len(want))
	}
	for i := range got {
		if string(got[i]) != want[i] {
			t.Fatalf("unexpected %s at %d: got %q want %q", name, i, got[i], want[i])
		}
	}
}

type stubRepo struct {
	plans     []*entity.ExecutionPlan
	runs      []*entity.PlaygroundRun
	steps     []*entity.RuntimeStep
	artifacts []*entity.RuntimeArtifact
	actions   []*entity.RecoveryAction
}

type recordingRepo struct {
	*stubRepo
	stepStatusLog map[string][]entity.StepStatus
}

var (
	errAlreadyExists = errors.New("already exists")
	errNotFound      = errors.New("not found")
)

func (s *stubRepo) SavePlan(_ context.Context, plan *entity.ExecutionPlan) error {
	if s.findPlan(plan.PlanID) >= 0 {
		return errAlreadyExists
	}
	s.plans = append(s.plans, plan)
	return nil
}

func (s *stubRepo) UpdatePlan(_ context.Context, plan *entity.ExecutionPlan) error {
	idx := s.findPlan(plan.PlanID)
	if idx < 0 {
		return errNotFound
	}
	s.plans[idx] = plan
	return nil
}

func (s *stubRepo) GetPlan(_ context.Context, planID string) (*entity.ExecutionPlan, error) {
	for _, plan := range s.plans {
		if plan.PlanID == planID {
			return plan, nil
		}
	}
	return nil, errNotFound
}

func (s *stubRepo) SaveRun(_ context.Context, run *entity.PlaygroundRun) error {
	if s.findRun(run.RunID) >= 0 {
		return errAlreadyExists
	}
	s.runs = append(s.runs, run)
	return nil
}

func (s *stubRepo) UpdateRun(_ context.Context, run *entity.PlaygroundRun) error {
	idx := s.findRun(run.RunID)
	if idx < 0 {
		return errNotFound
	}
	s.runs[idx] = run
	return nil
}

func (s *stubRepo) GetRun(_ context.Context, runID string) (*entity.PlaygroundRun, error) {
	for _, run := range s.runs {
		if run.RunID == runID {
			return run, nil
		}
	}
	return nil, errNotFound
}

func (s *stubRepo) SaveSteps(_ context.Context, steps []*entity.RuntimeStep) error {
	for _, step := range steps {
		if s.findStep(step.RunID, step.StepID) >= 0 {
			return errAlreadyExists
		}
	}
	s.steps = append(s.steps, steps...)
	return nil
}

func (r *recordingRepo) SaveSteps(ctx context.Context, steps []*entity.RuntimeStep) error {
	if r.stepStatusLog == nil {
		r.stepStatusLog = make(map[string][]entity.StepStatus)
	}
	for _, step := range steps {
		r.stepStatusLog[step.StepID] = append(r.stepStatusLog[step.StepID], step.Status)
	}
	return r.stubRepo.SaveSteps(ctx, steps)
}

func (s *stubRepo) UpdateStep(_ context.Context, step *entity.RuntimeStep) error {
	idx := s.findStep(step.RunID, step.StepID)
	if idx < 0 {
		return errNotFound
	}
	s.steps[idx] = step
	return nil
}

func (r *recordingRepo) UpdateStep(ctx context.Context, step *entity.RuntimeStep) error {
	if r.stepStatusLog == nil {
		r.stepStatusLog = make(map[string][]entity.StepStatus)
	}
	r.stepStatusLog[step.StepID] = append(r.stepStatusLog[step.StepID], step.Status)
	return r.stubRepo.UpdateStep(ctx, step)
}

func (s *stubRepo) ListSteps(_ context.Context, runID string) ([]*entity.RuntimeStep, error) {
	var steps []*entity.RuntimeStep
	for _, step := range s.steps {
		if step.RunID == runID {
			steps = append(steps, step)
		}
	}
	return steps, nil
}

func (s *stubRepo) SaveArtifact(_ context.Context, artifact *entity.RuntimeArtifact) error {
	if s.findArtifact(artifact.ArtifactID) >= 0 {
		return errAlreadyExists
	}
	s.artifacts = append(s.artifacts, artifact)
	return nil
}

func (s *stubRepo) UpdateArtifact(_ context.Context, artifact *entity.RuntimeArtifact) error {
	idx := s.findArtifact(artifact.ArtifactID)
	if idx < 0 {
		return errNotFound
	}
	s.artifacts[idx] = artifact
	return nil
}

func (s *stubRepo) ListArtifacts(_ context.Context, runID string) ([]*entity.RuntimeArtifact, error) {
	var artifacts []*entity.RuntimeArtifact
	for _, artifact := range s.artifacts {
		if artifact.RunID == runID {
			artifacts = append(artifacts, artifact)
		}
	}
	return artifacts, nil
}

func (s *stubRepo) SaveRecoveryActions(_ context.Context, actions []*entity.RecoveryAction) error {
	for _, action := range actions {
		if s.findAction(action.ID) >= 0 {
			return errAlreadyExists
		}
	}
	s.actions = append(s.actions, actions...)
	return nil
}

func (s *stubRepo) UpdateRecoveryAction(_ context.Context, action *entity.RecoveryAction) error {
	idx := s.findAction(action.ID)
	if idx < 0 {
		return errNotFound
	}
	s.actions[idx] = action
	return nil
}

func (s *stubRepo) ListRecoveryActions(_ context.Context, runID string) ([]*entity.RecoveryAction, error) {
	var actions []*entity.RecoveryAction
	for _, action := range s.actions {
		if action.RunID == runID {
			actions = append(actions, action)
		}
	}
	return actions, nil
}

func (s *stubRepo) findPlan(planID string) int {
	for i, plan := range s.plans {
		if plan.PlanID == planID {
			return i
		}
	}
	return -1
}

func (s *stubRepo) findRun(runID string) int {
	for i, run := range s.runs {
		if run.RunID == runID {
			return i
		}
	}
	return -1
}

func (s *stubRepo) findStep(runID, stepID string) int {
	for i, step := range s.steps {
		if step.RunID == runID && step.StepID == stepID {
			return i
		}
	}
	return -1
}

func (s *stubRepo) findArtifact(artifactID string) int {
	for i, artifact := range s.artifacts {
		if artifact.ArtifactID == artifactID {
			return i
		}
	}
	return -1
}

func (s *stubRepo) findAction(actionID string) int {
	for i, action := range s.actions {
		if action.ID == actionID {
			return i
		}
	}
	return -1
}

func (r *recordingRepo) hasStepStatusSequence(stepID string, seq ...entity.StepStatus) bool {
	got := r.stepStatusLog[stepID]
	if len(seq) == 0 {
		return true
	}
	index := 0
	for _, status := range got {
		if status == seq[index] {
			index++
			if index == len(seq) {
				return true
			}
		}
	}
	return false
}
