package collaboration

import (
	"context"
	"fmt"
	"strings"

	"ruleGoKratos/internal/biz/entity"
)

// PlanExecHandler 规划执行模式处理器
// 主 Agent 负责任务规划，然后依次分配给子 Agent 执行
type PlanExecHandler struct {
	scheme *entity.CollaborationScheme
	pool   *entity.AgentPool
	rt     *CollaborationRuntime
}

func NewPlanExecHandler() *PlanExecHandler {
	return &PlanExecHandler{}
}

func (h *PlanExecHandler) Init(ctx context.Context, scheme *entity.CollaborationScheme, pool *entity.AgentPool, rt *CollaborationRuntime) error {
	h.scheme = scheme
	h.pool = pool
	h.rt = rt
	return nil
}

func (h *PlanExecHandler) Name() string {
	return "plan_exec"
}

// Execute 规划执行执行逻辑：
// 1. 规划 Agent 分析任务，拆解为子任务
// 2. 按执行顺序依次分配给各个 Agent
// 3. 前一个 Agent 的输出作为下一个 Agent 的输入
// 4. 返回最终执行结果
func (h *PlanExecHandler) Execute(ctx context.Context, runID string, input string, trace TraceEmitter) (*entity.AgentInstance, error) {
	// 步骤1: 找到规划 Agent（可用方案配置覆盖默认 planner ID）
	planner := h.resolvePlannerAgent()
	if planner == nil {
		return nil, fmt.Errorf("planner agent not found")
	}

	trace.TaskAssigned(runID, planner.Definition.ID, "任务规划: "+input)
	trace.AgentEnterWorker(runID, planner.Definition.ID, "plan_node")

	// 步骤2: 规划 Agent 拆解任务（简化实现）
	trace.Thinking(runID, planner.Definition.ID, "分析并拆解任务...")
	subTasks := h.planSubTasks(input, trace, runID, planner)
	h.emitPlanSummary(ctx, trace, runID, planner.Definition.ID, subTasks)

	trace.AgentExitWorker(runID, planner.Definition.ID, "plan_node", "规划完成")

	// 步骤3: 按顺序执行子任务
	var lastOutput = input
	for _, task := range subTasks {
		trace.TaskAssigned(runID, task.agentID, task.desc)
		trace.AgentEnterWorker(runID, task.agentID, task.nodeID)

		output, err := h.executeTask(ctx, task, lastOutput, trace, runID)
		if err != nil {
			trace.Error(runID, task.agentID, err.Error())
			trace.AgentExitWorker(runID, task.agentID, task.nodeID, fmt.Sprintf("执行失败: %v", err))
			return nil, err
		}

		lastOutput = output
		h.emitStepOutput(ctx, trace, runID, task, output)
		trace.AgentExitWorker(runID, task.agentID, task.nodeID, "执行完成")
	}

	// 返回最后一个执行的 Agent 作为结果
	lastAgent := &entity.AgentInstance{
		Definition: h.findAgentDef(subTasks[len(subTasks)-1].agentID),
		State:     entity.AgentStateDone,
		Output:    lastOutput,
	}

	return lastAgent, nil
}

type subTask struct {
	agentID string
	nodeID  string
	desc    string
}

// resolvePlannerAgent 解析规划师 Agent（优先 PlanExecConfig.PlannerAgent）
func (h *PlanExecHandler) resolvePlannerAgent() *entity.AgentInstance {
	plannerID := "planner"
	if h.scheme.Config != nil && h.scheme.Config.ModeConfig != nil && h.scheme.Config.ModeConfig.PlanExecConfig != nil {
		if id := h.scheme.Config.ModeConfig.PlanExecConfig.PlannerAgent; id != "" {
			plannerID = id
		}
	}
	return h.findAgentByRole(plannerID, "规划师")
}

