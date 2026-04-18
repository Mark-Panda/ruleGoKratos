package collaboration

import (
	"context"
	"fmt"
	"ruleGoKratos/internal/biz/entity"
)

// SupervisionHandler 动态监督模式处理器
// 监督者 Agent 并行监控所有 Worker Agent 的执行状态，发现问题时及时干预
type SupervisionHandler struct {
	scheme *entity.CollaborationScheme
	pool   *entity.AgentPool
	rt     *CollaborationRuntime
}

func NewSupervisionHandler() *SupervisionHandler {
	return &SupervisionHandler{}
}

func (h *SupervisionHandler) Init(ctx context.Context, scheme *entity.CollaborationScheme, pool *entity.AgentPool, rt *CollaborationRuntime) error {
	h.scheme = scheme
	h.pool = pool
	h.rt = rt
	return nil
}

func (h *SupervisionHandler) Name() string {
	return "supervision"
}

// Execute 动态监督执行逻辑：
// 1. 监督 Agent 分析任务并制定执行计划
// 2. 并行启动多个 Worker Agent 执行子任务
// 3. 监督 Agent 实时监控进度，发现问题及时干预
// 4. 汇总各 Worker 结果，返回最终输出
func (h *SupervisionHandler) Execute(ctx context.Context, runID string, input string, trace TraceEmitter) (*entity.AgentInstance, error) {
	// 步骤1: 找到监督 Agent（可用 SupervisionConfig.SupervisorAgent 覆盖）
	supervisorID := "supervisor"
	if h.scheme.Config != nil && h.scheme.Config.ModeConfig != nil && h.scheme.Config.ModeConfig.SupervisionConfig != nil {
		if id := h.scheme.Config.ModeConfig.SupervisionConfig.SupervisorAgent; id != "" {
			supervisorID = id
		}
	}
	supervisor := h.findAgentByRole(supervisorID, "监督者")
	if supervisor == nil {
		return nil, fmt.Errorf("supervisor agent not found")
	}

	trace.TaskAssigned(runID, supervisor.Definition.ID, "任务监督: "+input)
	trace.AgentEnterWorker(runID, supervisor.Definition.ID, "supervisor_node")

	// 步骤2: 监督 Agent 分析任务并分配给 Workers
	trace.Thinking(runID, supervisor.Definition.ID, "分析任务并分配子任务...")
	workers := h.allocateWorkers(input, trace, runID, supervisor)
	if len(workers) == 0 {
		trace.AgentExitWorker(runID, supervisor.Definition.ID, "supervisor_node", "无可用 Worker")
		return nil, fmt.Errorf("no available workers")
	}

	// 记录委派事件
	for _, w := range workers {
		trace.WorkerDelegated(runID, supervisor.Definition.ID, w.agentID, w.reason)
	}

	// 步骤3: 并行执行子任务（简化：串行执行）
	var results []string
	for i, w := range workers {
		trace.AgentEnterWorker(runID, w.agentID, w.nodeID)
		trace.TaskAssigned(runID, w.agentID, w.taskDesc)

		// 模拟执行
		result, err := h.executeWorker(ctx, w, input, trace, runID)
		if err != nil {
			trace.Error(runID, w.agentID, err.Error())
			trace.AgentExitWorker(runID, w.agentID, w.nodeID, fmt.Sprintf("执行失败: %v", err))
			// 监督者干预
			trace.EmitEvent(ctx, entity.NewTraceEvent(
				runID,
				entity.TraceEventSupervisorIntervene,
				supervisor.Definition.ID,
				w.nodeID,
				w.taskDesc,
				fmt.Sprintf("检测到 Worker %s 执行失败，尝试恢复...", w.agentID),
			))
			continue
		}

		trace.AgentExitWorker(runID, w.agentID, w.nodeID, "执行完成")
		results = append(results, result)

		// 监督者检查
		trace.EmitEvent(ctx, entity.NewTraceEvent(
			runID,
			entity.TraceEventSupervisorCheck,
			supervisor.Definition.ID,
			w.nodeID,
			w.taskDesc,
			fmt.Sprintf("检查 Worker %d/%d 执行结果: %s", i+1, len(workers), truncate(result, 100)),
		))
	}

	trace.AgentExitWorker(runID, supervisor.Definition.ID, "supervisor_node", "监督完成")

	// 返回监督者作为结果持有者
	supervisor.Output = fmt.Sprintf("完成了 %d 个子任务", len(results))
	return supervisor, nil
}

