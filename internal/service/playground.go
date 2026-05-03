package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ruleGoKratos/internal/biz/entity"
	"ruleGoKratos/internal/biz/playground/agentpool"
	playgroundruntime "ruleGoKratos/internal/biz/playground/runtime"
	"ruleGoKratos/internal/biz/playground/workflow"

	"github.com/go-kratos/kratos/v2/log"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// PlaygroundService Agent Playground 服务
type PlaygroundService struct {
	agentPoolSvc *agentpool.AgentPoolService
	workflowSvc  *workflow.WorkflowService
	log          *log.Helper
}

// NewPlaygroundService 创建 Playground 服务
func NewPlaygroundService(
	agentPoolSvc *agentpool.AgentPoolService,
	workflowSvc *workflow.WorkflowService,
	logger log.Logger,
) *PlaygroundService {
	helper := log.NewHelper(logger)

	return &PlaygroundService{
		agentPoolSvc: agentPoolSvc,
		workflowSvc:  workflowSvc,
		log:          helper,
	}
}

// RegisterPlaygroundHTTPRoutes 注册 Playground HTTP 路由
func RegisterPlaygroundHTTPRoutes(s *khttp.Server, svc *PlaygroundService) {
	r := s.Route("/api/v1/playground")

	// Agent Pool APIs
	r.GET("/pools", svc.listAgentPools)
	r.GET("/pools/{id}", svc.getAgentPool)
	r.POST("/pools", svc.createAgentPool)
	r.PUT("/pools/{id}", svc.updateAgentPool)
	r.DELETE("/pools/{id}", svc.deleteAgentPool)

	// Scheme APIs
	r.GET("/schemes", svc.listSchemes)
	r.GET("/schemes/{id}", svc.getScheme)
	r.POST("/schemes", svc.createScheme)
	r.PUT("/schemes/{id}", svc.updateScheme)
	r.DELETE("/schemes/{id}", svc.deleteScheme)

	// Run APIs
	r.POST("/run", svc.runWorkflow)
	r.GET("/run/{runId}", svc.getRun)
	r.POST("/run/{runId}/recovery-actions/{actionId}", svc.applyRecoveryAction)
	r.GET("/run/{runId}/events/stream", svc.streamRunEvents)
	r.GET("/run/{runId}/events", svc.getRunEvents)

	// Run Workspace APIs
	r.GET("/run/{runId}/workspace/files", svc.listRunWorkspaceFiles)
	r.GET("/run/{runId}/workspace/file", svc.readRunWorkspaceFile)

	// Mode APIs
	r.GET("/modes", svc.getCollaborationModes)
}

// ========== Agent Pool Handlers ==========

type listPoolsResp struct {
	Pools []*agentPoolResp `json:"pools"`
}

type agentPoolResp struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Agents      []*agentDefResp `json:"agents"`
	CreatedAt   string          `json:"createdAt"`
	UpdatedAt   string          `json:"updatedAt"`
}

type agentDefResp struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Role           string   `json:"role"`
	Desc           string   `json:"desc"`
	Model          string   `json:"model"`
	Tools          []string `json:"tools"`
	Enabled        bool     `json:"enabled"`
	Priority       int      `json:"priority"`
	ManagedAgentID int64    `json:"managedAgentId,omitempty"`
}

func (s *PlaygroundService) listAgentPools(ctx khttp.Context) error {
	// 空库或删库后：首次拉取池列表前幂等创建 default 池（Run 之外也应可见）
	if _, err := s.agentPoolSvc.CreateDefaultAgentPool(ctx); err != nil {
		return err
	}
	pools, err := s.agentPoolSvc.ListPools(ctx)
	if err != nil {
		return err
	}

	resp := &listPoolsResp{Pools: make([]*agentPoolResp, 0, len(pools))}
	for _, p := range pools {
		resp.Pools = append(resp.Pools, s.poolToResp(p))
	}

	return ctx.JSON(http.StatusOK, resp)
}

func (s *PlaygroundService) getAgentPool(ctx khttp.Context) error {
	var req struct {
		ID string `json:"id"` // BindVars：form 解码默认使用 json tag
	}
	if err := ctx.BindVars(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	id := req.ID
	if strings.TrimSpace(id) == "default" {
		if _, err := s.agentPoolSvc.CreateDefaultAgentPool(ctx); err != nil {
			return err
		}
	}
	pool, err := s.agentPoolSvc.GetPool(ctx, id)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, map[string]string{"error": "pool not found"})
	}

	agents := make([]*agentDefResp, 0, len(pool.Agents))
	for _, a := range pool.Agents {
		if a == nil {
			continue
		}
		agents = append(agents, &agentDefResp{
			ID:             a.ID,
			Name:           a.Name,
			Role:           a.Role,
			Desc:           a.Desc,
			Model:          a.Model,
			Tools:          a.Tools,
			Enabled:        a.Enabled,
			Priority:       a.Priority,
			ManagedAgentID: a.ManagedAgentID,
		})
	}

	resp := &agentPoolResp{
		ID:          pool.ID,
		Name:        pool.Name,
		Description: pool.Description,
		Agents:      agents,
		CreatedAt:   formatTime(pool.CreatedAt),
		UpdatedAt:   formatTime(pool.UpdatedAt),
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{"pool": resp})
}

