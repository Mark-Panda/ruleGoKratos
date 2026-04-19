package workflow

import (
	"context"
	"ruleGoKratos/internal/biz/entity"
	"ruleGoKratos/internal/biz/playground/agentpool"
	"ruleGoKratos/internal/biz/playground/collaboration"
	playgroundruntime "ruleGoKratos/internal/biz/playground/runtime"
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

func TestNewWorkflowServiceInitializesRuntimeWithUsableRepo(t *testing.T) {
	agentPoolRepo := playgrounddata.NewAgentPoolRepo()
	schemeRepo := playgrounddata.NewSchemeRepo()
	traceRepo := playgrounddata.NewTraceRepo()

	agentPoolSvc := agentpool.NewAgentPoolService(agentPoolRepo)
	traceEngine := trace.NewTraceEngine(traceRepo)
	svc := NewWorkflowService(schemeRepo, agentPoolSvc, traceEngine, nil)

	rtSvc, ok := svc.runtimeSvc.(*playgroundruntime.Service)
	if !ok {
		t.Fatalf("expected runtime service type, got %T", svc.runtimeSvc)
	}
	if rtSvc.Repo() == nil {
		t.Fatal("expected workflow runtime service to have a repo")
	}
}

func TestNewWorkflowServiceWithRuntimeRepoUsesInjectedRepo(t *testing.T) {
	agentPoolRepo := playgrounddata.NewAgentPoolRepo()
	schemeRepo := playgrounddata.NewSchemeRepo()
	traceRepo := playgrounddata.NewTraceRepo()
	customRuntimeRepo := playgroundruntime.NewMemoryRepo()

	agentPoolSvc := agentpool.NewAgentPoolService(agentPoolRepo)
	traceEngine := trace.NewTraceEngine(traceRepo)
	svc := NewWorkflowServiceWithRuntimeRepo(schemeRepo, agentPoolSvc, traceEngine, nil, customRuntimeRepo)

	rtSvc, ok := svc.runtimeSvc.(*playgroundruntime.Service)
	if !ok {
		t.Fatalf("expected runtime service type, got %T", svc.runtimeSvc)
	}
	if svc.runtimeRepo != customRuntimeRepo {
		t.Fatal("expected workflow service to retain injected runtime repo")
	}
	if rtSvc.Repo() != customRuntimeRepo {
		t.Fatal("expected runtime service to use injected runtime repo")
	}
}

func TestNewWorkflowServiceWithRuntimeRepoDoesNotInjectDefaultWhenNil(t *testing.T) {
	agentPoolRepo := playgrounddata.NewAgentPoolRepo()
	schemeRepo := playgrounddata.NewSchemeRepo()
	traceRepo := playgrounddata.NewTraceRepo()

	agentPoolSvc := agentpool.NewAgentPoolService(agentPoolRepo)
	traceEngine := trace.NewTraceEngine(traceRepo)
	svc := NewWorkflowServiceWithRuntimeRepo(schemeRepo, agentPoolSvc, traceEngine, nil, nil)

	if svc.runtimeRepo != nil {
		t.Fatalf("expected workflow service runtime repo to stay nil, got %T", svc.runtimeRepo)
	}
	if svc.runtimeSvc != nil {
		t.Fatalf("expected runtime service to stay nil when runtime repo is nil, got %T", svc.runtimeSvc)
	}
}

func TestWorkflowService_RunFailsFastWhenRuntimeNotConfigured(t *testing.T) {
	agentPoolRepo := playgrounddata.NewAgentPoolRepo()
	schemeRepo := playgrounddata.NewSchemeRepo()
	traceRepo := playgrounddata.NewTraceRepo()

	agentPoolSvc := agentpool.NewAgentPoolService(agentPoolRepo)
	traceEngine := trace.NewTraceEngine(traceRepo)
	svc := NewWorkflowServiceWithRuntimeRepo(schemeRepo, agentPoolSvc, traceEngine, nil, nil)

	ctx := context.Background()
	if _, err := agentPoolSvc.CreateDefaultAgentPool(ctx); err != nil {
		t.Fatalf("CreateDefaultAgentPool failed: %v", err)
	}
	scheme, err := svc.CreateScheme(ctx, "未配置 runtime", "测试", entity.ModeRouterExpert, []*entity.AgentBinding{
		{AgentID: "engineer", Role: "工程师"},
	})
	if err != nil {
		t.Fatalf("CreateScheme failed: %v", err)
	}

	runID, err := svc.Run(ctx, scheme.ID, "请实现接口")
	if err == nil {
		t.Fatalf("expected run to fail fast without runtime, got runID=%q", runID)
	}
	if runID != "" {
		t.Fatalf("expected empty runID, got %q", runID)
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

func TestExecuteRunUsesRuntimeForRouterExpert(t *testing.T) {
	rt := &fakeRuntimeExecutor{output: "runtime-output"}
	handler := &fakeHandler{output: "legacy-output"}
	factory := collaboration.NewFactory()
	factory.Register(entity.ModeRouterExpert, handler)

	svc := &WorkflowService{
		collabFactory:     factory,
		runtimeRepo:       playgroundruntime.NewMemoryRepo(),
		runtimeSvc:        rt,
		runtimeEnabledSet: map[entity.CollaborationMode]struct{}{entity.ModeRouterExpert: {}},
	}

	output, err := svc.executeRun(
		context.Background(),
		"run-1",
		&entity.ExecutionPlan{PlanID: "plan-1"},
		&entity.CollaborationScheme{Mode: entity.ModeRouterExpert},
		nil,
		"设计登录页",
		nil,
	)
	if err != nil {
		t.Fatalf("executeRun failed: %v", err)
	}
	if output != "runtime-output" {
		t.Fatalf("expected runtime output, got %q", output)
	}
	if !rt.called {
		t.Fatal("expected runtime executor to be called")
	}
	if handler.executeCalled {
		t.Fatal("expected legacy handler not to run for router_expert")
	}
}

func TestExecuteRunUsesRuntimeForCurrentModes(t *testing.T) {
	testCases := []struct {
		name string
		mode entity.CollaborationMode
	}{
		{name: "plan_exec", mode: entity.ModePlanExec},
		{name: "supervision", mode: entity.ModeSupervision},
		{name: "peer_handoff", mode: entity.ModePeerHandoff},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rt := &fakeRuntimeExecutor{output: "runtime-output"}
			handler := &fakeHandler{output: "legacy-output"}
			factory := collaboration.NewFactory()
			factory.Register(tc.mode, handler)

			svc := &WorkflowService{
				collabFactory: factory,
				runtimeRepo:   playgroundruntime.NewMemoryRepo(),
				runtimeSvc:    rt,
			}

			output, err := svc.executeRun(
				context.Background(),
				"run-"+tc.name,
				&entity.ExecutionPlan{PlanID: "plan-" + tc.name},
				&entity.CollaborationScheme{Mode: tc.mode},
				nil,
				"执行任务",
				nil,
			)
			if err != nil {
				t.Fatalf("executeRun failed: %v", err)
			}
			if output != "runtime-output" {
				t.Fatalf("expected runtime output, got %q", output)
			}
			if !rt.called {
				t.Fatal("expected runtime executor to be called")
			}
			if handler.initCalled || handler.executeCalled {
				t.Fatal("expected legacy handler not to initialize or execute")
			}
		})
	}
}

func TestExecuteRunRejectsMissingRuntimePlanForCurrentModes(t *testing.T) {
	testCases := []struct {
		name string
		mode entity.CollaborationMode
	}{
		{name: "plan_exec", mode: entity.ModePlanExec},
		{name: "supervision", mode: entity.ModeSupervision},
		{name: "peer_handoff", mode: entity.ModePeerHandoff},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rt := &fakeRuntimeExecutor{output: "runtime-output"}
			handler := &fakeHandler{output: "legacy-output"}
			factory := collaboration.NewFactory()
			factory.Register(tc.mode, handler)

			svc := &WorkflowService{
				collabFactory: factory,
				runtimeSvc:    rt,
			}

			output, err := svc.executeRun(
				context.Background(),
				"run-"+tc.name,
				nil,
				&entity.CollaborationScheme{Mode: tc.mode},
				nil,
				"规划任务",
				nil,
			)
			if err == nil {
				t.Fatal("expected missing runtime plan error")
			}
			if output != "" {
				t.Fatalf("expected empty output, got %q", output)
			}
			if rt.called {
				t.Fatal("expected runtime executor not to run without plan")
			}
			if handler.initCalled || handler.executeCalled {
				t.Fatal("expected legacy handler not to initialize or execute")
			}
		})
	}
}

