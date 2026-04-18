package workflow

import "ruleGoKratos/internal/biz/entity"

// EnsurePlanExecModeConfig 根据方案的 bindAgents 补全规划执行专用配置，避免仍走默认的 designer/pm/engineer 全量链路
// 与用户在「协作编排」中实际绑定的成员不一致。
func EnsurePlanExecModeConfig(scheme *entity.CollaborationScheme) {
	if scheme == nil || scheme.Mode != entity.ModePlanExec {
		return
	}
	if scheme.Config == nil {
		c := *entity.DefaultSchemeConfig
		scheme.Config = &c
	}
	if scheme.Config.ModeConfig == nil {
		scheme.Config.ModeConfig = &entity.ModeConfig{}
	}
	if scheme.Config.ModeConfig.PlanExecConfig == nil {
		scheme.Config.ModeConfig.PlanExecConfig = &entity.PlanExecConfig{}
	}
	pec := scheme.Config.ModeConfig.PlanExecConfig

	// 规划师：绑定里显式出现 planner 则用其 id，否则默认池中的 "planner"
	plannerID := "planner"
	for _, b := range scheme.BindAgents {
		if b != nil && b.AgentID == "planner" {
			plannerID = "planner"
			break
		}
	}
	pec.PlannerAgent = plannerID

	// 执行顺序：绑定列表中除规划师外的成员，顺序与绑定一致（与示意图、用户预期对齐）
	var exec []string
	for _, b := range scheme.BindAgents {
		if b == nil || b.AgentID == "" {
			continue
		}
		if b.AgentID == plannerID {
			continue
		}
		exec = append(exec, b.AgentID)
	}
	if len(exec) > 0 {
		pec.ExecutionOrder = exec
	}
}
