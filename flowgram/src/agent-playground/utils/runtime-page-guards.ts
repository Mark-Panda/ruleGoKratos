import { RuntimeRunDetail, RuntimeRunStatus, TraceEvent } from '../../services/api-playground';

export interface RequestGuardSnapshot {
  runId: string;
  version: number;
}

export interface DisplayedRuntimeState {
  currentRunDetail?: RuntimeRunDetail;
  events: TraceEvent[];
  running: boolean;
  isBoundToSelectedScheme: boolean;
}

export interface InvalidatedRequestGuards {
  activeRunId?: undefined;
  runDetailVersion: number;
  eventsVersion: number;
}

const AUTO_REFRESH_RUN_STATUSES = new Set<RuntimeRunStatus>([
  'pending',
  'ready',
  'running',
]);

export function createRequestGuardSnapshot(runId: string, version: number): RequestGuardSnapshot {
  return { runId, version };
}

export function isRequestGuardCurrent(
  snapshot: RequestGuardSnapshot,
  activeRunId?: string,
  activeVersion?: number,
): boolean {
  return snapshot.runId === activeRunId && snapshot.version === activeVersion;
}

export function shouldDisplayRuntimeForScheme(selectedSchemeId?: string, runSchemeId?: string): boolean {
  if (!selectedSchemeId || !runSchemeId) {
    return false;
  }
  return selectedSchemeId === runSchemeId;
}

export function getDisplayedRuntimeState(input: {
  selectedSchemeId?: string;
  currentRunSchemeId?: string;
  currentRunDetail?: RuntimeRunDetail;
  events: TraceEvent[];
  running: boolean;
}): DisplayedRuntimeState {
  const isBoundToSelectedScheme = shouldDisplayRuntimeForScheme(
    input.selectedSchemeId,
    input.currentRunDetail?.run.schemeId ?? input.currentRunSchemeId,
  );

  if (!isBoundToSelectedScheme) {
    return {
      currentRunDetail: undefined,
      events: [],
      running: false,
      isBoundToSelectedScheme,
    };
  }

  const displayedRunId = input.currentRunDetail?.run.runId;
  if (!displayedRunId) {
    return {
      currentRunDetail: undefined,
      events: [],
      running: input.running,
      isBoundToSelectedScheme,
    };
  }

  return {
    currentRunDetail: input.currentRunDetail,
    events: input.events.filter(event => event.runId === displayedRunId),
    running: input.running,
    isBoundToSelectedScheme,
  };
}

export function invalidateRequestGuards(input: {
  runDetailVersion: number;
  eventsVersion: number;
}): InvalidatedRequestGuards {
  return {
    activeRunId: undefined,
    runDetailVersion: input.runDetailVersion + 1,
    eventsVersion: input.eventsVersion + 1,
  };
}

export function shouldAutoRefreshRun(status?: RuntimeRunStatus): boolean {
  return status ? AUTO_REFRESH_RUN_STATUSES.has(status) : false;
}