type agentDefReq struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Role           string   `json:"role"`
	Desc           string   `json:"desc"`
	Model          string   `json:"model"`
	Tools          []string `json:"tools"`
	Enabled        bool     `json:"enabled"`
	Priority       int      `json:"priority"`
	ManagedAgentID int64    `json:"managedAgentId"`
}

func (s *PlaygroundService) createAgentPool(ctx khttp.Context) error {
	// 协作运行固定使用 id=default 的池（见 WorkflowService.Run）；不提供再建其它池。
	return ctx.JSON(http.StatusBadRequest, map[string]string{
		"error": "协作仅使用内置 default Agent 池，不支持新建其它池；请通过 PUT /playground/pools/default 维护成员与绑定",
	})
}

type updatePoolReq struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Agents      []*agentDefReq `json:"agents"`
}

func (s *PlaygroundService) updateAgentPool(ctx khttp.Context) error {
	var path struct {
		ID string `json:"id"` // BindVars：form 解码默认使用 json tag
	}
	if err := ctx.BindVars(&path); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	var req updatePoolReq
	if err := json.NewDecoder(ctx.Request().Body).Decode(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	agents := make([]*entity.AgentDefinition, 0)
	if req.Agents != nil {
		for _, a := range req.Agents {
			if a == nil {
				continue
			}
			agents = append(agents, &entity.AgentDefinition{
				ID:             a.ID,
				Name:           a.Name,
				Role:           a.Role,
				Desc:           a.Desc,
				Model:          a.Model,
				Tools:          a.Tools,
				Enabled:        a.Enabled,
				Priority:       a.Priority,
				ManagedAgentID: a.ManagedAgentID,
			})
		}
	}

	pool, err := s.agentPoolSvc.UpdatePool(ctx, path.ID, req.Name, req.Description, agents)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "pool not found") || strings.Contains(msg, "find pool") || strings.Contains(msg, "get pool") {
			return ctx.JSON(http.StatusNotFound, map[string]string{"error": msg})
		}
		if strings.Contains(msg, "pool id is empty") || strings.Contains(msg, "id is empty") {
			return ctx.JSON(http.StatusBadRequest, map[string]string{"error": msg})
		}
		s.log.Errorf("updateAgentPool: %v", err)
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": msg})
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{"pool": s.poolToResp(pool)})
}

func (s *PlaygroundService) deleteAgentPool(ctx khttp.Context) error {
	var path struct {
		ID string `json:"id"` // BindVars：form 解码默认使用 json tag
	}
	if err := ctx.BindVars(&path); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if err := s.agentPoolSvc.DeletePool(ctx, path.ID); err != nil {
		if strings.Contains(err.Error(), "cannot delete") {
			return ctx.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		return ctx.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}
	return ctx.JSON(http.StatusOK, map[string]string{"ok": "true"})
}

func (s *PlaygroundService) poolToResp(pool *entity.AgentPool) *agentPoolResp {
	if pool == nil {
		return &agentPoolResp{}
	}
	agents := make([]*agentDefResp, 0, len(pool.Agents))
	for _, a := range pool.Agents {
		if a == nil {
			continue
		}
		agents = append(agents, &agentDefResp{
			ID:             a.ID,
			Name:           a.Name,
			Role:           a.Role,
			Desc:           a.Desc,
			Model:          a.Model,
			Tools:          a.Tools,
			Enabled:        a.Enabled,
			Priority:       a.Priority,
			ManagedAgentID: a.ManagedAgentID,
		})
	}
	return &agentPoolResp{
		ID:          pool.ID,
		Name:        pool.Name,
		Description: pool.Description,
		Agents:      agents,
		CreatedAt:   formatTime(pool.CreatedAt),
		UpdatedAt:   formatTime(pool.UpdatedAt),
	}
}

// ========== Scheme Handlers ==========

type listSchemesResp struct {
	Schemes []*schemeResp `json:"schemes"`
}

type schemeResp struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Mode            string            `json:"mode"`
	BindAgents      []*agentBindResp  `json:"bindAgents"`
	Config          *schemeConfigResp `json:"config"`
	Enabled         bool              `json:"enabled"`
	EnableFinalizer bool              `json:"enableFinalizer"`
	CreatedAt       string            `json:"createdAt"`
	UpdatedAt       string            `json:"updatedAt"`
}

type agentBindResp struct {
	AgentID string   `json:"agentId"`
	Role    string   `json:"role"`
	Model   string   `json:"model"`
	Tools   []string `json:"tools"`
}

type routerConfigResp struct {
	FallbackAgent string `json:"fallbackAgent"`
	RoutingPrompt string `json:"routingPrompt"`
}

type planExecConfigResp struct {
	PlannerAgent   string   `json:"plannerAgent"`
	ExecutionOrder []string `json:"executionOrder"`
}

type supervisionConfigResp struct {
	SupervisorAgent string   `json:"supervisorAgent"`
	WorkerAgents    []string `json:"workerAgents"`
	CheckInterval   int      `json:"checkInterval"`
}

type peerHandoffConfigResp struct {
	EntryAgent   string   `json:"entryAgent"`
	MeshAgents   []string `json:"meshAgents"`
	HandoffRules string   `json:"handoffRules"`
}

