/**
 * 可编排 Agent（系统提示、SKILL、MCP、模型站点与范围）
 */

import { requestJSON } from './http';

export interface SkillPackageItem {
  id: string;
  skillFileCount: number;
}

export interface ManagedAgentItem {
  id: number;
  name: string;
  description: string;
  systemPrompt: string;
  workspaceId?: string;
  skillPackageIds: string[];
  mcpIds: number[];
  llmConfigId: number;
  modelScope: 'all' | 'explicit';
  modelEntryIds: number[];
  enabled: boolean;
  createdAt?: string;
  updatedAt?: string;
}

export interface ManagedAgentPayload {
  name: string;
  description?: string;
  systemPrompt?: string;
  workspaceId?: string;
  skillPackageIds?: string[];
  mcpIds?: number[];
  llmConfigId: number;
  modelScope: 'all' | 'explicit';
  modelEntryIds?: number[];
  enabled?: boolean;
}

function normalizeManagedAgentItem(raw: Record<string, unknown>): ManagedAgentItem {
  const modelEntryIdsRaw = raw.modelEntryIds ?? raw.model_entry_ids;
  const modelEntryIds = Array.isArray(modelEntryIdsRaw)
    ? modelEntryIdsRaw.map((id) => Number(id))
    : [];
  const mcpIdsRaw = raw.mcpIds ?? raw.mcp_ids;
  const mcpIds = Array.isArray(mcpIdsRaw) ? mcpIdsRaw.map((id) => Number(id)) : [];
  const skillPkgRaw = raw.skillPackageIds ?? raw.skill_package_ids;
  const skillPackageIds = Array.isArray(skillPkgRaw)
    ? skillPkgRaw.map((s) => String(s))
    : [];
  const created =
    raw.createdAt != null
      ? String(raw.createdAt)
      : raw.created_at != null
        ? String(raw.created_at)
        : undefined;
  const updated =
    raw.updatedAt != null
      ? String(raw.updatedAt)
      : raw.updated_at != null
        ? String(raw.updated_at)
        : undefined;
  return {
    id: Number(raw.id ?? 0),
    name: String(raw.name ?? ''),
    description: String(raw.description ?? ''),
    systemPrompt: String(raw.systemPrompt ?? raw.system_prompt ?? ''),
    workspaceId: String(raw.workspaceId ?? raw.workspace_id ?? ''),
    skillPackageIds,
    mcpIds,
    llmConfigId: Number(raw.llmConfigId ?? raw.llm_config_id ?? 0),
    modelScope: (raw.modelScope ?? raw.model_scope ?? 'all') as 'all' | 'explicit',
    modelEntryIds,
    enabled: raw.enabled === true,
    ...(created != null ? { createdAt: created } : {}),
    ...(updated != null ? { updatedAt: updated } : {}),
  };
}

export const listManagedAgents = async () => {
  const r = await requestJSON<{ items: Record<string, unknown>[] }>('/admin/managed-agents');
  return (Array.isArray(r.items) ? r.items : []).map(normalizeManagedAgentItem);
};

export const listEnabledManagedAgents = async () =>
  (await listManagedAgents()).filter((item) => item.enabled !== false);

export const getManagedAgent = async (id: number) => {
  const r = await requestJSON<{ item: Record<string, unknown> }>(
    `/admin/managed-agents/${encodeURIComponent(String(id))}`
  );
  return normalizeManagedAgentItem(r.item || {});
};

export const createManagedAgent = async (body: ManagedAgentPayload) =>
  requestJSON<{ item: ManagedAgentItem }>('/admin/managed-agents', { method: 'POST', body });

export const updateManagedAgent = async (id: number, body: ManagedAgentPayload) =>
  requestJSON<{ item: ManagedAgentItem }>(
    `/admin/managed-agents/${encodeURIComponent(String(id))}`,
    { method: 'PUT', body }
  );

export const deleteManagedAgent = async (id: number) =>
  requestJSON<{ ok: boolean }>(`/admin/managed-agents/${encodeURIComponent(String(id))}`, {
    method: 'DELETE',
  });

/** 技能包列表（按 skill 根目录下路径首段聚合，与官方 Skill name 一致） */
export const listSkillPackages = async () => {
  const r = await requestJSON<{ root: string; items: SkillPackageItem[] }>('/admin/skill-packages');
  return r;
};
