import { requestJSON } from './http';

export type ScheduledTaskRunStatus = number | string;

export interface ScheduledTask {
  id: number | string;
  name: string;
  description?: string;
  ruleChainId: string;
  cronExpr: string;
  scheduleType: string;
  scheduleConfig: string;
  disabled: boolean;
  lastRunAt?: string | Record<string, unknown> | null;
  lastStatus?: ScheduledTaskRunStatus;
  lastError?: string;
  createdAt?: string | Record<string, unknown> | null;
  updatedAt?: string | Record<string, unknown> | null;
  deletedAt?: string | Record<string, unknown> | null;
  payloadTemplate?: string;
}

export interface ScheduledTaskRun {
  id: number | string;
  taskId: number | string;
  ruleChainId: string;
  status: ScheduledTaskRunStatus;
  triggerPayload?: string;
  errorMessage?: string;
  startedAt?: string | Record<string, unknown> | null;
  finishedAt?: string | Record<string, unknown> | null;
  createdAt?: string | Record<string, unknown> | null;
}

export interface ListScheduledTasksParams {
  name?: string;
  ruleChainId?: string;
  disabled?: boolean;
  page?: number;
  pageSize?: number;
}

export interface ListScheduledTasksReply {
  tasks: ScheduledTask[];
  total: number | string;
}

export interface ScheduledTaskPayload {
  name: string;
  description: string;
  ruleChainId: string;
  cronExpr: string;
  scheduleType: string;
  scheduleConfig: string;
  payloadTemplate?: string;
}

export type UpdateScheduledTaskPayload = ScheduledTaskPayload;

export interface ScheduledTaskReply {
  task: ScheduledTask;
}

export interface ListScheduledTaskRunsParams {
  page?: number;
  pageSize?: number;
}

export interface ListScheduledTaskRunsReply {
  runs: ScheduledTaskRun[];
  total: number | string;
}

const taskPath = (id: number | string): string => `/scheduled-tasks/${encodeURIComponent(String(id))}`;

export const listScheduledTasks = (
  params: ListScheduledTasksParams = {}
): Promise<ListScheduledTasksReply> =>
  requestJSON('/scheduled-tasks', { method: 'GET', params });

export const getScheduledTask = (id: number): Promise<ScheduledTaskReply> =>
  requestJSON(taskPath(id), { method: 'GET' });

export const createScheduledTask = (
  payload: ScheduledTaskPayload
): Promise<ScheduledTaskReply> =>
  requestJSON('/scheduled-tasks', { method: 'POST', body: payload });

export const updateScheduledTask = (
  id: number | string,
  payload: UpdateScheduledTaskPayload
): Promise<ScheduledTaskReply> =>
  requestJSON(taskPath(id), { method: 'PUT', body: payload });

export const deleteScheduledTask = (id: number | string): Promise<{ success?: boolean }> =>
  requestJSON(taskPath(id), { method: 'DELETE' });

export const enableScheduledTask = (id: number | string): Promise<ScheduledTaskReply> =>
  requestJSON(`${taskPath(id)}/enable`, { method: 'POST', body: {} });

export const disableScheduledTask = (id: number | string): Promise<ScheduledTaskReply> =>
  requestJSON(`${taskPath(id)}/disable`, { method: 'POST', body: {} });

export const listScheduledTaskRuns = (
  taskId: number | string,
  params: ListScheduledTaskRunsParams = {}
): Promise<ListScheduledTaskRunsReply> =>
  requestJSON(`${taskPath(taskId)}/runs`, { method: 'GET', params });
