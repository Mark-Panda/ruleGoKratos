package collaboration

import (
	"context"
	"fmt"
	"ruleGoKratos/internal/biz/entity"
)

// RouterExpertHandler 路由专家模式处理器
// 根据用户输入，LLM 决策应该由哪个 Agent 处理
type RouterExpertHandler struct {
	scheme *entity.CollaborationScheme
	pool   *entity.AgentPool
	rt     *CollaborationRuntime
}

func NewRouterExpertHandler() *RouterExpertHandler {
	return &RouterExpertHandler{}
}

func (h *RouterExpertHandler) Init(ctx context.Context, scheme *entity.CollaborationScheme, pool *entity.AgentPool, rt *CollaborationRuntime) error {
	h.scheme = scheme
	h.pool = pool
	h.rt = rt
	return nil
}

func (h *RouterExpertHandler) Name() string {
	return "router_expert"
}

// Execute 路由专家执行逻辑：
// 1. 接收用户输入
// 2. LLM 分析输入，决定由哪个 Agent 处理
// 3. 将任务分配给选定的 Agent
// 4. 返回 Agent 执行结果
func (h *RouterExpertHandler) Execute(ctx context.Context, runID string, input string, trace TraceEmitter) (*entity.AgentInstance, error) {
	// 步骤1: 分析用户输入，决定路由到哪个 Agent
	selectedAgent := h.selectAgent(input)
	if selectedAgent == nil {
		return nil, fmt.Errorf("no suitable agent found for input")
	}

	// 记录任务分配
	trace.TaskAssigned(runID, selectedAgent.Definition.ID, input)

	// 步骤2: Agent 进入工作状态
	trace.AgentEnterWorker(runID, selectedAgent.Definition.ID, "router_node")

	// 步骤3: 执行任务（这里简化处理，实际会调用 Agent 的 LLM）
	result, err := h.executeAgent(ctx, selectedAgent, input, trace, runID)
	if err != nil {
		trace.Error(runID, selectedAgent.Definition.ID, err.Error())
		trace.AgentExitWorker(runID, selectedAgent.Definition.ID, "router_node", fmt.Sprintf("执行失败: %v", err))
		return nil, err
	}

	// 步骤4: Agent 退出工作状态
	trace.AgentExitWorker(runID, selectedAgent.Definition.ID, "router_node", "执行完成")
	selectedAgent.Output = result

	return selectedAgent, nil
}

// selectAgent 根据输入选择最合适的 Agent
func (h *RouterExpertHandler) selectAgent(input string) *entity.AgentInstance {
	// 简化版：基于关键词路由；同一输入可能命中多类关键词时按固定优先级判定（与 BindAgents 顺序无关）
	inputLower := toLower(input)

	bound := make(map[string]bool, len(h.scheme.BindAgents))
	for _, binding := range h.scheme.BindAgents {
		bound[binding.AgentID] = true
	}

	// 优先匹配更「专精」的角色，避免例如「编写代码实现功能」被「功能」先路由到 PM
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
		if !bound[route.id] {
			continue
		}
		def := h.findAgentDef(route.id)
		if def == nil || !def.Enabled {
			continue
		}
		if containsAny(inputLower, route.keywords) {
			return newAgentInstance(def)
		}
	}

	// 默认返回第一个启用的 Agent
	for _, binding := range h.scheme.BindAgents {
		def := h.findAgentDef(binding.AgentID)
		if def != nil && def.Enabled {
			return newAgentInstance(def)
		}
	}

	return nil
}

// findAgentDef 根据 ID 查找 Agent 定义
func (h *RouterExpertHandler) findAgentDef(agentID string) *entity.AgentDefinition {
	for _, agent := range h.pool.Agents {
		if agent.ID == agentID {
			return agent
		}
	}
	return nil
}

// executeAgent 执行 Agent 任务
func (h *RouterExpertHandler) executeAgent(ctx context.Context, agent *entity.AgentInstance, input string, trace TraceEmitter, runID string) (string, error) {
	trace.Thinking(runID, agent.Definition.ID, "分析任务...")

	var cfg *entity.SchemeConfig
	if h.scheme != nil {
		cfg = h.scheme.Config
	}
	return RunAgentHarness(ctx, h.rt, runID, agent.Definition, input, nil, trace, cfg)
}

// toLower 字符串转小写
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

// containsAny 检查字符串是否包含列表中任意元素
func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

// newAgentInstance 根据定义创建新的 Agent 实例
func newAgentInstance(def *entity.AgentDefinition) *entity.AgentInstance {
	return &entity.AgentInstance{
		Definition: def,
		State:      entity.AgentStateIdle,
		History:    make([]entity.Turn, 0),
	}
}
