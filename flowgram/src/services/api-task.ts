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

export interface CreateTaskParams {
  name: string;
  priority: number;
  type: TaskType;
  handler_user_id: string;
  description: string;
}

export interface UpdateTaskParams {
  name?: string;
  priority?: number;
  status?: TaskStatus;
  handler_user_id?: string;
  description?: string;
}

// 获取任务列表
export const listTasks = (params: ListTasksParams): Promise<ListTasksReply> =>
  requestJSON('/tasks', { method: 'GET', params });

// 获取任务详情
export const getTask = (id: number): Promise<{ item: TaskItem }> => requestJSON(`/tasks/${id}`);

// 创建任务
export const createTask = (params: CreateTaskParams): Promise<{ item: TaskItem }> =>
  requestJSON('/tasks', { method: 'POST', body: params });

// 更新任务
export const updateTask = (id: number, params: UpdateTaskParams): Promise<{ item: TaskItem }> =>
  requestJSON(`/tasks/${id}`, { method: 'PUT', body: params });

// 删除任务
export const deleteTask = (id: number): Promise<{ success: boolean }> =>
  requestJSON(`/tasks/${id}`, { method: 'DELETE' });

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

export const priorityOptions = Array.from({ length: 100 }, (_, i) => ({
  value: i + 1,
  label: String(i + 1),
}));
