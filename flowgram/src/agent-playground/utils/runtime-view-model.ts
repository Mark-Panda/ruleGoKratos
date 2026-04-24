import {
  RecoveryAction,
  RuntimeArtifact,
  RuntimeRun,
  RuntimeRunStatus,
  RuntimeStep,
  RuntimeStepStatus,
  TraceEvent,
} from '../../services/api-playground';

export type RuntimeNodeDisplayStatus = 'pending' | 'active' | 'completed' | 'failed' | 'skipped';

export interface RuntimePlanNode {
  stepId: string;
  kind: RuntimeStep['kind'];
  name: string;
  agentBinding?: string;
  runtimeStatus: RuntimeStepStatus;
  status: RuntimeNodeDisplayStatus;
  isCurrent: boolean;
  isFailed: boolean;
  failureSummary?: string;
  inputRefs: string[];
  outputRef?: string;
  artifacts: RuntimeArtifact[];
  recoveryActions: RecoveryAction[];
}

export interface RuntimeViewModel {
  run: {
    raw?: RuntimeRun;
    runId?: string;
    schemeId?: string;
    planId?: string;
    status: RuntimeRunStatus | 'idle';
    label: string;
    isRunning: boolean;
    isWaitingRecovery: boolean;
    isFinished: boolean;
    userInput: string;
    finalOutput: string;
    failureSummary: string;
    currentStepIds: string[];
  };
  planNodes: RuntimePlanNode[];
  activeStep?: RuntimePlanNode;
  failedStep?: RuntimePlanNode;
  artifacts: {
    all: RuntimeArtifact[];
    total: number;
  };
  recovery: {
    actions: RecoveryAction[];
    summary: {
      count: number;
      waiting: boolean;
      failedStepId?: string;
    };
  };
  trace: {
    events: TraceEvent[];
    total: number;
    errorCount: number;
  };
}

export interface BuildRuntimeViewModelInput {
  run?: RuntimeRun;
  steps?: RuntimeStep[];
  artifacts?: RuntimeArtifact[];
  recoveryActions?: RecoveryAction[];
  events?: TraceEvent[];
}

const IN_PROGRESS_STATUSES = new Set<RuntimeRunStatus>(['pending', 'ready', 'running']);

const RUN_STATUS_LABELS: Record<RuntimeRunStatus | 'idle', string> = {
  idle: '未开始',
  pending: '待启动',
  ready: '就绪',
  running: '运行中',
  waiting_recovery: '等待恢复',
  completed: '已完成',
  failed: '失败',
  cancelled: '已取消',
};

function mapPlanNodeStatus(
  step: RuntimeStep,
  currentStepIds: Set<string>
): RuntimeNodeDisplayStatus {
  if (step.status === 'failed') {
    return 'failed';
  }
  if (step.status === 'skipped') {
    return 'skipped';
  }
  if (step.status === 'succeeded') {
    return 'completed';
  }
  if (step.status === 'running' || currentStepIds.has(step.stepId)) {
    return 'active';
  }
  return 'pending';
}

export function buildRuntimeViewModel(input: BuildRuntimeViewModelInput): RuntimeViewModel {
  const run = input.run;
  const steps = input.steps ?? [];
  const artifacts = input.artifacts ?? [];
  const recoveryActions = input.recoveryActions ?? [];
  const events = input.events ?? [];
  const currentStepIds = new Set(run?.currentStepIds ?? []);
  const actionsByStepId = new Map<string, RecoveryAction[]>();

  for (const action of recoveryActions) {
    const existing = actionsByStepId.get(action.stepId) ?? [];
    existing.push(action);
    actionsByStepId.set(action.stepId, existing);
  }

  const artifactsByStepId = new Map<string, RuntimeArtifact[]>();
  for (const artifact of artifacts) {
    const existing = artifactsByStepId.get(artifact.producerStepId) ?? [];
    existing.push(artifact);
    artifactsByStepId.set(artifact.producerStepId, existing);
  }

  const planNodes: RuntimePlanNode[] = steps.map((step) => ({
    stepId: step.stepId,
    kind: step.kind,
    name: step.name,
    agentBinding: step.agentBinding,
    runtimeStatus: step.status,
    status: mapPlanNodeStatus(step, currentStepIds),
    isCurrent: currentStepIds.has(step.stepId) || step.status === 'running',
    isFailed: step.status === 'failed',
    failureSummary: step.failureSummary,
    inputRefs: step.inputRefs ?? [],
    outputRef: step.outputRef,
    artifacts: artifactsByStepId.get(step.stepId) ?? [],
    recoveryActions: actionsByStepId.get(step.stepId) ?? [],
  }));

  const failedStep =
    planNodes.find((node) => node.isFailed) ??
    (recoveryActions.length > 0
      ? planNodes.find((node) => node.stepId === recoveryActions[0]?.stepId)
      : undefined);

  const activeStep = planNodes.find((node) => node.isCurrent);
  const runStatus = run?.status ?? 'idle';

  return {
    run: {
      raw: run,
      runId: run?.runId,
      schemeId: run?.schemeId,
      planId: run?.planId,
      status: runStatus,
      label: RUN_STATUS_LABELS[runStatus],
      isRunning: run ? IN_PROGRESS_STATUSES.has(run.status) : false,
      isWaitingRecovery: run?.status === 'waiting_recovery',
      isFinished: run ? !IN_PROGRESS_STATUSES.has(run.status) : false,
      userInput: run?.userInput?.trim() ?? '',
      finalOutput: run?.finalOutput?.trim() ?? '',
      failureSummary: run?.failureSummary?.trim() || failedStep?.failureSummary || '',
      currentStepIds: [...currentStepIds],
    },
    planNodes,
    activeStep,
    failedStep,
    artifacts: {
      all: artifacts,
      total: artifacts.length,
    },
    recovery: {
      actions: recoveryActions,
      summary: {
        count: recoveryActions.length,
        waiting: run?.status === 'waiting_recovery',
        failedStepId: failedStep?.stepId,
      },
    },
    trace: {
      events,
      total: events.length,
      errorCount: events.filter((event) => event.type === 'ERROR').length,
    },
  };
}
