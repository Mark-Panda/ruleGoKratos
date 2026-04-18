package workflow

import (
	"context"
	"ruleGoKratos/internal/biz/playground/agentpool"
	"ruleGoKratos/internal/biz/entity"
	"ruleGoKratos/internal/biz/playground/trace"
	playgrounddata "ruleGoKratos/internal/data/playground"
	"testing"
	"time"
)

func newTestAgentPoolRepo() *playgrounddata.AgentPoolRepo {
	return playgrounddata.NewAgentPoolRepo()
}

func waitRunTerminal(ctx context.Context, t *testing.T, svc *WorkflowService, runID string) *entity.TraceRun {
	t.Helper()
	deadline := time.After(8 * time.Second)
	for {
		run, err := svc.GetRun(ctx, runID)
		if err != nil {
			t.Fatalf("GetRun: %v", err)
		}
		if run.Status != "running" {
			return run
		}
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for run %s to finish", runID)
		case <-time.After(8 * time.Millisecond):
		}
	}
}

func TestWorkflowService_CreateAndRunScheme(t *testing.T) {
	// 初始化各组件
	agentPoolRepo := playgrounddata.NewAgentPoolRepo()
	schemeRepo := playgrounddata.NewSchemeRepo()
	traceRepo := playgrounddata.NewTraceRepo()

	agentPoolSvc := agentpool.NewAgentPoolService(agentPoolRepo)
	traceEngine := trace.NewTraceEngine(traceRepo)

	svc := NewWorkflowService(schemeRepo, agentPoolSvc, traceEngine, nil)

	ctx := context.Background()

	// 创建默认 Agent 池
	pool, err := agentPoolSvc.CreateDefaultAgentPool(ctx)
	if err != nil {
		t.Fatalf("CreateDefaultAgentPool failed: %v", err)
	}

	if pool.ID != "default" {
		t.Errorf("expected pool ID 'default', got '%s'", pool.ID)
	}

	// 创建协作方案
	scheme, err := svc.CreateScheme(ctx, "测试方案", "用于测试", entity.ModeRouterExpert, []*entity.AgentBinding{
		{AgentID: "designer", Role: "设计师"},
		{AgentID: "pm", Role: "产品经理"},
		{AgentID: "engineer", Role: "工程师"},
	})
	if err != nil {
		t.Fatalf("CreateScheme failed: %v", err)
	}

	if scheme.Name != "测试方案" {
		t.Errorf("expected scheme name '测试方案', got '%s'", scheme.Name)
	}

	// 运行工作流
	runID, err := svc.Run(ctx, scheme.ID, "设计一个好看的界面")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if runID == "" {
		t.Error("Run returned empty runID")
	}

	run := waitRunTerminal(ctx, t, svc, runID)
	if run.Status != "completed" {
		t.Errorf("expected status 'completed', got '%s'", run.Status)
	}
}

func TestWorkflowService_AllCollaborationModes(t *testing.T) {
	agentPoolRepo := playgrounddata.NewAgentPoolRepo()
	schemeRepo := playgrounddata.NewSchemeRepo()
	traceRepo := playgrounddata.NewTraceRepo()

	agentPoolSvc := agentpool.NewAgentPoolService(agentPoolRepo)
	traceEngine := trace.NewTraceEngine(traceRepo)

	svc := NewWorkflowService(schemeRepo, agentPoolSvc, traceEngine, nil)

	ctx := context.Background()

	// 创建默认 Agent 池
	agentPoolSvc.CreateDefaultAgentPool(ctx)

	bindAgents := []*entity.AgentBinding{
		{AgentID: "designer", Role: "设计师"},
		{AgentID: "pm", Role: "产品经理"},
		{AgentID: "engineer", Role: "工程师"},
	}

	modes := []entity.CollaborationMode{
		entity.ModeRouterExpert,
		entity.ModePlanExec,
		entity.ModeSupervision,
		entity.ModePeerHandoff,
	}

	for _, mode := range modes {
		t.Run(string(mode), func(t *testing.T) {
			scheme, err := svc.CreateScheme(ctx, string(mode)+"测试", "测试", mode, bindAgents)
			if err != nil {
				t.Fatalf("CreateScheme failed for mode %s: %v", mode, err)
			}

			runID, err := svc.Run(ctx, scheme.ID, "开发一个计算器")
			if err != nil {
				t.Fatalf("Run failed for mode %s: %v", mode, err)
			}

			run := waitRunTerminal(ctx, t, svc, runID)
			if run.Status != "completed" {
				t.Errorf("expected status 'completed', got '%s' for mode %s", run.Status, mode)
			}
		})
	}
}

