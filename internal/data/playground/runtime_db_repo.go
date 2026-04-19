package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"ruleGoKratos/internal/biz/entity"
	playgroundruntime "ruleGoKratos/internal/biz/playground/runtime"

	"gorm.io/gorm"
)

// GormRuntimeRepo 提供 runtime 仓储的 PostgreSQL 持久化接点。
type GormRuntimeRepo struct {
	db        *gorm.DB
	initOnce  sync.Once
	schemaErr error
}

type runtimePlanRecord struct {
	PlanID    string     `gorm:"column:plan_id;primaryKey"`
	PlanJSON  string     `gorm:"column:plan_json;type:text;not null"`
	CreatedAt *time.Time `gorm:"column:created_at"`
	UpdatedAt *time.Time `gorm:"column:updated_at"`
}

func (runtimePlanRecord) TableName() string {
	return "playground_runtime_plan"
}

type runtimeRunRecord struct {
	RunID      string     `gorm:"column:run_id;primaryKey"`
	Status     string     `gorm:"column:status"`
	RunJSON    string     `gorm:"column:run_json;type:text;not null"`
	CreatedAt  *time.Time `gorm:"column:created_at"`
	UpdatedAt  *time.Time `gorm:"column:updated_at"`
	FinishedAt *time.Time `gorm:"column:finished_at"`
}

func (runtimeRunRecord) TableName() string {
	return "playground_runtime_run"
}

type runtimeStepRecord struct {
	RunID      string     `gorm:"column:run_id;primaryKey"`
	StepID     string     `gorm:"column:step_id;primaryKey"`
	Status     string     `gorm:"column:status"`
	StepJSON   string     `gorm:"column:step_json;type:text;not null"`
	StartedAt  *time.Time `gorm:"column:started_at"`
	FinishedAt *time.Time `gorm:"column:finished_at"`
	CreatedAt  *time.Time `gorm:"column:created_at"`
	UpdatedAt  *time.Time `gorm:"column:updated_at"`
}

func (runtimeStepRecord) TableName() string {
	return "playground_runtime_step"
}

type runtimeArtifactRecord struct {
	RunID        string     `gorm:"column:run_id;index"`
	ArtifactID   string     `gorm:"column:artifact_id;primaryKey"`
	Type         string     `gorm:"column:type"`
	ArtifactJSON string     `gorm:"column:artifact_json;type:text;not null"`
	CreatedAt    *time.Time `gorm:"column:created_at"`
}

func (runtimeArtifactRecord) TableName() string {
	return "playground_runtime_artifact"
}

type runtimeRecoveryActionRecord struct {
	RunID      string     `gorm:"column:run_id;index"`
	ActionID   string     `gorm:"column:action_id;primaryKey"`
	Type       string     `gorm:"column:type"`
	ActionJSON string     `gorm:"column:action_json;type:text;not null"`
	CreatedAt  *time.Time `gorm:"column:created_at"`
	UpdatedAt  *time.Time `gorm:"column:updated_at"`
}

func (runtimeRecoveryActionRecord) TableName() string {
	return "playground_runtime_recovery_action"
}

// NewGormRuntimeRepo 由 Wire 注入 *gorm.DB，供后续切换真实持久化时复用。
func NewGormRuntimeRepo(db *gorm.DB) *GormRuntimeRepo {
	return &GormRuntimeRepo{db: db}
}

func (r *GormRuntimeRepo) ensureSchema(ctx context.Context) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("runtime repo db is nil")
	}
	r.initOnce.Do(func() {
		db := r.db
		if ctx != nil {
			db = db.WithContext(ctx)
		}
		r.schemaErr = db.AutoMigrate(
			&runtimePlanRecord{},
			&runtimeRunRecord{},
			&runtimeStepRecord{},
			&runtimeArtifactRecord{},
			&runtimeRecoveryActionRecord{},
		)
	})
	return r.schemaErr
}

