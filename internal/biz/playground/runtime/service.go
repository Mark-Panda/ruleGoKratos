package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ruleGoKratos/internal/biz/entity"
	"ruleGoKratos/internal/biz/playground/collaboration"

	"github.com/google/uuid"
)

// AgentRunner 负责执行单个 Agent step。
type AgentRunner func(
	ctx context.Context,
	runID string,
	def *entity.AgentDefinition,
	userInput string,
	trace collaboration.TraceEmitter,
	schemeCfg *entity.SchemeConfig,
) (string, error)

// Service 提供最小可运行的 plan runtime 外壳。
type Service struct {
	repo             Repo
	routeExecutor    *RouteStepExecutor
	agentExecutor    *AgentStepExecutor
	parallelExecutor *ParallelStepExecutor
	reviewExecutor   *ReviewStepExecutor
	handoffExecutor  *HandoffStepExecutor
	finalizeExecutor *FinalizeStepExecutor
}

type failureState struct {
	runStatus       entity.RunStatus
	failureSummary  string
	recoveryActions []*entity.RecoveryAction
}

type stepExecutionError struct {
	err      error
	artifact *entity.RuntimeArtifact
}

// RunError 表示 runtime 已确定的非终态/终态收口结果。
type RunError struct {
	status         entity.RunStatus
	failureSummary string
	cause          error
}

type executionContext struct {
	runID       string
	plan        *entity.ExecutionPlan
	scheme      *entity.CollaborationScheme
	pool        *entity.AgentPool
	repo        Repo
	userInput   string
	trace       collaboration.TraceEmitter
	artifacts   map[string]*entity.RuntimeArtifact
	finalOutput string
	// ForcedRouteAgent 仅用于 recovery：reroute_step 时强制路由到指定 Agent（不走关键词匹配）。
	ForcedRouteAgent string
}

// NewService 使用默认 Harness 适配器创建 runtime 服务。
func NewService(repo Repo, collabRT *collaboration.CollaborationRuntime) *Service {
	return NewServiceWithAgentRunner(repo, func(
		ctx context.Context,
		runID string,
		def *entity.AgentDefinition,
		userInput string,
		trace collaboration.TraceEmitter,
		schemeCfg *entity.SchemeConfig,
	) (string, error) {
		return collaboration.RunAgentHarness(ctx, collabRT, runID, def, userInput, nil, trace, schemeCfg)
	})
}

// NewServiceWithAgentRunner 使用自定义 AgentRunner 创建 runtime 服务。
func NewServiceWithAgentRunner(repo Repo, runner AgentRunner) *Service {
	return &Service{
		repo:             repo,
		routeExecutor:    NewRouteStepExecutor(),
		agentExecutor:    NewAgentStepExecutor(runner),
		parallelExecutor: NewParallelStepExecutor(runner),
		reviewExecutor:   NewReviewStepExecutor(runner),
		handoffExecutor:  NewHandoffStepExecutor(runner),
		finalizeExecutor: NewFinalizeStepExecutor(),
	}
}

// Repo 返回 runtime 当前使用的仓储实现。
func (s *Service) Repo() Repo {
	if s == nil {
		return nil
	}
	return s.repo
}

// NewRunError 创建带运行态语义的 runtime 错误。
func NewRunError(status entity.RunStatus, failureSummary string) *RunError {
	return &RunError{
		status:         status,
		failureSummary: failureSummary,
	}
}

// Error 实现 error 接口。
func (e *RunError) Error() string {
	if e == nil {
		return ""
	}
	if e.failureSummary != "" {
		return e.failureSummary
	}
	if e.cause != nil {
		return e.cause.Error()
	}
	return string(e.status)
}

