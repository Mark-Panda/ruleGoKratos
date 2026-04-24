import { describe, expect, it } from 'vitest';

import {
  createRequestGuardSnapshot,
  getDisplayedRuntimeState,
  invalidateRequestGuards,
  isRequestGuardCurrent,
  shouldAutoRefreshRun,
} from '../runtime-page-guards';

describe('runtime page guards', () => {
  it('拒绝过期请求版本或旧 run 回包', () => {
    const stale = createRequestGuardSnapshot('run-1', 1);
    const latest = createRequestGuardSnapshot('run-2', 3);

    expect(isRequestGuardCurrent(stale, 'run-2', 3)).toBe(false);
    expect(isRequestGuardCurrent(createRequestGuardSnapshot('run-2', 2), 'run-2', 3)).toBe(false);
    expect(isRequestGuardCurrent(latest, 'run-2', 3)).toBe(true);
  });

  it('切换方案后隐藏不匹配的 runtime 数据', () => {
    const displayed = getDisplayedRuntimeState({
      selectedSchemeId: 'scheme-2',
      currentRunDetail: {
        run: {
          runId: 'run-1',
          schemeId: 'scheme-1',
          planId: 'plan-1',
          status: 'running',
          currentStepIds: ['agent'],
          failureSummary: '',
          startedAt: '2026-04-19T10:00:00Z',
          finishedAt: '',
          userInput: 'hello',
          finalOutput: '',
        },
        steps: [],
        artifacts: [],
        recoveryActions: [],
      },
      events: [
        {
          id: 'event-1',
          runId: 'run-1',
          timestamp: Date.now(),
          type: 'WORKFLOW_START',
          agentId: '',
          nodeId: '',
          taskDesc: '',
          message: 'start',
          metadata: {},
        },
      ],
      running: true,
    });

    expect(displayed.isBoundToSelectedScheme).toBe(false);
    expect(displayed.currentRunDetail).toBeUndefined();
    expect(displayed.events).toEqual([]);
    expect(displayed.running).toBe(false);
  });

  it('只保留当前展示 run 的事件流', () => {
    const displayed = getDisplayedRuntimeState({
      selectedSchemeId: 'scheme-1',
      currentRunSchemeId: 'scheme-1',
      currentRunDetail: {
        run: {
          runId: 'run-2',
          schemeId: 'scheme-1',
          planId: 'plan-1',
          status: 'running',
          currentStepIds: ['agent'],
          failureSummary: '',
          startedAt: '2026-04-19T10:00:00Z',
          finishedAt: '',
          userInput: 'hello',
          finalOutput: '',
        },
        steps: [],
        artifacts: [],
        recoveryActions: [],
      },
      events: [
        {
          id: 'event-1',
          runId: 'run-1',
          timestamp: 1,
          type: 'WORKFLOW_START',
          agentId: '',
          nodeId: '',
          taskDesc: '',
          message: 'old',
          metadata: {},
        },
        {
          id: 'event-2',
          runId: 'run-2',
          timestamp: 2,
          type: 'WORKFLOW_START',
          agentId: '',
          nodeId: '',
          taskDesc: '',
          message: 'current',
          metadata: {},
        },
      ],
      running: true,
    });

    expect(displayed.events.map((event) => event.id)).toEqual(['event-2']);
  });

  it('启动新 run 前可立即让旧 guard 失效', () => {
    const oldRunSnapshot = createRequestGuardSnapshot('run-old', 4);
    const invalidated = invalidateRequestGuards({
      runDetailVersion: 4,
      eventsVersion: 9,
    });

    expect(invalidated.activeRunId).toBeUndefined();
    expect(
      isRequestGuardCurrent(oldRunSnapshot, invalidated.activeRunId, invalidated.runDetailVersion)
    ).toBe(false);
    expect(invalidated.runDetailVersion).toBe(5);
    expect(invalidated.eventsVersion).toBe(10);
  });

  it('waiting_recovery 不应继续触发自动刷新', () => {
    expect(shouldAutoRefreshRun('waiting_recovery')).toBe(false);
    expect(shouldAutoRefreshRun('completed')).toBe(false);
  });
});
