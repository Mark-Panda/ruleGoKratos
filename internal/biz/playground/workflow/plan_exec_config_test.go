package workflow

import (
	"testing"

	"ruleGoKratos/internal/biz/entity"
)

func TestEnsurePlanExecModeConfig_OrderFromBindings(t *testing.T) {
	cfg := *entity.DefaultSchemeConfig
	scheme := &entity.CollaborationScheme{
		Mode: entity.ModePlanExec,
		BindAgents: []*entity.AgentBinding{
			{AgentID: "planner", Role: "规划师"},
			{AgentID: "designer", Role: "设计师"},
			{AgentID: "pm", Role: "产品经理"},
		},
		Config: &cfg,
	}
	EnsurePlanExecModeConfig(scheme)
	pec := scheme.Config.ModeConfig.PlanExecConfig
	if pec.PlannerAgent != "planner" {
		t.Fatalf("PlannerAgent: want planner, got %q", pec.PlannerAgent)
	}
	if len(pec.ExecutionOrder) != 2 || pec.ExecutionOrder[0] != "designer" || pec.ExecutionOrder[1] != "pm" {
		t.Fatalf("ExecutionOrder: %+v", pec.ExecutionOrder)
	}
}
