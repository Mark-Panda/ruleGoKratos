import { describe, expect, it } from 'vitest';

import {
  createDefaultSchemeConfig,
  normalizeSchemeConfig,
  resolveSchemeBindAgentsForSave,
} from '../api-playground';

describe('scheme config helpers', () => {
  it('creates mode-specific defaults for supervision', () => {
    const config = createDefaultSchemeConfig('supervision');

    expect(config.maxIterations).toBe(32);
    expect(config.maxToolCalls).toBe(64);
    expect(config.timeoutSeconds).toBe(300);
    expect(config.modeConfig?.supervisionConfig).toEqual({
      supervisorAgent: '',
      workerAgents: [],
      checkInterval: 15,
    });
  });

  it('keeps only current mode config when normalizing', () => {
    const config = normalizeSchemeConfig('router_expert', {
      maxIterations: 12,
      maxToolCalls: 24,
      timeoutSeconds: 120,
      finalizerPrompt: '整理输出',
      modeConfig: {
        routerConfig: {
          fallbackAgent: 'designer',
          routingPrompt: 'route by specialty',
        },
        supervisionConfig: {
          supervisorAgent: 'supervisor',
          workerAgents: ['engineer'],
          checkInterval: 20,
        },
      },
    });

    expect(config.modeConfig?.routerConfig).toEqual({
      fallbackAgent: 'designer',
      routingPrompt: 'route by specialty',
    });
    expect(config.modeConfig?.supervisionConfig).toBeUndefined();
  });

  it('rebuilds bindAgents when editing mode changes', () => {
    const bindAgents = resolveSchemeBindAgentsForSave({
      mode: 'plan_exec',
      originalMode: 'router_expert',
      existingBindAgents: [
        { agentId: 'designer', role: '设计师' },
        { agentId: 'pm', role: '产品经理' },
      ],
      pool: {
        id: 'pool-1',
        name: '默认池',
        description: '',
        createdAt: '',
        updatedAt: '',
        agents: [
          {
            id: 'planner',
            name: '规划师',
            role: '',
            desc: '',
            model: '',
            tools: [],
            enabled: true,
            priority: 1,
          },
          {
            id: 'designer',
            name: '设计师',
            role: '',
            desc: '',
            model: '',
            tools: [],
            enabled: true,
            priority: 1,
          },
          {
            id: 'engineer',
            name: '工程师',
            role: '',
            desc: '',
            model: '',
            tools: [],
            enabled: true,
            priority: 1,
          },
        ],
      },
    });

    expect(bindAgents.map((item) => item.agentId)).toEqual(['planner', 'designer', 'engineer']);
  });
});
