package workflow

import (
	"context"
	"errors"
	"fmt"
	"log"
	"ruleGoKratos/internal/biz"
	"ruleGoKratos/internal/biz/entity"
	"ruleGoKratos/internal/biz/playground/agentpool"
	"ruleGoKratos/internal/biz/playground/collaboration"
	"ruleGoKratos/internal/biz/playground/planbuilder"
	playgroundruntime "ruleGoKratos/internal/biz/playground/runtime"
	"ruleGoKratos/internal/biz/playground/trace"
	"runtime/debug"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// WorkflowRepo 工作流仓储接口
type WorkflowRepo interface {
	SaveScheme(ctx context.Context, scheme *entity.CollaborationScheme) error
	UpdateScheme(ctx context.Context, scheme *entity.CollaborationScheme) error
	DeleteScheme(ctx context.Context, id string) error
	FindSchemeByID(ctx context.Context, id string) (*entity.CollaborationScheme, error)
	FindAllSchemes(ctx context.Context) ([]*entity.CollaborationScheme, error)
}

// WorkflowService 工作流服务
type WorkflowService struct {
	repo              WorkflowRepo
	agentPoolSvc      *agentpool.AgentPoolService
	traceEngine       *trace.TraceEngine
	agentUC           *biz.AgentUsecase
	collabFactory     *collaboration.Factory
	builderRegistry   *planbuilder.Registry
	runtimeEnabledSet map[entity.CollaborationMode]struct{}
	runtimeRepo       playgroundruntime.Repo
	runtimeSvc        runtimeExecutor
	schemes           map[string]*entity.CollaborationScheme
	schemesMu         sync.RWMutex
	activeRuns        map[string]*ActiveRun
	activeRunsMu      sync.RWMutex
}

type ActiveRun struct {
	ID       string
	SchemeID string
	Status   string
	StartAt  *time.Time
}

type runtimeExecutor interface {
	Execute(
		ctx context.Context,
		runID string,
		plan *entity.ExecutionPlan,
		scheme *entity.CollaborationScheme,
		pool *entity.AgentPool,
		userInput string,
		trace collaboration.TraceEmitter,
	) (string, error)
	ApplyRecoveryAction(
		ctx context.Context,
		runID string,
		actionID string,
		scheme *entity.CollaborationScheme,
		pool *entity.AgentPool,
		userInput string,
		trace collaboration.TraceEmitter,
		optTargetRef string,
	) (string, error)
}

func NewWorkflowService(
	repo WorkflowRepo,
	agentPoolSvc *agentpool.AgentPoolService,
	traceEngine *trace.TraceEngine,
	agentUC *biz.AgentUsecase,
) *WorkflowService {
	return NewWorkflowServiceWithRuntimeRepo(repo, agentPoolSvc, traceEngine, agentUC, playgroundruntime.NewMemoryRepo())
}

// NewWorkflowServiceWithRuntimeRepo 创建 WorkflowService，并显式暴露 runtime repo 注入点。
func NewWorkflowServiceWithRuntimeRepo(
	repo WorkflowRepo,
	agentPoolSvc *agentpool.AgentPoolService,
	traceEngine *trace.TraceEngine,
	agentUC *biz.AgentUsecase,
	runtimeRepo playgroundruntime.Repo,
) *WorkflowService {
	svc := &WorkflowService{
		repo:            repo,
		agentPoolSvc:    agentPoolSvc,
		traceEngine:     traceEngine,
		agentUC:         agentUC,
		collabFactory:   collaboration.NewFactory(),
		builderRegistry: planbuilder.NewRegistry(),
		runtimeRepo:     runtimeRepo,
		runtimeEnabledSet: map[entity.CollaborationMode]struct{}{
			entity.ModeRouterExpert: {},
			entity.ModePlanExec:     {},
			entity.ModeSupervision:  {},
			entity.ModePeerHandoff:  {},
		},
		schemes:    make(map[string]*entity.CollaborationScheme),
		activeRuns: make(map[string]*ActiveRun),
	}
	if svc.runtimeRepo != nil {
		svc.runtimeSvc = playgroundruntime.NewService(svc.runtimeRepo, &collaboration.CollaborationRuntime{AgentUC: agentUC})
	}

	// 注册协作处理器
	svc.collabFactory.Register(entity.ModeRouterExpert, collaboration.NewRouterExpertHandler())
	svc.collabFactory.Register(entity.ModePlanExec, collaboration.NewPlanExecHandler())
	svc.collabFactory.Register(entity.ModeSupervision, collaboration.NewSupervisionHandler())
	svc.collabFactory.Register(entity.ModePeerHandoff, collaboration.NewPeerHandoffHandler())
	svc.builderRegistry.Register(planbuilder.NewRouterExpertBuilder())
	svc.builderRegistry.Register(planbuilder.NewPlanExecBuilder())
	svc.builderRegistry.Register(planbuilder.NewSupervisionBuilder())
	svc.builderRegistry.Register(planbuilder.NewPeerHandoffBuilder())

	return svc
}

// CreateScheme 创建协作方案
func (s *WorkflowService) CreateScheme(ctx context.Context, name, desc string, mode entity.CollaborationMode, bindAgents []*entity.AgentBinding) (*entity.CollaborationScheme, error) {
	cfgCopy := *entity.DefaultSchemeConfig
	scheme := &entity.CollaborationScheme{
		ID:              uuid.NewString(),
		Name:            name,
		Description:     desc,
		Mode:            mode,
		BindAgents:      bindAgents,
		Config:          &cfgCopy,
		Enabled:         true,
		EnableFinalizer: false,
		CreatedAt:       nowPtr(),
		UpdatedAt:       nowPtr(),
	}

	EnsurePlanExecModeConfig(scheme)

	if err := s.repo.SaveScheme(ctx, scheme); err != nil {
		return nil, fmt.Errorf("save scheme: %w", err)
	}

	s.schemesMu.Lock()
	s.schemes[scheme.ID] = scheme
	s.schemesMu.Unlock()

	return scheme, nil
}

// UpdateScheme 更新协作方案
func (s *WorkflowService) UpdateScheme(ctx context.Context, scheme *entity.CollaborationScheme) error {
	scheme.UpdatedAt = nowPtr()
	EnsurePlanExecModeConfig(scheme)

	if err := s.repo.UpdateScheme(ctx, scheme); err != nil {
		return fmt.Errorf("update scheme: %w", err)
	}

	s.schemesMu.Lock()
	s.schemes[scheme.ID] = scheme
	s.schemesMu.Unlock()

	return nil
}

// DeleteScheme 删除协作方案
func (s *WorkflowService) DeleteScheme(ctx context.Context, id string) error {
	if err := s.repo.DeleteScheme(ctx, id); err != nil {
		return fmt.Errorf("delete scheme: %w", err)
	}

	s.schemesMu.Lock()
	delete(s.schemes, id)
	s.schemesMu.Unlock()

	return nil
}

// GetScheme 获取协作方案
func (s *WorkflowService) GetScheme(ctx context.Context, id string) (*entity.CollaborationScheme, error) {
	s.schemesMu.RLock()
	if scheme, ok := s.schemes[id]; ok {
		s.schemesMu.RUnlock()
		return scheme, nil
	}
	s.schemesMu.RUnlock()

	scheme, err := s.repo.FindSchemeByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find scheme: %w", err)
	}

	s.schemesMu.Lock()
	s.schemes[id] = scheme
	s.schemesMu.Unlock()

	return scheme, nil
}

