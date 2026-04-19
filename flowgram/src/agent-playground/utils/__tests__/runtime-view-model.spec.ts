import { describe, expect, it } from 'vitest';

import { buildRuntimeViewModel } from '../runtime-view-model';

describe('buildRuntimeViewModel', () => {
  it('暴露失败步骤与恢复动作', () => {
    const vm = buildRuntimeViewModel({
      run: {
        runId: 'run-1',
        schemeId: 'scheme-1',
        planId: 'plan-1',
        status: 'waiting_recovery',
        currentStepIds: ['agent'],
        failureSummary: 'agent timeout',
        startedAt: '2026-04-19T10:00:00Z',
        finishedAt: '',
        userInput: '生成登录页',
        finalOutput: '',
      },
      steps: [
        {
          stepId: 'route',
          kind: 'route',
          name: 'Route',
          status: 'succeeded',
          outputRef: 'route_result',
        },
        {
          stepId: 'agent',
          kind: 'agent',
          name: 'Engineer',
          status: 'failed',
          agentBinding: 'engineer',
          failureSummary: 'agent timeout',
          inputRefs: ['route_result'],
          outputRef: 'draft_output',
        },
      ],
      artifacts: [
        {
          artifactId: 'artifact-1',
          type: 'route_result',
          producerStepId: 'route',
          summary: '选择 engineer',
        },
      ],
      recoveryActions: [
        {
          id: 'ra-1',
          type: 'retry_step',
          stepId: 'agent',
          reason: '重新执行当前步骤',
        },
      ],
      events: [],
    });

    expect(vm.failedStep?.stepId).toBe('agent');
    expect(vm.failedStep?.recoveryActions).toHaveLength(1);
    expect(vm.recovery.summary.count).toBe(1);
    expect(vm.run.failureSummary).toBe('agent timeout');
  });

  it('将 plan 节点映射为 UI 状态并归集产物', () => {
    const vm = buildRuntimeViewModel({
      run: {
        runId: 'run-2',
        schemeId: 'scheme-2',
        planId: 'plan-2',
        status: 'running',
        currentStepIds: ['review'],
        failureSummary: '',
        startedAt: '2026-04-19T10:01:00Z',
        finishedAt: '',
        userInput: '做一次评审',
        finalOutput: '',
      },
      steps: [
        {
          stepId: 'route',
          kind: 'route',
          name: 'Route',
          status: 'succeeded',
          outputRef: 'route_result',
        },
        {
          stepId: 'review',
          kind: 'review',
          name: 'Supervisor Review',
          status: 'running',
          inputRefs: ['route_result'],
          outputRef: 'review_result',
        },
        {
          stepId: 'finalize',
          kind: 'finalize',
          name: 'Finalize',
          status: 'pending',
          inputRefs: ['review_result'],
          outputRef: 'final_output',
        },
      ],
      artifacts: [
        {
          artifactId: 'artifact-2',
          type: 'route_result',
          producerStepId: 'route',
          summary: '路由到 supervisor',
        },
        {
          artifactId: 'artifact-3',
          type: 'review_note',
          producerStepId: 'review',
          summary: '正在汇总 worker 结果',
        },
      ],
      recoveryActions: [],
      events: [],
    });

    expect(vm.planNodes.map(node => node.status)).toEqual(['completed', 'active', 'pending']);
    expect(vm.planNodes[0]?.artifacts).toHaveLength(1);
    expect(vm.planNodes[1]?.isCurrent).toBe(true);
    expect(vm.planNodes[2]?.artifacts).toHaveLength(0);
  });
});