type schemeModeConfigResp struct {
	RouterConfig      *routerConfigResp      `json:"routerConfig,omitempty"`
	PlanExecConfig    *planExecConfigResp    `json:"planExecConfig,omitempty"`
	SupervisionConfig *supervisionConfigResp `json:"supervisionConfig,omitempty"`
	PeerHandoffConfig *peerHandoffConfigResp `json:"peerHandoffConfig,omitempty"`
}

type schemeConfigResp struct {
	MaxIterations   int                   `json:"maxIterations"`
	MaxToolCalls    int                   `json:"maxToolCalls"`
	TimeoutSeconds  int                   `json:"timeoutSeconds"`
	FinalizerPrompt string                `json:"finalizerPrompt"`
	ModeConfig      *schemeModeConfigResp `json:"modeConfig,omitempty"`
}

func (s *PlaygroundService) listSchemes(ctx khttp.Context) error {
	schemes, err := s.workflowSvc.ListSchemes(ctx)
	if err != nil {
		return err
	}

	resp := &listSchemesResp{Schemes: make([]*schemeResp, 0, len(schemes))}
	for _, sc := range schemes {
		resp.Schemes = append(resp.Schemes, s.schemeToResp(sc))
	}

	return ctx.JSON(http.StatusOK, resp)
}

func (s *PlaygroundService) getScheme(ctx khttp.Context) error {
	var req struct {
		ID string `json:"id"` // BindVars：form 解码默认使用 json tag
	}
	if err := ctx.BindVars(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	id := req.ID
	sc, err := s.workflowSvc.GetScheme(ctx, id)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, map[string]string{"error": "scheme not found"})
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{"scheme": s.schemeToResp(sc)})
}

type createSchemeReq struct {
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Mode            string            `json:"mode"`
	BindAgents      []*agentBindResp  `json:"bindAgents"`
	Config          *schemeConfigResp `json:"config"`
	EnableFinalizer bool              `json:"enableFinalizer"`
}

func (s *PlaygroundService) createScheme(ctx khttp.Context) error {
	var req createSchemeReq
	if err := json.NewDecoder(ctx.Request().Body).Decode(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	bindings := make([]*entity.AgentBinding, 0, len(req.BindAgents))
	for _, b := range req.BindAgents {
		bindings = append(bindings, &entity.AgentBinding{
			AgentID: b.AgentID,
			Role:    b.Role,
			Model:   b.Model,
			Tools:   b.Tools,
		})
	}

	sc, err := s.workflowSvc.CreateScheme(ctx, req.Name, req.Description, entity.CollaborationMode(req.Mode), bindings)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	sc.EnableFinalizer = req.EnableFinalizer
	if req.Config != nil {
		patchSchemeConfig(sc, req.Config)
	}
	normalizeSchemeModeConfig(sc)
	if req.EnableFinalizer || req.Config != nil {
		if err := s.workflowSvc.UpdateScheme(ctx, sc); err != nil {
			return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	return ctx.JSON(http.StatusCreated, map[string]interface{}{"scheme": s.schemeToResp(sc)})
}

func (s *PlaygroundService) updateScheme(ctx khttp.Context) error {
	var req struct {
		ID string `json:"id"` // BindVars：form 解码默认使用 json tag
	}
	if err := ctx.BindVars(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	id := req.ID
	var req2 createSchemeReq
	if err := json.NewDecoder(ctx.Request().Body).Decode(&req2); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	sc, err := s.workflowSvc.GetScheme(ctx, id)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, map[string]string{"error": "scheme not found"})
	}

	sc.Name = req2.Name
	sc.Description = req2.Description
	sc.Mode = entity.CollaborationMode(req2.Mode)
	sc.EnableFinalizer = req2.EnableFinalizer

	if req2.BindAgents != nil {
		sc.BindAgents = make([]*entity.AgentBinding, 0, len(req2.BindAgents))
		for _, b := range req2.BindAgents {
			sc.BindAgents = append(sc.BindAgents, &entity.AgentBinding{
				AgentID: b.AgentID,
				Role:    b.Role,
				Model:   b.Model,
				Tools:   b.Tools,
			})
		}
	}

	if req2.Config != nil {
		patchSchemeConfig(sc, req2.Config)
	}
	normalizeSchemeModeConfig(sc)

	if err := s.workflowSvc.UpdateScheme(ctx, sc); err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{"scheme": s.schemeToResp(sc)})
}

func (s *PlaygroundService) deleteScheme(ctx khttp.Context) error {
	var req struct {
		ID string `json:"id"` // BindVars：form 解码默认使用 json tag
	}
	if err := ctx.BindVars(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	id := req.ID
	if err := s.workflowSvc.DeleteScheme(ctx, id); err != nil {
		return ctx.JSON(http.StatusNotFound, map[string]string{"error": "scheme not found"})
	}
	return ctx.JSON(http.StatusOK, map[string]string{})
}

func (s *PlaygroundService) schemeToResp(sc *entity.CollaborationScheme) *schemeResp {
	bindings := make([]*agentBindResp, 0, len(sc.BindAgents))
	for _, b := range sc.BindAgents {
		bindings = append(bindings, &agentBindResp{
			AgentID: b.AgentID,
			Role:    b.Role,
			Model:   b.Model,
			Tools:   b.Tools,
		})
	}

	cfg := &schemeConfigResp{}
	if sc.Config != nil {
		cfg.MaxIterations = sc.Config.MaxIterations
		cfg.MaxToolCalls = sc.Config.MaxToolCalls
		cfg.TimeoutSeconds = sc.Config.TimeoutSeconds
		cfg.FinalizerPrompt = sc.Config.FinalizerPrompt
		cfg.ModeConfig = schemeModeConfigToResp(sc.Mode, sc.Config.ModeConfig)
	}

	return &schemeResp{
		ID:              sc.ID,
		Name:            sc.Name,
		Description:     sc.Description,
		Mode:            string(sc.Mode),
		BindAgents:      bindings,
		Config:          cfg,
		Enabled:         sc.Enabled,
		EnableFinalizer: sc.EnableFinalizer,
		CreatedAt:       formatTime(sc.CreatedAt),
		UpdatedAt:       formatTime(sc.UpdatedAt),
	}
}

// ========== Run Handlers ==========

type runWorkflowReq struct {
	SchemeID  string `json:"schemeId"`
	UserInput string `json:"userInput"`
}

func (s *PlaygroundService) runWorkflow(ctx khttp.Context) error {
	var req runWorkflowReq
	if err := json.NewDecoder(ctx.Request().Body).Decode(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	runID, err := s.workflowSvc.Run(ctx, req.SchemeID, req.UserInput)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{"runId": runID})
}

func (s *PlaygroundService) getRun(ctx khttp.Context) error {
	var req struct {
		RunID string `json:"runId"`
	}
	if err := ctx.BindVars(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	runID := req.RunID
	traceRun, err := s.workflowSvc.GetRun(ctx, runID)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, map[string]string{"error": "run not found"})
	}
	runtimeRun, err := s.workflowSvc.GetRuntimeRun(ctx, runID)
	if err != nil {
		if errors.Is(err, playgroundruntime.ErrRunNotFound) {
			runtimeRun = buildFallbackRuntimeRun(traceRun)
		} else {
			s.log.Errorf("getRun runtime detail failed runID=%s: %v", runID, err)
			return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "load runtime run failed"})
		}
	}
	steps, err := s.workflowSvc.ListRuntimeSteps(ctx, runID)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	artifacts, err := s.workflowSvc.ListRuntimeArtifacts(ctx, runID)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	recoveryActions, err := s.workflowSvc.ListRecoveryActions(ctx, runID)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	resp := s.buildRunDetailResp(runtimeRun, steps, artifacts, recoveryActions)
	if resp.Run != nil {
		resp.Run.UserInput = traceRun.UserInput
		resp.Run.FinalOutput = traceRun.FinalOutput
	}
	return ctx.JSON(http.StatusOK, resp)
}

