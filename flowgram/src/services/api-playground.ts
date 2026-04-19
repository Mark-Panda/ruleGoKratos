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

export interface RouterSchemeConfig {
  fallbackAgent?: string;
  routingPrompt?: string;
}

export interface PlanExecSchemeConfig {
  plannerAgent?: string;
  executionOrder?: string[];
}

export interface SupervisionSchemeConfig {
  supervisorAgent?: string;
  workerAgents?: string[];
  checkInterval?: number;
}

export interface PeerHandoffSchemeConfig {
  entryAgent?: string;
  meshAgents?: string[];
  handoffRules?: string;
}

export interface SchemeModeConfig {
  routerConfig?: RouterSchemeConfig;
  planExecConfig?: PlanExecSchemeConfig;
  supervisionConfig?: SupervisionSchemeConfig;
  peerHandoffConfig?: PeerHandoffSchemeConfig;
}

export interface SchemeConfig {
  maxIterations: number;
  maxToolCalls: number;
  timeoutSeconds: number;
  finalizerPrompt?: string;
  modeConfig?: SchemeModeConfig;
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

export type RuntimeRunStatus =
  | 'pending'
  | 'ready'
  | 'running'
  | 'waiting_recovery'
  | 'completed'
  | 'failed'
  | 'cancelled';

export type RuntimeStepKind = 'route' | 'agent' | 'parallel' | 'review' | 'handoff' | 'finalize';

export type RuntimeStepStatus = 'pending' | 'ready' | 'running' | 'succeeded' | 'failed' | 'skipped';

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

export interface RuntimeRun {
  runId: string;
  schemeId: string;
  planId: string;
  status: RuntimeRunStatus;
  currentStepIds?: string[];
  /** 最近一次成功步骤的检查点标识（用于 retry_from_checkpoint） */
  lastCheckpointId?: string;
  failureSummary?: string;
  startedAt?: string;
  finishedAt?: string;
  userInput?: string;
  finalOutput?: string;
}

export interface RuntimeStep {
  stepId: string;
  kind: RuntimeStepKind;
  name: string;
  status: RuntimeStepStatus;
  agentBinding?: string;
  failureSummary?: string;
  inputRefs?: string[];
  outputRef?: string;
}

export interface RuntimeArtifact {
  artifactId: string;
  type: string;
  producerStepId: string;
  summary: string;
}

export type RecoveryActionType =
  | 'retry_step'
  | 'reroute_step'
  | 'skip_step'
  | 'retry_from_checkpoint';

export interface RecoveryAction {
  id: string;
  type: RecoveryActionType;
  stepId: string;
  /** 语义随 type 变化：reroute 时为 Agent ID；retry_from_checkpoint 时为检查点步骤 ID */
  targetRef?: string;
  reason: string;
}

export interface RuntimeRunDetail {
  run: RuntimeRun;
  steps: RuntimeStep[];
  artifacts: RuntimeArtifact[];
  recoveryActions: RecoveryAction[];
}

const PLAN_EXEC_BIND_TEMPLATE: AgentBinding[] = [
  { agentId: 'planner', role: '规划师' },
  { agentId: 'designer', role: '设计师' },
  { agentId: 'pm', role: '产品经理' },
  { agentId: 'engineer', role: '工程师' },
];

const DEFAULT_SCHEME_CONFIG_BASE = {
  maxIterations: 32,
  maxToolCalls: 64,
  timeoutSeconds: 300,
  finalizerPrompt: '',
} satisfies Omit<SchemeConfig, 'modeConfig'>;

const cloneStringArray = (items?: string[]) => (items ? [...items] : []);
const cloneAgentBinding = (agent: AgentBinding): AgentBinding => ({
  ...agent,
  tools: cloneStringArray(agent.tools),
});

export const createDefaultSchemeModeConfig = (mode: CollaborationMode): SchemeModeConfig => {
  switch (mode) {
    case 'router_expert':
      return {
        routerConfig: {
          fallbackAgent: '',
          routingPrompt: '',
        },
      };
    case 'plan_exec':
      return {
        planExecConfig: {
          plannerAgent: '',
          executionOrder: [],
        },
      };
    case 'supervision':
      return {
        supervisionConfig: {
          supervisorAgent: '',
          workerAgents: [],
          checkInterval: 15,
        },
      };
    case 'peer_handoff':
      return {
        peerHandoffConfig: {
          entryAgent: '',
          meshAgents: [],
          handoffRules: '',
        },
      };
  }
};

export const createDefaultSchemeConfig = (mode: CollaborationMode): SchemeConfig => ({
  ...DEFAULT_SCHEME_CONFIG_BASE,
  modeConfig: createDefaultSchemeModeConfig(mode),
});

export const normalizeSchemeConfig = (mode: CollaborationMode, config?: Partial<SchemeConfig>): SchemeConfig => {
  const defaults = createDefaultSchemeConfig(mode);

  switch (mode) {
    case 'router_expert':
      return {
        maxIterations: config?.maxIterations ?? defaults.maxIterations,
        maxToolCalls: config?.maxToolCalls ?? defaults.maxToolCalls,
        timeoutSeconds: config?.timeoutSeconds ?? defaults.timeoutSeconds,
        finalizerPrompt: config?.finalizerPrompt ?? defaults.finalizerPrompt,
        modeConfig: {
          routerConfig: {
            fallbackAgent: config?.modeConfig?.routerConfig?.fallbackAgent ?? defaults.modeConfig?.routerConfig?.fallbackAgent ?? '',
            routingPrompt: config?.modeConfig?.routerConfig?.routingPrompt ?? defaults.modeConfig?.routerConfig?.routingPrompt ?? '',
          },
        },
      };
    case 'plan_exec':
      return {
        maxIterations: config?.maxIterations ?? defaults.maxIterations,
        maxToolCalls: config?.maxToolCalls ?? defaults.maxToolCalls,
        timeoutSeconds: config?.timeoutSeconds ?? defaults.timeoutSeconds,
        finalizerPrompt: config?.finalizerPrompt ?? defaults.finalizerPrompt,
        modeConfig: {
          planExecConfig: {
            plannerAgent: config?.modeConfig?.planExecConfig?.plannerAgent ?? defaults.modeConfig?.planExecConfig?.plannerAgent ?? '',
            executionOrder: cloneStringArray(config?.modeConfig?.planExecConfig?.executionOrder ?? defaults.modeConfig?.planExecConfig?.executionOrder),
          },
        },
      };
    case 'supervision':
      return {
        maxIterations: config?.maxIterations ?? defaults.maxIterations,
        maxToolCalls: config?.maxToolCalls ?? defaults.maxToolCalls,
        timeoutSeconds: config?.timeoutSeconds ?? defaults.timeoutSeconds,
        finalizerPrompt: config?.finalizerPrompt ?? defaults.finalizerPrompt,
        modeConfig: {
          supervisionConfig: {
            supervisorAgent: config?.modeConfig?.supervisionConfig?.supervisorAgent ?? defaults.modeConfig?.supervisionConfig?.supervisorAgent ?? '',
            workerAgents: cloneStringArray(config?.modeConfig?.supervisionConfig?.workerAgents ?? defaults.modeConfig?.supervisionConfig?.workerAgents),
            checkInterval: config?.modeConfig?.supervisionConfig?.checkInterval ?? defaults.modeConfig?.supervisionConfig?.checkInterval ?? 15,
          },
        },
      };
    case 'peer_handoff':
      return {
        maxIterations: config?.maxIterations ?? defaults.maxIterations,
        maxToolCalls: config?.maxToolCalls ?? defaults.maxToolCalls,
        timeoutSeconds: config?.timeoutSeconds ?? defaults.timeoutSeconds,
        finalizerPrompt: config?.finalizerPrompt ?? defaults.finalizerPrompt,
        modeConfig: {
          peerHandoffConfig: {
            entryAgent: config?.modeConfig?.peerHandoffConfig?.entryAgent ?? defaults.modeConfig?.peerHandoffConfig?.entryAgent ?? '',
            meshAgents: cloneStringArray(config?.modeConfig?.peerHandoffConfig?.meshAgents ?? defaults.modeConfig?.peerHandoffConfig?.meshAgents),
            handoffRules: config?.modeConfig?.peerHandoffConfig?.handoffRules ?? defaults.modeConfig?.peerHandoffConfig?.handoffRules ?? '',
          },
        },
      };
  }
};

export const buildSchemeBindAgents = (mode: CollaborationMode, pool?: AgentPool): AgentBinding[] => {
  const defs = pool?.agents || [];
  const byId = new Map(defs.map(agent => [agent.id, agent]));

  if (mode === 'plan_exec') {
    const list: AgentBinding[] = [];
    for (const template of PLAN_EXEC_BIND_TEMPLATE) {
      const def = byId.get(template.agentId);
      if (def) {
        list.push({ agentId: def.id, role: def.name });
      }
    }
    if (list.length > 0) {
      return list;
    }
  }

  return defs.map(agent => ({ agentId: agent.id, role: agent.name }));
};

const filterBindAgentsForMode = (mode: CollaborationMode, bindAgents?: AgentBinding[]): AgentBinding[] => {
  const current = bindAgents ? bindAgents.map(cloneAgentBinding) : [];
  if (mode !== 'plan_exec') {
    return current;
  }

  const byId = new Map(current.map(agent => [agent.agentId, agent]));
  return PLAN_EXEC_BIND_TEMPLATE
    .map(template => byId.get(template.agentId))
    .filter((agent): agent is AgentBinding => agent !== undefined);
};

export const resolveSchemeBindAgentsForSave = (input: {
  mode: CollaborationMode;
  originalMode?: CollaborationMode;
  existingBindAgents?: AgentBinding[];
  pool?: AgentPool;
}): AgentBinding[] => {
  const existing = input.existingBindAgents ?? [];
  if (!input.originalMode) {
    return buildSchemeBindAgents(input.mode, input.pool);
  }
  if (input.originalMode === input.mode) {
    return existing.map(cloneAgentBinding);
  }

  const rebuilt = buildSchemeBindAgents(input.mode, input.pool);
  if (rebuilt.length > 0) {
    return rebuilt;
  }
  return filterBindAgentsForMode(input.mode, existing);
};

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
  return requestJSON<RuntimeRunDetail>(`/playground/run/${encodeURIComponent(runId)}`);
};

export const getRunEvents = async (runId: string, params?: { limit?: number; offset?: number }) => {
  return requestJSON<{ events: TraceEvent[] }>(`/playground/run/${encodeURIComponent(runId)}/events`, { params });
};

export const applyRecoveryAction = async (
  runId: string,
  actionId: string,
  body?: { targetRef?: string },
) => {
  const payload =
    body?.targetRef != null && body.targetRef.trim() !== ''
      ? { targetRef: body.targetRef.trim() }
      : undefined;
  return requestJSON<{ runId: string; actionId: string; status: string }>(
    `/playground/run/${encodeURIComponent(runId)}/recovery-actions/${encodeURIComponent(actionId)}`,
    { method: 'POST', body: payload },
  );
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
