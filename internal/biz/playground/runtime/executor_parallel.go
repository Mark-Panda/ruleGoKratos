package runtime

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"ruleGoKratos/internal/biz/entity"
)

// ParallelStepExecutor 执行最小 fan-out / fan-in 并行步骤。
type ParallelStepExecutor struct {
	runner AgentRunner
}

// NewParallelStepExecutor 创建 ParallelStepExecutor。
func NewParallelStepExecutor(runner AgentRunner) *ParallelStepExecutor {
	return &ParallelStepExecutor{runner: runner}
}

// Execute 并行运行配置中的 worker agent，并产出聚合 artifact。
func (e *ParallelStepExecutor) Execute(
	ctx context.Context,
	execCtx *executionContext,
	step *entity.PlanStep,
) (*entity.RuntimeArtifact, error) {
	if e == nil || e.runner == nil {
		return nil, fmt.Errorf("parallel runner is nil")
	}
	if step == nil {
		return nil, fmt.Errorf("parallel step is nil")
	}

	workerIDs := stringSliceConfig(step.Config["workers"])
	if len(workerIDs) == 0 {
		return nil, fmt.Errorf("parallel step %s has no workers", step.StepID)
	}
	userInput, err := buildStepUserInput(ctx, execCtx, step)
	if err != nil {
		return nil, err
	}

	type workerResult struct {
		index   int
		agentID string
		output  string
		err     error
	}
	resultCh := make(chan workerResult, len(workerIDs))

	var wg sync.WaitGroup
	for idx, agentID := range workerIDs {
		wg.Add(1)
		go func(index int, id string) {
			defer wg.Done()
			defer func() {
				if recovered := recover(); recovered != nil {
					panicErr := fmt.Errorf("worker %s panic: %v", id, recovered)
					if execCtx.trace != nil {
						execCtx.trace.Error(execCtx.runID, id, panicErr.Error())
						execCtx.trace.AgentExitWorker(execCtx.runID, id, step.StepID, fmt.Sprintf("执行 panic: %v", recovered))
					}
					resultCh <- workerResult{index: index, agentID: id, err: panicErr}
				}
			}()
			def := findEnabledAgent(execCtx.pool, id)
			if def == nil {
				resultCh <- workerResult{index: index, agentID: id, err: fmt.Errorf("agent %s is not available in pool", id)}
				return
			}
			if execCtx.trace != nil {
				execCtx.trace.TaskAssigned(execCtx.runID, def.ID, userInput)
				execCtx.trace.AgentEnterWorker(execCtx.runID, def.ID, step.StepID)
				execCtx.trace.Thinking(execCtx.runID, def.ID, "并行处理任务...")
			}
			output, runErr := e.runner(ctx, execCtx.runID, def, userInput, execCtx.trace, schemeConfigOf(execCtx.scheme))
			if runErr != nil {
				if execCtx.trace != nil {
					execCtx.trace.Error(execCtx.runID, def.ID, runErr.Error())
					execCtx.trace.AgentExitWorker(execCtx.runID, def.ID, step.StepID, fmt.Sprintf("执行失败: %v", runErr))
				}
				resultCh <- workerResult{index: index, agentID: def.ID, err: runErr}
				return
			}
			if execCtx.trace != nil {
				execCtx.trace.AgentExitWorker(execCtx.runID, def.ID, step.StepID, "执行完成")
			}
			resultCh <- workerResult{index: index, agentID: def.ID, output: output}
		}(idx, agentID)
	}

	wg.Wait()
	close(resultCh)

	results := make([]workerResult, len(workerIDs))
	for result := range resultCh {
		results[result.index] = result
	}

	payloadResults := make([]map[string]any, 0, len(results))
	failures := make([]map[string]any, 0)
	summaries := make([]string, 0, len(results))
	for _, result := range results {
		if result.err != nil {
			failures = append(failures, map[string]any{
				"agent_id": result.agentID,
				"error":    result.err.Error(),
			})
			continue
		}
		payloadResults = append(payloadResults, map[string]any{
			"agent_id": result.agentID,
			"output":   result.output,
		})
		summaries = append(summaries, fmt.Sprintf("[%s] %s", result.agentID, strings.TrimSpace(result.output)))
	}

	summary := strings.Join(summaries, "\n")
	artifactType := "parallel_result"
	if len(failures) > 0 {
		artifactType = "parallel_partial_result"
	}
	artifact := newArtifact(execCtx.runID, step.StepID, artifactType, summary, map[string]any{
		"workers": workerIDs,
		"results": payloadResults,
	})
	if len(failures) == 0 {
		return artifact, nil
	}
	artifact.Payload["failures"] = failures
	failureSummaries := make([]string, 0, len(failures))
	for _, failure := range failures {
		failureSummaries = append(failureSummaries, fmt.Sprintf("%s: %s", failure["agent_id"], failure["error"]))
	}
	if artifact.Summary == "" {
		artifact.Summary = "parallel step partially failed"
	}
	return nil, &stepExecutionError{
		err:      fmt.Errorf("parallel step %s partially failed: %s", step.StepID, strings.Join(failureSummaries, "; ")),
		artifact: artifact,
	}
}