type workerTask struct {
	agentID  string
	nodeID   string
	taskDesc string
	reason   string
}

// allocateWorkers 分配 Worker 任务
func (h *SupervisionHandler) allocateWorkers(input string, trace TraceEmitter, runID string, supervisor *entity.AgentInstance) []workerTask {
	tasks := make([]workerTask, 0)

	if h.scheme.Config != nil && h.scheme.Config.ModeConfig != nil && h.scheme.Config.ModeConfig.SupervisionConfig != nil {
		cfg := h.scheme.Config.ModeConfig.SupervisionConfig
		if len(cfg.WorkerAgents) > 0 {
			for _, agentID := range cfg.WorkerAgents {
				def := h.findAgentDef(agentID)
				if def == nil || !def.Enabled || agentID == supervisor.Definition.ID {
					continue
				}
				tasks = append(tasks, workerTask{
					agentID:  def.ID,
					nodeID:   fmt.Sprintf("worker_%s", def.ID),
					taskDesc: fmt.Sprintf("执行 %s 相关任务", def.Name),
					reason:   fmt.Sprintf("监督配置指定的 Worker：%s", def.Name),
				})
			}
			if len(tasks) > 0 {
				return tasks
			}
		}
	}

	// 根据绑定 Agent 分配任务
	for _, binding := range h.scheme.BindAgents {
		def := h.findAgentDef(binding.AgentID)
		if def == nil || !def.Enabled || def.ID == "supervisor" {
			continue
		}

		tasks = append(tasks, workerTask{
			agentID:  def.ID,
			nodeID:   fmt.Sprintf("worker_%s", def.ID),
			taskDesc: fmt.Sprintf("执行 %s 相关任务", def.Name),
			reason:   fmt.Sprintf("任务需要 %s 能力", def.Name),
		})
	}

	return tasks
}

// findAgentByRole 根据角色查找 Agent
func (h *SupervisionHandler) findAgentByRole(agentID, roleName string) *entity.AgentInstance {
	def := h.findAgentDef(agentID)
	if def != nil && def.Enabled {
		return newAgentInstance(def)
	}

	for _, agent := range h.pool.Agents {
		if agent.Enabled && (agent.ID == agentID || agent.Name == roleName) {
			return newAgentInstance(agent)
		}
	}

	return nil
}

// findAgentDef 根据 ID 查找 Agent 定义
func (h *SupervisionHandler) findAgentDef(agentID string) *entity.AgentDefinition {
	for _, agent := range h.pool.Agents {
		if agent.ID == agentID {
			return agent
		}
	}
	return nil
}

// executeWorker 执行 Worker 任务
func (h *SupervisionHandler) executeWorker(ctx context.Context, w workerTask, input string, trace TraceEmitter, runID string) (string, error) {
	def := h.findAgentDef(w.agentID)
	if def == nil {
		return "", fmt.Errorf("agent not found: %s", w.agentID)
	}

	trace.Thinking(runID, w.agentID, fmt.Sprintf("处理任务: %s", w.taskDesc))

	userPayload := fmt.Sprintf("%s\n\n监督上下文（原始需求）：\n%s", w.taskDesc, input)
	var cfg *entity.SchemeConfig
	if h.scheme != nil {
		cfg = h.scheme.Config
	}
	return RunAgentHarness(ctx, h.rt, runID, def, userPayload, nil, trace, cfg)
}
