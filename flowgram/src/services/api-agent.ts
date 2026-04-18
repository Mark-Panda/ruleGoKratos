import { requestJSON } from './http';

export interface SkillItem {
  name: string;
  path: string;
  size: number;
  updatedAt: string;
}

export interface MCPConfigItem {
  id: number;
  name: string;
  server: string;
  endpoint: string;
  headers?: Record<string, any>;
  enabled: boolean;
  description: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface MCPConfigPayload {
  name: string;
  server: string;
  endpoint: string;
  headers?: Record<string, any>;
  enabled: boolean;
  description: string;
}

export const listSkills = () => requestJSON<{ root: string; items: SkillItem[] }>('/admin/skills');

export const uploadSkill = async (file: File, path?: string) => {
  const buf = await file.arrayBuffer();
  const bytes = new Uint8Array(buf);
  let binary = '';
  const chunk = 0x8000;
  for (let i = 0; i < bytes.length; i += chunk) {
    binary += String.fromCharCode(...bytes.subarray(i, i + chunk));
  }
  const contentBase64 = btoa(binary);
  return requestJSON<{ path: string }>('/admin/skills/upload', {
    method: 'POST',
    body: {
      path: path || file.name,
      contentBase64,
    },
  });
};

export const listMCPConfigs = () =>
  requestJSON<{ items: MCPConfigItem[] }>('/admin/mcps').then((r) => r.items || []);

export const createMCPConfig = (payload: MCPConfigPayload) =>
  requestJSON<MCPConfigItem>('/admin/mcps', { method: 'POST', body: payload });

export const updateMCPConfig = (id: number, payload: MCPConfigPayload) =>
  requestJSON(`/admin/mcps/${id}`, { method: 'PUT', body: payload });

export const deleteMCPConfig = (id: number) =>
  requestJSON(`/admin/mcps/${id}`, { method: 'DELETE' });

/** 一条模型记录（隶属于某个 LLM 配置，共享凭证） */
export interface LlmModelEntryItem {
  id: number;
  configId: number;
  modelName: string;
  description: string;
  enabled: boolean;
  createdAt?: string;
  updatedAt?: string;
}

/** LLM 配置（名称、厂商、BaseURL、API Key 等），其下可挂多条模型 ID */
export interface LlmConfigItem {
  id: number;
  name: string;
  provider: string;
  baseUrl: string;
  apiKey: string;
  enabled: boolean;
  description: string;
  models: LlmModelEntryItem[];
  createdAt?: string;
  updatedAt?: string;
}

export interface LlmConfigPayload {
  name: string;
  provider: string;
  baseUrl: string;
  apiKey: string;
  enabled: boolean;
  description: string;
  /** 仅创建配置时提交，可一次写入多条模型 ID */
  models?: LlmModelEntryPayload[];
}

export interface LlmModelEntryPayload {
  modelName: string;
  description: string;
  enabled: boolean;
}

/**
 * 后端 JSON 来自 protobuf 生成结构体的 `encoding/json` 标签，嵌套字段多为 snake_case（如 model_name、base_url）。
 * 前端约定 camelCase，不映射则下拉选项文案为空、enabled 判断异常。
 */
function normalizeLlmModelEntryItem(raw: Record<string, unknown>): LlmModelEntryItem {
  const created =
    raw.created_at != null ? String(raw.created_at) : raw.createdAt != null ? String(raw.createdAt) : undefined;
  const updated =
    raw.updated_at != null ? String(raw.updated_at) : raw.updatedAt != null ? String(raw.updatedAt) : undefined;
  return {
    id: Number(raw.id ?? 0),
    configId: Number(raw.config_id ?? raw.configId ?? 0),
    modelName: String(raw.model_name ?? raw.modelName ?? '').trim(),
    description: String(raw.description ?? ''),
    enabled: raw.enabled === true,
    ...(created != null ? { createdAt: created } : {}),
    ...(updated != null ? { updatedAt: updated } : {}),
  };
}

function normalizeLlmConfigItem(raw: Record<string, unknown>): LlmConfigItem {
  const modelsRaw = raw.models;
  const models = Array.isArray(modelsRaw)
    ? modelsRaw.map((m) => normalizeLlmModelEntryItem(m as Record<string, unknown>))
    : [];
  const created =
    raw.created_at != null ? String(raw.created_at) : raw.createdAt != null ? String(raw.createdAt) : undefined;
  const updated =
    raw.updated_at != null ? String(raw.updated_at) : raw.updatedAt != null ? String(raw.updatedAt) : undefined;
  return {
    id: Number(raw.id ?? 0),
    name: String(raw.name ?? ''),
    provider: String(raw.provider ?? ''),
    baseUrl: String(raw.base_url ?? raw.baseUrl ?? ''),
    apiKey: String(raw.api_key ?? raw.apiKey ?? ''),
    enabled: raw.enabled === true,
    description: String(raw.description ?? ''),
    models,
    ...(created != null ? { createdAt: created } : {}),
    ...(updated != null ? { updatedAt: updated } : {}),
  };
}

export const listLlmConfigs = () =>
  requestJSON<{ items: unknown[] }>('/admin/llm-configs').then((r) =>
    (Array.isArray(r.items) ? r.items : []).map((row) => normalizeLlmConfigItem(row as Record<string, unknown>))
  );

export const createLlmConfig = (payload: LlmConfigPayload) =>
  requestJSON<unknown>('/admin/llm-configs', { method: 'POST', body: payload }).then((row) =>
    normalizeLlmConfigItem(row as Record<string, unknown>)
  );

export const updateLlmConfig = (id: number, payload: LlmConfigPayload) =>
  requestJSON(`/admin/llm-configs/${id}`, { method: 'PUT', body: payload });

export const deleteLlmConfig = (id: number) =>
  requestJSON(`/admin/llm-configs/${id}`, { method: 'DELETE' });

export const createLlmModelEntry = (configId: number, payload: LlmModelEntryPayload) =>
  requestJSON<unknown>(`/admin/llm-configs/${configId}/models`, {
    method: 'POST',
    body: payload,
  }).then((row) => normalizeLlmModelEntryItem(row as Record<string, unknown>));

/** 服务端返回空的 UpdateLlmModelEntryReply，不做 normalize */
export const updateLlmModelEntry = (id: number, payload: LlmModelEntryPayload) =>
  requestJSON(`/admin/llm-model-entries/${id}`, { method: 'PUT', body: payload });

export const deleteLlmModelEntry = (id: number) =>
  requestJSON(`/admin/llm-model-entries/${id}`, { method: 'DELETE' });
