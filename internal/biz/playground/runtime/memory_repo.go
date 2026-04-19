package runtime

import (
	"context"
	"fmt"
	"sync"

	"ruleGoKratos/internal/biz/entity"
)

// MemoryRepo 提供 Task 2 阶段可用的轻量内存 runtime repo。
type MemoryRepo struct {
	mu sync.RWMutex

	plans     map[string]*entity.ExecutionPlan
	runs      map[string]*entity.PlaygroundRun
	steps     map[string]map[string]*entity.RuntimeStep
	artifacts map[string]map[string]*entity.RuntimeArtifact
	actions   map[string]map[string]*entity.RecoveryAction
}

// NewMemoryRepo 创建轻量内存 runtime repo。
func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		plans:     make(map[string]*entity.ExecutionPlan),
		runs:      make(map[string]*entity.PlaygroundRun),
		steps:     make(map[string]map[string]*entity.RuntimeStep),
		artifacts: make(map[string]map[string]*entity.RuntimeArtifact),
		actions:   make(map[string]map[string]*entity.RecoveryAction),
	}
}

func (r *MemoryRepo) SavePlan(_ context.Context, plan *entity.ExecutionPlan) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.plans[plan.PlanID]; ok {
		return fmt.Errorf("plan already exists: %s", plan.PlanID)
	}
	r.plans[plan.PlanID] = clonePlan(plan)
	return nil
}

func (r *MemoryRepo) UpdatePlan(_ context.Context, plan *entity.ExecutionPlan) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.plans[plan.PlanID]; !ok {
		return fmt.Errorf("plan not found: %s", plan.PlanID)
	}
	r.plans[plan.PlanID] = clonePlan(plan)
	return nil
}

func (r *MemoryRepo) GetPlan(_ context.Context, planID string) (*entity.ExecutionPlan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	plan, ok := r.plans[planID]
	if !ok {
		return nil, fmt.Errorf("plan not found: %s", planID)
	}
	return clonePlan(plan), nil
}

func (r *MemoryRepo) SaveRun(_ context.Context, run *entity.PlaygroundRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.runs[run.RunID]; ok {
		return fmt.Errorf("run already exists: %s", run.RunID)
	}
	r.runs[run.RunID] = cloneRun(run)
	return nil
}

func (r *MemoryRepo) UpdateRun(_ context.Context, run *entity.PlaygroundRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.runs[run.RunID]; !ok {
		return fmt.Errorf("run not found: %s", run.RunID)
	}
	r.runs[run.RunID] = cloneRun(run)
	return nil
}

func (r *MemoryRepo) GetRun(_ context.Context, runID string) (*entity.PlaygroundRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	run, ok := r.runs[runID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrRunNotFound, runID)
	}
	return cloneRun(run), nil
}

func (r *MemoryRepo) SaveSteps(_ context.Context, steps []*entity.RuntimeStep) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, step := range steps {
		if _, ok := r.steps[step.RunID]; !ok {
			r.steps[step.RunID] = make(map[string]*entity.RuntimeStep)
		}
		if _, ok := r.steps[step.RunID][step.StepID]; ok {
			return fmt.Errorf("step already exists: %s", step.StepID)
		}
		r.steps[step.RunID][step.StepID] = cloneStep(step)
	}
	return nil
}

func (r *MemoryRepo) UpdateStep(_ context.Context, step *entity.RuntimeStep) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.steps[step.RunID]; !ok {
		return fmt.Errorf("step run not found: %s", step.RunID)
	}
	if _, ok := r.steps[step.RunID][step.StepID]; !ok {
		return fmt.Errorf("step not found: %s", step.StepID)
	}
	r.steps[step.RunID][step.StepID] = cloneStep(step)
	return nil
}

func (r *MemoryRepo) ListSteps(_ context.Context, runID string) ([]*entity.RuntimeStep, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	stepMap := r.steps[runID]
	out := make([]*entity.RuntimeStep, 0, len(stepMap))
	for _, step := range stepMap {
		out = append(out, cloneStep(step))
	}
	return out, nil
}

func (r *MemoryRepo) SaveArtifact(_ context.Context, artifact *entity.RuntimeArtifact) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.artifacts[artifact.RunID]; !ok {
		r.artifacts[artifact.RunID] = make(map[string]*entity.RuntimeArtifact)
	}
	if _, ok := r.artifacts[artifact.RunID][artifact.ArtifactID]; ok {
		return fmt.Errorf("artifact already exists: %s", artifact.ArtifactID)
	}
	r.artifacts[artifact.RunID][artifact.ArtifactID] = cloneArtifact(artifact)
	return nil
}

