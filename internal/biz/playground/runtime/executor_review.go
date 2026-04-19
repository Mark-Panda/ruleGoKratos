package runtime

import (
	"context"
	"fmt"
	"strings"

	"ruleGoKratos/internal/biz/entity"
)

// ReviewStepExecutor 执行最小审查/汇总步骤。
type ReviewStepExecutor struct {
	runner AgentRunner
}

// NewReviewStepExecutor 创建 ReviewStepExecutor。
func NewReviewStepExecutor(runner AgentRunner) *ReviewStepExecutor {
	return &ReviewStepExecutor{runner: runner}
}

// Execute 汇总输入 artifact，并输出 review_result。
func (e *ReviewStepExecutor) Execute(
	ctx context.Context,
	execCtx *executionContext,
	step *entity.PlanStep,
) (*entity.RuntimeArtifact, error) {
	if e == nil || e.runner == nil {
		return nil, fmt.Errorf("review runner is nil")
	}
	if step == nil {
		return nil, fmt.Errorf("review step is nil")
	}

	reviewer := strings.TrimSpace(step.AgentBinding)
	agentDef, err := resolveAgentByID(execCtx, reviewer, step.StepID)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"reviewer_agent": agentDef.ID,
		"step_name":      step.Name,
	}
	if workers := stringSliceConfig(step.Config["workers"]); len(workers) > 0 {
		payload["workers"] = workers
	}

	userInput, err := buildReviewUserInput(ctx, execCtx, step, agentDef.ID)
	if err != nil {
		return nil, err
	}
	output, err := e.runner(ctx, execCtx.runID, agentDef, userInput, execCtx.trace, schemeConfigOf(execCtx.scheme))
	if err != nil {
		return nil, err
	}
	summary := strings.TrimSpace(output)
	if summary == "" {
		summary = fmt.Sprintf("[%s] 已完成审查", agentDef.ID)
	}
	if len(step.InputRefs) > 0 {
		payload["input_refs"] = append([]string(nil), step.InputRefs...)
	}
	payload["review_summary"] = summary

	artifactType := "review_result"
	if strings.Contains(step.StepID, "assign") {
		artifactType = "assignment_result"
	}
	return newArtifact(execCtx.runID, step.StepID, artifactType, summary, payload), nil
}

func buildReviewUserInput(
	ctx context.Context,
	execCtx *executionContext,
	step *entity.PlanStep,
	reviewer string,
) (string, error) {
	baseInput, err := buildStepUserInput(ctx, execCtx, step)
	if err != nil {
		return "", err
	}
	var prompt strings.Builder
	prompt.WriteString("请作为 ")
	prompt.WriteString(reviewer)
	prompt.WriteString(" 完成当前审查/汇总步骤。请直接输出简洁结论。")
	if workers := stringSliceConfig(step.Config["workers"]); len(workers) > 0 {
		prompt.WriteString("\n并行参与者：")
		prompt.WriteString(strings.Join(workers, ", "))
	}
	if strings.TrimSpace(baseInput) != "" {
		prompt.WriteString("\n\n输入上下文：\n")
		prompt.WriteString(baseInput)
	}
	return prompt.String(), nil
}