func (s *PlaygroundService) applyRecoveryAction(ctx khttp.Context) error {
	var req struct {
		RunID    string `json:"runId"`
		ActionID string `json:"actionId"`
	}
	if err := ctx.BindVars(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	var body struct {
		TargetRef string `json:"targetRef"`
	}
	if ctx.Request() != nil && ctx.Request().Body != nil {
		raw, err := io.ReadAll(ctx.Request().Body)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if len(strings.TrimSpace(string(raw))) > 0 {
			if err := json.Unmarshal(raw, &body); err != nil {
				return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
			}
		}
	}
	if err := s.workflowSvc.ApplyRecoveryAction(ctx, req.RunID, req.ActionID, strings.TrimSpace(body.TargetRef)); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return ctx.JSON(http.StatusOK, map[string]string{
		"runId":    req.RunID,
		"actionId": req.ActionID,
		"status":   "accepted",
	})
}

func (s *PlaygroundService) getRunEvents(ctx khttp.Context) error {
	var req struct {
		RunID string `json:"runId"`
	}
	if err := ctx.BindVars(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	runID := req.RunID

	events, err := s.workflowSvc.GetRunEvents(ctx, runID)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	eventResp := make([]*traceEventResp, 0, len(events))
	for _, e := range events {
		eventResp = append(eventResp, s.traceEventToResp(e))
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"events": eventResp,
	})
}

//lint:ignore U1000 "kept for future use"
type traceRunResp struct {
	ID          string            `json:"id"`
	RunID       string            `json:"runId"`
	SchemeID    string            `json:"schemeId"`
	UserInput   string            `json:"userInput"`
	Status      string            `json:"status"`
	StartTime   string            `json:"startTime"`
	EndTime     string            `json:"endTime"`
	TotalMs     int64             `json:"totalMs"`
	Events      []*traceEventResp `json:"events"`
	FinalOutput string            `json:"finalOutput"`
}

type runtimeRunResp struct {
	RunID            string   `json:"runId"`
	SchemeID         string   `json:"schemeId"`
	PlanID           string   `json:"planId"`
	Status           string   `json:"status"`
	CurrentStepIDs   []string `json:"currentStepIds,omitempty"`
	LastCheckpointID string   `json:"lastCheckpointId,omitempty"`
	FailureSummary   string   `json:"failureSummary,omitempty"`
	StartedAt        string   `json:"startedAt"`
	FinishedAt       string   `json:"finishedAt"`
	UserInput        string   `json:"userInput,omitempty"`
	FinalOutput      string   `json:"finalOutput,omitempty"`
	WorkspacePath    string   `json:"workspacePath,omitempty"`
}

type runtimeStepResp struct {
	StepID         string   `json:"stepId"`
	Kind           string   `json:"kind"`
	Name           string   `json:"name"`
	Status         string   `json:"status"`
	AgentBinding   string   `json:"agentBinding,omitempty"`
	FailureSummary string   `json:"failureSummary,omitempty"`
	InputRefs      []string `json:"inputRefs,omitempty"`
	OutputRef      string   `json:"outputRef,omitempty"`
}

type runtimeArtifactResp struct {
	ArtifactID     string `json:"artifactId"`
	Type           string `json:"type"`
	ProducerStepID string `json:"producerStepId"`
	Summary        string `json:"summary"`
}

type recoveryActionResp struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	StepID    string `json:"stepId"`
	TargetRef string `json:"targetRef,omitempty"`
	Reason    string `json:"reason"`
}