func TestWorkflowService_GetScheme(t *testing.T) {
	agentPoolRepo := playgrounddata.NewAgentPoolRepo()
	schemeRepo := playgrounddata.NewSchemeRepo()
	traceRepo := playgrounddata.NewTraceRepo()

	agentPoolSvc := agentpool.NewAgentPoolService(agentPoolRepo)
	traceEngine := trace.NewTraceEngine(traceRepo)

	svc := NewWorkflowService(schemeRepo, agentPoolSvc, traceEngine, nil)

	ctx := context.Background()
	agentPoolSvc.CreateDefaultAgentPool(ctx)

	// 创建方案
	scheme, err := svc.CreateScheme(ctx, "测试获取", "测试描述", entity.ModeRouterExpert, nil)
	if err != nil {
		t.Fatalf("CreateScheme failed: %v", err)
	}

	// 获取方案
	found, err := svc.GetScheme(ctx, scheme.ID)
	if err != nil {
		t.Fatalf("GetScheme failed: %v", err)
	}

	if found.ID != scheme.ID {
		t.Errorf("expected ID '%s', got '%s'", scheme.ID, found.ID)
	}

	if found.Name != "测试获取" {
		t.Errorf("expected name '测试获取', got '%s'", found.Name)
	}
}

func TestWorkflowService_ListSchemes(t *testing.T) {
	agentPoolRepo := playgrounddata.NewAgentPoolRepo()
	schemeRepo := playgrounddata.NewSchemeRepo()
	traceRepo := playgrounddata.NewTraceRepo()

	agentPoolSvc := agentpool.NewAgentPoolService(agentPoolRepo)
	traceEngine := trace.NewTraceEngine(traceRepo)

	svc := NewWorkflowService(schemeRepo, agentPoolSvc, traceEngine, nil)

	ctx := context.Background()
	agentPoolSvc.CreateDefaultAgentPool(ctx)

	// 创建多个方案
	schemes := make([]*entity.CollaborationScheme, 0, 3)
	for i := 0; i < 3; i++ {
		s, err := svc.CreateScheme(ctx, "测试方案", "测试", entity.ModeRouterExpert, nil)
		if err != nil {
			t.Fatalf("CreateScheme failed: %v", err)
		}
		schemes = append(schemes, s)
	}

	// 列出方案
	list, err := svc.ListSchemes(ctx)
	if err != nil {
		t.Fatalf("ListSchemes failed: %v", err)
	}

	if len(list) < 3 {
		t.Errorf("expected at least 3 schemes, got %d", len(list))
	}
}