func (r *GormRuntimeRepo) SavePlan(ctx context.Context, plan *entity.ExecutionPlan) error {
	if plan == nil || plan.PlanID == "" {
		return fmt.Errorf("plan id is empty")
	}
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}
	var n int64
	if err := r.db.WithContext(ctx).Model(&runtimePlanRecord{}).Where("plan_id = ?", plan.PlanID).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("plan already exists: %s", plan.PlanID)
	}
	row, err := runtimePlanToRecord(plan)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *GormRuntimeRepo) UpdatePlan(ctx context.Context, plan *entity.ExecutionPlan) error {
	if plan == nil || plan.PlanID == "" {
		return fmt.Errorf("plan id is empty")
	}
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}
	var n int64
	if err := r.db.WithContext(ctx).Model(&runtimePlanRecord{}).Where("plan_id = ?", plan.PlanID).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("plan not found: %s", plan.PlanID)
	}
	row, err := runtimePlanToRecord(plan)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *GormRuntimeRepo) GetPlan(ctx context.Context, planID string) (*entity.ExecutionPlan, error) {
	if err := r.ensureSchema(ctx); err != nil {
		return nil, err
	}
	var row runtimePlanRecord
	if err := r.db.WithContext(ctx).Where("plan_id = ?", planID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("plan not found: %s", planID)
		}
		return nil, err
	}
	return recordToRuntimePlan(&row)
}

func (r *GormRuntimeRepo) SaveRun(ctx context.Context, run *entity.PlaygroundRun) error {
	if run == nil || run.RunID == "" {
		return fmt.Errorf("run id is empty")
	}
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}
	var n int64
	if err := r.db.WithContext(ctx).Model(&runtimeRunRecord{}).Where("run_id = ?", run.RunID).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("run already exists: %s", run.RunID)
	}
	row, err := runtimeRunToRecord(run)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *GormRuntimeRepo) UpdateRun(ctx context.Context, run *entity.PlaygroundRun) error {
	if run == nil || run.RunID == "" {
		return fmt.Errorf("run id is empty")
	}
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}
	var n int64
	if err := r.db.WithContext(ctx).Model(&runtimeRunRecord{}).Where("run_id = ?", run.RunID).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("run not found: %s", run.RunID)
	}
	row, err := runtimeRunToRecord(run)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *GormRuntimeRepo) GetRun(ctx context.Context, runID string) (*entity.PlaygroundRun, error) {
	if err := r.ensureSchema(ctx); err != nil {
		return nil, err
	}
	var row runtimeRunRecord
	if err := r.db.WithContext(ctx).Where("run_id = ?", runID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: %s", playgroundruntime.ErrRunNotFound, runID)
		}
		return nil, err
	}
	return recordToRuntimeRun(&row)
}

