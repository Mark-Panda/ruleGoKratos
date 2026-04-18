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
  skillPackageIds?: string[];
  mcpIds?: number[];
  llmConfigId: number;
  modelScope: 'all' | 'explicit';
  modelEntryIds?: number[];
  enabled?: boolean;
}

export const listManagedAgents = async () => {
  const r = await requestJSON<{ items: ManagedAgentItem[] }>('/admin/managed-agents');
  return r.items || [];
};

export const getManagedAgent = async (id: number) => {
  const r = await requestJSON<{ item: ManagedAgentItem }>(
    `/admin/managed-agents/${encodeURIComponent(String(id))}`
  );
  return r.item;
};

export const createManagedAgent = async (body: ManagedAgentPayload) =>
  requestJSON<{ item: ManagedAgentItem }>('/admin/managed-agents', { method: 'POST', body });

export const updateManagedAgent = async (id: number, body: ManagedAgentPayload) =>
  requestJSON<{ item: ManagedAgentItem }>(
    `/admin/managed-agents/${encodeURIComponent(String(id))}`,
    { method: 'PUT', body }
  );

export const deleteManagedAgent = async (id: number) =>
  requestJSON<{ ok: string }>(`/admin/managed-agents/${encodeURIComponent(String(id))}`, {
    method: 'DELETE',
  });

/** 技能包列表（按 skill 根目录下路径首段聚合，与 run_skill 命名一致） */
export const listSkillPackages = async () => {
  const r = await requestJSON<{ root: string; items: SkillPackageItem[] }>(
    '/admin/skill-packages'
  );
  return r;
};
