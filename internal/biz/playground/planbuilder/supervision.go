package planbuilder

import (
	"fmt"

	"ruleGoKratos/internal/biz/entity"

	"github.com/google/uuid"
)

// SupervisionBuilder 将 supervision 模式编译为 review -> parallel -> review -> finalize。
type SupervisionBuilder struct{}

// NewSupervisionBuilder 创建 supervision 的计划构建器。
func NewSupervisionBuilder() *SupervisionBuilder {
	return &SupervisionBuilder{}
}

// Mode 返回当前 Builder 对应的协作模式。
func (b *SupervisionBuilder) Mode() entity.CollaborationMode {
	return entity.ModeSupervision
}

// Build 生成监督分工、并行执行、复核与收口的计划。
func (b *SupervisionBuilder) Build(scheme *entity.CollaborationScheme, _ string) (*entity.ExecutionPlan, error) {
	if scheme == nil {
		return nil, fmt.Errorf("scheme is nil")
	}

	supervisorID, err := resolveSupervisorAgentID(scheme)
	if err != nil {
		return nil, err
	}
	workerAgents, err := resolveWorkerAgents(scheme, supervisorID)
	if err != nil {
		return nil, err
	}

	return &entity.ExecutionPlan{
		PlanID:       uuid.NewString(),
		PlanVersion:  1,
		SourceMode:   string(entity.ModeSupervision),
		EntryStepIDs: []string{"supervisor_assign"},
		Steps: []*entity.PlanStep{
			{
				StepID:       "supervisor_assign",
				Kind:         entity.StepKindReview,
				Name:         "supervisor_assign",
				AgentBinding: supervisorID,
				OutputRef:    "assignment",
				Config: map[string]any{
					"workers": workerAgents,
				},
			},
			{
				StepID:    "workers",
				Kind:      entity.StepKindParallel,
				Name:      "workers",
				DependsOn: []string{"supervisor_assign"},
				InputRefs: []string{"assignment"},
				OutputRef: "worker_results",
				Config: map[string]any{
					"workers": workerAgents,
				},
			},
			{
				StepID:       "supervisor_review",
				Kind:         entity.StepKindReview,
				Name:         "supervisor_review",
				DependsOn:    []string{"workers"},
				AgentBinding: supervisorID,
				InputRefs:    []string{"worker_results"},
				OutputRef:    "review_result",
			},
			{
				StepID:    "finalize",
				Kind:      entity.StepKindFinalize,
				Name:      "finalize",
				DependsOn: []string{"supervisor_review"},
				InputRefs: []string{"review_result"},
				OutputRef: "final_output",
			},
		},
		Metadata: map[string]any{
			"schemeId": scheme.ID,
		},
	}, nil
}

func resolveSupervisorAgentID(scheme *entity.CollaborationScheme) (string, error) {
	if scheme == nil {
		return "", fmt.Errorf("scheme is nil")
	}
	if scheme.Config != nil &&
		scheme.Config.ModeConfig != nil &&
		scheme.Config.ModeConfig.SupervisionConfig != nil &&
		scheme.Config.ModeConfig.SupervisionConfig.SupervisorAgent != "" {
		return scheme.Config.ModeConfig.SupervisionConfig.SupervisorAgent, nil
	}
	for _, binding := range scheme.BindAgents {
		if binding != nil && binding.AgentID == "supervisor" {
			return binding.AgentID, nil
		}
	}
	for _, binding := range scheme.BindAgents {
		if binding != nil && binding.AgentID != "" {
			return binding.AgentID, nil
		}
	}
	return "", fmt.Errorf("supervision requires at least one bound agent")
}

func resolveWorkerAgents(scheme *entity.CollaborationScheme, supervisorID string) ([]string, error) {
	if scheme == nil {
		return nil, fmt.Errorf("scheme is nil")
	}
	if scheme.Config != nil &&
		scheme.Config.ModeConfig != nil &&
		scheme.Config.ModeConfig.SupervisionConfig != nil &&
		len(scheme.Config.ModeConfig.SupervisionConfig.WorkerAgents) > 0 {
		return append([]string(nil), scheme.Config.ModeConfig.SupervisionConfig.WorkerAgents...), nil
	}

	workers := make([]string, 0, len(scheme.BindAgents))
	for _, binding := range scheme.BindAgents {
		if binding == nil || binding.AgentID == "" || binding.AgentID == supervisorID {
			continue
		}
		workers = append(workers, binding.AgentID)
	}
	if len(workers) == 0 {
		return nil, fmt.Errorf("supervision requires worker agents")
	}
	return workers, nil
}