type runDetailResp struct {
	Run             *runtimeRunResp        `json:"run"`
	Steps           []*runtimeStepResp     `json:"steps"`
	Artifacts       []*runtimeArtifactResp `json:"artifacts"`
	RecoveryActions []*recoveryActionResp  `json:"recoveryActions"`
}

type traceEventResp struct {
	ID        string            `json:"id"`
	RunID     string            `json:"runId"`
	Timestamp int64             `json:"timestamp"`
	Type      string            `json:"type"`
	AgentID   string            `json:"agentId"`
	NodeID    string            `json:"nodeId"`
	TaskDesc  string            `json:"taskDesc"`
	Message   string            `json:"message"`
	Metadata  map[string]string `json:"metadata"`
}

//lint:ignore U1000 "kept for future use"
func (s *PlaygroundService) runToResp(run *entity.TraceRun) *traceRunResp {
	events := make([]*traceEventResp, 0, len(run.Events))
	for _, e := range run.Events {
		events = append(events, s.traceEventToResp(e))
	}

	return &traceRunResp{
		ID:          run.ID,
		RunID:       run.RunID,
		SchemeID:    run.SchemeID,
		UserInput:   run.UserInput,
		Status:      run.Status,
		StartTime:   formatTime(run.StartTime),
		EndTime:     formatTime(run.EndTime),
		TotalMs:     run.TotalMs,
		Events:      events,
		FinalOutput: run.FinalOutput,
	}
}

func (s *PlaygroundService) buildRunDetailResp(
	run *entity.PlaygroundRun,
	steps []*entity.RuntimeStep,
	artifacts []*entity.RuntimeArtifact,
	actions []*entity.RecoveryAction,
) *runDetailResp {
	stepResp := make([]*runtimeStepResp, 0, len(steps))
	for _, step := range steps {
		if step == nil {
			continue
		}
		stepResp = append(stepResp, &runtimeStepResp{
			StepID:         step.StepID,
			Kind:           string(step.Kind),
			Name:           step.Name,
			Status:         string(step.Status),
			AgentBinding:   step.AgentBinding,
			FailureSummary: step.FailureSummary,
			InputRefs:      append([]string(nil), step.InputRefs...),
			OutputRef:      step.OutputRef,
		})
	}

	artifactResp := make([]*runtimeArtifactResp, 0, len(artifacts))
	for _, artifact := range artifacts {
		if artifact == nil {
			continue
		}
		artifactResp = append(artifactResp, &runtimeArtifactResp{
			ArtifactID:     artifact.ArtifactID,
			Type:           artifact.Type,
			ProducerStepID: artifact.ProducerStepID,
			Summary:        artifact.Summary,
		})
	}

	actionResp := make([]*recoveryActionResp, 0, len(actions))
	for _, action := range actions {
		if action == nil {
			continue
		}
		actionResp = append(actionResp, &recoveryActionResp{
			ID:        action.ID,
			Type:      string(action.Type),
			StepID:    action.StepID,
			TargetRef: action.TargetRef,
			Reason:    action.Reason,
		})
	}

	return &runDetailResp{
		Run:             s.runtimeRunToResp(run),
		Steps:           stepResp,
		Artifacts:       artifactResp,
		RecoveryActions: actionResp,
	}
}

func (s *PlaygroundService) traceEventToResp(e *entity.TraceEvent) *traceEventResp {
	metadata := make(map[string]string)
	for k, v := range e.Metadata {
		if str, ok := v.(string); ok {
			metadata[k] = str
		} else {
			metadata[k] = fmt.Sprintf("%v", v)
		}
	}

	return &traceEventResp{
		ID:        e.ID,
		RunID:     e.RunID,
		Timestamp: e.Timestamp,
		Type:      string(e.Type),
		AgentID:   e.AgentID,
		NodeID:    e.NodeID,
		TaskDesc:  e.TaskDesc,
		Message:   e.Message,
		Metadata:  metadata,
	}
}

