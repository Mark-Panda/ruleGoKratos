package runtime

import (
	"context"
	"errors"

	"ruleGoKratos/internal/biz/entity"
)

// ErrRunNotFound 表示指定 runID 的 runtime run 不存在。
var ErrRunNotFound = errors.New("runtime run not found")

// Repo 定义运行时计划、执行态与恢复动作的持久化能力。
type Repo interface {
	// SavePlan 创建执行计划。
	SavePlan(ctx context.Context, plan *entity.ExecutionPlan) error
	// UpdatePlan 更新已有执行计划。
	UpdatePlan(ctx context.Context, plan *entity.ExecutionPlan) error
	// GetPlan 读取单个执行计划。
	GetPlan(ctx context.Context, planID string) (*entity.ExecutionPlan, error)

	// SaveRun 创建运行实例。
	SaveRun(ctx context.Context, run *entity.PlaygroundRun) error
	// UpdateRun 更新已有运行实例。
	UpdateRun(ctx context.Context, run *entity.PlaygroundRun) error
	// GetRun 读取单个运行实例。
	GetRun(ctx context.Context, runID string) (*entity.PlaygroundRun, error)

	// SaveSteps 批量创建运行步骤。
	SaveSteps(ctx context.Context, steps []*entity.RuntimeStep) error
	// UpdateStep 更新已有运行步骤。
	UpdateStep(ctx context.Context, step *entity.RuntimeStep) error
	// ListSteps 列出某次运行的全部步骤。
	ListSteps(ctx context.Context, runID string) ([]*entity.RuntimeStep, error)

	// SaveArtifact 创建运行产物。
	SaveArtifact(ctx context.Context, artifact *entity.RuntimeArtifact) error
	// UpdateArtifact 更新已有运行产物。
	UpdateArtifact(ctx context.Context, artifact *entity.RuntimeArtifact) error
	ListArtifacts(ctx context.Context, runID string) ([]*entity.RuntimeArtifact, error)

	// SaveRecoveryActions 批量创建恢复动作。
	SaveRecoveryActions(ctx context.Context, actions []*entity.RecoveryAction) error
	// UpdateRecoveryAction 更新已有恢复动作。
	UpdateRecoveryAction(ctx context.Context, action *entity.RecoveryAction) error
	ListRecoveryActions(ctx context.Context, runID string) ([]*entity.RecoveryAction, error)
}
