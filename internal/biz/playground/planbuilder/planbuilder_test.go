package planbuilder_test

import (
	"testing"

	"ruleGoKratos/internal/biz/entity"
	"ruleGoKratos/internal/biz/playground/planbuilder"
)

func TestRegistryRegistersAndResolvesBuilder(t *testing.T) {
	registry := planbuilder.NewRegistry()
	builder := planbuilder.NewRouterExpertBuilder()

	registry.Register(builder)

	got, ok := registry.Get(entity.ModeRouterExpert)
	if !ok {
		t.Fatal("expected router_expert builder to be registered")
	}
	if got.Mode() != entity.ModeRouterExpert {
		t.Fatalf("expected mode %q, got %q", entity.ModeRouterExpert, got.Mode())
	}
}

func TestRouterExpertBuilderBuildsThreeSteps(t *testing.T) {
	builder := planbuilder.NewRouterExpertBuilder()
	scheme := &entity.CollaborationScheme{
		ID:   "scheme-1",
		Mode: entity.ModeRouterExpert,
		BindAgents: []*entity.AgentBinding{
			{AgentID: "designer", Role: "设计师"},
			{AgentID: "engineer", Role: "工程师"},
		},
		Config: &entity.SchemeConfig{
			ModeConfig: &entity.ModeConfig{
				RouterConfig: &entity.RouterConfig{
					FallbackAgent: "designer",
				},
			},
		},
	}

	plan, err := builder.Build(scheme, "请实现一个登录页")
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if got, want := len(plan.Steps), 3; got != want {
		t.Fatalf("expected %d steps, got %d", want, got)
	}
	if got, want := plan.EntryStepIDs, []string{"route"}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("unexpected entry steps: %#v", got)
	}

	kinds := []entity.StepKind{plan.Steps[0].Kind, plan.Steps[1].Kind, plan.Steps[2].Kind}
	wantKinds := []entity.StepKind{entity.StepKindRoute, entity.StepKindAgent, entity.StepKindFinalize}
	for i := range wantKinds {
		if kinds[i] != wantKinds[i] {
			t.Fatalf("step %d kind = %q, want %q", i, kinds[i], wantKinds[i])
		}
	}

	if got := plan.Steps[0].OutputRef; got != "route_result" {
		t.Fatalf("expected route output ref, got %q", got)
	}
	if got := plan.Steps[1].InputRefs; len(got) != 1 || got[0] != "route_result" {
		t.Fatalf("expected agent step to depend on route_result, got %#v", got)
	}
	if got := plan.Steps[2].InputRefs; len(got) != 1 || got[0] != "agent_output" {
		t.Fatalf("expected finalize step to depend on agent_output, got %#v", got)
	}
}

func TestRouterExpertBuilderEmbedsRouteConfig(t *testing.T) {
	builder := planbuilder.NewRouterExpertBuilder()
	scheme := &entity.CollaborationScheme{
		ID:   "scheme-1",
		Mode: entity.ModeRouterExpert,
		BindAgents: []*entity.AgentBinding{
			{AgentID: "designer", Role: "设计师"},
			{AgentID: "engineer", Role: "工程师"},
		},
	}

	plan, err := builder.Build(scheme, "实现一个登录页")
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	routeStep := plan.Steps[0]
	candidateAgents, ok := routeStep.Config["candidateAgents"].([]string)
	if !ok {
		t.Fatalf("expected candidateAgents in route config, got %#v", routeStep.Config["candidateAgents"])
	}
	if got, want := candidateAgents, []string{"designer", "engineer"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("unexpected candidateAgents: %#v", got)
	}

	fallbackAgent, ok := routeStep.Config["fallbackAgent"].(string)
	if !ok {
		t.Fatalf("expected fallbackAgent in route config, got %#v", routeStep.Config["fallbackAgent"])
	}
	if fallbackAgent != "designer" {
		t.Fatalf("expected default fallback agent %q, got %q", "designer", fallbackAgent)
	}
}