// ListSchemes 列出所有协作方案
func (s *WorkflowService) ListSchemes(ctx context.Context) ([]*entity.CollaborationScheme, error) {
	s.schemesMu.RLock()
	if len(s.schemes) > 0 {
		result := make([]*entity.CollaborationScheme, 0, len(s.schemes))
		for _, p := range s.schemes {
			result = append(result, p)
		}
		s.schemesMu.RUnlock()
		sortSchemesByUpdatedDesc(result)
		return result, nil
	}
	s.schemesMu.RUnlock()

	schemes, err := s.repo.FindAllSchemes(ctx)
	if err != nil {
		return nil, fmt.Errorf("find all schemes: %w", err)
	}

	s.schemesMu.Lock()
	for _, p := range schemes {
		s.schemes[p.ID] = p
	}
	s.schemesMu.Unlock()

	sortSchemesByUpdatedDesc(schemes)
	return schemes, nil
}

// Run 执行工作流
func (s *WorkflowService) Run(ctx context.Context, schemeID, userInput string) (string, error) {
	scheme, err := s.GetScheme(ctx, schemeID)
	if err != nil {
		return "", fmt.Errorf("get scheme: %w", err)
	}

	// 默认 Agent 池：首次使用 PostgreSQL 时可能尚无 `default` 行，需幂等创建（与测试里显式 CreateDefaultAgentPool 对齐）
	pool, err := s.agentPoolSvc.CreateDefaultAgentPool(ctx)
	if err != nil {
		return "", fmt.Errorf("default agent pool: %w", err)
	}

	EnsurePlanExecModeConfig(scheme)
	if err := s.validateAgentsForHarness(scheme, pool); err != nil {
		return "", err
	}
	if err := s.ensureRuntimeConfigured(); err != nil {
		return "", err
	}

	builder, ok := s.builderRegistry.Get(scheme.Mode)
	if !ok {
		return "", fmt.Errorf("runtime builder not registered for mode: %s", scheme.Mode)
	}
	plan, err := builder.Build(scheme, userInput)
	if err != nil {
		return "", fmt.Errorf("build plan: %w", err)
	}

	// 创建 Trace
	run, err := s.traceEngine.StartRun(ctx, schemeID, userInput)
	if err != nil {
		return "", fmt.Errorf("start run: %w", err)
	}

	runID := run.RunID

	// 记录为活跃运行
	s.activeRunsMu.Lock()
	s.activeRuns[runID] = &ActiveRun{
		ID:       runID,
		SchemeID: schemeID,
		Status:   "running",
		StartAt:  nowPtr(),
	}
	s.activeRunsMu.Unlock()

	// 异步执行：HTTP 立即返回 runId，前端可轮询 Trace 实时展示规划执行等模式的任务流。
	execCtx := context.WithoutCancel(ctx)
	traceSink := newTraceAdapter(execCtx, s.traceEngine)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("playground workflow panic runID=%s: %v\n%s", runID, r, debug.Stack())
				_ = s.traceEngine.EndRun(execCtx, runID, "", "failed")
				s.activeRunsMu.Lock()
				if ar, ok := s.activeRuns[runID]; ok {
					ar.Status = "failed"
				}
				s.activeRunsMu.Unlock()
			}
		}()

		finalOutput, err := s.executeRun(execCtx, runID, plan, scheme, pool, userInput, traceSink)
		if err != nil {
			status, finalOutput := runtimeErrorOutcome(err)
			_ = s.traceEngine.EndRun(execCtx, runID, finalOutput, status)
			s.activeRunsMu.Lock()
			if r, ok := s.activeRuns[runID]; ok {
				r.Status = status
			}
			s.activeRunsMu.Unlock()
			return
		}

		_ = s.traceEngine.EndRun(execCtx, runID, finalOutput, "completed")

		s.activeRunsMu.Lock()
		if r, ok := s.activeRuns[runID]; ok {
			r.Status = "completed"
		}
		s.activeRunsMu.Unlock()
	}()

	return runID, nil
}

