package collaboration

import (
	"context"
	"errors"
	"fmt"
	"ruleGoKratos/internal/biz/entity"
)

// ErrLegacyExecuteDeprecated 标记旧协作 handler 的 Execute 已废弃。
var ErrLegacyExecuteDeprecated = errors.New("legacy collaboration handler execute is deprecated; use runtime plan execution")

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

// Execute 仅保留兼容层职责；真实执行已迁移到 runtime。
func (h *RouterExpertHandler) Execute(ctx context.Context, runID string, input string, trace TraceEmitter) (*entity.AgentInstance, error) {
	return nil, fmt.Errorf("%w: %s", ErrLegacyExecuteDeprecated, h.Name())
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