// planSubTasks 规划子任务
func (h *PlanExecHandler) planSubTasks(input string, trace TraceEmitter, runID string, planner *entity.AgentInstance) []subTask {
	tasks := make([]subTask, 0)

	if h.scheme.Config != nil && h.scheme.Config.ModeConfig != nil && h.scheme.Config.ModeConfig.PlanExecConfig != nil {
		order := h.scheme.Config.ModeConfig.PlanExecConfig.ExecutionOrder
		for i, agentID := range order {
			def := h.findAgentDef(agentID)
			if def == nil || !def.Enabled {
				continue
			}
			tasks = append(tasks, subTask{
				agentID: agentID,
				nodeID:  fmt.Sprintf("exec_%d_%s", i, agentID),
				desc:    fmt.Sprintf("%s 执行步骤", def.Name),
			})
		}
		if len(tasks) > 0 {
			return tasks
		}
	}

	// 默认流程：设计 -> 产品规划 -> 开发
	defaultFlow := []struct {
		agentID string
		nodeID  string
		desc    string
	}{
		{"designer", "design_node", "界面设计"},
		{"pm", "pm_node", "需求分析"},
		{"engineer", "dev_node", "代码开发"},
	}

	for _, step := range defaultFlow {
		def := h.findAgentDef(step.agentID)
		if def != nil && def.Enabled {
			tasks = append(tasks, subTask{
				agentID: step.agentID,
				nodeID:  step.nodeID,
				desc:    step.desc,
			})
		}
	}

	return tasks
}

// findAgentByRole 根据角色查找 Agent
func (h *PlanExecHandler) findAgentByRole(agentID, roleName string) *entity.AgentInstance {
	def := h.findAgentDef(agentID)
	if def != nil && def.Enabled {
		return newAgentInstance(def)
	}

	// 按角色名称查找
	for _, agent := range h.pool.Agents {
		if agent.Enabled && (agent.ID == agentID || agent.Name == roleName) {
			return newAgentInstance(agent)
		}
	}

	return nil
}

// findAgentDef 根据 ID 查找 Agent 定义
func (h *PlanExecHandler) findAgentDef(agentID string) *entity.AgentDefinition {
	for _, agent := range h.pool.Agents {
		if agent.ID == agentID {
			return agent
		}
	}
	return nil
}

// executeTask 执行单个任务
func (h *PlanExecHandler) executeTask(ctx context.Context, task subTask, input string, trace TraceEmitter, runID string) (string, error) {
	def := h.findAgentDef(task.agentID)
	if def == nil {
		return "", fmt.Errorf("agent not found: %s", task.agentID)
	}

	trace.Thinking(runID, task.agentID, fmt.Sprintf("执行任务: %s", task.desc))

	userPayload := fmt.Sprintf("%s\n\n上游产出 / 上下文：\n%s", task.desc, input)
	var cfg *entity.SchemeConfig
	if h.scheme != nil {
		cfg = h.scheme.Config
	}
	return RunAgentHarness(ctx, h.rt, runID, def, userPayload, nil, trace, cfg)
}

func truncateStepPreview(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + fmt.Sprintf("\n…(truncated，全文 %d 字符)", len(s))
}

func (h *PlanExecHandler) emitPlanSummary(ctx context.Context, trace TraceEmitter, runID, plannerAgentID string, tasks []subTask) {
	if trace == nil || len(tasks) == 0 {
		return
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("已确定执行顺序，共 %d 步：\n", len(tasks)))
	for i, t := range tasks {
		b.WriteString(fmt.Sprintf("%d. [%s] %s · %s\n", i+1, t.agentID, t.desc, t.nodeID))
	}
	ev := entity.NewTraceEvent(runID, entity.TraceEventPlanSummary, plannerAgentID, "plan_node", "", strings.TrimSpace(b.String()))
	trace.EmitEvent(ctx, ev)
}

func (h *PlanExecHandler) emitStepOutput(ctx context.Context, trace TraceEmitter, runID string, task subTask, output string) {
	if trace == nil {
		return
	}
	body := truncateStepPreview(strings.TrimSpace(output), 16000)
	ev := entity.NewTraceEvent(runID, entity.TraceEventStepOutput, task.agentID, task.nodeID, task.desc, body)
	trace.EmitEvent(ctx, ev)
}
