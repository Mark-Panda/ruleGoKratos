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

// Execute 仅保留兼容层职责；真实执行已迁移到 runtime。
func (h *SupervisionHandler) Execute(ctx context.Context, runID string, input string, trace TraceEmitter) (*entity.AgentInstance, error) {
	return nil, fmt.Errorf("%w: %s", ErrLegacyExecuteDeprecated, h.Name())
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
