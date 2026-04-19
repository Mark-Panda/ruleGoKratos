package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ruleGoKratos/internal/biz/entity"
	"ruleGoKratos/internal/biz/playground/agentpool"
	"ruleGoKratos/internal/biz/playground/collaboration"
	playgroundruntime "ruleGoKratos/internal/biz/playground/runtime"
	"ruleGoKratos/internal/biz/playground/trace"
	"ruleGoKratos/internal/biz/playground/workflow"
	playgrounddata "ruleGoKratos/internal/data/playground"

	"github.com/go-kratos/kratos/v2/log"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

func TestBuildRunDetailRespIncludesRecoveryActions(t *testing.T) {
	svc := &PlaygroundService{}
	run := &entity.PlaygroundRun{
		RunID:          "run-1",
		SchemeID:       "scheme-1",
		PlanID:         "plan-1",
		Status:         entity.RunStatusWaitingRecovery,
		CurrentStepIDs: []string{"agent"},
		FailureSummary: "agent timeout",
	}
	steps := []*entity.RuntimeStep{
		{
			RunID:          "run-1",
			StepID:         "agent",
			Kind:           entity.StepKindAgent,
			Name:           "agent",
			Status:         entity.StepStatusFailed,
			AgentBinding:   "engineer",
			FailureSummary: "agent timeout",
			InputRefs:      []string{"route_result"},
			OutputRef:      "agent_output",
		},
	}
	artifacts := []*entity.RuntimeArtifact{
		{
			ArtifactID:     "artifact-1",
			RunID:          "run-1",
			Type:           "route_result",
			ProducerStepID: "route",
			Summary:        "选择 engineer",
		},
	}
	actions := []*entity.RecoveryAction{
		{
			ID:     "action-1",
			RunID:  "run-1",
			StepID: "agent",
			Type:   entity.RecoveryActionRetryStep,
			Reason: "重新执行当前步骤",
		},
	}

	resp := svc.buildRunDetailResp(run, steps, artifacts, actions)
	if resp.Run == nil {
		t.Fatal("expected run detail")
	}
	if len(resp.Steps) != 1 || resp.Steps[0].FailureSummary != "agent timeout" {
		t.Fatalf("expected failed runtime step, got %#v", resp.Steps)
	}
	if len(resp.RecoveryActions) != 1 || resp.RecoveryActions[0].StepID != "agent" {
		t.Fatalf("expected recovery action for failed step, got %#v", resp.RecoveryActions)
	}
}

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
					CheckInterval:   15,
				},
			},
		},
	}

	resp := svc.schemeToResp(scheme)
	if resp.Config == nil || resp.Config.ModeConfig == nil || resp.Config.ModeConfig.SupervisionConfig == nil {
		t.Fatalf("expected supervision mode config, got %#v", resp.Config)
	}
	if resp.Config.ModeConfig.SupervisionConfig.SupervisorAgent != "supervisor" {
		t.Fatalf("expected supervisor agent, got %#v", resp.Config.ModeConfig.SupervisionConfig)
	}
	if len(resp.Config.ModeConfig.SupervisionConfig.WorkerAgents) != 1 || resp.Config.ModeConfig.SupervisionConfig.WorkerAgents[0] != "engineer" {
		t.Fatalf("expected worker agents, got %#v", resp.Config.ModeConfig.SupervisionConfig)
	}
}

func TestPatchSchemeConfigTrimsMixedModeConfigBySchemeMode(t *testing.T) {
	scheme := &entity.CollaborationScheme{
		Mode: entity.ModeSupervision,
		Config: &entity.SchemeConfig{
			ModeConfig: &entity.ModeConfig{
				RouterConfig: &entity.RouterConfig{
					FallbackAgent: "designer",
					RoutingPrompt: "route",
				},
			},
		},
	}

	patchSchemeConfig(scheme, &schemeConfigResp{
		MaxIterations:  16,
		MaxToolCalls:   32,
		TimeoutSeconds: 120,
		ModeConfig: &schemeModeConfigResp{
			RouterConfig: &routerConfigResp{
				FallbackAgent: "pm",
				RoutingPrompt: "should be trimmed",
			},
			SupervisionConfig: &supervisionConfigResp{
				SupervisorAgent: "supervisor",
				WorkerAgents:    []string{"engineer"},
				CheckInterval:   10,
			},
		},
	})

	if scheme.Config == nil || scheme.Config.ModeConfig == nil {
		t.Fatalf("expected mode config, got %#v", scheme.Config)
	}
	if scheme.Config.ModeConfig.RouterConfig != nil {
		t.Fatalf("expected router config trimmed, got %#v", scheme.Config.ModeConfig)
	}
	if scheme.Config.ModeConfig.SupervisionConfig == nil {
		t.Fatalf("expected supervision config kept, got %#v", scheme.Config.ModeConfig)
	}
	if scheme.Config.ModeConfig.SupervisionConfig.SupervisorAgent != "supervisor" {
		t.Fatalf("expected supervisor agent, got %#v", scheme.Config.ModeConfig.SupervisionConfig)
	}
}

