package runtime

import (
	"context"
	"fmt"

	"ruleGoKratos/internal/biz/entity"
)

// FinalizeStepExecutor 执行最终结果收口。
type FinalizeStepExecutor struct{}

// NewFinalizeStepExecutor 创建 FinalizeStepExecutor。
func NewFinalizeStepExecutor() *FinalizeStepExecutor {
	return &FinalizeStepExecutor{}
}

// Execute 将最后一个输入 artifact 收束为 final_answer。
func (e *FinalizeStepExecutor) Execute(
	ctx context.Context,
	execCtx *executionContext,
	step *entity.PlanStep,
) (*entity.RuntimeArtifact, string, error) {
	source, err := loadArtifactForRefs(ctx, execCtx, step.InputRefs)
	if err != nil {
		return nil, "", err
	}
	if source == nil {
		return nil, "", fmt.Errorf("finalize step %s missing input artifact", step.StepID)
	}

	output := artifactText(source)
	if output == "" {
		output = source.Summary
	}
	artifact := newArtifact(execCtx.runID, step.StepID, "final_answer", output, map[string]any{
		"text": output,
	})
	return artifact, output, nil
}