func (s *WorkflowService) executeRun(
	ctx context.Context,
	runID string,
	plan *entity.ExecutionPlan,
	scheme *entity.CollaborationScheme,
	pool *entity.AgentPool,
	userInput string,
	traceSink collaboration.TraceEmitter,
) (string, error) {
	if plan == nil {
		return "", fmt.Errorf("runtime plan is nil for mode: %s", scheme.Mode)
	}
	return s.executePlanWithRuntime(ctx, runID, plan, scheme, pool, userInput, traceSink)
}

func (s *WorkflowService) executePlanWithRuntime(
	ctx context.Context,
	runID string,
	plan *entity.ExecutionPlan,
	scheme *entity.CollaborationScheme,
	pool *entity.AgentPool,
	userInput string,
	traceSink collaboration.TraceEmitter,
) (string, error) {
	if err := s.ensureRuntimeConfigured(); err != nil {
		return "", err
	}
	return s.runtimeSvc.Execute(ctx, runID, plan, scheme, pool, userInput, traceSink)
}

func (s *WorkflowService) ensureRuntimeConfigured() error {
	if s.runtimeRepo == nil {
		return fmt.Errorf("runtime repo is nil")
	}
	if s.runtimeSvc == nil {
		return fmt.Errorf("runtime service is nil")
	}
	return nil
}

