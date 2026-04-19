package runtime

import (
	"context"
	"fmt"
	"strings"

	"ruleGoKratos/internal/biz/entity"
)

// AgentStepExecutor 执行单个 Agent 步骤。
type AgentStepExecutor struct {
	runner AgentRunner
}

// NewAgentStepExecutor 创建 AgentStepExecutor。
func NewAgentStepExecutor(runner AgentRunner) *AgentStepExecutor {
	return &AgentStepExecutor{runner: runner}
}

// Execute 执行单个 Agent，并返回结构化 worker_result。
func (e *AgentStepExecutor) Execute(
	ctx context.Context,
	execCtx *executionContext,
	step *entity.PlanStep,
) (*entity.RuntimeArtifact, error) {
	if e == nil || e.runner == nil {
		return nil, fmt.Errorf("agent runner is nil")
	}

	agentDef, err := resolveAgentDefinition(execCtx, step)
	if err != nil {
		return nil, err
	}

	if execCtx.trace != nil {
		execCtx.trace.TaskAssigned(execCtx.runID, agentDef.ID, execCtx.userInput)
		execCtx.trace.AgentEnterWorker(execCtx.runID, agentDef.ID, step.StepID)
		execCtx.trace.Thinking(execCtx.runID, agentDef.ID, "分析任务...")
	}

	userInput, err := buildStepUserInput(ctx, execCtx, step)
	if err != nil {
		return nil, err
	}
	output, err := e.runner(ctx, execCtx.runID, agentDef, userInput, execCtx.trace, schemeConfigOf(execCtx.scheme))
	if err != nil {
		if execCtx.trace != nil {
			execCtx.trace.Error(execCtx.runID, agentDef.ID, err.Error())
			execCtx.trace.AgentExitWorker(execCtx.runID, agentDef.ID, step.StepID, fmt.Sprintf("执行失败: %v", err))
		}
		return nil, err
	}

	if execCtx.trace != nil {
		execCtx.trace.AgentExitWorker(execCtx.runID, agentDef.ID, step.StepID, "执行完成")
	}

	return newArtifact(execCtx.runID, step.StepID, "worker_result", output, map[string]any{
		"agent_id": agentDef.ID,
		"output":   output,
	}), nil
}

func resolveAgentDefinition(execCtx *executionContext, step *entity.PlanStep) (*entity.AgentDefinition, error) {
	if execCtx == nil || step == nil {
		return nil, fmt.Errorf("agent step context is incomplete")
	}

	agentID, err := resolveAgentIDFromInputs(execCtx, step)
	if err != nil {
		return nil, err
	}
	if agentID == "" {
		agentID = step.AgentBinding
	}
	if agentID == "" {
		return nil, fmt.Errorf("agent step %s missing selected agent", step.StepID)
	}
	return resolveAgentByID(execCtx, agentID, step.StepID)
}

func resolveAgentIDFromInputs(execCtx *executionContext, step *entity.PlanStep) (string, error) {
	if execCtx == nil || step == nil {
		return "", nil
	}
	selectedAgent := ""
	for _, inputRef := range step.InputRefs {
		artifact, err := loadArtifactForRefs(context.Background(), execCtx, []string{inputRef})
		if err != nil {
			return "", err
		}
		if artifact == nil {
			continue
		}
		if nextAgent, ok := artifact.Payload["next_agent"].(string); ok && strings.TrimSpace(nextAgent) != "" {
			return nextAgent, nil
		}
		if selectedAgent == "" {
			if routedAgent, ok := artifact.Payload["selected_agent"].(string); ok && strings.TrimSpace(routedAgent) != "" {
				selectedAgent = routedAgent
			}
		}
	}
	return selectedAgent, nil
}

func resolveAgentByID(execCtx *executionContext, agentID, stepID string) (*entity.AgentDefinition, error) {
	if execCtx == nil {
		return nil, fmt.Errorf("agent step context is incomplete")
	}
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return nil, fmt.Errorf("agent step %s missing selected agent", stepID)
	}

	def := findEnabledAgent(execCtx.pool, agentID)
	if def == nil {
		return nil, fmt.Errorf("agent %s is not available in pool", agentID)
	}
	return def, nil
}

func schemeConfigOf(scheme *entity.CollaborationScheme) *entity.SchemeConfig {
	if scheme == nil {
		return nil
	}
	return scheme.Config
}

func buildStepUserInput(ctx context.Context, execCtx *executionContext, step *entity.PlanStep) (string, error) {
	if execCtx == nil {
		return "", fmt.Errorf("execution context is nil")
	}

	baseInput := strings.TrimSpace(execCtx.userInput)
	if step == nil || len(step.InputRefs) == 0 {
		return baseInput, nil
	}

	snippets, err := collectArtifactSnippets(ctx, execCtx, step.InputRefs)
	if err != nil {
		return "", err
	}
	if len(snippets) == 0 {
		return baseInput, nil
	}
	if baseInput == "" {
		return strings.Join(snippets, "\n\n"), nil
	}
	return baseInput + "\n\n上游上下文：\n" + strings.Join(snippets, "\n\n"), nil
}

func collectArtifactSnippets(
	ctx context.Context,
	execCtx *executionContext,
	refs []string,
) ([]string, error) {
	snippets := make([]string, 0, len(refs))
	for _, ref := range refs {
		artifact, err := loadArtifactForRefs(ctx, execCtx, []string{ref})
		if err != nil {
			return nil, err
		}
		snippet := artifactSnippet(artifact)
		if snippet == "" {
			continue
		}
		snippets = append(snippets, snippet)
	}
	return snippets, nil
}

func artifactSnippet(artifact *entity.RuntimeArtifact) string {
	if artifact == nil {
		return ""
	}
	// route_result 只用于决策选人，不应污染后续 Agent 的工作输入。
	if artifact.Type == "route_result" {
		return ""
	}
	text := strings.TrimSpace(artifactText(artifact))
	if text == "" {
		return ""
	}
	return fmt.Sprintf("[%s] %s", artifact.Type, text)
}
