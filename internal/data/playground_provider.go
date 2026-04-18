package data

import (
	"ruleGoKratos/internal/biz/playground/agentpool"
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

// PlaygroundProviderSet Playground 数据层依赖注入集合
var PlaygroundProviderSet = wire.NewSet(
	NewPlaygroundAgentPoolRepo,
	NewPlaygroundSchemeRepo,
	playgrounddata.NewTraceRepo,
	wire.Bind(new(workflow.WorkflowRepo), new(*playgrounddata.GormSchemeRepo)),
	wire.Bind(new(trace.TraceRepo), new(*playgrounddata.TraceRepo)),
	agentpool.NewAgentPoolService,
	trace.NewTraceEngine,
	workflow.NewWorkflowService,
)
