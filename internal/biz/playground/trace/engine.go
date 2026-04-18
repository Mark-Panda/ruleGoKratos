package trace

import (
	"context"
	"fmt"
	"ruleGoKratos/internal/biz/entity"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TraceEventHandler 事件处理函数类型
type TraceEventHandler func(event *entity.TraceEvent)

// TraceRepo Trace 仓储接口
type TraceRepo interface {
	Save(ctx context.Context, run *entity.TraceRun) error
	Update(ctx context.Context, run *entity.TraceRun) error
	FindByID(ctx context.Context, id string) (*entity.TraceRun, error)
	FindByRunID(ctx context.Context, runID string) (*entity.TraceRun, error)
	FindEvents(ctx context.Context, filter *entity.TraceFilter) ([]*entity.TraceEvent, error)
	AppendEvent(ctx context.Context, event *entity.TraceEvent) error
}

// TraceEngine Trace 引擎
type TraceEngine struct {
	repo     TraceRepo
	runs     map[string]*entity.TraceRun
	runsMu   sync.RWMutex
	handlers []TraceEventHandler

	// subs：按 runID 订阅实时事件（SSE）；EmitEvent 时 fan-out
	subMu sync.Mutex
	subs  map[string][]chan *entity.TraceEvent
}

func NewTraceEngine(repo TraceRepo) *TraceEngine {
	return &TraceEngine{
		repo:     repo,
		runs:     make(map[string]*entity.TraceRun),
		handlers: make([]TraceEventHandler, 0),
		subs:     make(map[string][]chan *entity.TraceEvent),
	}
}

// RegisterHandler 注册事件处理器
func (e *TraceEngine) RegisterHandler(handler TraceEventHandler) {
	e.handlers = append(e.handlers, handler)
}

// StartRun 开始一次运行
func (e *TraceEngine) StartRun(ctx context.Context, schemeID, userInput string) (*entity.TraceRun, error) {
	runID := uuid.NewString()
	run := &entity.TraceRun{
		ID:        uuid.NewString(),
		RunID:     runID,
		SchemeID:  schemeID,
		UserInput: userInput,
		Status:    "running",
		StartTime: nowPtr(),
		Events:    make([]*entity.TraceEvent, 0),
	}

	if err := e.repo.Save(ctx, run); err != nil {
		return nil, fmt.Errorf("save run: %w", err)
	}

	e.runsMu.Lock()
	e.runs[runID] = run
	e.runsMu.Unlock()

	// 记录开始事件
	e.EmitEvent(ctx, entity.NewTraceEvent(
		runID,
		entity.TraceEventWorkflowStart,
		"",
		"",
		"",
		"工作流开始",
	))

	return run, nil
}

// EndRun 结束一次运行
func (e *TraceEngine) EndRun(ctx context.Context, runID, finalOutput string, status string) error {
	e.runsMu.Lock()
	run, ok := e.runs[runID]
	if ok {
		run.Status = status
		run.FinalOutput = finalOutput
		run.EndTime = nowPtr()
		run.TotalMs = run.EndTime.UnixMilli() - run.StartTime.UnixMilli()
	}
	e.runsMu.Unlock()

	if run == nil {
		var err error
		run, err = e.repo.FindByRunID(ctx, runID)
		if err != nil {
			return fmt.Errorf("find run: %w", err)
		}
		run.Status = status
		run.FinalOutput = finalOutput
		run.EndTime = nowPtr()
		run.TotalMs = run.EndTime.UnixMilli() - run.StartTime.UnixMilli()
	}

	if err := e.repo.Update(ctx, run); err != nil {
		return fmt.Errorf("update run: %w", err)
	}

	// 记录结束事件
	e.EmitEvent(ctx, entity.NewTraceEvent(
		runID,
		entity.TraceEventWorkflowEnd,
		"",
		"",
		"",
		fmt.Sprintf("工作流结束: %s", status),
	))

	return nil
}

// EmitEvent 发送事件
func (e *TraceEngine) EmitEvent(ctx context.Context, event *entity.TraceEvent) {
	// 保存到运行记录
	e.runsMu.Lock()
	if run, ok := e.runs[event.RunID]; ok {
		run.Events = append(run.Events, event)
	}
	e.runsMu.Unlock()

	// 持久化
	if err := e.repo.AppendEvent(ctx, event); err != nil {
		// 日志记录，但不阻塞主流程
		fmt.Printf("append event error: %v\n", err)
	}

	// 调用处理器
	for _, h := range e.handlers {
		h(event)
	}

	e.fanoutEvent(event)
}

// SubscribeRun 订阅某 run 的新增事件（含 fan-out 副本，避免并发读同一指针）。
// 返回的取消函数会从订阅表移除该 channel（不关闭 channel，避免向已关闭 chan 发送）。
func (e *TraceEngine) SubscribeRun(runID string) (<-chan *entity.TraceEvent, func()) {
	ch := make(chan *entity.TraceEvent, 256)
	e.subMu.Lock()
	e.subs[runID] = append(e.subs[runID], ch)
	e.subMu.Unlock()
	unsub := func() {
		e.subMu.Lock()
		defer e.subMu.Unlock()
		list := e.subs[runID]
		for i, c := range list {
			if c == ch {
				e.subs[runID] = append(list[:i], list[i+1:]...)
				break
			}
		}
		if len(e.subs[runID]) == 0 {
			delete(e.subs, runID)
		}
	}
	return ch, unsub
}

func (e *TraceEngine) fanoutEvent(ev *entity.TraceEvent) {
	if ev == nil {
		return
	}
	e.subMu.Lock()
	list := append([]chan *entity.TraceEvent(nil), e.subs[ev.RunID]...)
	e.subMu.Unlock()
	for _, ch := range list {
		msg := cloneTraceEvent(ev)
		select {
		case ch <- msg:
		default:
			// 消费者过慢时丢弃，避免阻塞协作主流程
		}
	}
}

// TaskAssigned 记录任务分配
func (e *TraceEngine) TaskAssigned(ctx context.Context, runID, agentID, taskDesc string) {
	e.EmitEvent(ctx, entity.NewTraceEvent(
		runID,
		entity.TraceEventTaskAssigned,
		agentID,
		"",
		taskDesc,
		fmt.Sprintf("任务分配给 %s: %s", agentID, taskDesc),
	))
}

// AgentEnterWorker 记录 Agent 进入工作状态
func (e *TraceEngine) AgentEnterWorker(ctx context.Context, runID, agentID, nodeID string) {
	e.EmitEvent(ctx, entity.NewTraceEvent(
		runID,
		entity.TraceEventAgentEnterWorker,
		agentID,
		nodeID,
		"",
		fmt.Sprintf("Agent %s 进入工作状态", agentID),
	))
}

// AgentExitWorker 记录 Agent 退出工作状态
func (e *TraceEngine) AgentExitWorker(ctx context.Context, runID, agentID, nodeID, message string) {
	e.EmitEvent(ctx, entity.NewTraceEvent(
		runID,
		entity.TraceEventAgentExitWorker,
		agentID,
		nodeID,
		"",
		message,
	))
}

// WorkerDelegated 记录任务委派
func (e *TraceEngine) WorkerDelegated(ctx context.Context, runID, agentID, workerAgentID, reason string) {
	e.EmitEvent(ctx, entity.NewTraceEvent(
		runID,
		entity.TraceEventWorkerDelegated,
		agentID,
		"",
		"",
		fmt.Sprintf("任务委派给 %s: %s", workerAgentID, reason),
	).WithMetadata("workerAgentId", workerAgentID).WithMetadata("reason", reason))
}

// Thinking 记录思考状态
func (e *TraceEngine) Thinking(ctx context.Context, runID, agentID, message string) {
	e.EmitEvent(ctx, entity.NewTraceEvent(
		runID,
		entity.TraceEventThinking,
		agentID,
		"",
		"",
		message,
	))
}

// ToolCall 记录工具调用
func (e *TraceEngine) ToolCall(ctx context.Context, runID, agentID, toolName, args string) {
	e.EmitEvent(ctx, entity.NewTraceEvent(
		runID,
		entity.TraceEventToolCall,
		agentID,
		"",
		"",
		fmt.Sprintf("调用工具 %s", toolName),
	).WithMetadata("toolName", toolName).WithMetadata("args", args))
}

// ToolResult 记录工具结果
func (e *TraceEngine) ToolResult(ctx context.Context, runID, agentID, toolName, result string, success bool) {
	e.EmitEvent(ctx, entity.NewTraceEvent(
		runID,
		entity.TraceEventToolResult,
		agentID,
		"",
		"",
		fmt.Sprintf("工具 %s 执行%s", toolName, map[bool]string{true: "成功", false: "失败"}[success]),
	).WithMetadata("toolName", toolName).WithMetadata("result", truncate(result, 500)).WithMetadata("success", success))
}

// Handoff 记录任务交接
func (e *TraceEngine) Handoff(ctx context.Context, runID, fromAgent, toAgent, reason string) {
	e.EmitEvent(ctx, entity.NewTraceEvent(
		runID,
		entity.TraceEventHandoff,
		fromAgent,
		"",
		"",
		fmt.Sprintf("任务从 %s 交接给 %s", fromAgent, toAgent),
	).WithMetadata("toAgent", toAgent).WithMetadata("reason", reason))
}

// Error 记录错误
func (e *TraceEngine) Error(ctx context.Context, runID, agentID, message string) {
	e.EmitEvent(ctx, entity.NewTraceEvent(
		runID,
		entity.TraceEventError,
		agentID,
		"",
		"",
		message,
	))
}

// GetRun 获取运行记录
func (e *TraceEngine) GetRun(ctx context.Context, runID string) (*entity.TraceRun, error) {
	e.runsMu.RLock()
	if run, ok := e.runs[runID]; ok {
		e.runsMu.RUnlock()
		return run, nil
	}
	e.runsMu.RUnlock()

	return e.repo.FindByRunID(ctx, runID)
}

// GetEvents 获取事件列表
func (e *TraceEngine) GetEvents(ctx context.Context, filter *entity.TraceFilter) ([]*entity.TraceEvent, error) {
	return e.repo.FindEvents(ctx, filter)
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// nowPtr 返回当前时间指针
func nowPtr() *time.Time {
	now := time.Now()
	return &now
}
