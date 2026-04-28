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

/** JSON 可能为数字或 SERVICE_STATUS_* 字符串 */
export function parseServiceStatus(raw: unknown): ServiceStatus {
  if (typeof raw === 'number' && (raw === ServiceStatus.RUNNING || raw === ServiceStatus.STOPPED)) {
    return raw as ServiceStatus;
  }
  if (typeof raw === 'string') {
    const t = raw.trim();
    if (t === '1' || t === '2') return Number(t) as ServiceStatus;
    const map: Record<string, ServiceStatus> = {
      SERVICE_STATUS_UNSPECIFIED: ServiceStatus.STOPPED,
      SERVICE_STATUS_RUNNING: ServiceStatus.RUNNING,
      SERVICE_STATUS_STOPPED: ServiceStatus.STOPPED,
    };
    if (map[t] !== undefined) return map[t];
  }
  return ServiceStatus.STOPPED;
}

function formatServiceTimestamp(v: unknown): string {
  if (v == null || v === '') return '';
  if (typeof v === 'string') return v;
  if (typeof v === 'object' && v !== null && 'seconds' in (v as Record<string, unknown>)) {
    const sec = Number((v as { seconds?: unknown }).seconds);
    if (!Number.isFinite(sec)) return '';
    return new Date(sec * 1000).toISOString();
  }
  return String(v);
}

/** 将 protobuf 返回的服务行规范为表格用 ServiceItem */
function normalizeServiceFromApi(raw: Record<string, unknown> | null | undefined): ServiceItem {
  const r = raw ?? {};
  const volc =
    (r.volc_log_service_id as string) ?? (r.volcLogServiceId as string) ?? '';
  const git = (r.git_repo_url as string) ?? (r.gitRepoUrl as string) ?? '';
  const created = r.created_at ?? r.createdAt;
  const updated = r.updated_at ?? r.updatedAt;
  return {
    id: Number(r.id),
    name: String(r.name ?? ''),
    status: parseServiceStatus(r.status),
    volc_log_service_id: volc,
    git_repo_url: git,
    created_at: formatServiceTimestamp(created),
    updated_at: formatServiceTimestamp(updated),
    description: String(r.description ?? ''),
  };
}

/** ListServices 返回 services（proto）；兼容 items */
function normalizeListServicesReply(raw: Record<string, unknown>): ListServicesReply {
  const list = (raw.services ?? raw.items) as unknown[] | undefined;
  const arr = Array.isArray(list) ? list : [];
  return {
    items: arr.map((row) => normalizeServiceFromApi(row as Record<string, unknown>)),
    total: Number(raw.total ?? 0),
  };
}

export interface CreateServiceParams {
  name: string;
  status: ServiceStatus;
  volc_log_service_id: string;
  git_repo_url: string;
  description: string;
}

export type SaveServiceParams = CreateServiceParams;

export interface UpdateServiceParams {
  name?: string;
  status?: ServiceStatus;
  volc_log_service_id?: string;
  git_repo_url?: string;
  description?: string;
}

// 获取服务列表
export const listServices = async (params: ListServicesParams): Promise<ListServicesReply> => {
  const raw = await requestJSON<Record<string, unknown>>('/services', { method: 'GET', params });
  return normalizeListServicesReply(raw);
};

// 获取服务详情（GetServiceReply.service）
export const getService = async (id: number): Promise<{ item: ServiceItem }> => {
  const raw = await requestJSON<Record<string, unknown>>(`/services/${id}`);
  const s = (raw.service ?? raw.item) as Record<string, unknown> | undefined;
  return { item: normalizeServiceFromApi(s) };
};

// 按名称保存服务（CreateServiceReply.service；同名更新，不存在新建）
export const saveServiceByName = async (params: SaveServiceParams): Promise<{ item: ServiceItem }> => {
  const raw = await requestJSON<Record<string, unknown>>('/services', { method: 'POST', body: params });
  const s = (raw.service ?? raw.item) as Record<string, unknown> | undefined;
  return { item: normalizeServiceFromApi(s) };
};

// 兼容旧调用
export const createService = async (params: CreateServiceParams): Promise<{ item: ServiceItem }> =>
  saveServiceByName(params);

// 更新服务（UpdateServiceReply.service）
export const updateService = async (
  id: number,
  params: UpdateServiceParams
): Promise<{ item: ServiceItem }> => {
  const raw = await requestJSON<Record<string, unknown>>(`/services/${id}`, { method: 'PUT', body: params });
  const s = (raw.service ?? raw.item) as Record<string, unknown> | undefined;
  return { item: normalizeServiceFromApi(s) };
};

// 删除服务
export const deleteService = (id: number): Promise<{ success: boolean }> =>
  requestJSON(`/services/${id}`, { method: 'DELETE' });

export const serviceStatusOptions = [
  { value: ServiceStatus.RUNNING, label: '运行中', color: 'green' },
  { value: ServiceStatus.STOPPED, label: '已停止', color: 'red' },
];
