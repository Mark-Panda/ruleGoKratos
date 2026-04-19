package planbuilder

import (
	"fmt"

	"ruleGoKratos/internal/biz/entity"

	"github.com/google/uuid"
)

// RouterExpertBuilder 将 router_expert 模式编译为 route -> agent -> finalize。
type RouterExpertBuilder struct{}

// NewRouterExpertBuilder 创建 router_expert 的计划构建器。
func NewRouterExpertBuilder() *RouterExpertBuilder {
	return &RouterExpertBuilder{}
}

// Mode 返回当前 Builder 对应的协作模式。
func (b *RouterExpertBuilder) Mode() entity.CollaborationMode {
	return entity.ModeRouterExpert
}

// Build 生成 router_expert 的最小可执行计划。
func (b *RouterExpertBuilder) Build(scheme *entity.CollaborationScheme, _ string) (*entity.ExecutionPlan, error) {
	if scheme == nil {
		return nil, fmt.Errorf("scheme is nil")
	}
	if len(scheme.BindAgents) == 0 {
		return nil, fmt.Errorf("router_expert requires at least one bound agent")
	}

	candidateAgents := make([]string, 0, len(scheme.BindAgents))
	for _, binding := range scheme.BindAgents {
		if binding == nil || binding.AgentID == "" {
			continue
		}
		candidateAgents = append(candidateAgents, binding.AgentID)
	}
	if len(candidateAgents) == 0 {
		return nil, fmt.Errorf("router_expert requires non-empty candidate agents")
	}

	fallbackAgent := ""
	if scheme.Config != nil && scheme.Config.ModeConfig != nil && scheme.Config.ModeConfig.RouterConfig != nil {
		fallbackAgent = scheme.Config.ModeConfig.RouterConfig.FallbackAgent
	}
	if fallbackAgent == "" {
		fallbackAgent = candidateAgents[0]
	}

	return &entity.ExecutionPlan{
		PlanID:       uuid.NewString(),
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
					"candidateAgents": candidateAgents,
					"fallbackAgent":   fallbackAgent,
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
		Metadata: map[string]any{
			"schemeId": scheme.ID,
		},
	}, nil
}
