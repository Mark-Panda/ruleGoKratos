package collaboration

import "ruleGoKratos/internal/biz"

// CollaborationRuntime 协作执行期依赖（由 WorkflowService 注入）。
type CollaborationRuntime struct {
	AgentUC *biz.AgentUsecase // 非 nil 时走真实 StreamHarness；nil 时 handler 使用占位输出（单测）
}
