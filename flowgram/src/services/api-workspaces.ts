import { requestJSON } from './http';

export interface WorkspaceRepoItem {
  url: string;
  dir: string;
}

export interface WorkspaceItem {
  id: string;
  name: string;
  description: string;
  rootDir: string;
  configFile: string;
  repositoryUrls: string[];
  repositories: WorkspaceRepoItem[];
  createdAt: string;
  updatedAt: string;
}

export interface WorkspacePayload {
  id?: string;
  name: string;
  description?: string;
  repositoryUrls: string[];
}

export const listWorkspaces = async () => {
  const r = await requestJSON<{ items: WorkspaceItem[] }>('/admin/workspaces');
  return r.items || [];
};

export const getWorkspace = async (id: string) => {
  const r = await requestJSON<{ item: WorkspaceItem }>(
    `/admin/workspaces/${encodeURIComponent(id)}`
  );
  return r.item;
};

export const createWorkspace = async (body: WorkspacePayload) =>
  requestJSON<{ item: WorkspaceItem }>('/admin/workspaces', { method: 'POST', body });

export const updateWorkspace = async (id: string, body: WorkspacePayload) =>
  requestJSON<{ item: WorkspaceItem }>(`/admin/workspaces/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body,
  });

export const syncWorkspaceRepos = async (id: string) =>
  requestJSON<{ item: WorkspaceItem; message: string }>(
    `/admin/workspaces/${encodeURIComponent(id)}/sync`,
    {
      method: 'POST',
      body: {},
    }
  );

export const deleteWorkspace = async (id: string) =>
  requestJSON<{ ok: string }>(`/admin/workspaces/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  });