func (s *PlaygroundService) runtimeRunToResp(run *entity.PlaygroundRun) *runtimeRunResp {
	if run == nil {
		return &runtimeRunResp{}
	}
	return &runtimeRunResp{
		RunID:            run.RunID,
		SchemeID:         run.SchemeID,
		PlanID:           run.PlanID,
		Status:           string(run.Status),
		CurrentStepIDs:   append([]string(nil), run.CurrentStepIDs...),
		LastCheckpointID: run.LastCheckpointID,
		FailureSummary:   run.FailureSummary,
		StartedAt:        formatTime(run.StartedAt),
		FinishedAt:       formatTime(run.FinishedAt),
		WorkspacePath:    s.workflowSvc.ResolveRunWorkspacePath(run.RunID),
	}
}

func buildFallbackRuntimeRun(traceRun *entity.TraceRun) *entity.PlaygroundRun {
	if traceRun == nil {
		return &entity.PlaygroundRun{}
	}
	return &entity.PlaygroundRun{
		RunID:      traceRun.RunID,
		SchemeID:   traceRun.SchemeID,
		Status:     entity.RunStatus(traceRun.Status),
		StartedAt:  traceRun.StartTime,
		FinishedAt: traceRun.EndTime,
	}
}

// streamRunEvents SSE：先回放已有事件，再推送实时 Trace（与轮询互补，前端可仅用其一）。
func (s *PlaygroundService) streamRunEvents(ctx khttp.Context) error {
	var path struct {
		RunID string `json:"runId"`
	}
	if err := ctx.BindVars(&path); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	runID := path.RunID

	reqCtx := ctx.Request().Context()
	if _, err := s.workflowSvc.GetRun(reqCtx, runID); err != nil {
		return ctx.JSON(http.StatusNotFound, map[string]string{"error": "run not found"})
	}

	w := ctx.Response()
	h := w.Header()
	h.Set("Content-Type", "text/event-stream; charset=utf-8")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.log.Errorf("streamRunEvents: ResponseWriter not http.Flusher")
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "streaming unsupported"})
	}

	ch, unsub := s.workflowSvc.SubscribeRunEvents(runID)
	defer unsub()

	events, err := s.workflowSvc.GetRunEvents(reqCtx, runID)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	for _, ev := range events {
		if err := s.writeSSETraceEvent(w, ev); err != nil {
			return nil
		}
		flusher.Flush()
	}

	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-reqCtx.Done():
			return nil
		case <-ticker.C:
			_, _ = fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		case ev, ok := <-ch:
			if !ok {
				return nil
			}
			if ev == nil {
				continue
			}
			if err := s.writeSSETraceEvent(w, ev); err != nil {
				return nil
			}
			flusher.Flush()
			if ev.Type == entity.TraceEventWorkflowEnd {
				return nil
			}
		}
	}
}

func (s *PlaygroundService) writeSSETraceEvent(w http.ResponseWriter, ev *entity.TraceEvent) error {
	resp := s.traceEventToResp(ev)
	b, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", b)
	return err
}

// ========== Mode Handler ==========

type modeResp struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (s *PlaygroundService) getCollaborationModes(ctx khttp.Context) error {
	modes := []modeResp{
		{ID: "router_expert", Name: "路由专家", Description: "根据输入智能路由到最合适的 Agent"},
		{ID: "plan_exec", Name: "规划执行", Description: "规划师拆解任务后依次执行"},
		{ID: "supervision", Name: "动态监督", Description: "监督者并行监控 Workers"},
		{ID: "peer_handoff", Name: "同伴交接", Description: "Agent 之间自主协商交接任务"},
	}
	return ctx.JSON(http.StatusOK, map[string]interface{}{"modes": modes})
}

// ========== Run Workspace Handlers ==========

const maxWorkspaceFileReadBytes = 4 << 20 // 4 MiB

// workspaceFileItem 工作区文件列表项
type workspaceFileItem struct {
	Name    string `json:"name"`
	Type    string `json:"type"` // "file" | "dir"
	Size    int64  `json:"size"`
	ModTime string `json:"modTime"`
}

// listRunWorkspaceFiles 列出运行工作区内指定子路径的文件和目录。
func (s *PlaygroundService) listRunWorkspaceFiles(ctx khttp.Context) error {
	var path struct {
		RunID string `json:"runId"`
	}
	if err := ctx.BindVars(&path); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	runID := path.RunID

	// 验证 run 存在
	if _, err := s.workflowSvc.GetRun(ctx.Request().Context(), runID); err != nil {
		return ctx.JSON(http.StatusNotFound, map[string]string{"error": "run not found"})
	}

	runDir := s.workflowSvc.ResolveRunWorkspacePath(runID)
	if runDir == "" {
		return ctx.JSON(http.StatusOK, map[string]interface{}{"items": []workspaceFileItem{}})
	}

	// 解析子路径，防止路径穿越
	subPath := strings.TrimSpace(ctx.Request().URL.Query().Get("path"))

	var absDir string
	if subPath == "" {
		absDir = runDir
	} else {
		resolved, err := pgAbsPathUnderWorkspace(runDir, subPath)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "invalid path: " + err.Error()})
		}
		absDir = resolved
	}

	entries, err := os.ReadDir(absDir)
	if err != nil {
		if os.IsNotExist(err) {
			return ctx.JSON(http.StatusOK, map[string]interface{}{"items": []workspaceFileItem{}})
		}
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	items := make([]workspaceFileItem, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		itemType := "file"
		if e.IsDir() {
			itemType = "dir"
		}
		items = append(items, workspaceFileItem{
			Name:    e.Name(),
			Type:    itemType,
			Size:    info.Size(),
			ModTime: info.ModTime().Format(time.RFC3339),
		})
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{"items": items})
}

