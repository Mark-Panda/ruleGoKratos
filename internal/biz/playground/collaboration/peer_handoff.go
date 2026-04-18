package collaboration

import (
	"context"
	"fmt"
	"strings"

	"ruleGoKratos/internal/biz/entity"
)

// PeerHandoffHandler 同伴交接模式处理器
// 多个 Agent 形成 Mesh 网络，自主协商任务交接
type PeerHandoffHandler struct {
	scheme *entity.CollaborationScheme
	pool   *entity.AgentPool
	rt     *CollaborationRuntime
}

func NewPeerHandoffHandler() *PeerHandoffHandler {
	return &PeerHandoffHandler{}
}

func (h *PeerHandoffHandler) Init(ctx context.Context, scheme *entity.CollaborationScheme, pool *entity.AgentPool, rt *CollaborationRuntime) error {
	h.scheme = scheme
	h.pool = pool
	h.rt = rt
	return nil
}

func (h *PeerHandoffHandler) Name() string {
	return "peer_handoff"
}

// findAgentDef 根据 ID 查找 Agent 定义
func (h *PeerHandoffHandler) findAgentDef(agentID string) *entity.AgentDefinition {
	for _, agent := range h.pool.Agents {
		if agent.ID == agentID {
			return agent
		}
	}
	return nil
}

// Execute 同伴交接执行逻辑：
// 1. 确定入口 Agent 作为起点
// 2. Agent 完成后，根据任务状态自主判断交接给下一个 Agent
// 3. 通过 Peer Mesh 协议进行任务传递
// 4. 直到所有任务完成
func (h *PeerHandoffHandler) Execute(ctx context.Context, runID string, input string, trace TraceEmitter) (*entity.AgentInstance, error) {
	// 步骤1: 确定入口 Agent
	entryAgent := h.findEntryAgent()
	if entryAgent == nil {
		return nil, fmt.Errorf("entry agent not found")
	}

	currentAgent := entryAgent
	var lastOutput = input
	var taskDesc = input

	// 步骤2: 循环执行直到完成
	maxHandoffs := 10 // 防止无限循环
	for i := 0; i < maxHandoffs; i++ {
		trace.TaskAssigned(runID, currentAgent.Definition.ID, taskDesc)
		trace.AgentEnterWorker(runID, currentAgent.Definition.ID, fmt.Sprintf("peer_node_%d", i))

		trace.Thinking(runID, currentAgent.Definition.ID, "处理任务并决定是否交接...")

		// 模拟 Agent 执行
		output, err := h.executeAgent(ctx, currentAgent, taskDesc, lastOutput, trace, runID)
		if err != nil {
			trace.Error(runID, currentAgent.Definition.ID, err.Error())
			trace.AgentExitWorker(runID, currentAgent.Definition.ID, fmt.Sprintf("peer_node_%d", i), fmt.Sprintf("执行失败: %v", err))
			return nil, err
		}

		trace.AgentExitWorker(runID, currentAgent.Definition.ID, fmt.Sprintf("peer_node_%d", i), "执行完成")

		// 步骤3: 判断是否需要交接（round 用于在可选同伴间轮询，避免错误地固定到同一目标）
		nextAgent := h.findNextAgent(currentAgent, i)
		if nextAgent == nil {
			// 无需交接，任务完成
			currentAgent.Output = output
			break
		}

		// 步骤4: 执行交接
		trace.Handoff(runID, currentAgent.Definition.ID, nextAgent.Definition.ID, "任务交接")
		lastOutput = output
		taskDesc = h.generateNextTaskDesc(currentAgent, nextAgent, output)
		currentAgent = nextAgent
	}

	return currentAgent, nil
}

// findEntryAgent 找到入口 Agent
func (h *PeerHandoffHandler) findEntryAgent() *entity.AgentInstance {
	// 优先使用配置中的入口 Agent
	if h.scheme.Config != nil && h.scheme.Config.ModeConfig != nil {
		if config := h.scheme.Config.ModeConfig.PeerHandoffConfig; config != nil {
			def := h.findAgentDef(config.EntryAgent)
			if def != nil && def.Enabled {
				return newAgentInstance(def)
			}
		}
	}

	// 默认返回第一个启用的非监督者 Agent
	for _, agent := range h.pool.Agents {
		if agent.Enabled && agent.ID != "supervisor" {
			return newAgentInstance(agent)
		}
	}

	return nil
}

// findNextAgent 找到下一个应该接收任务的 Agent。
// round 为循环轮次：当前 Agent 已从候选列表排除，因此在 available 上做 round-robin。
func (h *PeerHandoffHandler) findNextAgent(current *entity.AgentInstance, round int) *entity.AgentInstance {
	// 构建可用的 Agent 列表（排除当前 Agent）
	available := make([]*entity.AgentDefinition, 0)
	var mesh []string
	if h.scheme.Config != nil && h.scheme.Config.ModeConfig != nil && h.scheme.Config.ModeConfig.PeerHandoffConfig != nil {
		mesh = h.scheme.Config.ModeConfig.PeerHandoffConfig.MeshAgents
	}

	for _, agent := range h.pool.Agents {
		if !agent.Enabled || agent.ID == current.Definition.ID || agent.ID == "supervisor" {
			continue
		}
		if len(mesh) > 0 && !stringSliceContains(mesh, agent.ID) {
			continue
		}
		available = append(available, agent)
	}

	if len(available) == 0 {
		return nil
	}

	// 简化实现：按轮次在同伴间轮询；后续可替换为基于 LLM 的交接决策
	nextDef := available[round%len(available)]
	return newAgentInstance(nextDef)
}

// executeAgent 执行 Agent 任务
func (h *PeerHandoffHandler) executeAgent(ctx context.Context, agent *entity.AgentInstance, taskDesc, input string, trace TraceEmitter, runID string) (string, error) {
	userPayload := taskDesc
	if strings.TrimSpace(input) != "" && input != taskDesc {
		userPayload = taskDesc + "\n\n上下文：\n" + input
	}
	var cfg *entity.SchemeConfig
	if h.scheme != nil {
		cfg = h.scheme.Config
	}
	return RunAgentHarness(ctx, h.rt, runID, agent.Definition, userPayload, nil, trace, cfg)
}

// generateNextTaskDesc 生成下一个任务的描述
func (h *PeerHandoffHandler) generateNextTaskDesc(current, next *entity.AgentInstance, output string) string {
	return fmt.Sprintf("根据 %s 的输出继续处理: %s", current.Definition.Name, truncate(output, 100))
}