func TestWorkflowService_UpdateAndDeleteScheme(t *testing.T) {
	agentPoolRepo := playgrounddata.NewAgentPoolRepo()
	schemeRepo := playgrounddata.NewSchemeRepo()
	traceRepo := playgrounddata.NewTraceRepo()

	agentPoolSvc := agentpool.NewAgentPoolService(agentPoolRepo)
	traceEngine := trace.NewTraceEngine(traceRepo)

	svc := NewWorkflowService(schemeRepo, agentPoolSvc, traceEngine, nil)

	ctx := context.Background()
	agentPoolSvc.CreateDefaultAgentPool(ctx)

	// 创建方案
	scheme, _ := svc.CreateScheme(ctx, "待更新", "原始描述", entity.ModeRouterExpert, nil)

	// 更新方案
	scheme.Name = "已更新"
	scheme.Description = "新描述"
	err := svc.UpdateScheme(ctx, scheme)
	if err != nil {
		t.Fatalf("UpdateScheme failed: %v", err)
	}

	// 验证更新
	updated, _ := svc.GetScheme(ctx, scheme.ID)
	if updated.Name != "已更新" {
		t.Errorf("expected name '已更新', got '%s'", updated.Name)
	}

	// 删除方案
	err = svc.DeleteScheme(ctx, scheme.ID)
	if err != nil {
		t.Fatalf("DeleteScheme failed: %v", err)
	}

	// 验证删除
	_, err = svc.GetScheme(ctx, scheme.ID)
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}

func TestWorkflowService_GetRunEvents(t *testing.T) {
	agentPoolRepo := playgrounddata.NewAgentPoolRepo()
	schemeRepo := playgrounddata.NewSchemeRepo()
	traceRepo := playgrounddata.NewTraceRepo()

	agentPoolSvc := agentpool.NewAgentPoolService(agentPoolRepo)
	traceEngine := trace.NewTraceEngine(traceRepo)

	svc := NewWorkflowService(schemeRepo, agentPoolSvc, traceEngine, nil)

	ctx := context.Background()
	agentPoolSvc.CreateDefaultAgentPool(ctx)

	// 创建并运行方案
	scheme, _ := svc.CreateScheme(ctx, "测试事件", "测试", entity.ModeRouterExpert, []*entity.AgentBinding{
		{AgentID: "designer", Role: "设计师"},
	})

	runID, _ := svc.Run(ctx, scheme.ID, "设计界面")
	waitRunTerminal(ctx, t, svc, runID)

	// 获取事件
	events, err := svc.GetRunEvents(ctx, runID)
	if err != nil {
		t.Fatalf("GetRunEvents failed: %v", err)
	}

	if len(events) == 0 {
		t.Error("expected events, got none")
	}

	// 验证事件包含必要的类型
	eventTypes := make(map[entity.TraceEventType]bool)
	for _, e := range events {
		eventTypes[e.Type] = true
	}

	if !eventTypes[entity.TraceEventWorkflowStart] {
		t.Error("missing WorkflowStart event")
	}
}

func TestSchemeRepo_InMemory(t *testing.T) {
	repo := playgrounddata.NewSchemeRepo()
	ctx := context.Background()

	scheme := &entity.CollaborationScheme{
		ID:   "test-scheme",
		Name: "测试",
		Mode: entity.ModeRouterExpert,
		BindAgents: []*entity.AgentBinding{
			{AgentID: "agent-1", Role: "角色1"},
		},
	}

	// 保存
	err := repo.SaveScheme(ctx, scheme)
	if err != nil {
		t.Fatalf("SaveScheme failed: %v", err)
	}

	// 查询
	found, err := repo.FindSchemeByID(ctx, "test-scheme")
	if err != nil {
		t.Fatalf("FindSchemeByID failed: %v", err)
	}

	if found.ID != "test-scheme" {
		t.Errorf("expected ID 'test-scheme', got '%s'", found.ID)
	}

	// 更新
	found.Name = "已更新"
	err = repo.UpdateScheme(ctx, found)
	if err != nil {
		t.Fatalf("UpdateScheme failed: %v", err)
	}

	updated, _ := repo.FindSchemeByID(ctx, "test-scheme")
	if updated.Name != "已更新" {
		t.Errorf("expected name '已更新', got '%s'", updated.Name)
	}

	// 删除
	err = repo.DeleteScheme(ctx, "test-scheme")
	if err != nil {
		t.Fatalf("DeleteScheme failed: %v", err)
	}

	_, err = repo.FindSchemeByID(ctx, "test-scheme")
	if err == nil {
		t.Error("expected error after delete")
	}
}