func TestPatchSchemeConfigClearsOldModeConfigWhenSourceModeConfigOmitted(t *testing.T) {
	scheme := &entity.CollaborationScheme{
		Mode: entity.ModePlanExec,
		Config: &entity.SchemeConfig{
			ModeConfig: &entity.ModeConfig{
				RouterConfig: &entity.RouterConfig{
					FallbackAgent: "designer",
					RoutingPrompt: "legacy route config",
				},
			},
		},
	}

	patchSchemeConfig(scheme, &schemeConfigResp{
		MaxIterations:  20,
		MaxToolCalls:   40,
		TimeoutSeconds: 180,
	})

	if scheme.Config == nil {
		t.Fatal("expected scheme config")
	}
	if scheme.Config.ModeConfig != nil {
		t.Fatalf("expected old mode config cleared, got %#v", scheme.Config.ModeConfig)
	}
}

func TestGetRunReturnsRuntimeDetailSections(t *testing.T) {
	ctx := context.Background()
	playgroundSvc, traceEngine, runtimeRepo := newPlaygroundServiceWithRuntimeRepoForTest(playgrounddata.NewMemoryRuntimeRepo())

	traceRun, err := traceEngine.StartRun(ctx, "scheme-1", "请实现登录页")
	if err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}
	if err := traceEngine.EndRun(ctx, traceRun.RunID, "final output", string(entity.RunStatusWaitingRecovery)); err != nil {
		t.Fatalf("EndRun failed: %v", err)
	}

	runtimeRun := &entity.PlaygroundRun{
		RunID:          traceRun.RunID,
		SchemeID:       "scheme-1",
		PlanID:         "plan-1",
		Status:         entity.RunStatusWaitingRecovery,
		CurrentStepIDs: []string{"agent"},
		FailureSummary: "agent timeout",
	}
	runtimeSteps := []*entity.RuntimeStep{
		{
			RunID:     traceRun.RunID,
			StepID:    "route",
			Kind:      entity.StepKindRoute,
			Name:      "route",
			Status:    entity.StepStatusSucceeded,
			OutputRef: "route_result",
		},
		{
			RunID:          traceRun.RunID,
			StepID:         "agent",
			Kind:           entity.StepKindAgent,
			Name:           "agent",
			Status:         entity.StepStatusFailed,
			AgentBinding:   "engineer",
			FailureSummary: "agent timeout",
			InputRefs:      []string{"route_result"},
			OutputRef:      "agent_output",
		},
	}
	runtimeArtifact := &entity.RuntimeArtifact{
		ArtifactID:     "artifact-1",
		RunID:          traceRun.RunID,
		Type:           "route_result",
		ProducerStepID: "route",
		SchemaVersion:  1,
		Summary:        "选择 engineer",
	}
	recoveryAction := &entity.RecoveryAction{
		ID:     "action-1",
		RunID:  traceRun.RunID,
		StepID: "agent",
		Type:   entity.RecoveryActionRetryStep,
		Reason: "重新执行当前步骤",
	}

	if err := runtimeRepo.SaveRun(ctx, runtimeRun); err != nil {
		t.Fatalf("SaveRun failed: %v", err)
	}
	if err := runtimeRepo.SaveSteps(ctx, runtimeSteps); err != nil {
		t.Fatalf("SaveSteps failed: %v", err)
	}
	if err := runtimeRepo.SaveArtifact(ctx, runtimeArtifact); err != nil {
		t.Fatalf("SaveArtifact failed: %v", err)
	}
	if err := runtimeRepo.SaveRecoveryActions(ctx, []*entity.RecoveryAction{recoveryAction}); err != nil {
		t.Fatalf("SaveRecoveryActions failed: %v", err)
	}

	server := khttp.NewServer()
	RegisterPlaygroundHTTPRoutes(server, playgroundSvc)
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	resp, err := http.Get(httpServer.URL + "/api/v1/playground/run/" + traceRun.RunID)
	if err != nil {
		t.Fatalf("GET run detail failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Run             map[string]any   `json:"run"`
		Steps           []map[string]any `json:"steps"`
		Artifacts       []map[string]any `json:"artifacts"`
		RecoveryActions []map[string]any `json:"recoveryActions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if payload.Run == nil {
		t.Fatal("expected run detail in response")
	}
	if len(payload.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %#v", payload.Steps)
	}
	if len(payload.Artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %#v", payload.Artifacts)
	}
	if len(payload.RecoveryActions) != 1 {
		t.Fatalf("expected 1 recovery action, got %#v", payload.RecoveryActions)
	}
}

func TestGetRunFallsBackWhenRuntimeRunNotFound(t *testing.T) {
	ctx := context.Background()
	repo := &failingRuntimeRunRepo{
		Repo:      playgrounddata.NewMemoryRuntimeRepo(),
		getRunErr: playgroundruntime.ErrRunNotFound,
	}
	playgroundSvc, traceEngine, _ := newPlaygroundServiceWithRuntimeRepoForTest(repo)

	traceRun, err := traceEngine.StartRun(ctx, "scheme-1", "请实现登录页")
	if err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}
	if err := traceEngine.EndRun(ctx, traceRun.RunID, "final output", string(entity.RunStatusCompleted)); err != nil {
		t.Fatalf("EndRun failed: %v", err)
	}

	server := khttp.NewServer()
	RegisterPlaygroundHTTPRoutes(server, playgroundSvc)
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	resp, err := http.Get(httpServer.URL + "/api/v1/playground/run/" + traceRun.RunID)
	if err != nil {
		t.Fatalf("GET run detail failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Run             map[string]any   `json:"run"`
		Steps           []map[string]any `json:"steps"`
		Artifacts       []map[string]any `json:"artifacts"`
		RecoveryActions []map[string]any `json:"recoveryActions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if payload.Run["runId"] != traceRun.RunID {
		t.Fatalf("expected fallback run id %q, got %#v", traceRun.RunID, payload.Run["runId"])
	}
	if payload.Run["status"] != traceRun.Status {
		t.Fatalf("expected fallback status %q, got %#v", traceRun.Status, payload.Run["status"])
	}
}

