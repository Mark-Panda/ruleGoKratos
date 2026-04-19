package runtime

import (
	"context"
	"fmt"
	"strings"

	"ruleGoKratos/internal/biz/entity"
)

// RouteStepExecutor 执行最小路由步骤。
type RouteStepExecutor struct{}

// NewRouteStepExecutor 创建 RouteStepExecutor。
func NewRouteStepExecutor() *RouteStepExecutor {
	return &RouteStepExecutor{}
}

// Execute 产出结构化路由选择结果。
func (e *RouteStepExecutor) Execute(
	_ context.Context,
	execCtx *executionContext,
	step *entity.PlanStep,
) (*entity.RuntimeArtifact, error) {
	if forced := strings.TrimSpace(execCtx.ForcedRouteAgent); forced != "" {
		if def := findEnabledAgent(execCtx.pool, forced); def == nil {
			return nil, fmt.Errorf("forced route agent %q is not available in pool", forced)
		}
		if execCtx.trace != nil {
			execCtx.trace.Thinking(execCtx.runID, "router", "按恢复动作强制路由到 "+forced)
		}
		return newArtifact(execCtx.runID, step.StepID, "route_result", forced, map[string]any{
			"selected_agent":   forced,
			"candidate_agents": routeCandidateAgents(execCtx.scheme, step),
			"reason":           "recovery_forced_route",
			"confidence":       float64(1),
		}), nil
	}

	candidateAgents := routeCandidateAgents(execCtx.scheme, step)
	if len(candidateAgents) == 0 {
		return nil, fmt.Errorf("route step %s has no candidate agents", step.StepID)
	}

	selectedAgent, reason, confidence, err := selectRouteAgent(execCtx.pool, candidateAgents, fallbackAgent(step), execCtx.userInput)
	if err != nil {
		return nil, err
	}

	if execCtx.trace != nil {
		execCtx.trace.Thinking(execCtx.runID, "router", "分析任务路由")
	}

	return newArtifact(execCtx.runID, step.StepID, "route_result", selectedAgent, map[string]any{
		"selected_agent":   selectedAgent,
		"candidate_agents": candidateAgents,
		"reason":           reason,
		"confidence":       confidence,
	}), nil
}

func routeCandidateAgents(scheme *entity.CollaborationScheme, step *entity.PlanStep) []string {
	if step != nil && step.Config != nil {
		if values := stringSliceConfig(step.Config["candidateAgents"]); len(values) > 0 {
			return values
		}
	}
	if scheme == nil {
		return nil
	}
	candidates := make([]string, 0, len(scheme.BindAgents))
	for _, binding := range scheme.BindAgents {
		if binding == nil || binding.AgentID == "" {
			continue
		}
		candidates = append(candidates, binding.AgentID)
	}
	return candidates
}

func fallbackAgent(step *entity.PlanStep) string {
	if step == nil || step.Config == nil {
		return ""
	}
	if value, ok := step.Config["fallbackAgent"].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func selectRouteAgent(
	pool *entity.AgentPool,
	candidateAgents []string,
	fallbackAgentID string,
	input string,
) (string, string, float64, error) {
	allowed := make(map[string]struct{}, len(candidateAgents))
	for _, agentID := range candidateAgents {
		allowed[agentID] = struct{}{}
	}

	inputLower := strings.ToLower(input)
	routeOrder := []struct {
		id       string
		keywords []string
	}{
		{"engineer", []string{"代码", "开发", "实现", "bug", "修复", "html", "css", "js"}},
		{"planner", []string{"规划", "拆解", "步骤", "计划"}},
		{"designer", []string{"设计", "界面", "ui", "ux", "样式", "布局", "颜色"}},
		{"pm", []string{"需求", "功能", "产品", "prd", "mrd"}},
	}

	for _, route := range routeOrder {
		if _, ok := allowed[route.id]; !ok {
			continue
		}
		def := findEnabledAgent(pool, route.id)
		if def == nil {
			continue
		}
		if containsAny(inputLower, route.keywords) {
			return def.ID, "keyword_match", 0.9, nil
		}
	}

	if fallbackAgentID != "" {
		if def := findEnabledAgent(pool, fallbackAgentID); def != nil {
			if _, ok := allowed[def.ID]; ok {
				return def.ID, "fallback_agent", 0.5, nil
			}
		}
	}

	for _, agentID := range candidateAgents {
		if def := findEnabledAgent(pool, agentID); def != nil {
			return def.ID, "first_enabled_candidate", 0.4, nil
		}
	}

	return "", "", 0, fmt.Errorf("no enabled route candidate found")
}

func findEnabledAgent(pool *entity.AgentPool, agentID string) *entity.AgentDefinition {
	if pool == nil || agentID == "" {
		return nil
	}
	for _, agent := range pool.Agents {
		if agent != nil && agent.ID == agentID && agent.Enabled {
			return agent
		}
	}
	return nil
}

func stringSliceConfig(value any) []string {
	switch values := value.(type) {
	case []string:
		return append([]string(nil), values...)
	case []any:
		out := make([]string, 0, len(values))
		for _, value := range values {
			str, ok := value.(string)
			if !ok || strings.TrimSpace(str) == "" {
				continue
			}
			out = append(out, str)
		}
		return out
	default:
		return nil
	}
}

func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
