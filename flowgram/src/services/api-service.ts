import { requestJSON } from './http';

export enum ServiceStatus {
  RUNNING = 1,
  STOPPED = 2,
}

export interface ServiceItem {
  id: number;
  name: string;
  status: ServiceStatus;
  volc_log_service_id: string;
  git_repo_url: string;
  created_at: string;
  updated_at: string;
  description: string;
}

export interface ListServicesParams {
  status?: ServiceStatus;
  page?: number;
  page_size?: number;
}

export interface ListServicesReply {
  items: ServiceItem[];
  total: number;
}

export interface CreateServiceParams {
  name: string;
  status: ServiceStatus;
  volc_log_service_id: string;
  git_repo_url: string;
  description: string;
}

export interface UpdateServiceParams {
  name?: string;
  status?: ServiceStatus;
  volc_log_service_id?: string;
  git_repo_url?: string;
  description?: string;
}

// 获取服务列表
export const listServices = (params: ListServicesParams): Promise<ListServicesReply> =>
  requestJSON('/services', { method: 'GET', params });

// 获取服务详情
export const getService = (id: number): Promise<{ item: ServiceItem }> =>
  requestJSON(`/services/${id}`);

// 创建服务
export const createService = (params: CreateServiceParams): Promise<{ item: ServiceItem }> =>
  requestJSON('/services', { method: 'POST', body: params });

// 更新服务
export const updateService = (
  id: number,
  params: UpdateServiceParams
): Promise<{ item: ServiceItem }> =>
  requestJSON(`/services/${id}`, { method: 'PUT', body: params });

// 删除服务
export const deleteService = (id: number): Promise<{ success: boolean }> =>
  requestJSON(`/services/${id}`, { method: 'DELETE' });

export const serviceStatusOptions = [
  { value: ServiceStatus.RUNNING, label: '运行中', color: 'green' },
  { value: ServiceStatus.STOPPED, label: '已停止', color: 'red' },
];
