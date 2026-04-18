package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ruleGoKratos/internal/biz/entity"
	"ruleGoKratos/internal/biz/playground/agentpool"
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
	r.GET("/run/{runId}/events/stream", svc.streamRunEvents)
	r.GET("/run/{runId}/events", svc.getRunEvents)

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
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Role             string   `json:"role"`
	Desc             string   `json:"desc"`
	Model            string   `json:"model"`
	Tools            []string `json:"tools"`
	Enabled          bool     `json:"enabled"`
	Priority         int      `json:"priority"`
	ManagedAgentID   int64    `json:"managedAgentId,omitempty"`
}

func (s *PlaygroundService) listAgentPools(ctx khttp.Context) error {
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
			ID:               a.ID,
			Name:             a.Name,
			Role:             a.Role,
			Desc:             a.Desc,
			Model:            a.Model,
			Tools:            a.Tools,
			Enabled:          a.Enabled,
			Priority:         a.Priority,
			ManagedAgentID:   a.ManagedAgentID,
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

type createPoolReq struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Agents      []*agentDefReq `json:"agents"`
}

type agentDefReq struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Role             string   `json:"role"`
	Desc             string   `json:"desc"`
	Model            string   `json:"model"`
	Tools            []string `json:"tools"`
	Enabled          bool     `json:"enabled"`
	Priority         int      `json:"priority"`
	ManagedAgentID   int64    `json:"managedAgentId"`
}

func (s *PlaygroundService) createAgentPool(ctx khttp.Context) error {
	var req createPoolReq
	if err := json.NewDecoder(ctx.Request().Body).Decode(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	agents := make([]*entity.AgentDefinition, 0)
	if req.Agents != nil {
		for _, a := range req.Agents {
			agents = append(agents, &entity.AgentDefinition{
				ID:               a.ID,
				Name:             a.Name,
				Role:             a.Role,
				Desc:             a.Desc,
				Model:            a.Model,
				Tools:            a.Tools,
				Enabled:          a.Enabled,
				Priority:         a.Priority,
				ManagedAgentID:   a.ManagedAgentID,
			})
		}
	}

	pool, err := s.agentPoolSvc.CreatePool(ctx, req.Name, req.Description, agents)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return ctx.JSON(http.StatusCreated, map[string]interface{}{"pool": s.poolToResp(pool)})
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
				ID:               a.ID,
				Name:             a.Name,
				Role:             a.Role,
				Desc:             a.Desc,
				Model:            a.Model,
				Tools:            a.Tools,
				Enabled:          a.Enabled,
				Priority:         a.Priority,
				ManagedAgentID:   a.ManagedAgentID,
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
			ID:               a.ID,
			Name:             a.Name,
			Role:             a.Role,
			Desc:             a.Desc,
			Model:            a.Model,
			Tools:            a.Tools,
			Enabled:          a.Enabled,
			Priority:         a.Priority,
			ManagedAgentID:   a.ManagedAgentID,
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

type schemeConfigResp struct {
	MaxIterations   int    `json:"maxIterations"`
	MaxToolCalls    int    `json:"maxToolCalls"`
	TimeoutSeconds  int    `json:"timeoutSeconds"`
	FinalizerPrompt string `json:"finalizerPrompt"`
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
	run, err := s.workflowSvc.GetRun(ctx, runID)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, map[string]string{"error": "run not found"})
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"run": s.runToResp(run),
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
}
