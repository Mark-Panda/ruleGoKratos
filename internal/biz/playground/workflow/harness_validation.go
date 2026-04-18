package workflow

import (
	"fmt"
	"strings"

	"ruleGoKratos/internal/biz/entity"
)

func findAgentDefInPool(pool *entity.AgentPool, agentID string) *entity.AgentDefinition {
	if pool == nil || agentID == "" {
		return nil
	}
	for _, a := range pool.Agents {
		if a != nil && a.ID == agentID {
			return a
		}
	}
	return nil
}

// validateAgentsForHarness 在创建 Trace 前校验：真实执行 Harness 时每个参与成员必须已绑定 managedAgentId。
func (s *WorkflowService) validateAgentsForHarness(scheme *entity.CollaborationScheme, pool *entity.AgentPool) error {
	if s.agentUC == nil {
		return nil
	}
	if scheme == nil || pool == nil {
		return nil
	}

	var agentIDs []string
	switch scheme.Mode {
	case entity.ModePlanExec:
		EnsurePlanExecModeConfig(scheme)
		var pec *entity.PlanExecConfig
		if scheme.Config != nil && scheme.Config.ModeConfig != nil {
			pec = scheme.Config.ModeConfig.PlanExecConfig
		}
		if pec != nil && strings.TrimSpace(pec.PlannerAgent) != "" {
			agentIDs = append(agentIDs, pec.PlannerAgent)
		}
		if pec != nil && len(pec.ExecutionOrder) > 0 {
			agentIDs = append(agentIDs, pec.ExecutionOrder...)
		} else {
			for _, id := range []string{"designer", "pm", "engineer"} {
				if d := findAgentDefInPool(pool, id); d != nil && d.Enabled {
					agentIDs = append(agentIDs, id)
				}
			}
		}
	default:
		for _, b := range scheme.BindAgents {
			if b != nil && b.AgentID != "" {
				agentIDs = append(agentIDs, b.AgentID)
			}
		}
	}

	seen := make(map[string]struct{})
	var problems []string
	for _, id := range agentIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		def := findAgentDefInPool(pool, id)
		if def == nil {
			problems = append(problems, fmt.Sprintf("%s（池内无定义）", id))
			continue
		}
		if def.ManagedAgentID <= 0 {
			problems = append(problems, fmt.Sprintf("「%s」(%s)", def.Name, def.ID))
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf(
		"以下成员未关联主站「Agent 配置」(managedAgentId)，无法调用模型。请在 Playground「智能体」页为对应池成员选择托管 Agent 后再运行：%s",
		strings.Join(problems, "、"),
	)
}