func TestWorkflowRunPreservesWaitingRecoveryFromRuntime(t *testing.T) {
	agentPoolRepo := playgrounddata.NewAgentPoolRepo()
	schemeRepo := playgrounddata.NewSchemeRepo()
	traceRepo := playgrounddata.NewTraceRepo()

	agentPoolSvc := agentpool.NewAgentPoolService(agentPoolRepo)
	traceEngine := trace.NewTraceEngine(traceRepo)
	svc := NewWorkflowService(schemeRepo, agentPoolSvc, traceEngine, nil)
	svc.runtimeSvc = &fakeRuntimeExecutor{
		err: playgroundruntime.NewRunError(entity.RunStatusWaitingRecovery, "agent step failed"),
	}

	ctx := context.Background()
	if _, err := agentPoolSvc.CreateDefaultAgentPool(ctx); err != nil {
		t.Fatalf("CreateDefaultAgentPool failed: %v", err)
	}
	scheme, err := svc.CreateScheme(ctx, "router", "test", entity.ModeRouterExpert, []*entity.AgentBinding{
		{AgentID: "engineer", Role: "工程师"},
	})
	if err != nil {
		t.Fatalf("CreateScheme failed: %v", err)
	}

	runID, err := svc.Run(ctx, scheme.ID, "请实现接口")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	run := waitRunTerminal(ctx, t, svc, runID)
	if run.Status != string(entity.RunStatusWaitingRecovery) {
		t.Fatalf("expected trace run status waiting_recovery, got %q", run.Status)
	}
	if got := svc.activeRuns[runID].Status; got != string(entity.RunStatusWaitingRecovery) {
		t.Fatalf("expected active run status waiting_recovery, got %q", got)
	}
}