func (r *MemoryRepo) UpdateArtifact(_ context.Context, artifact *entity.RuntimeArtifact) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.artifacts[artifact.RunID]; !ok {
		return fmt.Errorf("artifact run not found: %s", artifact.RunID)
	}
	if _, ok := r.artifacts[artifact.RunID][artifact.ArtifactID]; !ok {
		return fmt.Errorf("artifact not found: %s", artifact.ArtifactID)
	}
	r.artifacts[artifact.RunID][artifact.ArtifactID] = cloneArtifact(artifact)
	return nil
}

func (r *MemoryRepo) ListArtifacts(_ context.Context, runID string) ([]*entity.RuntimeArtifact, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	artifactMap := r.artifacts[runID]
	out := make([]*entity.RuntimeArtifact, 0, len(artifactMap))
	for _, artifact := range artifactMap {
		out = append(out, cloneArtifact(artifact))
	}
	return out, nil
}

func (r *MemoryRepo) SaveRecoveryActions(_ context.Context, actions []*entity.RecoveryAction) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, action := range actions {
		if _, ok := r.actions[action.RunID]; !ok {
			r.actions[action.RunID] = make(map[string]*entity.RecoveryAction)
		}
		if _, ok := r.actions[action.RunID][action.ID]; ok {
			return fmt.Errorf("recovery action already exists: %s", action.ID)
		}
		r.actions[action.RunID][action.ID] = cloneRecoveryAction(action)
	}
	return nil
}

func (r *MemoryRepo) UpdateRecoveryAction(_ context.Context, action *entity.RecoveryAction) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.actions[action.RunID]; !ok {
		return fmt.Errorf("recovery action run not found: %s", action.RunID)
	}
	if _, ok := r.actions[action.RunID][action.ID]; !ok {
		return fmt.Errorf("recovery action not found: %s", action.ID)
	}
	r.actions[action.RunID][action.ID] = cloneRecoveryAction(action)
	return nil
}

func (r *MemoryRepo) ListRecoveryActions(_ context.Context, runID string) ([]*entity.RecoveryAction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	actionMap := r.actions[runID]
	out := make([]*entity.RecoveryAction, 0, len(actionMap))
	for _, action := range actionMap {
		out = append(out, cloneRecoveryAction(action))
	}
	return out, nil
}

func clonePlan(plan *entity.ExecutionPlan) *entity.ExecutionPlan {
	if plan == nil {
		return nil
	}
	cp := *plan
	if plan.EntryStepIDs != nil {
		cp.EntryStepIDs = append([]string(nil), plan.EntryStepIDs...)
	}
	if plan.Steps != nil {
		cp.Steps = make([]*entity.PlanStep, 0, len(plan.Steps))
		for _, step := range plan.Steps {
			cpStep := *step
			if step.DependsOn != nil {
				cpStep.DependsOn = append([]string(nil), step.DependsOn...)
			}
			if step.InputRefs != nil {
				cpStep.InputRefs = append([]string(nil), step.InputRefs...)
			}
			if step.Config != nil {
				cpStep.Config = cloneMap(step.Config)
			}
			cp.Steps = append(cp.Steps, &cpStep)
		}
	}
	if plan.Metadata != nil {
		cp.Metadata = cloneMap(plan.Metadata)
	}
	return &cp
}

func cloneRun(run *entity.PlaygroundRun) *entity.PlaygroundRun {
	if run == nil {
		return nil
	}
	cp := *run
	if run.CurrentStepIDs != nil {
		cp.CurrentStepIDs = append([]string(nil), run.CurrentStepIDs...)
	}
	if run.Metadata != nil {
		cp.Metadata = cloneMap(run.Metadata)
	}
	return &cp
}

func cloneStep(step *entity.RuntimeStep) *entity.RuntimeStep {
	if step == nil {
		return nil
	}
	cp := *step
	if step.InputRefs != nil {
		cp.InputRefs = append([]string(nil), step.InputRefs...)
	}
	if step.Metadata != nil {
		cp.Metadata = cloneMap(step.Metadata)
	}
	return &cp
}

func cloneArtifact(artifact *entity.RuntimeArtifact) *entity.RuntimeArtifact {
	if artifact == nil {
		return nil
	}
	cp := *artifact
	if artifact.Payload != nil {
		cp.Payload = cloneMap(artifact.Payload)
	}
	if artifact.Metadata != nil {
		cp.Metadata = cloneMap(artifact.Metadata)
	}
	return &cp
}

func cloneRecoveryAction(action *entity.RecoveryAction) *entity.RecoveryAction {
	if action == nil {
		return nil
	}
	cp := *action
	if action.Metadata != nil {
		cp.Metadata = cloneMap(action.Metadata)
	}
	return &cp
}

func cloneMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
