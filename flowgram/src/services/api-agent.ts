import { requestJSON } from './http';

export interface SkillItem {
  name: string;
  path: string;
  size: number;
  updatedAt: string;
}

/**
 * 列表/详情接口经 normalize 后统一为 snake_case（表单用）。
 * 原始 HTTP 为 protojson：stdio 字段为 stdioCommand / stdioArgsJson / stdioEnvJson。
 */
export interface MCPConfigItem {
  id: number;
  name: string;
  server: string;
  endpoint: string;
  headers?: Record<string, any>;
  enabled: boolean;
  description: string;
  /** 缺省或空视为 http */
  transport?: string;
  stdio_command?: string;
  stdio_args_json?: string;
  stdio_env_json?: string;
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
  transport?: string;
  stdio_command?: string;
  stdio_args_json?: string;
  stdio_env_json?: string;
}

/** 将 protojson 响应（camelCase）合并为表单使用的 MCPConfigItem */
export function normalizeMcpConfigItem(raw: Record<string, unknown>): MCPConfigItem {
  const r = raw as Record<string, any>;
  const headersRaw = r.headers;
  const headers =
    headersRaw && typeof headersRaw === 'object' && !Array.isArray(headersRaw)
      ? (headersRaw as Record<string, any>)
      : {};
  const stdioCmd = r.stdioCommand ?? r.stdio_command;
  const stdioArgs = r.stdioArgsJson ?? r.stdio_args_json;
  const stdioEnv = r.stdioEnvJson ?? r.stdio_env_json;
  return {
    id: Number(r.id),
    name: String(r.name ?? ''),
    server: String(r.server ?? ''),
    endpoint: String(r.endpoint ?? ''),
    headers,
    enabled: Boolean(r.enabled),
    description: String(r.description ?? ''),
    transport: r.transport != null && r.transport !== '' ? String(r.transport) : undefined,
    stdio_command: stdioCmd != null ? String(stdioCmd) : '',
    stdio_args_json:
      stdioArgs != null && String(stdioArgs).trim() !== '' ? String(stdioArgs) : undefined,
    stdio_env_json:
      stdioEnv != null && String(stdioEnv).trim() !== '' ? String(stdioEnv) : undefined,
    createdAt:
      r.createdAt != null
        ? String(r.createdAt)
        : r.created_at != null
        ? String(r.created_at)
        : undefined,
    updatedAt:
      r.updatedAt != null
        ? String(r.updatedAt)
        : r.updated_at != null
        ? String(r.updated_at)
        : undefined,
  };
}

/** Kratos/protojson 请求体须使用 proto 的 json_name（camelCase） */
function mcpPayloadToApiBody(payload: MCPConfigPayload): Record<string, unknown> {
  return {
    name: payload.name,
    server: payload.server,
    endpoint: payload.endpoint,
    headers: payload.headers ?? {},
    enabled: payload.enabled,
    description: payload.description,
    transport: payload.transport,
    stdioCommand: payload.stdio_command,
    stdioArgsJson: payload.stdio_args_json,
    stdioEnvJson: payload.stdio_env_json,
  };
}

/** POST /admin/mcps/:id/test：http 走 SSE；stdio 拉起子进程；均 initialize + tools/list */
export interface TestMcpConfigReply {
  ok: boolean;
  message: string;
  toolNames?: string[];
  serverName?: string;
  protocolVersion?: string;
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
  requestJSON<{ items: Record<string, unknown>[] }>('/admin/mcps').then((r) =>
    (r.items || []).map((row) => normalizeMcpConfigItem(row))
  );

export const createMCPConfig = (payload: MCPConfigPayload) =>
  requestJSON<Record<string, unknown>>('/admin/mcps', {
    method: 'POST',
    body: mcpPayloadToApiBody(payload),
  }).then((row) => normalizeMcpConfigItem(row));

export const updateMCPConfig = (id: number, payload: MCPConfigPayload) =>
  requestJSON(`/admin/mcps/${id}`, { method: 'PUT', body: mcpPayloadToApiBody(payload) });

export const deleteMCPConfig = (id: number) =>
  requestJSON(`/admin/mcps/${id}`, { method: 'DELETE' });

export const testMCPConfig = (id: number) =>
  requestJSON<TestMcpConfigReply>(`/admin/mcps/${id}/test`, { method: 'POST', body: {} });

/** POST /admin/terminal/run：服务端执行 shell（cwd 须为白名单绝对路径，空则 /app） */
export interface RunTerminalReply {
  exitCode?: number;
  exit_code?: number;
  stdout?: string;
  stderr?: string;
  diagnostic?: string;
}

export const runTerminal = async (command: string, cwd?: string): Promise<RunTerminalReply> => {
  const raw = await requestJSON<RunTerminalReply>('/admin/terminal/run', {
    method: 'POST',
    body: { command, cwd: cwd ?? '' },
  });
  const exit = raw.exitCode ?? raw.exit_code;
  return { ...raw, exitCode: exit != null ? Number(exit) : 0 };
};

/** GET /admin/lark-cli/config：服务端 lark-cli 配置文件与终端超时（来自 configs） */
export interface LarkCliConfigMeta {
  content: string;
  resolvedPath: string;
  exists: boolean;
  terminalExecTimeoutSec: number;
}

export async function getLarkCliConfig(): Promise<LarkCliConfigMeta> {
  const raw = await requestJSON<Record<string, unknown>>('/admin/lark-cli/config');
  const sec = Number(raw.terminalExecTimeoutSec ?? raw.terminal_exec_timeout_sec ?? 120);
  return {
    content: String(raw.content ?? ''),
    resolvedPath: String(raw.resolvedPath ?? raw.resolved_path ?? ''),
    exists: Boolean(raw.exists),
    terminalExecTimeoutSec: Number.isFinite(sec) && sec > 0 ? sec : 120,
  };
}

/** PUT /admin/lark-cli/config：写入合法 JSON（默认路径见 get 返回的 resolvedPath） */
export async function saveLarkCliConfig(jsonRaw: string): Promise<void> {
  await requestJSON('/admin/lark-cli/config', {
    method: 'PUT',
    body: { jsonRaw },
  });
}

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
    raw.created_at != null
      ? String(raw.created_at)
      : raw.createdAt != null
      ? String(raw.createdAt)
      : undefined;
  const updated =
    raw.updated_at != null
      ? String(raw.updated_at)
      : raw.updatedAt != null
      ? String(raw.updatedAt)
      : undefined;
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
    raw.created_at != null
      ? String(raw.created_at)
      : raw.createdAt != null
      ? String(raw.createdAt)
      : undefined;
  const updated =
    raw.updated_at != null
      ? String(raw.updated_at)
      : raw.updatedAt != null
      ? String(raw.updatedAt)
      : undefined;
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
    (Array.isArray(r.items) ? r.items : []).map((row) =>
      normalizeLlmConfigItem(row as Record<string, unknown>)
    )
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