func TestPlanExecBuilderUsesSequentialAgents(t *testing.T) {
	builder := planbuilder.NewPlanExecBuilder()
	scheme := &entity.CollaborationScheme{
		ID:   "scheme-plan-exec",
		Mode: entity.ModePlanExec,
		BindAgents: []*entity.AgentBinding{
			{AgentID: "planner", Role: "规划师"},
			{AgentID: "designer", Role: "设计师"},
			{AgentID: "engineer", Role: "工程师"},
		},
		Config: &entity.SchemeConfig{
			ModeConfig: &entity.ModeConfig{
				PlanExecConfig: &entity.PlanExecConfig{
					PlannerAgent:   "planner",
					ExecutionOrder: []string{"designer", "engineer"},
				},
			},
		},
	}

	plan, err := builder.Build(scheme, "做一个搜索页")
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	if got, want := len(plan.Steps), 4; got != want {
		t.Fatalf("expected %d steps, got %d", want, got)
	}
	wantKinds := []entity.StepKind{
		entity.StepKindAgent,
		entity.StepKindAgent,
		entity.StepKindAgent,
		entity.StepKindFinalize,
	}
	for i, want := range wantKinds {
		if plan.Steps[i].Kind != want {
			t.Fatalf("step %d kind = %q, want %q", i, plan.Steps[i].Kind, want)
		}
	}
	if got := plan.Steps[0].AgentBinding; got != "planner" {
		t.Fatalf("expected planner step to bind planner, got %q", got)
	}
	if got := plan.Steps[1].AgentBinding; got != "designer" {
		t.Fatalf("expected first execution step to bind designer, got %q", got)
	}
	if got := plan.Steps[2].AgentBinding; got != "engineer" {
		t.Fatalf("expected second execution step to bind engineer, got %q", got)
	}
	if got := plan.Steps[1].InputRefs; len(got) != 1 || got[0] != "plan_outline" {
		t.Fatalf("expected first execution step to consume plan_outline, got %#v", got)
	}
	if got := plan.Steps[2].InputRefs; len(got) != 1 || got[0] != "step_1_output" {
		t.Fatalf("expected second execution step to consume step_1_output, got %#v", got)
	}
	if got := plan.Steps[3].InputRefs; len(got) != 1 || got[0] != "step_2_output" {
		t.Fatalf("expected finalize step to consume step_2_output, got %#v", got)
	}
}

func TestSupervisionBuilderUsesParallelAndReview(t *testing.T) {
	builder := planbuilder.NewSupervisionBuilder()
	scheme := &entity.CollaborationScheme{
		ID:   "scheme-supervision",
		Mode: entity.ModeSupervision,
		BindAgents: []*entity.AgentBinding{
			{AgentID: "supervisor", Role: "监督者"},
			{AgentID: "designer", Role: "设计师"},
			{AgentID: "engineer", Role: "工程师"},
		},
	}

	plan, err := builder.Build(scheme, "并发分析")
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	if got, want := len(plan.Steps), 4; got != want {
		t.Fatalf("expected %d steps, got %d", want, got)
	}
	wantKinds := []entity.StepKind{
		entity.StepKindReview,
		entity.StepKindParallel,
		entity.StepKindReview,
		entity.StepKindFinalize,
	}
	for i, want := range wantKinds {
		if plan.Steps[i].Kind != want {
			t.Fatalf("step %d kind = %q, want %q", i, plan.Steps[i].Kind, want)
		}
	}
	if got := plan.Steps[0].AgentBinding; got != "supervisor" {
		t.Fatalf("expected first review to bind supervisor, got %q", got)
	}
	if got := plan.Steps[1].InputRefs; len(got) != 1 || got[0] != "assignment" {
		t.Fatalf("expected parallel step to consume assignment, got %#v", got)
	}
	workers, ok := plan.Steps[1].Config["workers"].([]string)
	if !ok {
		t.Fatalf("expected workers config, got %#v", plan.Steps[1].Config["workers"])
	}
	if got, want := workers, []string{"designer", "engineer"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("unexpected workers config: %#v", got)
	}
}

func TestPeerHandoffBuilderIncludesHandoffChain(t *testing.T) {
	builder := planbuilder.NewPeerHandoffBuilder()
	scheme := &entity.CollaborationScheme{
		ID:   "scheme-peer-handoff",
		Mode: entity.ModePeerHandoff,
		BindAgents: []*entity.AgentBinding{
			{AgentID: "designer", Role: "设计师"},
			{AgentID: "pm", Role: "产品经理"},
			{AgentID: "engineer", Role: "工程师"},
		},
		Config: &entity.SchemeConfig{
			ModeConfig: &entity.ModeConfig{
				PeerHandoffConfig: &entity.PeerHandoffConfig{
					EntryAgent: "designer",
					MeshAgents: []string{"designer", "pm", "engineer"},
				},
			},
		},
	}

	plan, err := builder.Build(scheme, "开始接力")
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	wantKinds := []entity.StepKind{
		entity.StepKindAgent,
		entity.StepKindHandoff,
		entity.StepKindAgent,
		entity.StepKindHandoff,
		entity.StepKindAgent,
		entity.StepKindHandoff,
		entity.StepKindFinalize,
	}
	if got, want := len(plan.Steps), len(wantKinds); got != want {
		t.Fatalf("expected %d steps, got %d", want, got)
	}
	for i, want := range wantKinds {
		if plan.Steps[i].Kind != want {
			t.Fatalf("step %d kind = %q, want %q", i, plan.Steps[i].Kind, want)
		}
	}
	if got := plan.Steps[0].AgentBinding; got != "designer" {
		t.Fatalf("expected entry agent designer, got %q", got)
	}
	if got := plan.Steps[2].AgentBinding; got != "pm" {
		t.Fatalf("expected second agent pm, got %q", got)
	}
	if got := plan.Steps[4].AgentBinding; got != "engineer" {
		t.Fatalf("expected third agent engineer, got %q", got)
	}
	if got := plan.Steps[6].DependsOn; len(got) != 1 || got[0] != plan.Steps[5].StepID {
		t.Fatalf("expected finalize to depend on last handoff, got %#v", got)
	}
}