// readRunWorkspaceFile 读取运行工作区内指定文件的内容。
func (s *PlaygroundService) readRunWorkspaceFile(ctx khttp.Context) error {
	var path struct {
		RunID string `json:"runId"`
	}
	if err := ctx.BindVars(&path); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	runID := path.RunID

	// 验证 run 存在
	if _, err := s.workflowSvc.GetRun(ctx.Request().Context(), runID); err != nil {
		return ctx.JSON(http.StatusNotFound, map[string]string{"error": "run not found"})
	}

	runDir := s.workflowSvc.ResolveRunWorkspacePath(runID)
	if runDir == "" {
		return ctx.JSON(http.StatusNotFound, map[string]string{"error": "run workspace directory not found"})
	}

	filePath := strings.TrimSpace(ctx.Request().URL.Query().Get("path"))
	if filePath == "" {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "path parameter is required"})
	}

	absPath, err := pgAbsPathUnderWorkspace(runDir, filePath)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "invalid path: " + err.Error()})
	}

	st, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ctx.JSON(http.StatusNotFound, map[string]string{"error": "file not found"})
		}
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if st.IsDir() {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "path is a directory, not a file"})
	}
	if st.Size() > maxWorkspaceFileReadBytes {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "file too large (exceeds 4 MiB)"})
	}

	b, err := os.ReadFile(absPath)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if !pgIsMostlyText(b) {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "file appears to be binary, only text files are supported"})
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"content": string(b),
		"path":    filePath,
	})
}

// pgIsMostlyText 判断字节内容是否主要为文本。
func pgIsMostlyText(b []byte) bool {
	if len(b) == 0 {
		return true
	}
	n := len(b)
	if n > 8192 {
		n = 8192
	}
	ctl := 0
	for i := 0; i < n; i++ {
		c := b[i]
		if c == 0 || (c < 0x09 && c != 0x09 && c != 0x0a && c != 0x0d) {
			ctl++
		}
	}
	return ctl*20 < n
}

// pgAbsPathUnderWorkspace 确保路径在 workspace 根目录内，防止路径穿越。
func pgAbsPathUnderWorkspace(rootAbs, userPath string) (string, error) {
	userPath = strings.TrimSpace(userPath)
	if userPath == "" {
		return "", errors.New("path cannot be empty")
	}
	p := userPath
	if !filepath.IsAbs(p) {
		p = filepath.Join(rootAbs, p)
	}
	p = filepath.Clean(p)
	rootAbs = filepath.Clean(rootAbs)
	rel, err := filepath.Rel(rootAbs, p)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("path must be within workspace directory")
	}
	return p, nil
}

// ========== Helper ==========

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

// patchSchemeConfig 将 HTTP 请求中的配置合并到实体（保留 ModeConfig 等未在 JSON 中出现的字段）
func patchSchemeConfig(dst *entity.CollaborationScheme, src *schemeConfigResp) {
	if dst == nil || src == nil {
		return
	}
	if dst.Config == nil {
		c := *entity.DefaultSchemeConfig
		dst.Config = &c
	}
	dst.Config.MaxIterations = src.MaxIterations
	dst.Config.MaxToolCalls = src.MaxToolCalls
	dst.Config.TimeoutSeconds = src.TimeoutSeconds
	dst.Config.FinalizerPrompt = src.FinalizerPrompt
	dst.Config.ModeConfig = schemeModeConfigToEntity(dst.Mode, src.ModeConfig)
}

func schemeModeConfigToResp(mode entity.CollaborationMode, cfg *entity.ModeConfig) *schemeModeConfigResp {
	if cfg == nil {
		return nil
	}

	switch mode {
	case entity.ModeRouterExpert:
		if cfg.RouterConfig == nil {
			return nil
		}
		return &schemeModeConfigResp{
			RouterConfig: &routerConfigResp{
				FallbackAgent: cfg.RouterConfig.FallbackAgent,
				RoutingPrompt: cfg.RouterConfig.RoutingPrompt,
			},
		}
	case entity.ModePlanExec:
		if cfg.PlanExecConfig == nil {
			return nil
		}
		return &schemeModeConfigResp{
			PlanExecConfig: &planExecConfigResp{
				PlannerAgent:   cfg.PlanExecConfig.PlannerAgent,
				ExecutionOrder: append([]string(nil), cfg.PlanExecConfig.ExecutionOrder...),
			},
		}
	case entity.ModeSupervision:
		if cfg.SupervisionConfig == nil {
			return nil
		}
		return &schemeModeConfigResp{
			SupervisionConfig: &supervisionConfigResp{
				SupervisorAgent: cfg.SupervisionConfig.SupervisorAgent,
				WorkerAgents:    append([]string(nil), cfg.SupervisionConfig.WorkerAgents...),
				CheckInterval:   cfg.SupervisionConfig.CheckInterval,
			},
		}
	case entity.ModePeerHandoff:
		if cfg.PeerHandoffConfig == nil {
			return nil
		}
		return &schemeModeConfigResp{
			PeerHandoffConfig: &peerHandoffConfigResp{
				EntryAgent:   cfg.PeerHandoffConfig.EntryAgent,
				MeshAgents:   append([]string(nil), cfg.PeerHandoffConfig.MeshAgents...),
				HandoffRules: cfg.PeerHandoffConfig.HandoffRules,
			},
		}
	default:
		return nil
	}
}

