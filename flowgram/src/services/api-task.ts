import { requestJSON } from './http';

export enum TaskStatus {
  PENDING = 1,
  PROCESSING = 2,
  COMPLETED = 3,
  FAILED = 4,
}

export enum TaskType {
  BUG = 1,
  REQUIRE = 2,
  FEATURE = 3,
  OTHER = 4,
}

export interface TaskItem {
  id: number;
  name: string;
  priority: number;
  status: TaskStatus;
  type: TaskType;
  created_at: string;
  updated_at: string;
  handler_user_id: string;
  description: string;
  rule_chain_id?: string;
  parent_id?: number;
  last_run_id?: string;
}

export interface ListTasksParams {
  status?: TaskStatus;
  type?: TaskType;
  handler_user_id?: string;
  page?: number;
  page_size?: number;
}

export interface ListTasksReply {
  items: TaskItem[];
  total: number;
}

/** HTTP JSON 可能为数字或 proto 枚举名（如 TASK_STATUS_PENDING），供列表/表单回显 */
export function parseTaskStatus(raw: unknown): TaskStatus {
  if (typeof raw === 'number' && raw >= TaskStatus.PENDING && raw <= TaskStatus.FAILED) {
    return raw as TaskStatus;
  }
  if (typeof raw === 'string') {
    const t = raw.trim();
    if (t !== '' && !Number.isNaN(Number(t)) && !t.includes('_')) {
      const n = Number(t);
      if (n >= TaskStatus.PENDING && n <= TaskStatus.FAILED) return n as TaskStatus;
    }
    const map: Record<string, TaskStatus> = {
      TASK_STATUS_UNSPECIFIED: TaskStatus.PENDING,
      TASK_STATUS_PENDING: TaskStatus.PENDING,
      TASK_STATUS_PROCESSING: TaskStatus.PROCESSING,
      TASK_STATUS_COMPLETED: TaskStatus.COMPLETED,
      TASK_STATUS_FAILED: TaskStatus.FAILED,
    };
    if (map[t] !== undefined) return map[t];
  }
  return TaskStatus.PENDING;
}

export function parseTaskType(raw: unknown): TaskType {
  if (typeof raw === 'number' && raw >= TaskType.BUG && raw <= TaskType.OTHER) {
    return raw as TaskType;
  }
  if (typeof raw === 'string') {
    const t = raw.trim();
    if (t !== '' && !Number.isNaN(Number(t)) && !t.includes('_')) {
      const n = Number(t);
      if (n >= TaskType.BUG && n <= TaskType.OTHER) return n as TaskType;
    }
    const map: Record<string, TaskType> = {
      TASK_TYPE_UNSPECIFIED: TaskType.OTHER,
      TASK_TYPE_BUG: TaskType.BUG,
      TASK_TYPE_REQUIRE: TaskType.REQUIRE,
      TASK_TYPE_FEATURE: TaskType.FEATURE,
      TASK_TYPE_OTHER: TaskType.OTHER,
    };
    if (map[t] !== undefined) return map[t];
  }
  return TaskType.OTHER;
}

function parseTaskPriority(raw: unknown): number {
  const n = Number(raw);
  if (!Number.isFinite(n)) return 99;
  return Math.min(99, Math.max(0, Math.floor(n)));
}

/** 将 protobuf / Kratos 返回的任务行规范为表格用的 TaskItem */
function normalizeTaskFromApi(raw: Record<string, unknown> | null | undefined): TaskItem {
  const r = raw ?? {};
  const handler =
    (r.handler_user_id as string) ??
    (r.handlerUserId as string) ??
    '';
  const created = r.created_at ?? r.createdAt;
  const updated = r.updated_at ?? r.updatedAt;
  const parentIdRaw = r.parent_id ?? r.parentId;
  const parentIdNum = typeof parentIdRaw === 'number' ? parentIdRaw : (typeof parentIdRaw === 'string' && parentIdRaw !== '' ? Number(parentIdRaw) : undefined);
  return {
    id: Number(r.id),
    name: String(r.name ?? ''),
    priority: parseTaskPriority(r.priority),
    status: parseTaskStatus(r.status),
    type: parseTaskType(r.type),
    created_at: formatTaskTimestamp(created),
    updated_at: formatTaskTimestamp(updated),
    handler_user_id: handler,
    description: String(r.description ?? ''),
    rule_chain_id: typeof r.rule_chain_id === 'string' ? r.rule_chain_id : typeof r.ruleChainId === 'string' ? r.ruleChainId : undefined,
    parent_id: parentIdNum != null && Number.isFinite(parentIdNum) ? parentIdNum : undefined,
    last_run_id: typeof r.last_run_id === 'string' ? r.last_run_id : typeof r.lastRunId === 'string' ? r.lastRunId : undefined,
  };
}

