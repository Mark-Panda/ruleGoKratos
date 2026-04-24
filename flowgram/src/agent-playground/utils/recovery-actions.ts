import { RecoveryAction, RuntimeRunDetail, RuntimeRunStatus } from '../../services/api-playground';

const DEFAULT_REFRESH_ATTEMPTS = 6;
const DEFAULT_RETRY_DELAY_MS = 250;

const APPLICABLE_RECOVERY_TYPES = new Set<RecoveryAction['type']>([
  'retry_step',
  'skip_step',
  'reroute_step',
  'retry_from_checkpoint',
]);

export function canApplyRecoveryAction(
  action: RecoveryAction,
  runStatus: RuntimeRunStatus | 'idle',
  applyingActionId?: string
): boolean {
  if (applyingActionId === action.id) {
    return false;
  }
  if (runStatus !== 'waiting_recovery') {
    return false;
  }
  return APPLICABLE_RECOVERY_TYPES.has(action.type);
}

/** 按钮文案：与后端 recovery action 类型对齐 */
/** POST body：与需要显式目标引用的 recovery 类型配合（与 action 上的 targetRef 对齐） */
export function recoveryActionRequestBody(
  action: RecoveryAction
): { targetRef?: string } | undefined {
  const ref = action.targetRef?.trim();
  if (!ref) {
    return undefined;
  }
  if (action.type === 'reroute_step' || action.type === 'retry_from_checkpoint') {
    return { targetRef: ref };
  }
  return undefined;
}

export function recoveryActionButtonLabel(action: RecoveryAction): string {
  switch (action.type) {
    case 'retry_step':
      return '重试该步骤';
    case 'skip_step':
      return '跳过该步骤';
    case 'reroute_step':
      return action.targetRef?.trim() ? `强制路由到 ${action.targetRef.trim()}` : '强制路由并重试';
    case 'retry_from_checkpoint':
      return action.targetRef?.trim()
        ? `从检查点「${action.targetRef.trim()}」恢复`
        : '从检查点恢复';
    default:
      return '执行';
  }
}

export async function applyRecoveryActionAndRefresh(input: {
  runId: string;
  action: RecoveryAction;
  submit: (runId: string, actionId: string, body?: { targetRef?: string }) => Promise<unknown>;
  refresh: (runId: string) => Promise<RuntimeRunDetail>;
  wait?: (ms: number) => Promise<void>;
  maxRefreshAttempts?: number;
  retryDelayMs?: number;
}): Promise<RuntimeRunDetail> {
  const wait = input.wait ?? defaultWait;
  const maxRefreshAttempts = input.maxRefreshAttempts ?? DEFAULT_REFRESH_ATTEMPTS;
  const retryDelayMs = input.retryDelayMs ?? DEFAULT_RETRY_DELAY_MS;

  await input.submit(input.runId, input.action.id, recoveryActionRequestBody(input.action));

  let lastDetail: RuntimeRunDetail | undefined;
  for (let attempt = 0; attempt < maxRefreshAttempts; attempt += 1) {
    lastDetail = await input.refresh(input.runId);
    if (lastDetail.run.status !== 'waiting_recovery') {
      return lastDetail;
    }
    if (attempt < maxRefreshAttempts - 1) {
      await wait(retryDelayMs);
    }
  }

  if (!lastDetail) {
    throw new Error('refresh runtime detail failed');
  }
  return lastDetail;
}

async function defaultWait(ms: number): Promise<void> {
  await new Promise((resolve) => {
    window.setTimeout(resolve, ms);
  });
}