func TestGetRunReturns500WhenRuntimeRunLookupFails(t *testing.T) {
	ctx := context.Background()
	repo := &failingRuntimeRunRepo{
		Repo:      playgrounddata.NewMemoryRuntimeRepo(),
		getRunErr: errors.New("runtime repo unavailable"),
	}
	playgroundSvc, traceEngine, _ := newPlaygroundServiceWithRuntimeRepoForTest(repo)

	traceRun, err := traceEngine.StartRun(ctx, "scheme-1", "请实现登录页")
	if err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}
	if err := traceEngine.EndRun(ctx, traceRun.RunID, "final output", string(entity.RunStatusCompleted)); err != nil {
		t.Fatalf("EndRun failed: %v", err)
	}

	server := khttp.NewServer()
	RegisterPlaygroundHTTPRoutes(server, playgroundSvc)
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	resp, err := http.Get(httpServer.URL + "/api/v1/playground/run/" + traceRun.RunID)
	if err != nil {
		t.Fatalf("GET run detail failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 500, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestApplyRecoveryActionEndpointAcceptsRetryAndCompletesRun(t *testing.T) {
	ctx := context.Background()
	runtimeRepo := playgrounddata.NewMemoryRuntimeRepo()
	playgroundSvc, _, _ := newPlaygroundServiceWithRuntimeRepoForTest(runtimeRepo)

	if _, err := playgroundSvc.agentPoolSvc.CreateDefaultAgentPool(ctx); err != nil {
		t.Fatalf("CreateDefaultAgentPool failed: %v", err)
	}
	scheme, err := playgroundSvc.workflowSvc.CreateScheme(ctx, "恢复测试", "recovery", entity.ModeRouterExpert, []*entity.AgentBinding{
		{AgentID: "engineer", Role: "工程师"},
	})
	if err != nil {
		t.Fatalf("CreateScheme failed: %v", err)
	}

	agentCalls := 0
	playgroundSvc.workflowSvc.SetRuntimeServiceForTest(playgroundruntime.NewServiceWithAgentRunner(runtimeRepo, func(
		_ context.Context,
		_ string,
		def *entity.AgentDefinition,
		userInput string,
		_ collaboration.TraceEmitter,
		_ *entity.SchemeConfig,
	) (string, error) {
		if def.ID != "engineer" {
			return "", errors.New("unexpected agent")
		}
		agentCalls++
		if agentCalls == 1 {
			return "", errors.New("temporary failure")
		}
		return "[engineer] recovered: " + userInput, nil
	}))

	runID, err := playgroundSvc.workflowSvc.Run(ctx, scheme.ID, "请重试这个任务")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	waitRuntimeStatus(t, ctx, runtimeRepo, runID, entity.RunStatusWaitingRecovery)

	actions, err := playgroundSvc.workflowSvc.ListRecoveryActions(ctx, runID)
	if err != nil {
		t.Fatalf("ListRecoveryActions failed: %v", err)
	}
	var retryID string
	for _, a := range actions {
		if a.Type == entity.RecoveryActionRetryStep {
			retryID = a.ID
			break
		}
	}
	if retryID == "" {
		t.Fatalf("expected retry_step in recovery actions, got %d", len(actions))
	}

	server := khttp.NewServer()
	RegisterPlaygroundHTTPRoutes(server, playgroundSvc)
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	req, err := http.NewRequest(http.MethodPost, httpServer.URL+"/api/v1/playground/run/"+runID+"/recovery-actions/"+retryID, nil)
	if err != nil {
		t.Fatalf("build POST request failed: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST recovery action failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	waitRuntimeStatus(t, ctx, runtimeRepo, runID, entity.RunStatusCompleted)
	run, err := runtimeRepo.GetRun(ctx, runID)
	if err != nil {
		t.Fatalf("GetRun failed: %v", err)
	}
	if run.FailureSummary != "" {
		t.Fatalf("expected failure summary cleared after recovery, got %q", run.FailureSummary)
	}
	if agentCalls != 2 {
		t.Fatalf("expected retry to execute agent twice, got %d", agentCalls)
	}
}

func newPlaygroundServiceWithRuntimeRepoForTest(runtimeRepo playgroundruntime.Repo) (*PlaygroundService, *trace.TraceEngine, playgroundruntime.Repo) {
	agentPoolSvc := agentpool.NewAgentPoolService(playgrounddata.NewAgentPoolRepo())
	traceEngine := trace.NewTraceEngine(playgrounddata.NewTraceRepo())
	workflowSvc := workflow.NewWorkflowServiceWithRuntimeRepo(
		playgrounddata.NewSchemeRepo(),
		agentPoolSvc,
		traceEngine,
		nil,
		runtimeRepo,
	)
	playgroundSvc := NewPlaygroundService(agentPoolSvc, workflowSvc, log.NewStdLogger(io.Discard))
	return playgroundSvc, traceEngine, runtimeRepo
}

type failingRuntimeRunRepo struct {
	playgroundruntime.Repo
	getRunErr error
}

func (r *failingRuntimeRunRepo) GetRun(context.Context, string) (*entity.PlaygroundRun, error) {
	return nil, r.getRunErr
}

func waitRuntimeStatus(t *testing.T, ctx context.Context, repo playgroundruntime.Repo, runID string, want entity.RunStatus) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		run, err := repo.GetRun(ctx, runID)
		if err == nil && run != nil && run.Status == want {
			return
		}
		select {
		case <-deadline:
			if err != nil {
				t.Fatalf("timeout waiting run %s status %s: %v", runID, want, err)
			}
			t.Fatalf("timeout waiting run %s status %s", runID, want)
		case <-time.After(20 * time.Millisecond):
		}
	}
}