function formatTaskTimestamp(v: unknown): string {
  if (v == null || v === '') return '';
  if (typeof v === 'string') return v;
  if (typeof v === 'object' && v !== null && 'seconds' in (v as Record<string, unknown>)) {
    const sec = Number((v as { seconds?: unknown }).seconds);
    if (!Number.isFinite(sec)) return '';
    return new Date(sec * 1000).toISOString();
  }
  return String(v);
}

/** ListTasks 接口返回 tasks（proto）；历史前端曾用 items，此处兼容两种字段名 */
function normalizeListTasksReply(raw: Record<string, unknown>): ListTasksReply {
  const list = (raw.tasks ?? raw.items) as unknown[] | undefined;
  const arr = Array.isArray(list) ? list : [];
  return {
    items: arr.map((row) => normalizeTaskFromApi(row as Record<string, unknown>)),
    total: Number(raw.total ?? 0),
  };
}

export interface CreateTaskParams {
  name: string;
  priority: number;
  type: TaskType;
  handler_user_id: string;
  description: string;
  rule_chain_id?: string;
}

export interface UpdateTaskParams {
  name?: string;
  priority?: number;
  status?: TaskStatus;
  handler_user_id?: string;
  description?: string;
  rule_chain_id?: string;
  clear_rule_chain_id?: boolean;
}

export interface CreateChildTaskParams {
  name_suffix?: string;
}

export interface ListChildTasksParams {
  page?: number;
  page_size?: number;
}

// 获取任务列表
export const listTasks = async (params: ListTasksParams): Promise<ListTasksReply> => {
  const raw = await requestJSON<Record<string, unknown>>('/tasks', { method: 'GET', params });
  return normalizeListTasksReply(raw);
};

// 获取任务详情（后端字段为 task）
export const getTask = async (id: number): Promise<{ item: TaskItem }> => {
  const raw = await requestJSON<Record<string, unknown>>(`/tasks/${id}`);
  const t = (raw.task ?? raw.item) as Record<string, unknown> | undefined;
  return { item: normalizeTaskFromApi(t) };
};

// 创建任务（CreateTaskReply.task）
export const createTask = async (params: CreateTaskParams): Promise<{ item: TaskItem }> => {
  const raw = await requestJSON<Record<string, unknown>>('/tasks', { method: 'POST', body: params });
  const t = (raw.task ?? raw.item) as Record<string, unknown> | undefined;
  return { item: normalizeTaskFromApi(t) };
};

// 更新任务（UpdateTaskReply.task）
export const updateTask = async (id: number, params: UpdateTaskParams): Promise<{ item: TaskItem }> => {
  const raw = await requestJSON<Record<string, unknown>>(`/tasks/${id}`, { method: 'PUT', body: params });
  const t = (raw.task ?? raw.item) as Record<string, unknown> | undefined;
  return { item: normalizeTaskFromApi(t) };
};

// 删除任务
export const deleteTask = (id: number): Promise<{ success: boolean }> =>
  requestJSON(`/tasks/${id}`, { method: 'DELETE' });

// 执行任务关联的规则链
export const executeTaskRuleChain = (id: number): Promise<{ success: boolean; message: string }> =>
  requestJSON(`/tasks/${id}/execute`, { method: 'POST', body: {} });

// 创建子任务
export const createChildTask = async (parentId: number, params?: CreateChildTaskParams): Promise<{ item: TaskItem }> => {
  const raw = await requestJSON<Record<string, unknown>>(`/tasks/${parentId}/children`, {
    method: 'POST',
    body: params ?? {},
  });
  const t = (raw.task ?? raw.item) as Record<string, unknown> | undefined;
  return { item: normalizeTaskFromApi(t) };
};

// 查询子任务列表
export const listChildTasks = async (parentId: number, params?: ListChildTasksParams): Promise<{ items: TaskItem[]; total: number }> => {
  const raw = await requestJSON<Record<string, unknown>>(`/tasks/${parentId}/children`, { params });
  const list = (raw.tasks ?? raw.items) as unknown[] | undefined;
  return {
    items: (Array.isArray(list) ? list : []).map((row) => normalizeTaskFromApi(row as Record<string, unknown>)),
    total: Number(raw.total ?? 0),
  };
};

export const taskStatusOptions = [
  { value: TaskStatus.PENDING, label: '待处理', color: 'orange' },
  { value: TaskStatus.PROCESSING, label: '处理中', color: 'blue' },
  { value: TaskStatus.COMPLETED, label: '已完成', color: 'green' },
  { value: TaskStatus.FAILED, label: '处理失败', color: 'red' },
];

export const taskTypeOptions = [
  { value: TaskType.BUG, label: '缺陷', color: 'red' },
  { value: TaskType.REQUIRE, label: '需求', color: 'purple' },
  { value: TaskType.FEATURE, label: '功能', color: 'blue' },
  { value: TaskType.OTHER, label: '其他', color: 'grey' },
];

/** 与 proto 一致：priority 取值 0–99 */
export const priorityOptions = Array.from({ length: 100 }, (_, i) => ({
  value: i,
  label: String(i),
}));