func TestExecutePlanUsesRuntimeService(t *testing.T) {
	rt := &fakeRuntimeExecutor{output: "runtime-output"}
	svc := &WorkflowService{
		runtimeRepo: playgroundruntime.NewMemoryRepo(),
		runtimeSvc:  rt,
	}

	output, err := svc.executePlanWithRuntime(
		context.Background(),
		"run-1",
		&entity.ExecutionPlan{PlanID: "plan-1"},
		&entity.CollaborationScheme{ID: "scheme-router", Mode: entity.ModeRouterExpert},
		nil,
		"设计一个表单",
		nil,
	)
	if err != nil {
		t.Fatalf("executePlanWithRuntime failed: %v", err)
	}
	if output != "runtime-output" {
		t.Fatalf("expected runtime output, got %q", output)
	}
	if !rt.called {
		t.Fatal("expected runtime path instead of legacy handler execute")
	}
}

type fakeRuntimeExecutor struct {
	called bool
	output string
	err    error
}

func (f *fakeRuntimeExecutor) Execute(
	_ context.Context,
	_ string,
	_ *entity.ExecutionPlan,
	_ *entity.CollaborationScheme,
	_ *entity.AgentPool,
	_ string,
	_ collaboration.TraceEmitter,
) (string, error) {
	f.called = true
	return f.output, f.err
}

func (f *fakeRuntimeExecutor) ApplyRecoveryAction(
	_ context.Context,
	_ string,
	_ string,
	_ *entity.CollaborationScheme,
	_ *entity.AgentPool,
	_ string,
	_ collaboration.TraceEmitter,
	_ string,
) (string, error) {
	f.called = true
	return f.output, f.err
}

type fakeHandler struct {
	initCalled    bool
	executeCalled bool
	output        string
	err           error
}

func (f *fakeHandler) Init(context.Context, *entity.CollaborationScheme, *entity.AgentPool, *collaboration.CollaborationRuntime) error {
	f.initCalled = true
	return nil
}

func (f *fakeHandler) Execute(context.Context, string, string, collaboration.TraceEmitter) (*entity.AgentInstance, error) {
	f.executeCalled = true
	if f.err != nil {
		return nil, f.err
	}
	return &entity.AgentInstance{Output: f.output}, nil
}

func (f *fakeHandler) Name() string {
	return "fake"
}