func (r *GormRuntimeRepo) SaveSteps(ctx context.Context, steps []*entity.RuntimeStep) error {
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, step := range steps {
			if step == nil || step.RunID == "" || step.StepID == "" {
				return fmt.Errorf("runtime step id is empty")
			}
			var n int64
			if err := tx.Model(&runtimeStepRecord{}).
				Where("run_id = ? AND step_id = ?", step.RunID, step.StepID).
				Count(&n).Error; err != nil {
				return err
			}
			if n > 0 {
				return fmt.Errorf("step already exists: %s", step.StepID)
			}
			row, err := runtimeStepToRecord(step)
			if err != nil {
				return err
			}
			if err := tx.Create(row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *GormRuntimeRepo) UpdateStep(ctx context.Context, step *entity.RuntimeStep) error {
	if step == nil || step.RunID == "" || step.StepID == "" {
		return fmt.Errorf("runtime step id is empty")
	}
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}
	var n int64
	if err := r.db.WithContext(ctx).Model(&runtimeStepRecord{}).
		Where("run_id = ? AND step_id = ?", step.RunID, step.StepID).
		Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("step not found: %s", step.StepID)
	}
	row, err := runtimeStepToRecord(step)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *GormRuntimeRepo) ListSteps(ctx context.Context, runID string) ([]*entity.RuntimeStep, error) {
	if err := r.ensureSchema(ctx); err != nil {
		return nil, err
	}
	var rows []runtimeStepRecord
	if err := r.db.WithContext(ctx).
		Where("run_id = ?", runID).
		Order("created_at ASC NULLS LAST, step_id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	steps := make([]*entity.RuntimeStep, 0, len(rows))
	for i := range rows {
		step, err := recordToRuntimeStep(&rows[i])
		if err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}
	return steps, nil
}

func (r *GormRuntimeRepo) SaveArtifact(ctx context.Context, artifact *entity.RuntimeArtifact) error {
	if artifact == nil || artifact.ArtifactID == "" {
		return fmt.Errorf("artifact id is empty")
	}
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}
	var n int64
	if err := r.db.WithContext(ctx).Model(&runtimeArtifactRecord{}).
		Where("artifact_id = ?", artifact.ArtifactID).
		Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("artifact already exists: %s", artifact.ArtifactID)
	}
	row, err := runtimeArtifactToRecord(artifact)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *GormRuntimeRepo) UpdateArtifact(ctx context.Context, artifact *entity.RuntimeArtifact) error {
	if artifact == nil || artifact.ArtifactID == "" {
		return fmt.Errorf("artifact id is empty")
	}
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}
	var n int64
	if err := r.db.WithContext(ctx).Model(&runtimeArtifactRecord{}).
		Where("artifact_id = ?", artifact.ArtifactID).
		Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("artifact not found: %s", artifact.ArtifactID)
	}
	row, err := runtimeArtifactToRecord(artifact)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *GormRuntimeRepo) ListArtifacts(ctx context.Context, runID string) ([]*entity.RuntimeArtifact, error) {
	if err := r.ensureSchema(ctx); err != nil {
		return nil, err
	}
	var rows []runtimeArtifactRecord
	if err := r.db.WithContext(ctx).
		Where("run_id = ?", runID).
		Order("created_at ASC NULLS LAST, artifact_id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	artifacts := make([]*entity.RuntimeArtifact, 0, len(rows))
	for i := range rows {
		artifact, err := recordToRuntimeArtifact(&rows[i])
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts, nil
}

func (r *GormRuntimeRepo) SaveRecoveryActions(ctx context.Context, actions []*entity.RecoveryAction) error {
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, action := range actions {
			if action == nil || action.ID == "" {
				return fmt.Errorf("recovery action id is empty")
			}
			var n int64
			if err := tx.Model(&runtimeRecoveryActionRecord{}).
				Where("action_id = ?", action.ID).
				Count(&n).Error; err != nil {
				return err
			}
			if n > 0 {
				return fmt.Errorf("recovery action already exists: %s", action.ID)
			}
			row, err := runtimeRecoveryActionToRecord(action)
			if err != nil {
				return err
			}
			if err := tx.Create(row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *GormRuntimeRepo) UpdateRecoveryAction(ctx context.Context, action *entity.RecoveryAction) error {
	if action == nil || action.ID == "" {
		return fmt.Errorf("recovery action id is empty")
	}
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}
	var n int64
	if err := r.db.WithContext(ctx).Model(&runtimeRecoveryActionRecord{}).
		Where("action_id = ?", action.ID).
		Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("recovery action not found: %s", action.ID)
	}
	row, err := runtimeRecoveryActionToRecord(action)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *GormRuntimeRepo) ListRecoveryActions(ctx context.Context, runID string) ([]*entity.RecoveryAction, error) {
	if err := r.ensureSchema(ctx); err != nil {
		return nil, err
	}
	var rows []runtimeRecoveryActionRecord
	if err := r.db.WithContext(ctx).
		Where("run_id = ?", runID).
		Order("created_at ASC NULLS LAST, action_id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	actions := make([]*entity.RecoveryAction, 0, len(rows))
	for i := range rows {
		action, err := recordToRuntimeRecoveryAction(&rows[i])
		if err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}
	return actions, nil
}

func runtimePlanToRecord(plan *entity.ExecutionPlan) (*runtimePlanRecord, error) {
	raw, err := json.Marshal(plan)
	if err != nil {
		return nil, fmt.Errorf("marshal runtime plan: %w", err)
	}
	now := time.Now()
	return &runtimePlanRecord{
		PlanID:    plan.PlanID,
		PlanJSON:  string(raw),
		CreatedAt: &now,
		UpdatedAt: &now,
	}, nil
}

func recordToRuntimePlan(row *runtimePlanRecord) (*entity.ExecutionPlan, error) {
	var plan entity.ExecutionPlan
	if err := json.Unmarshal([]byte(row.PlanJSON), &plan); err != nil {
		return nil, fmt.Errorf("decode plan_json: %w", err)
	}
	return &plan, nil
}

func runtimeRunToRecord(run *entity.PlaygroundRun) (*runtimeRunRecord, error) {
	raw, err := json.Marshal(run)
	if err != nil {
		return nil, fmt.Errorf("marshal runtime run: %w", err)
	}
	now := time.Now()
	createdAt := run.StartedAt
	if createdAt == nil {
		createdAt = &now
	}
	return &runtimeRunRecord{
		RunID:      run.RunID,
		Status:     string(run.Status),
		RunJSON:    string(raw),
		CreatedAt:  createdAt,
		UpdatedAt:  &now,
		FinishedAt: run.FinishedAt,
	}, nil
}

func recordToRuntimeRun(row *runtimeRunRecord) (*entity.PlaygroundRun, error) {
	var run entity.PlaygroundRun
	if err := json.Unmarshal([]byte(row.RunJSON), &run); err != nil {
		return nil, fmt.Errorf("decode run_json: %w", err)
	}
	return &run, nil
}

func runtimeStepToRecord(step *entity.RuntimeStep) (*runtimeStepRecord, error) {
	raw, err := json.Marshal(step)
	if err != nil {
		return nil, fmt.Errorf("marshal runtime step: %w", err)
	}
	now := time.Now()
	createdAt := step.StartedAt
	if createdAt == nil {
		createdAt = &now
	}
	return &runtimeStepRecord{
		RunID:      step.RunID,
		StepID:     step.StepID,
		Status:     string(step.Status),
		StepJSON:   string(raw),
		StartedAt:  step.StartedAt,
		FinishedAt: step.FinishedAt,
		CreatedAt:  createdAt,
		UpdatedAt:  &now,
	}, nil
}

func recordToRuntimeStep(row *runtimeStepRecord) (*entity.RuntimeStep, error) {
	var step entity.RuntimeStep
	if err := json.Unmarshal([]byte(row.StepJSON), &step); err != nil {
		return nil, fmt.Errorf("decode step_json: %w", err)
	}
	return &step, nil
}

func runtimeArtifactToRecord(artifact *entity.RuntimeArtifact) (*runtimeArtifactRecord, error) {
	raw, err := json.Marshal(artifact)
	if err != nil {
		return nil, fmt.Errorf("marshal runtime artifact: %w", err)
	}
	now := time.Now()
	createdAt := artifact.CreatedAt
	if createdAt == nil {
		createdAt = &now
	}
	return &runtimeArtifactRecord{
		RunID:        artifact.RunID,
		ArtifactID:   artifact.ArtifactID,
		Type:         artifact.Type,
		ArtifactJSON: string(raw),
		CreatedAt:    createdAt,
	}, nil
}

func recordToRuntimeArtifact(row *runtimeArtifactRecord) (*entity.RuntimeArtifact, error) {
	var artifact entity.RuntimeArtifact
	if err := json.Unmarshal([]byte(row.ArtifactJSON), &artifact); err != nil {
		return nil, fmt.Errorf("decode artifact_json: %w", err)
	}
	return &artifact, nil
}

func runtimeRecoveryActionToRecord(action *entity.RecoveryAction) (*runtimeRecoveryActionRecord, error) {
	raw, err := json.Marshal(action)
	if err != nil {
		return nil, fmt.Errorf("marshal runtime recovery action: %w", err)
	}
	now := time.Now()
	createdAt := action.CreatedAt
	if createdAt == nil {
		createdAt = &now
	}
	return &runtimeRecoveryActionRecord{
		RunID:      action.RunID,
		ActionID:   action.ID,
		Type:       string(action.Type),
		ActionJSON: string(raw),
		CreatedAt:  createdAt,
		UpdatedAt:  &now,
	}, nil
}

func recordToRuntimeRecoveryAction(row *runtimeRecoveryActionRecord) (*entity.RecoveryAction, error) {
	var action entity.RecoveryAction
	if err := json.Unmarshal([]byte(row.ActionJSON), &action); err != nil {
		return nil, fmt.Errorf("decode action_json: %w", err)
	}
	return &action, nil
}
