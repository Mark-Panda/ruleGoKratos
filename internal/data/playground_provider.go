package data

import (
	"ruleGoKratos/internal/biz/playground/agentpool"
	playgroundruntime "ruleGoKratos/internal/biz/playground/runtime"
	"ruleGoKratos/internal/biz/playground/trace"
	"ruleGoKratos/internal/biz/playground/workflow"
	playgrounddata "ruleGoKratos/internal/data/playground"

	"github.com/google/wire"
)

// NewPlaygroundAgentPoolRepo Playground Agent 池持久化到 PostgreSQL（表 playground_agent_pool）。
func NewPlaygroundAgentPoolRepo(d *Data) agentpool.AgentPoolRepo {
	return playgrounddata.NewGormAgentPoolRepo(d.DB())
}

// NewPlaygroundSchemeRepo 协作编排方案持久化到 PostgreSQL（表 playground_collaboration_scheme）。
func NewPlaygroundSchemeRepo(d *Data) *playgrounddata.GormSchemeRepo {
	return playgrounddata.NewGormSchemeRepo(d.DB())
}

// NewPlaygroundRuntimeRepo 提供 Playground runtime 的数据层接入点。
func NewPlaygroundRuntimeRepo(d *Data) playgroundruntime.Repo {
	if d == nil {
		return playgrounddata.NewGormRuntimeRepo(nil)
	}
	return playgrounddata.NewGormRuntimeRepo(d.DB())
}

// PlaygroundProviderSet Playground 数据层依赖注入集合
var PlaygroundProviderSet = wire.NewSet(
	NewPlaygroundAgentPoolRepo,
	NewPlaygroundSchemeRepo,
	NewPlaygroundRuntimeRepo,
	playgrounddata.NewTraceRepo,
	wire.Bind(new(workflow.WorkflowRepo), new(*playgrounddata.GormSchemeRepo)),
	wire.Bind(new(trace.TraceRepo), new(*playgrounddata.TraceRepo)),
	agentpool.NewAgentPoolService,
	trace.NewTraceEngine,
	workflow.NewWorkflowServiceWithRuntimeRepo,
)
