package planbuilder

import (
	"fmt"

	"ruleGoKratos/internal/biz/entity"

	"github.com/google/uuid"
)

// PeerHandoffBuilder 将 peer_handoff 模式编译为链式 agent / handoff 计划。
type PeerHandoffBuilder struct{}

// NewPeerHandoffBuilder 创建 peer_handoff 的计划构建器。
func NewPeerHandoffBuilder() *PeerHandoffBuilder {
	return &PeerHandoffBuilder{}
}

// Mode 返回当前 Builder 对应的协作模式。
func (b *PeerHandoffBuilder) Mode() entity.CollaborationMode {
	return entity.ModePeerHandoff
}

// Build 生成 entry agent -> handoff -> ... -> finalize 的链式计划。
func (b *PeerHandoffBuilder) Build(scheme *entity.CollaborationScheme, _ string) (*entity.ExecutionPlan, error) {
	if scheme == nil {
		return nil, fmt.Errorf("scheme is nil")
	}

	sequence, err := resolvePeerHandoffSequence(scheme)
	if err != nil {
		return nil, err
	}

	steps := make([]*entity.PlanStep, 0, len(sequence)*2+1)
	prevDependency := ""
	prevInputRef := ""
	for i, agentID := range sequence {
		agentStepID := fmt.Sprintf("agent_%d", i+1)
		agentOutputRef := fmt.Sprintf("agent_%d_output", i+1)

		agentStep := &entity.PlanStep{
			StepID:       agentStepID,
			Kind:         entity.StepKindAgent,
			Name:         agentStepID,
			AgentBinding: agentID,
			OutputRef:    agentOutputRef,
		}
		if prevDependency != "" {
			agentStep.DependsOn = []string{prevDependency}
			agentStep.InputRefs = []string{prevInputRef}
		}
		steps = append(steps, agentStep)

		handoffStepID := fmt.Sprintf("handoff_%d", i+1)
		handoffOutputRef := fmt.Sprintf("handoff_%d_payload", i+1)
		nextAgent := ""
		stopOrContinue := "stop"
		handoffReason := "当前链路已完成，准备收口"
		if i+1 < len(sequence) {
			nextAgent = sequence[i+1]
			stopOrContinue = "continue"
			handoffReason = fmt.Sprintf("%s 阶段完成，交由 %s 继续处理", agentID, nextAgent)
		}
		steps = append(steps, &entity.PlanStep{
			StepID:    handoffStepID,
			Kind:      entity.StepKindHandoff,
			Name:      handoffStepID,
			DependsOn: []string{agentStepID},
			InputRefs: []string{agentOutputRef},
			OutputRef: handoffOutputRef,
			Config: map[string]any{
				"current_agent":    agentID,
				"next_agent":       nextAgent,
				"handoff_reason":   handoffReason,
				"stop_or_continue": stopOrContinue,
			},
		})

		prevDependency = handoffStepID
		prevInputRef = handoffOutputRef
	}

	steps = append(steps, &entity.PlanStep{
		StepID:    "finalize",
		Kind:      entity.StepKindFinalize,
		Name:      "finalize",
		DependsOn: []string{prevDependency},
		InputRefs: []string{prevInputRef},
		OutputRef: "final_output",
	})

	return &entity.ExecutionPlan{
		PlanID:       uuid.NewString(),
		PlanVersion:  1,
		SourceMode:   string(entity.ModePeerHandoff),
		EntryStepIDs: []string{"agent_1"},
		Steps:        steps,
		Metadata: map[string]any{
			"schemeId": scheme.ID,
		},
	}, nil
}

func resolvePeerHandoffSequence(scheme *entity.CollaborationScheme) ([]string, error) {
	if scheme == nil {
		return nil, fmt.Errorf("scheme is nil")
	}

	ordered := make([]string, 0, len(scheme.BindAgents))
	seen := make(map[string]struct{}, len(scheme.BindAgents))
	appendAgent := func(agentID string) {
		if agentID == "" {
			return
		}
		if _, ok := seen[agentID]; ok {
			return
		}
		seen[agentID] = struct{}{}
		ordered = append(ordered, agentID)
	}

	entryAgent := ""
	meshAgents := make([]string, 0, len(scheme.BindAgents))
	if scheme.Config != nil && scheme.Config.ModeConfig != nil && scheme.Config.ModeConfig.PeerHandoffConfig != nil {
		entryAgent = scheme.Config.ModeConfig.PeerHandoffConfig.EntryAgent
		meshAgents = append(meshAgents, scheme.Config.ModeConfig.PeerHandoffConfig.MeshAgents...)
	}
	if entryAgent == "" {
		for _, binding := range scheme.BindAgents {
			if binding != nil && binding.AgentID != "" {
				entryAgent = binding.AgentID
				break
			}
		}
	}
	appendAgent(entryAgent)

	if len(meshAgents) == 0 {
		for _, binding := range scheme.BindAgents {
			if binding != nil {
				meshAgents = append(meshAgents, binding.AgentID)
			}
		}
	}
	for _, agentID := range meshAgents {
		appendAgent(agentID)
	}
	if len(ordered) == 0 {
		return nil, fmt.Errorf("peer_handoff requires at least one agent")
	}
	return ordered, nil
}