// Unwrap 返回底层错误。
func (e *RunError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// Status 返回 runtime 收口状态。
func (e *RunError) Status() entity.RunStatus {
	if e == nil {
		return ""
	}
	return e.status
}

// FailureSummary 返回失败摘要。
func (e *RunError) FailureSummary() string {
	if e == nil {
		return ""
	}
	return e.failureSummary
}

func (e *stepExecutionError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *stepExecutionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// Execute 执行最小 runtime happy path。
func (s *Service) Execute(
	ctx context.Context,
	runID string,
	plan *entity.ExecutionPlan,
	scheme *entity.CollaborationScheme,
	pool *entity.AgentPool,
	userInput string,
	trace collaboration.TraceEmitter,
) (string, error) {
	if s == nil {
		return "", fmt.Errorf("runtime service is nil")
	}
	if strings.TrimSpace(runID) == "" {
		return "", fmt.Errorf("run id is empty")
	}
	if plan == nil {
		return "", fmt.Errorf("plan is nil")
	}
	if len(plan.Steps) == 0 {
		return "", fmt.Errorf("plan has no steps")
	}

	run := &entity.PlaygroundRun{
		RunID:          runID,
		SchemeID:       schemeIDOf(scheme),
		PlanID:         plan.PlanID,
		Status:         entity.RunStatusReady,
		CurrentStepIDs: append([]string(nil), plan.EntryStepIDs...),
		StartedAt:      nowPtr(),
		Metadata: map[string]any{
			"sourceMode": plan.SourceMode,
		},
	}
	execCtx := &executionContext{
		runID:     runID,
		plan:      plan,
		scheme:    scheme,
		pool:      pool,
		repo:      s.repo,
		userInput: userInput,
		trace:     trace,
		artifacts: make(map[string]*entity.RuntimeArtifact),
	}
	runtimeSteps := buildRuntimeSteps(runID, plan)
	if err := s.saveInitialState(ctx, execCtx, plan, run, runtimeSteps, userInput); err != nil {
		return "", err
	}
	stepStates := make(map[string]*entity.RuntimeStep, len(runtimeSteps))
	for _, step := range runtimeSteps {
		stepStates[step.StepID] = step
	}
	return s.continueExecution(ctx, execCtx, run, plan, stepStates)
}

// ApplyRecoveryAction 对失败后的恢复动作做最小执行闭环。
func (s *Service) ApplyRecoveryAction(
	ctx context.Context,
	runID string,
	actionID string,
	scheme *entity.CollaborationScheme,
	pool *entity.AgentPool,
	userInput string,
	trace collaboration.TraceEmitter,
	optTargetRef string,
) (string, error) {
	if s == nil {
		return "", fmt.Errorf("runtime service is nil")
	}
	if s.repo == nil {
		return "", fmt.Errorf("runtime repo is nil")
	}
	if strings.TrimSpace(runID) == "" {
		return "", fmt.Errorf("run id is empty")
	}
	if strings.TrimSpace(actionID) == "" {
		return "", fmt.Errorf("action id is empty")
	}
	run, err := s.repo.GetRun(ctx, runID)
	if err != nil {
		return "", fmt.Errorf("get run: %w", err)
	}
	plan, err := s.repo.GetPlan(ctx, run.PlanID)
	if err != nil {
		return "", fmt.Errorf("get plan: %w", err)
	}
	steps, err := s.repo.ListSteps(ctx, runID)
	if err != nil {
		return "", fmt.Errorf("list steps: %w", err)
	}
	artifacts, err := s.repo.ListArtifacts(ctx, runID)
	if err != nil {
		return "", fmt.Errorf("list artifacts: %w", err)
	}
	actions, err := s.repo.ListRecoveryActions(ctx, runID)
	if err != nil {
		return "", fmt.Errorf("list recovery actions: %w", err)
	}
	action := findVisibleRecoveryAction(actions, actionID)
	if action == nil {
		return "", fmt.Errorf("recovery action %s not found", actionID)
	}
	if run.Status != entity.RunStatusWaitingRecovery {
		return "", fmt.Errorf("run %s is not waiting recovery", runID)
	}

	stepStates := make(map[string]*entity.RuntimeStep, len(steps))
	for _, step := range steps {
		if step == nil {
			continue
		}
		stepStates[step.StepID] = step
	}
	targetStep := stepStates[action.StepID]
	if targetStep == nil {
		return "", fmt.Errorf("target step %s not found", action.StepID)
	}
	planStep := findPlanStep(plan, action.StepID)
	if planStep == nil {
		return "", fmt.Errorf("plan step %s not found", action.StepID)
	}

	if err := markRecoveryActionsApplied(ctx, s.repo, actions, action.ID); err != nil {
		return "", err
	}

	switch action.Type {
	case entity.RecoveryActionRetryStep:
		if err := resetStepsForRetry(ctx, s.repo, plan, stepStates, targetStep.StepID); err != nil {
			return "", err
		}
	case entity.RecoveryActionSkipStep:
		if planStep.Kind == entity.StepKindFinalize {
			return "", fmt.Errorf("skip_step is not supported for finalize step")
		}
	case entity.RecoveryActionRerouteStep:
		if planStep.Kind != entity.StepKindRoute {
			return "", fmt.Errorf("reroute_step only applies to route steps")
		}
		if err := resetStepsForRetry(ctx, s.repo, plan, stepStates, targetStep.StepID); err != nil {
			return "", err
		}
	case entity.RecoveryActionRetryFromCheckpoint:
		cp := strings.TrimSpace(action.TargetRef)
		if cp == "" {
			cp = strings.TrimSpace(run.LastCheckpointID)
		}
		if cp == "" {
			return "", fmt.Errorf("retry_from_checkpoint requires checkpoint step id")
		}
		if err := resetStepsAfterCheckpoint(ctx, s.repo, plan, stepStates, cp); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported recovery action type: %s", action.Type)
	}

	execCtx := &executionContext{
		runID:       runID,
		plan:        plan,
		scheme:      scheme,
		pool:        pool,
		repo:        s.repo,
		userInput:   userInput,
		trace:       trace,
		artifacts:   rebuildArtifactIndex(plan, artifacts),
		finalOutput: "",
	}

	switch action.Type {
	case entity.RecoveryActionSkipStep:
		aliasArtifact, err := loadArtifactForRefs(ctx, execCtx, planStep.InputRefs)
		if err != nil {
			return "", err
		}
		if aliasArtifact == nil && len(planStep.InputRefs) == 0 {
			aliasArtifact = execCtx.artifacts["user_input"]
		}
		run.Status = entity.RunStatusRunning
		run.FailureSummary = ""
		run.FinishedAt = nil
		targetStep.FailureSummary = ""
		if err := s.skipStep(ctx, execCtx, run, targetStep, planStep, aliasArtifact); err != nil {
			return "", err
		}
		if err := s.promoteReadySteps(ctx, run, plan, stepStates); err != nil {
			return "", err
		}
		return s.continueExecution(ctx, execCtx, run, plan, stepStates)

	case entity.RecoveryActionRerouteStep:
		forced := strings.TrimSpace(optTargetRef)
		if forced == "" {
			forced = strings.TrimSpace(action.TargetRef)
		}
		if forced == "" {
			return "", fmt.Errorf("reroute_step requires target agent id (targetRef)")
		}
		execCtx.ForcedRouteAgent = forced
		run.Status = entity.RunStatusReady
		run.FailureSummary = ""
		run.FinishedAt = nil
		run.CurrentStepIDs = []string{targetStep.StepID}
		if err := s.updateRun(ctx, run); err != nil {
			return "", err
		}
		return s.continueExecution(ctx, execCtx, run, plan, stepStates)

	case entity.RecoveryActionRetryStep:
		run.Status = entity.RunStatusReady
		run.FailureSummary = ""
		run.FinishedAt = nil
		run.CurrentStepIDs = []string{targetStep.StepID}
		if err := s.updateRun(ctx, run); err != nil {
			return "", err
		}
		return s.continueExecution(ctx, execCtx, run, plan, stepStates)

	case entity.RecoveryActionRetryFromCheckpoint:
		run.Status = entity.RunStatusReady
		run.FailureSummary = ""
		run.FinishedAt = nil
		run.CurrentStepIDs = nil
		if err := s.updateRun(ctx, run); err != nil {
			return "", err
		}
		if err := s.promoteReadySteps(ctx, run, plan, stepStates); err != nil {
			return "", err
		}
		return s.continueExecution(ctx, execCtx, run, plan, stepStates)

	default:
		return "", fmt.Errorf("unsupported recovery action type: %s", action.Type)
	}
}

func (s *Service) continueExecution(
	ctx context.Context,
	execCtx *executionContext,
	run *entity.PlaygroundRun,
	plan *entity.ExecutionPlan,
	stepStates map[string]*entity.RuntimeStep,
) (string, error) {
	for _, planStep := range plan.Steps {
		runtimeStep := stepStates[planStep.StepID]
		if runtimeStep == nil {
			return "", fmt.Errorf("runtime step %s not found", planStep.StepID)
		}
		if runtimeStep.Status == entity.StepStatusSucceeded || runtimeStep.Status == entity.StepStatusSkipped {
			continue
		}
		if runtimeStep.Status == entity.StepStatusFailed {
			return "", s.failRun(ctx, run, runtimeStep, planStep, fmt.Errorf("step %s remains failed before retry", runtimeStep.StepID))
		}
		if err := ensureDependenciesSatisfied(stepStates, planStep.DependsOn); err != nil {
			return "", s.failRun(ctx, run, runtimeStep, planStep, err)
		}
		skip, aliasArtifact, err := s.shouldSkipStep(ctx, execCtx, planStep, stepStates)
		if err != nil {
			return "", s.failRun(ctx, run, runtimeStep, planStep, err)
		}
		if skip {
			if err := s.skipStep(ctx, execCtx, run, runtimeStep, planStep, aliasArtifact); err != nil {
				return "", err
			}
			if err := s.promoteReadySteps(ctx, run, plan, stepStates); err != nil {
				return "", err
			}
			continue
		}
		if err := s.promoteStepToReady(ctx, run, runtimeStep); err != nil {
			return "", err
		}

		runtimeStep.Status = entity.StepStatusRunning
		runtimeStep.StartedAt = nowPtr()
		run.Status = entity.RunStatusRunning
		run.CurrentStepIDs = []string{runtimeStep.StepID}
		if err := s.persistRunAndStep(ctx, run, runtimeStep); err != nil {
			return "", err
		}

		finalOutput, err := s.executeStep(ctx, execCtx, planStep)
		if err != nil {
			if storeErr := s.storeArtifactForError(ctx, execCtx, planStep, err); storeErr != nil {
				return "", storeErr
			}
			return "", s.failRun(ctx, run, runtimeStep, planStep, err)
		}

		runtimeStep.Status = entity.StepStatusSucceeded
		runtimeStep.FinishedAt = nowPtr()
		run.LastCheckpointID = runtimeStep.StepID
		run.CurrentStepIDs = nil
		if finalOutput != "" {
			execCtx.finalOutput = finalOutput
		}
		if err := s.persistRunAndStep(ctx, run, runtimeStep); err != nil {
			return "", err
		}
		if err := s.promoteReadySteps(ctx, run, plan, stepStates); err != nil {
			return "", err
		}
	}

	run.Status = entity.RunStatusCompleted
	run.FinishedAt = nowPtr()
	if err := s.updateRun(ctx, run); err != nil {
		return "", err
	}
	return execCtx.finalOutput, nil
}

func (s *Service) executeStep(ctx context.Context, execCtx *executionContext, step *entity.PlanStep) (string, error) {
	switch step.Kind {
	case entity.StepKindRoute:
		artifact, err := s.routeExecutor.Execute(ctx, execCtx, step)
		if err != nil {
			return "", err
		}
		return "", s.storeArtifact(ctx, execCtx, step, artifact)
	case entity.StepKindAgent:
		artifact, err := s.agentExecutor.Execute(ctx, execCtx, step)
		if err != nil {
			return "", err
		}
		return "", s.storeArtifact(ctx, execCtx, step, artifact)
	case entity.StepKindParallel:
		artifact, err := s.parallelExecutor.Execute(ctx, execCtx, step)
		if err != nil {
			return "", err
		}
		return "", s.storeArtifact(ctx, execCtx, step, artifact)
	case entity.StepKindReview:
		artifact, err := s.reviewExecutor.Execute(ctx, execCtx, step)
		if err != nil {
			return "", err
		}
		return "", s.storeArtifact(ctx, execCtx, step, artifact)
	case entity.StepKindHandoff:
		artifact, err := s.handoffExecutor.Execute(ctx, execCtx, step)
		if err != nil {
			return "", err
		}
		return "", s.storeArtifact(ctx, execCtx, step, artifact)
	case entity.StepKindFinalize:
		artifact, finalOutput, err := s.finalizeExecutor.Execute(ctx, execCtx, step)
		if err != nil {
			return "", err
		}
		if err := s.storeArtifact(ctx, execCtx, step, artifact); err != nil {
			return "", err
		}
		return finalOutput, nil
	default:
		return "", fmt.Errorf("unsupported step kind: %s", step.Kind)
	}
}

func (s *Service) promoteStepToReady(
	ctx context.Context,
	run *entity.PlaygroundRun,
	step *entity.RuntimeStep,
) error {
	if step == nil || step.Status != entity.StepStatusPending {
		return nil
	}
	step.Status = entity.StepStatusReady
	run.CurrentStepIDs = []string{step.StepID}
	return s.persistRunAndStep(ctx, run, step)
}

func (s *Service) promoteReadySteps(
	ctx context.Context,
	run *entity.PlaygroundRun,
	plan *entity.ExecutionPlan,
	stepStates map[string]*entity.RuntimeStep,
) error {
	if run == nil || plan == nil {
		return nil
	}
	readyIDs := make([]string, 0)
	for _, planStep := range plan.Steps {
		runtimeStep := stepStates[planStep.StepID]
		if runtimeStep == nil || runtimeStep.Status != entity.StepStatusPending {
			continue
		}
		if err := ensureDependenciesSatisfied(stepStates, planStep.DependsOn); err != nil {
			continue
		}
		runtimeStep.Status = entity.StepStatusReady
		if s.repo != nil {
			if err := s.repo.UpdateStep(ctx, runtimeStep); err != nil {
				return fmt.Errorf("promote ready step %s: %w", runtimeStep.StepID, err)
			}
		}
		readyIDs = append(readyIDs, runtimeStep.StepID)
	}
	if len(readyIDs) == 0 {
		return nil
	}
	run.CurrentStepIDs = readyIDs
	return s.updateRun(ctx, run)
}

func (s *Service) shouldSkipStep(
	ctx context.Context,
	execCtx *executionContext,
	step *entity.PlanStep,
	stepStates map[string]*entity.RuntimeStep,
) (bool, *entity.RuntimeArtifact, error) {
	if step == nil || step.Kind == entity.StepKindFinalize {
		return false, nil, nil
	}
	for _, depID := range step.DependsOn {
		if dep := stepStates[depID]; dep != nil && dep.Status == entity.StepStatusSkipped {
			artifact, err := loadArtifactForRefs(ctx, execCtx, step.InputRefs)
			if err != nil {
				return false, nil, err
			}
			return true, artifact, nil
		}
	}
	for _, inputRef := range step.InputRefs {
		artifact, err := loadArtifactForRefs(ctx, execCtx, []string{inputRef})
		if err != nil {
			return false, nil, err
		}
		if shouldStopOnArtifact(artifact) {
			return true, artifact, nil
		}
	}
	return false, nil, nil
}

func (s *Service) skipStep(
	ctx context.Context,
	execCtx *executionContext,
	run *entity.PlaygroundRun,
	runtimeStep *entity.RuntimeStep,
	planStep *entity.PlanStep,
	aliasArtifact *entity.RuntimeArtifact,
) error {
	if runtimeStep == nil {
		return nil
	}
	if execCtx != nil && planStep != nil && planStep.OutputRef != "" && aliasArtifact != nil {
		skippedArtifact := cloneArtifactForSkippedStep(run.RunID, runtimeStep.StepID, aliasArtifact)
		if err := s.storeArtifact(ctx, execCtx, planStep, skippedArtifact); err != nil {
			return err
		}
	}
	runtimeStep.Status = entity.StepStatusSkipped
	runtimeStep.FinishedAt = nowPtr()
	run.CurrentStepIDs = nil
	if run.Status == entity.RunStatusReady {
		run.Status = entity.RunStatusRunning
	}
	return s.persistRunAndStep(ctx, run, runtimeStep)
}

func (s *Service) saveInitialState(
	ctx context.Context,
	execCtx *executionContext,
	plan *entity.ExecutionPlan,
	run *entity.PlaygroundRun,
	steps []*entity.RuntimeStep,
	userInput string,
) error {
	inputArtifact := newArtifact(run.RunID, "", "user_input", userInput, map[string]any{
		"text": userInput,
	})
	if execCtx != nil {
		execCtx.artifacts["user_input"] = inputArtifact
	}
	if s.repo == nil {
		return nil
	}
	if err := s.repo.SavePlan(ctx, plan); err != nil {
		return fmt.Errorf("save plan: %w", err)
	}
	if err := s.repo.SaveRun(ctx, run); err != nil {
		return fmt.Errorf("save run: %w", err)
	}
	if err := s.repo.SaveSteps(ctx, steps); err != nil {
		return fmt.Errorf("save runtime steps: %w", err)
	}
	if err := s.repo.SaveArtifact(ctx, inputArtifact); err != nil {
		return fmt.Errorf("save input artifact: %w", err)
	}
	return nil
}

func (s *Service) storeArtifact(
	ctx context.Context,
	execCtx *executionContext,
	step *entity.PlanStep,
	artifact *entity.RuntimeArtifact,
) error {
	if artifact == nil {
		return nil
	}
	if step != nil && step.OutputRef != "" {
		execCtx.artifacts[step.OutputRef] = artifact
	}
	if s.repo == nil {
		return nil
	}
	if err := s.repo.SaveArtifact(ctx, artifact); err != nil {
		return fmt.Errorf("save artifact: %w", err)
	}
	return nil
}

func (s *Service) storeArtifactForError(
	ctx context.Context,
	execCtx *executionContext,
	step *entity.PlanStep,
	err error,
) error {
	var stepErr *stepExecutionError
	if !errors.As(err, &stepErr) || stepErr == nil || stepErr.artifact == nil {
		return nil
	}
	return s.storeArtifact(ctx, execCtx, step, stepErr.artifact)
}

func (s *Service) persistRunAndStep(ctx context.Context, run *entity.PlaygroundRun, step *entity.RuntimeStep) error {
	if err := s.updateRun(ctx, run); err != nil {
		return err
	}
	if s.repo == nil {
		return nil
	}
	if err := s.repo.UpdateStep(ctx, step); err != nil {
		return fmt.Errorf("update runtime step %s: %w", step.StepID, err)
	}
	return nil
}

func (s *Service) updateRun(ctx context.Context, run *entity.PlaygroundRun) error {
	if s.repo == nil {
		return nil
	}
	if err := s.repo.UpdateRun(ctx, run); err != nil {
		return fmt.Errorf("update run: %w", err)
	}
	return nil
}

func (s *Service) failRun(ctx context.Context, run *entity.PlaygroundRun, step *entity.RuntimeStep, planStep *entity.PlanStep, err error) error {
	state := s.buildFailureState(run, step, planStep, err)
	if step != nil {
		step.Status = entity.StepStatusFailed
		step.FailureSummary = state.failureSummary
		step.FinishedAt = nowPtr()
		if s.repo != nil {
			if updateErr := s.repo.UpdateStep(ctx, step); updateErr != nil {
				return fmt.Errorf("update failed step: %w", updateErr)
			}
		}
	}
	run.Status = state.runStatus
	run.FailureSummary = state.failureSummary
	if step != nil {
		run.CurrentStepIDs = []string{step.StepID}
	}
	run.FinishedAt = nowPtr()
	if updateErr := s.updateRun(ctx, run); updateErr != nil {
		return updateErr
	}
	if len(state.recoveryActions) > 0 && s.repo != nil {
		if updateErr := s.repo.SaveRecoveryActions(ctx, state.recoveryActions); updateErr != nil {
			return fmt.Errorf("save recovery actions: %w", updateErr)
		}
	}
	return &RunError{
		status:         state.runStatus,
		failureSummary: state.failureSummary,
		cause:          err,
	}
}

func (s *Service) buildFailureState(run *entity.PlaygroundRun, step *entity.RuntimeStep, planStep *entity.PlanStep, err error) failureState {
	var actions []*entity.RecoveryAction
	if run != nil && step != nil {
		appendAction := func(actionType entity.RecoveryActionType, targetRef string, reason string) {
			actions = append(actions, &entity.RecoveryAction{
				ID:        uuid.NewString(),
				RunID:     run.RunID,
				StepID:    step.StepID,
				Type:      actionType,
				TargetRef: targetRef,
				Reason:    reason,
				CreatedAt: nowPtr(),
				Metadata: map[string]any{
					"state": "pending",
				},
			})
		}

		appendAction(entity.RecoveryActionRetryStep, step.StepID, err.Error())

		if planStep != nil && planStep.Kind != entity.StepKindFinalize {
			appendAction(entity.RecoveryActionSkipStep, "", "跳过该步骤并沿用上游产物继续执行（下游若依赖不满足可能会再次失败）")
		}

		if planStep != nil && planStep.Kind == entity.StepKindRoute {
			if fb := fallbackAgent(planStep); strings.TrimSpace(fb) != "" {
				appendAction(entity.RecoveryActionRerouteStep, fb, fmt.Sprintf("强制路由到兜底 Agent「%s」并重试路由步骤", fb))
			}
		}

		if run != nil && strings.TrimSpace(run.LastCheckpointID) != "" {
			appendAction(
				entity.RecoveryActionRetryFromCheckpoint,
				run.LastCheckpointID,
				fmt.Sprintf("从检查点「%s」之后的步骤重新开始执行", run.LastCheckpointID),
			)
		}
	}
	return failureState{
		runStatus:       entity.RunStatusWaitingRecovery,
		failureSummary:  err.Error(),
		recoveryActions: actions,
	}
}

func buildRuntimeSteps(runID string, plan *entity.ExecutionPlan) []*entity.RuntimeStep {
	ready := make(map[string]struct{}, len(plan.EntryStepIDs))
	for _, stepID := range plan.EntryStepIDs {
		ready[stepID] = struct{}{}
	}
	steps := make([]*entity.RuntimeStep, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		status := entity.StepStatusPending
		if _, ok := ready[step.StepID]; ok {
			status = entity.StepStatusReady
		}
		steps = append(steps, &entity.RuntimeStep{
			RunID:        runID,
			StepID:       step.StepID,
			Kind:         step.Kind,
			Name:         step.Name,
			Status:       status,
			AgentBinding: step.AgentBinding,
			InputRefs:    append([]string(nil), step.InputRefs...),
			OutputRef:    step.OutputRef,
		})
	}
	return steps
}

func ensureDependenciesSatisfied(stepStates map[string]*entity.RuntimeStep, dependsOn []string) error {
	for _, depID := range dependsOn {
		step := stepStates[depID]
		if step == nil {
			return fmt.Errorf("dependency step %s not found", depID)
		}
		if step.Status != entity.StepStatusSucceeded && step.Status != entity.StepStatusSkipped {
			return fmt.Errorf("dependency step %s is not completed", depID)
		}
	}
	return nil
}

func shouldStopOnArtifact(artifact *entity.RuntimeArtifact) bool {
	if artifact == nil || artifact.Type != "handoff_payload" {
		return false
	}
	stopOrContinue, ok := artifact.Payload["stop_or_continue"].(string)
	return ok && strings.EqualFold(strings.TrimSpace(stopOrContinue), "stop")
}

func latestArtifactForRefs(artifacts map[string]*entity.RuntimeArtifact, refs []string) *entity.RuntimeArtifact {
	for i := len(refs) - 1; i >= 0; i-- {
		if artifact, ok := artifacts[refs[i]]; ok {
			return artifact
		}
	}
	return nil
}

func loadArtifactForRefs(
	ctx context.Context,
	execCtx *executionContext,
	refs []string,
) (*entity.RuntimeArtifact, error) {
	if artifact := latestArtifactForRefs(execCtx.artifacts, refs); artifact != nil {
		return artifact, nil
	}
	if execCtx == nil || execCtx.repo == nil {
		return nil, nil
	}
	artifacts, err := execCtx.repo.ListArtifacts(ctx, execCtx.runID)
	if err != nil {
		return nil, fmt.Errorf("list artifacts: %w", err)
	}
	for i := len(refs) - 1; i >= 0; i-- {
		ref := refs[i]
		if artifact := findArtifactByOutputRef(execCtx.plan, artifacts, ref); artifact != nil {
			execCtx.artifacts[ref] = artifact
			return artifact, nil
		}
	}
	return nil, nil
}

func findArtifactByOutputRef(
	plan *entity.ExecutionPlan,
	artifacts []*entity.RuntimeArtifact,
	outputRef string,
) *entity.RuntimeArtifact {
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

func artifactText(artifact *entity.RuntimeArtifact) string {
	if artifact == nil {
		return ""
	}
	if text, ok := artifact.Payload["output"].(string); ok && strings.TrimSpace(text) != "" {
		return text
	}
	if text, ok := artifact.Payload["text"].(string); ok && strings.TrimSpace(text) != "" {
		return text
	}
	return artifact.Summary
}

func newArtifact(runID, producerStepID, artifactType, summary string, payload map[string]any) *entity.RuntimeArtifact {
	return &entity.RuntimeArtifact{
		ArtifactID:     uuid.NewString(),
		RunID:          runID,
		Type:           artifactType,
		ProducerStepID: producerStepID,
		SchemaVersion:  1,
		Payload:        payload,
		Summary:        summary,
		CreatedAt:      nowPtr(),
	}
}

func cloneArtifactForSkippedStep(runID, producerStepID string, source *entity.RuntimeArtifact) *entity.RuntimeArtifact {
	if source == nil {
		return nil
	}
	payload := make(map[string]any, len(source.Payload))
	for key, value := range source.Payload {
		payload[key] = value
	}
	return &entity.RuntimeArtifact{
		ArtifactID:     uuid.NewString(),
		RunID:          runID,
		Type:           source.Type,
		ProducerStepID: producerStepID,
		SchemaVersion:  source.SchemaVersion,
		Payload:        payload,
		Summary:        source.Summary,
		CreatedAt:      nowPtr(),
	}
}

func schemeIDOf(scheme *entity.CollaborationScheme) string {
	if scheme == nil {
		return ""
	}
	return scheme.ID
}

func resetStepsForRetry(
	ctx context.Context,
	repo Repo,
	plan *entity.ExecutionPlan,
	stepStates map[string]*entity.RuntimeStep,
	targetStepID string,
) error {
	if plan == nil {
		return fmt.Errorf("plan is nil")
	}
	targetSeen := false
	for _, planStep := range plan.Steps {
		if planStep == nil {
			continue
		}
		if planStep.StepID == targetStepID {
			targetSeen = true
		}
		if !targetSeen {
			continue
		}
		step := stepStates[planStep.StepID]
		if step == nil {
			continue
		}
		if step.Status == entity.StepStatusSucceeded || step.Status == entity.StepStatusSkipped {
			continue
		}
		step.Status = entity.StepStatusPending
		step.FailureSummary = ""
		step.StartedAt = nil
		step.FinishedAt = nil
		if repo != nil {
			if err := repo.UpdateStep(ctx, step); err != nil {
				return fmt.Errorf("reset retry step %s: %w", step.StepID, err)
			}
		}
	}
	return nil
}

func resetStepsAfterCheckpoint(
	ctx context.Context,
	repo Repo,
	plan *entity.ExecutionPlan,
	stepStates map[string]*entity.RuntimeStep,
	checkpointStepID string,
) error {
	if plan == nil {
		return fmt.Errorf("plan is nil")
	}
	cpIdx := -1
	for i, ps := range plan.Steps {
		if ps != nil && ps.StepID == checkpointStepID {
			cpIdx = i
			break
		}
	}
	if cpIdx < 0 {
		return fmt.Errorf("checkpoint step %s not found in plan", checkpointStepID)
	}
	for i := cpIdx + 1; i < len(plan.Steps); i++ {
		ps := plan.Steps[i]
		if ps == nil {
			continue
		}
		st := stepStates[ps.StepID]
		if st == nil {
			continue
		}
		st.Status = entity.StepStatusPending
		st.FailureSummary = ""
		st.StartedAt = nil
		st.FinishedAt = nil
		if repo != nil {
			if err := repo.UpdateStep(ctx, st); err != nil {
				return fmt.Errorf("reset checkpoint downstream step %s: %w", ps.StepID, err)
			}
		}
	}
	return nil
}

func findPlanStep(plan *entity.ExecutionPlan, stepID string) *entity.PlanStep {
	if plan == nil || strings.TrimSpace(stepID) == "" {
		return nil
	}
	for _, step := range plan.Steps {
		if step != nil && step.StepID == stepID {
			return step
		}
	}
	return nil
}

func rebuildArtifactIndex(plan *entity.ExecutionPlan, artifacts []*entity.RuntimeArtifact) map[string]*entity.RuntimeArtifact {
	index := make(map[string]*entity.RuntimeArtifact)
	for i := len(artifacts) - 1; i >= 0; i-- {
		artifact := artifacts[i]
		if artifact == nil {
			continue
		}
		if artifact.Type == "user_input" {
			if _, exists := index["user_input"]; !exists {
				index["user_input"] = artifact
			}
			continue
		}
	}
	if plan == nil {
		return index
	}
	for _, step := range plan.Steps {
		if step == nil || step.OutputRef == "" {
			continue
		}
		if _, exists := index[step.OutputRef]; exists {
			continue
		}
		if artifact := findArtifactByOutputRef(plan, artifacts, step.OutputRef); artifact != nil {
			index[step.OutputRef] = artifact
		}
	}
	return index
}

func findVisibleRecoveryAction(actions []*entity.RecoveryAction, actionID string) *entity.RecoveryAction {
	for _, action := range actions {
		if action != nil && action.ID == actionID && isRecoveryActionVisible(action) {
			return action
		}
	}
	return nil
}

func markRecoveryActionsApplied(ctx context.Context, repo Repo, actions []*entity.RecoveryAction, appliedActionID string) error {
	if repo == nil {
		return nil
	}
	for _, action := range actions {
		if action == nil || !isRecoveryActionVisible(action) {
			continue
		}
		action.Metadata = ensureActionMetadata(action.Metadata)
		if action.ID == appliedActionID {
			action.Metadata["state"] = "applied"
		} else {
			action.Metadata["state"] = "superseded"
			action.Metadata["supersededBy"] = appliedActionID
		}
		action.Metadata["updatedAt"] = time.Now().Format(time.RFC3339Nano)
		if err := repo.UpdateRecoveryAction(ctx, action); err != nil {
			return fmt.Errorf("update recovery action %s: %w", action.ID, err)
		}
	}
	return nil
}

func isRecoveryActionVisible(action *entity.RecoveryAction) bool {
	if action == nil {
		return false
	}
	state := recoveryActionState(action)
	return state == "" || state == "pending"
}

func recoveryActionState(action *entity.RecoveryAction) string {
	if action == nil || action.Metadata == nil {
		return ""
	}
	if state, ok := action.Metadata["state"].(string); ok {
		return strings.TrimSpace(state)
	}
	return ""
}

func ensureActionMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return make(map[string]any)
	}
	return metadata
}

func nowPtr() *time.Time {
	now := time.Now()
	return &now
}