func (s *WorkflowService) isRuntimeEnabledMode(mode entity.CollaborationMode) bool {
	if s == nil {
		return false
	}
	_, ok := s.runtimeEnabledSet[mode]
	return ok
}

// SetRuntimeServiceForTest 允许测试注入自定义 runtime service。
func (s *WorkflowService) SetRuntimeServiceForTest(rt *playgroundruntime.Service) {
	if s == nil {
		return
	}
	s.runtimeSvc = rt
}

func runtimeErrorOutcome(err error) (status string, finalOutput string) {
	var runErr *playgroundruntime.RunError
	if errors.As(err, &runErr) {
		return string(runErr.Status()), runErr.FailureSummary()
	}
	return "failed", ""
}

// GetRun 获取运行记录
func (s *WorkflowService) GetRun(ctx context.Context, runID string) (*entity.TraceRun, error) {
	return s.traceEngine.GetRun(ctx, runID)
}

// GetRuntimeRun 获取 runtime 运行详情。
func (s *WorkflowService) GetRuntimeRun(ctx context.Context, runID string) (*entity.PlaygroundRun, error) {
	if s.runtimeRepo == nil {
		return nil, fmt.Errorf("runtime repo is nil")
	}
	run, err := s.runtimeRepo.GetRun(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("get runtime run: %w", err)
	}
	return run, nil
}

// ListRuntimeSteps 列出某次运行的全部 runtime steps。
func (s *WorkflowService) ListRuntimeSteps(ctx context.Context, runID string) ([]*entity.RuntimeStep, error) {
	if s.runtimeRepo == nil {
		return nil, fmt.Errorf("runtime repo is nil")
	}
	return s.runtimeRepo.ListSteps(ctx, runID)
}

// ListRuntimeArtifacts 列出某次运行的全部 runtime artifacts。
func (s *WorkflowService) ListRuntimeArtifacts(ctx context.Context, runID string) ([]*entity.RuntimeArtifact, error) {
	if s.runtimeRepo == nil {
		return nil, fmt.Errorf("runtime repo is nil")
	}
	return s.runtimeRepo.ListArtifacts(ctx, runID)
}

// ListRecoveryActions 列出某次运行当前可见的恢复动作。
func (s *WorkflowService) ListRecoveryActions(ctx context.Context, runID string) ([]*entity.RecoveryAction, error) {
	if s.runtimeRepo == nil {
		return nil, fmt.Errorf("runtime repo is nil")
	}
	actions, err := s.runtimeRepo.ListRecoveryActions(ctx, runID)
	if err != nil {
		return nil, err
	}
	filtered := make([]*entity.RecoveryAction, 0, len(actions))
	for _, action := range actions {
		if action == nil {
			continue
		}
		if state, _ := action.Metadata["state"].(string); state != "" && state != "pending" {
			continue
		}
		filtered = append(filtered, action)
	}
	return filtered, nil
}

// ApplyRecoveryAction 异步触发指定 run 的恢复动作。
func (s *WorkflowService) ApplyRecoveryAction(ctx context.Context, runID, actionID, optTargetRef string) error {
	if err := s.ensureRuntimeConfigured(); err != nil {
		return err
	}
	traceRun, err := s.GetRun(ctx, runID)
	if err != nil {
		return fmt.Errorf("get trace run: %w", err)
	}
	runtimeRun, err := s.GetRuntimeRun(ctx, runID)
	if err != nil {
		return fmt.Errorf("get runtime run: %w", err)
	}
	scheme, err := s.GetScheme(ctx, runtimeRun.SchemeID)
	if err != nil {
		return fmt.Errorf("get scheme: %w", err)
	}
	pool, err := s.agentPoolSvc.CreateDefaultAgentPool(ctx)
	if err != nil {
		return fmt.Errorf("default agent pool: %w", err)
	}
	if err := s.validateAgentsForHarness(scheme, pool); err != nil {
		return err
	}

	s.activeRunsMu.Lock()
	s.activeRuns[runID] = &ActiveRun{
		ID:       runID,
		SchemeID: scheme.ID,
		Status:   "running",
		StartAt:  nowPtr(),
	}
	s.activeRunsMu.Unlock()

	execCtx := context.WithoutCancel(ctx)
	traceSink := newTraceAdapter(execCtx, s.traceEngine)
	go func() {
		finalOutput, err := s.runtimeSvc.ApplyRecoveryAction(execCtx, runID, actionID, scheme, pool, traceRun.UserInput, traceSink, optTargetRef)
		if err != nil {
			status, finalOutput := runtimeErrorOutcome(err)
			_ = s.traceEngine.EndRun(execCtx, runID, finalOutput, status)
			s.activeRunsMu.Lock()
			if r, ok := s.activeRuns[runID]; ok {
				r.Status = status
			}
			s.activeRunsMu.Unlock()
			return
		}
		_ = s.traceEngine.EndRun(execCtx, runID, finalOutput, "completed")
		s.activeRunsMu.Lock()
		if r, ok := s.activeRuns[runID]; ok {
			r.Status = "completed"
		}
		s.activeRunsMu.Unlock()
	}()
	return nil
}

