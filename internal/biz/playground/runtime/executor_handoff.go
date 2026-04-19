package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ruleGoKratos/internal/biz/entity"
)

// HandoffStepExecutor 执行最小结构化交接决策。
type HandoffStepExecutor struct {
	runner AgentRunner
}

// NewHandoffStepExecutor 创建 HandoffStepExecutor。
func NewHandoffStepExecutor(runner AgentRunner) *HandoffStepExecutor {
	return &HandoffStepExecutor{runner: runner}
}

// Execute 产出 next_agent / handoff_reason / payload_summary / stop_or_continue。
func (e *HandoffStepExecutor) Execute(
	ctx context.Context,
	execCtx *executionContext,
	step *entity.PlanStep,
) (*entity.RuntimeArtifact, error) {
	if e == nil || e.runner == nil {
		return nil, fmt.Errorf("handoff runner is nil")
	}
	if step == nil {
		return nil, fmt.Errorf("handoff step is nil")
	}

	source, err := loadArtifactForRefs(ctx, execCtx, step.InputRefs)
	if err != nil {
		return nil, err
	}

	currentAgent := configString(step.Config, "current_agent")
	if currentAgent == "" {
		currentAgent = strings.TrimSpace(step.AgentBinding)
	}
	payloadSummary := strings.TrimSpace(artifactText(source))
	if payloadSummary == "" {
		payloadSummary = strings.TrimSpace(execCtx.userInput)
	}

	agentDef, err := resolveAgentByID(execCtx, firstNonEmpty(strings.TrimSpace(step.AgentBinding), currentAgent), step.StepID)
	if err != nil {
		return nil, err
	}
	userInput, err := buildHandoffUserInput(ctx, execCtx, step, agentDef.ID, currentAgent, payloadSummary)
	if err != nil {
		return nil, err
	}
	output, err := e.runner(ctx, execCtx.runID, agentDef, userInput, execCtx.trace, schemeConfigOf(execCtx.scheme))
	if err != nil {
		return nil, err
	}

	decision := parseHandoffDecision(output)
	nextAgent := firstNonEmpty(decision.NextAgent, configString(step.Config, "next_agent"))
	handoffReason := firstNonEmpty(decision.HandoffReason, configString(step.Config, "handoff_reason"))
	payloadSummary = firstNonEmpty(decision.PayloadSummary, payloadSummary, strings.TrimSpace(output))
	stopOrContinue := normalizeStopOrContinue(firstNonEmpty(decision.StopOrContinue, configString(step.Config, "stop_or_continue")))
	if stopOrContinue == "" {
		if nextAgent == "" {
			stopOrContinue = "stop"
		} else {
			stopOrContinue = "continue"
		}
	}
	if handoffReason == "" {
		if nextAgent != "" {
			handoffReason = fmt.Sprintf("%s 已完成当前阶段，交由 %s 接力", currentAgent, nextAgent)
		} else {
			handoffReason = fmt.Sprintf("%s 已完成当前阶段，流程收口", currentAgent)
		}
	}
	if stopOrContinue == "stop" {
		nextAgent = ""
	}

	if execCtx.trace != nil && nextAgent != "" {
		execCtx.trace.Handoff(execCtx.runID, currentAgent, nextAgent, handoffReason)
	}

	return newArtifact(execCtx.runID, step.StepID, "handoff_payload", payloadSummary, map[string]any{
		"current_agent":    firstNonEmpty(currentAgent, agentDef.ID),
		"next_agent":       nextAgent,
		"handoff_reason":   handoffReason,
		"payload_summary":  payloadSummary,
		"stop_or_continue": stopOrContinue,
	}), nil
}

func configString(config map[string]any, key string) string {
	if config == nil {
		return ""
	}
	value, ok := config[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

type handoffDecision struct {
	NextAgent      string `json:"next_agent"`
	HandoffReason  string `json:"handoff_reason"`
	PayloadSummary string `json:"payload_summary"`
	StopOrContinue string `json:"stop_or_continue"`
}

func buildHandoffUserInput(
	ctx context.Context,
	execCtx *executionContext,
	step *entity.PlanStep,
	decisionAgent string,
	currentAgent string,
	payloadSummary string,
) (string, error) {
	baseInput, err := buildStepUserInput(ctx, execCtx, step)
	if err != nil {
		return "", err
	}

	var prompt strings.Builder
	prompt.WriteString("请作为交接决策代理输出 JSON，字段必须包含 next_agent、handoff_reason、payload_summary、stop_or_continue。")
	if currentAgent != "" {
		prompt.WriteString("\n当前处理代理：")
		prompt.WriteString(currentAgent)
	}
	prompt.WriteString("\n决策代理：")
	prompt.WriteString(decisionAgent)
	prompt.WriteString("\nstop_or_continue 只能为 continue 或 stop。若 stop，请将 next_agent 置空。")
	if strings.TrimSpace(payloadSummary) != "" {
		prompt.WriteString("\n当前可交接摘要：")
		prompt.WriteString(payloadSummary)
	}
	if strings.TrimSpace(baseInput) != "" {
		prompt.WriteString("\n\n输入上下文：\n")
		prompt.WriteString(baseInput)
	}
	return prompt.String(), nil
}

func parseHandoffDecision(output string) handoffDecision {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return handoffDecision{}
	}

	for _, candidate := range []string{trimmed, extractJSONObject(trimmed)} {
		candidate = strings.TrimSpace(strings.Trim(candidate, "`"))
		if candidate == "" {
			continue
		}
		var decision handoffDecision
		if err := json.Unmarshal([]byte(candidate), &decision); err == nil {
			return decision
		}
	}

	decision := handoffDecision{}
	for _, line := range strings.Split(trimmed, "\n") {
		key, value, ok := splitDecisionLine(line)
		if !ok {
			continue
		}
		switch key {
		case "next_agent":
			decision.NextAgent = value
		case "handoff_reason":
			decision.HandoffReason = value
		case "payload_summary":
			decision.PayloadSummary = value
		case "stop_or_continue":
			decision.StopOrContinue = value
		}
	}
	return decision
}

func extractJSONObject(text string) string {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end <= start {
		return ""
	}
	return text[start : end+1]
}

func splitDecisionLine(line string) (key string, value string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", "", false
	}
	for _, sep := range []string{":", "：", "="} {
		parts := strings.SplitN(line, sep, 2)
		if len(parts) != 2 {
			continue
		}
		key = strings.TrimSpace(parts[0])
		value = strings.TrimSpace(parts[1])
		if key == "" || value == "" {
			return "", "", false
		}
		return key, value, true
	}
	return "", "", false
}

func normalizeStopOrContinue(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "continue":
		return "continue"
	case "stop":
		return "stop"
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
