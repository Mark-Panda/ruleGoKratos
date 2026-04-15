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