// GetRunEvents 获取运行事件
func (s *WorkflowService) GetRunEvents(ctx context.Context, runID string) ([]*entity.TraceEvent, error) {
	filter := &entity.TraceFilter{
		RunID: runID,
		Limit: 1000,
	}
	return s.traceEngine.GetEvents(ctx, filter)
}

// SubscribeRunEvents 订阅实时 Trace（供 SSE）；与 GetRunEvents 叠加可做到先回放再接流。
func (s *WorkflowService) SubscribeRunEvents(runID string) (<-chan *entity.TraceEvent, func()) {
	return s.traceEngine.SubscribeRun(runID)
}

// traceAdapter Trace 追踪适配器，实现 TraceEmitter 接口
type traceAdapter struct {
	ctx    context.Context
	engine *trace.TraceEngine
}

func newTraceAdapter(ctx context.Context, engine *trace.TraceEngine) *traceAdapter {
	if ctx == nil {
		ctx = context.Background()
	}
	return &traceAdapter{ctx: ctx, engine: engine}
}

func (a *traceAdapter) TaskAssigned(runID, agentID, taskDesc string) {
	a.engine.TaskAssigned(a.ctx, runID, agentID, taskDesc)
}

func (a *traceAdapter) AgentEnterWorker(runID, agentID, nodeID string) {
	a.engine.AgentEnterWorker(a.ctx, runID, agentID, nodeID)
}

func (a *traceAdapter) AgentExitWorker(runID, agentID, nodeID, message string) {
	a.engine.AgentExitWorker(a.ctx, runID, agentID, nodeID, message)
}

func (a *traceAdapter) WorkerDelegated(runID, agentID, workerAgentID, reason string) {
	a.engine.WorkerDelegated(a.ctx, runID, agentID, workerAgentID, reason)
}

func (a *traceAdapter) Thinking(runID, agentID, message string) {
	a.engine.Thinking(a.ctx, runID, agentID, message)
}

func (a *traceAdapter) ToolCall(runID, agentID, toolName, args string) {
	a.engine.ToolCall(a.ctx, runID, agentID, toolName, args)
}

func (a *traceAdapter) ToolResult(runID, agentID, toolName, result string, success bool) {
	a.engine.ToolResult(a.ctx, runID, agentID, toolName, result, success)
}

func (a *traceAdapter) Handoff(runID, fromAgent, toAgent, reason string) {
	a.engine.Handoff(a.ctx, runID, fromAgent, toAgent, reason)
}

func (a *traceAdapter) Error(runID, agentID, message string) {
	a.engine.Error(a.ctx, runID, agentID, message)
}

func (a *traceAdapter) EmitEvent(ctx context.Context, event *entity.TraceEvent) {
	if ctx == nil {
		ctx = a.ctx
	}
	a.engine.EmitEvent(ctx, event)
}

// sortSchemesByUpdatedDesc 稳定展示顺序：最近更新的在前
func sortSchemesByUpdatedDesc(list []*entity.CollaborationScheme) {
	sort.SliceStable(list, func(i, j int) bool {
		ui, uj := list[i].UpdatedAt, list[j].UpdatedAt
		if ui == nil && uj == nil {
			return list[i].Name < list[j].Name
		}
		if ui == nil {
			return false
		}
		if uj == nil {
			return true
		}
		if ui.Equal(*uj) {
			return list[i].Name < list[j].Name
		}
		return ui.After(*uj)
	})
}

// nowPtr 返回当前时间指针
func nowPtr() *time.Time {
	now := time.Now()
	return &now
}