func schemeModeConfigToEntity(mode entity.CollaborationMode, cfg *schemeModeConfigResp) *entity.ModeConfig {
	if cfg == nil {
		return nil
	}

	switch mode {
	case entity.ModeRouterExpert:
		if cfg.RouterConfig == nil {
			return nil
		}
		return &entity.ModeConfig{
			RouterConfig: &entity.RouterConfig{
				FallbackAgent: cfg.RouterConfig.FallbackAgent,
				RoutingPrompt: cfg.RouterConfig.RoutingPrompt,
			},
		}
	case entity.ModePlanExec:
		if cfg.PlanExecConfig == nil {
			return nil
		}
		return &entity.ModeConfig{
			PlanExecConfig: &entity.PlanExecConfig{
				PlannerAgent:   cfg.PlanExecConfig.PlannerAgent,
				ExecutionOrder: append([]string(nil), cfg.PlanExecConfig.ExecutionOrder...),
			},
		}
	case entity.ModeSupervision:
		if cfg.SupervisionConfig == nil {
			return nil
		}
		return &entity.ModeConfig{
			SupervisionConfig: &entity.SupervisionConfig{
				SupervisorAgent: cfg.SupervisionConfig.SupervisorAgent,
				WorkerAgents:    append([]string(nil), cfg.SupervisionConfig.WorkerAgents...),
				CheckInterval:   cfg.SupervisionConfig.CheckInterval,
			},
		}
	case entity.ModePeerHandoff:
		if cfg.PeerHandoffConfig == nil {
			return nil
		}
		return &entity.ModeConfig{
			PeerHandoffConfig: &entity.PeerHandoffConfig{
				EntryAgent:   cfg.PeerHandoffConfig.EntryAgent,
				MeshAgents:   append([]string(nil), cfg.PeerHandoffConfig.MeshAgents...),
				HandoffRules: cfg.PeerHandoffConfig.HandoffRules,
			},
		}
	default:
		return nil
	}
}

func normalizeSchemeModeConfig(sc *entity.CollaborationScheme) {
	if sc == nil || sc.Config == nil {
		return
	}

	switch sc.Mode {
	case entity.ModeRouterExpert:
		if sc.Config.ModeConfig == nil || sc.Config.ModeConfig.RouterConfig == nil {
			sc.Config.ModeConfig = nil
			return
		}
		sc.Config.ModeConfig = &entity.ModeConfig{
			RouterConfig: &entity.RouterConfig{
				FallbackAgent: sc.Config.ModeConfig.RouterConfig.FallbackAgent,
				RoutingPrompt: sc.Config.ModeConfig.RouterConfig.RoutingPrompt,
			},
		}
	case entity.ModePlanExec:
		if sc.Config.ModeConfig == nil || sc.Config.ModeConfig.PlanExecConfig == nil {
			sc.Config.ModeConfig = nil
			return
		}
		sc.Config.ModeConfig = &entity.ModeConfig{
			PlanExecConfig: &entity.PlanExecConfig{
				PlannerAgent:   sc.Config.ModeConfig.PlanExecConfig.PlannerAgent,
				ExecutionOrder: append([]string(nil), sc.Config.ModeConfig.PlanExecConfig.ExecutionOrder...),
			},
		}
	case entity.ModeSupervision:
		if sc.Config.ModeConfig == nil || sc.Config.ModeConfig.SupervisionConfig == nil {
			sc.Config.ModeConfig = nil
			return
		}
		sc.Config.ModeConfig = &entity.ModeConfig{
			SupervisionConfig: &entity.SupervisionConfig{
				SupervisorAgent: sc.Config.ModeConfig.SupervisionConfig.SupervisorAgent,
				WorkerAgents:    append([]string(nil), sc.Config.ModeConfig.SupervisionConfig.WorkerAgents...),
				CheckInterval:   sc.Config.ModeConfig.SupervisionConfig.CheckInterval,
			},
		}
	case entity.ModePeerHandoff:
		if sc.Config.ModeConfig == nil || sc.Config.ModeConfig.PeerHandoffConfig == nil {
			sc.Config.ModeConfig = nil
			return
		}
		sc.Config.ModeConfig = &entity.ModeConfig{
			PeerHandoffConfig: &entity.PeerHandoffConfig{
				EntryAgent:   sc.Config.ModeConfig.PeerHandoffConfig.EntryAgent,
				MeshAgents:   append([]string(nil), sc.Config.ModeConfig.PeerHandoffConfig.MeshAgents...),
				HandoffRules: sc.Config.ModeConfig.PeerHandoffConfig.HandoffRules,
			},
		}
	default:
		sc.Config.ModeConfig = nil
	}
}
