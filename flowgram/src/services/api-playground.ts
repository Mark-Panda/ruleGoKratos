/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { requestJSON } from './http';

// ========== Types ==========

export interface AgentDefinition {
  id: string;
  name: string;
  role: string;
  desc: string;
  model: string;
  tools: string[];
  enabled: boolean;
  priority: number;
  /** 与「Agent 管理 → Agent 配置」中的 id 对应；非零表示运行模型/SKILL/MCP 以该托管配置为准 */
  managedAgentId?: number;
}

export interface AgentPool {
  id: string;
  name: string;
  description: string;
  agents: AgentDefinition[];
  createdAt: string;
  updatedAt: string;
}

export interface AgentBinding {
  agentId: string;
  role: string;
  model?: string;
  tools?: string[];
}

export interface SchemeConfig {
  maxIterations: number;
  maxToolCalls: number;
  timeoutSeconds: number;
  finalizerPrompt?: string;
}

export interface CollaborationScheme {
  id: string;
  name: string;
  description: string;
  mode: CollaborationMode;
  bindAgents: AgentBinding[];
  config: SchemeConfig;
  enabled: boolean;
  enableFinalizer: boolean;
  createdAt: string;
  updatedAt: string;
}

export type CollaborationMode = 'router_expert' | 'plan_exec' | 'supervision' | 'peer_handoff';

export interface CollaborationModeInfo {
  id: CollaborationMode;
  name: string;
  description: string;
}

export interface TraceEvent {
  id: string;
  runId: string;
  timestamp: number;
  type: string;
  agentId: string;
  nodeId: string;
  taskDesc: string;
  message: string;
  metadata: Record<string, string>;
}

export interface TraceRun {
  id: string;
  runId: string;
  schemeId: string;
  userInput: string;
  status: 'running' | 'completed' | 'failed';
  startTime: string;
  endTime?: string;
  totalMs: number;
  events: TraceEvent[];
  finalOutput: string;
}

// ========== Agent Pool APIs ==========

export const listAgentPools = async () => {
  return requestJSON<{ pools: AgentPool[] }>('/playground/pools');
};

export const getAgentPool = async (id: string) => {
  return requestJSON<{ pool: AgentPool }>(`/playground/pools/${encodeURIComponent(id)}`);
};

export const createAgentPool = async (data: { name: string; description: string; agents?: AgentDefinition[] }) => {
  return requestJSON<{ pool: AgentPool }>('/playground/pools', { method: 'POST', body: data });
};

export const updateAgentPool = async (id: string, data: { name?: string; description?: string; agents?: AgentDefinition[] }) => {
  return requestJSON<{ pool: AgentPool }>(`/playground/pools/${encodeURIComponent(id)}`, { method: 'PUT', body: data });
};

export const deleteAgentPool = async (id: string) => {
  return requestJSON<{ ok: string }>(`/playground/pools/${encodeURIComponent(id)}`, { method: 'DELETE' });
};

// ========== Scheme APIs ==========

export const listSchemes = async () => {
  return requestJSON<{ schemes: CollaborationScheme[] }>('/playground/schemes');
};

export const getScheme = async (id: string) => {
  return requestJSON<{ scheme: CollaborationScheme }>(`/playground/schemes/${encodeURIComponent(id)}`);
};

export const createScheme = async (data: {
  name: string;
  description?: string;
  mode: CollaborationMode;
  bindAgents?: AgentBinding[];
  config?: SchemeConfig;
  enableFinalizer?: boolean;
}) => {
  return requestJSON<{ scheme: CollaborationScheme }>('/playground/schemes', { method: 'POST', body: data });
};

export const updateScheme = async (id: string, data: Partial<{
  name: string;
  description: string;
  mode: CollaborationMode;
  bindAgents: AgentBinding[];
  config: SchemeConfig;
  enableFinalizer: boolean;
}>) => {
  return requestJSON<{ scheme: CollaborationScheme }>(`/playground/schemes/${encodeURIComponent(id)}`, { method: 'PUT', body: data });
};

export const deleteScheme = async (id: string) => {
  return requestJSON<{}>(`/playground/schemes/${encodeURIComponent(id)}`, { method: 'DELETE' });
};

// ========== Run APIs ==========

export const runWorkflow = async (data: { schemeId: string; userInput: string }) => {
  return requestJSON<{ runId: string }>('/playground/run', { method: 'POST', body: data });
};

export const getRun = async (runId: string) => {
  return requestJSON<{ run: TraceRun }>(`/playground/run/${encodeURIComponent(runId)}`);
};

export const getRunEvents = async (runId: string, params?: { limit?: number; offset?: number }) => {
  return requestJSON<{ events: TraceEvent[] }>(`/playground/run/${encodeURIComponent(runId)}/events`, { params });
};

// ========== Mode APIs ==========

export const getCollaborationModes = async () => {
  return requestJSON<{ modes: CollaborationModeInfo[] }>('/playground/modes');
};

// ========== Mode 名称映射 ==========

export const MODE_NAME_MAP: Record<CollaborationMode, string> = {
  router_expert: '路由专家',
  plan_exec: '规划执行',
  supervision: '动态监督',
  peer_handoff: '同伴交接',
};

export const MODE_DESC_MAP: Record<CollaborationMode, string> = {
  router_expert: '根据输入智能路由到最合适的 Agent',
  plan_exec: '规划师拆解任务后依次执行',
  supervision: '监督者并行监控 Workers',
  peer_handoff: 'Agent 之间自主协商交接任务',
};
