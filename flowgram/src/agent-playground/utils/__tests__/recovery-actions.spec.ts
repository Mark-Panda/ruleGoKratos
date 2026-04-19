import { describe, expect, it, vi } from 'vitest';

import {
  applyRecoveryActionAndRefresh,
  canApplyRecoveryAction,
  recoveryActionButtonLabel,
  recoveryActionRequestBody,
} from '../recovery-actions';
import { RecoveryAction, RuntimeRunDetail } from '../../../services/api-playground';

function buildDetail(status: RuntimeRunDetail['run']['status']): RuntimeRunDetail {
  return {
    run: {
      runId: 'run-1',
      schemeId: 'scheme-1',
      planId: 'plan-1',
      status,
    },
    steps: [],
    artifacts: [],
    recoveryActions: [],
  };
}

describe('recovery-actions', () => {
  const retryAction: RecoveryAction = {
    id: 'action-1',
    type: 'retry_step',
    stepId: 'step-1',
    reason: 'retry failed step',
  };

  const skipAction: RecoveryAction = {
    id: 'action-skip',
    type: 'skip_step',
    stepId: 'step-1',
    reason: 'skip',
  };

  it('enables supported recovery types only while waiting_recovery', () => {
    expect(canApplyRecoveryAction(retryAction, 'waiting_recovery')).toBe(true);
    expect(canApplyRecoveryAction(skipAction, 'waiting_recovery')).toBe(true);
    expect(canApplyRecoveryAction(retryAction, 'running')).toBe(false);
    expect(canApplyRecoveryAction(retryAction, 'waiting_recovery', retryAction.id)).toBe(false);
  });

  it('recoveryActionButtonLabel reflects type and targetRef', () => {
    expect(recoveryActionButtonLabel(retryAction)).toBe('重试该步骤');
    expect(recoveryActionButtonLabel(skipAction)).toBe('跳过该步骤');
    expect(
      recoveryActionButtonLabel({
        ...retryAction,
        id: 'r2',
        type: 'reroute_step',
        targetRef: 'agent-x',
      }),
    ).toBe('强制路由到 agent-x');
    expect(
      recoveryActionButtonLabel({
        ...retryAction,
        id: 'r3',
        type: 'retry_from_checkpoint',
        targetRef: 'chk-1',
      }),
    ).toBe('从检查点「chk-1」恢复');
  });

  it('recoveryActionRequestBody sends targetRef for reroute and checkpoint', () => {
    expect(recoveryActionRequestBody(retryAction)).toBeUndefined();
    expect(
      recoveryActionRequestBody({
        ...retryAction,
        type: 'reroute_step',
        targetRef: ' a ',
      }),
    ).toEqual({ targetRef: 'a' });
    expect(
      recoveryActionRequestBody({
        ...retryAction,
        type: 'retry_from_checkpoint',
        targetRef: 'step-x',
      }),
    ).toEqual({ targetRef: 'step-x' });
  });

  it('submits action then refreshes until run leaves waiting_recovery', async () => {
    const submit = vi.fn(async () => undefined);
    const refresh = vi
      .fn<(_: string) => Promise<RuntimeRunDetail>>()
      .mockResolvedValueOnce(buildDetail('waiting_recovery'))
      .mockResolvedValueOnce(buildDetail('running'));
    const wait = vi.fn(async () => undefined);

    const detail = await applyRecoveryActionAndRefresh({
      runId: 'run-1',
      action: retryAction,
      submit,
      refresh,
      wait,
      maxRefreshAttempts: 3,
      retryDelayMs: 1,
    });

    expect(submit).toHaveBeenCalledWith('run-1', 'action-1', undefined);
    expect(refresh).toHaveBeenCalledTimes(2);
    expect(wait).toHaveBeenCalledTimes(1);
    expect(detail.run.status).toBe('running');
  });

  it('passes body when reroute action has targetRef', async () => {
    const submit = vi.fn(async () => undefined);
    const refresh = vi
      .fn<(_: string) => Promise<RuntimeRunDetail>>()
      .mockResolvedValueOnce(buildDetail('running'));

    await applyRecoveryActionAndRefresh({
      runId: 'run-1',
      action: { ...retryAction, id: 'a2', type: 'reroute_step', targetRef: 'planner' },
      submit,
      refresh,
      wait: vi.fn(async () => undefined),
      maxRefreshAttempts: 3,
      retryDelayMs: 1,
    });

    expect(submit).toHaveBeenCalledWith('run-1', 'a2', { targetRef: 'planner' });
    expect(refresh).toHaveBeenCalledTimes(1);
  });
});
