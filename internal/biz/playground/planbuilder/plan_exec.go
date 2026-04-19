package planbuilder

import (
	"fmt"

	"ruleGoKratos/internal/biz/entity"

	"github.com/google/uuid"
)

// PlanExecBuilder 将 plan_exec 模式编译为规划后顺序执行的 Agent 链。
type PlanExecBuilder struct{}

// NewPlanExecBuilder 创建 plan_exec 的计划构建器。
func NewPlanExecBuilder() *PlanExecBuilder {
	return &PlanExecBuilder{}
}

// Mode 返回当前 Builder 对应的协作模式。
func (b *PlanExecBuilder) Mode() entity.CollaborationMode {
	return entity.ModePlanExec
}

// Build 生成 planner -> agent... -> finalize 的顺序执行计划。
func (b *PlanExecBuilder) Build(scheme *entity.CollaborationScheme, _ string) (*entity.ExecutionPlan, error) {
	if scheme == nil {
		return nil, fmt.Errorf("scheme is nil")
	}

	plannerID, err := resolvePlannerAgentID(scheme)
	if err != nil {
		return nil, err
	}
	executionOrder, err := resolvePlanExecOrder(scheme, plannerID)
	if err != nil {
		return nil, err
	}

	steps := make([]*entity.PlanStep, 0, len(executionOrder)+2)
	steps = append(steps, &entity.PlanStep{
		StepID:       "planner",
		Kind:         entity.StepKindAgent,
		Name:         "planner",
		AgentBinding: plannerID,
		OutputRef:    "plan_outline",
	})

	prevStepID := "planner"
	prevOutputRef := "plan_outline"
	for i, agentID := range executionOrder {
		stepID := fmt.Sprintf("step_%d", i+1)
		outputRef := fmt.Sprintf("step_%d_output", i+1)
		steps = append(steps, &entity.PlanStep{
			StepID:       stepID,
			Kind:         entity.StepKindAgent,
			Name:         stepID,
			DependsOn:    []string{prevStepID},
			AgentBinding: agentID,
			InputRefs:    []string{prevOutputRef},
			OutputRef:    outputRef,
		})
		prevStepID = stepID
		prevOutputRef = outputRef
	}

	steps = append(steps, &entity.PlanStep{
		StepID:    "finalize",
		Kind:      entity.StepKindFinalize,
		Name:      "finalize",
		DependsOn: []string{prevStepID},
		InputRefs: []string{prevOutputRef},
		OutputRef: "final_output",
	})

	return &entity.ExecutionPlan{
		PlanID:       uuid.NewString(),
		PlanVersion:  1,
		SourceMode:   string(entity.ModePlanExec),
		EntryStepIDs: []string{"planner"},
		Steps:        steps,
		Metadata: map[string]any{
			"schemeId": scheme.ID,
		},
	}, nil
}

func resolvePlannerAgentID(scheme *entity.CollaborationScheme) (string, error) {
	if scheme == nil {
		return "", fmt.Errorf("scheme is nil")
	}
	if scheme.Config != nil &&
		scheme.Config.ModeConfig != nil &&
		scheme.Config.ModeConfig.PlanExecConfig != nil &&
		scheme.Config.ModeConfig.PlanExecConfig.PlannerAgent != "" {
		return scheme.Config.ModeConfig.PlanExecConfig.PlannerAgent, nil
	}
	for _, binding := range scheme.BindAgents {
		if binding != nil && binding.AgentID == "planner" {
			return binding.AgentID, nil
		}
	}
	for _, binding := range scheme.BindAgents {
		if binding != nil && binding.AgentID != "" {
			return binding.AgentID, nil
		}
	}
	return "", fmt.Errorf("plan_exec requires at least one bound agent")
}

func resolvePlanExecOrder(scheme *entity.CollaborationScheme, plannerID string) ([]string, error) {
	if scheme == nil {
		return nil, fmt.Errorf("scheme is nil")
	}
	if scheme.Config != nil &&
		scheme.Config.ModeConfig != nil &&
		scheme.Config.ModeConfig.PlanExecConfig != nil &&
		len(scheme.Config.ModeConfig.PlanExecConfig.ExecutionOrder) > 0 {
		return append([]string(nil), scheme.Config.ModeConfig.PlanExecConfig.ExecutionOrder...), nil
	}

	order := make([]string, 0, len(scheme.BindAgents))
	for _, binding := range scheme.BindAgents {
		if binding == nil || binding.AgentID == "" || binding.AgentID == plannerID {
			continue
		}
		order = append(order, binding.AgentID)
	}
	if len(order) == 0 {
		return nil, fmt.Errorf("plan_exec requires execution agents")
	}
	return order, nil
}
