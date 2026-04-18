package collaboration

import (
	"context"
	"fmt"
	"ruleGoKratos/internal/biz/entity"
)

// CollaborationHandler 协作处理器接口
type CollaborationHandler interface {
	// Init 初始化（rt.AgentUC 非 nil 时使用真实 Harness；nil 则为占位输出，便于单测）
	Init(ctx context.Context, scheme *entity.CollaborationScheme, pool *entity.AgentPool, rt *CollaborationRuntime) error
	// Execute 执行协作
	Execute(ctx context.Context, runID string, input string, trace TraceEmitter) (*entity.AgentInstance, error)
	// Name 获取名称
	Name() string
}

// TraceEmitter Trace 发射接口
type TraceEmitter interface {
	TaskAssigned(runID, agentID, taskDesc string)
	AgentEnterWorker(runID, agentID, nodeID string)
	AgentExitWorker(runID, agentID, nodeID, message string)
	WorkerDelegated(runID, agentID, workerAgentID, reason string)
	Thinking(runID, agentID, message string)
	ToolCall(runID, agentID, toolName, args string)
	ToolResult(runID, agentID, toolName, result string, success bool)
	Handoff(runID, fromAgent, toAgent, reason string)
	Error(runID, agentID, message string)
	EmitEvent(ctx context.Context, event *entity.TraceEvent)
}

// Factory 协作模式工厂
type Factory struct {
	handlers map[entity.CollaborationMode]CollaborationHandler
}

func NewFactory() *Factory {
	return &Factory{
		handlers: make(map[entity.CollaborationMode]CollaborationHandler),
	}
}

// Register 注册协作处理器
func (f *Factory) Register(mode entity.CollaborationMode, handler CollaborationHandler) {
	f.handlers[mode] = handler
}

// Get 获取协作处理器
func (f *Factory) Get(mode entity.CollaborationMode) (CollaborationHandler, error) {
	handler, ok := f.handlers[mode]
	if !ok {
		return nil, fmt.Errorf("unsupported collaboration mode: %s", mode)
	}
	return handler, nil
}

// ListModes 列出所有支持的模式
func (f *Factory) ListModes() []entity.CollaborationMode {
	modes := make([]entity.CollaborationMode, 0, len(f.handlers))
	for m := range f.handlers {
		modes = append(modes, m)
	}
	return modes
}
