/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconLLM from '../../assets/icon-llm.jpg';
import { agentHarnessFormMeta } from './form-meta';

let index = 0;

export const AgentHarnessNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.AgentHarness,
  info: {
    icon: iconLLM,
    description:
      'Agent LLM：与 Chat Harness 一致的工具调用能力，可按节点配置 Skill / MCP / Workspace 工具与白名单；提示词与模型名支持 ${}。',
  },
  meta: {
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: {
      width: 320,
      height: 280,
    },
    defaultExpanded: false,
    expandable: true,
  },
  formMeta: agentHarnessFormMeta,
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.AgentHarness,
      data: {
        title: `AgentLLM_${++index}`,
        positionType: 'middle',
        inputsValues: {
          llmConfigId: { type: 'constant', content: 0 },
          llmModelEntryId: { type: 'constant', content: 0 },
          managedAgentId: { type: 'constant', content: 0 },
          model: { type: 'constant', content: '' },
          userPrompt: { type: 'template', content: '' },
          systemPrompt: {
            type: 'template',
            content:
              'You are a helpful assistant. You may call run_skill and call_mcp_tool when they help answer the user.',
          },
          enableSkillTool: { type: 'constant', content: true },
          enableMcpTool: { type: 'constant', content: true },
          enableUUIDTool: { type: 'constant', content: true },
          enableWorkspaceTools: { type: 'constant', content: false },
          skillAllowlist: { type: 'constant', content: [] as string[] },
          mcpAllowlist: { type: 'constant', content: [] as string[] },
          maxIterations: { type: 'constant', content: 0 },
          maxToolCalls: { type: 'constant', content: 0 },
          toolTimeoutSecs: { type: 'constant', content: 0 },
        },
        inputs: {
          type: 'object',
          required: [
            'model',
            'userPrompt',
            'systemPrompt',
            'enableSkillTool',
            'enableMcpTool',
            'enableUUIDTool',
            'enableWorkspaceTools',
            'skillAllowlist',
            'mcpAllowlist',
            'maxIterations',
            'maxToolCalls',
            'toolTimeoutSecs',
          ],
          properties: {
            model: {
              type: 'string',
              extra: {
                label: '模型名称',
                formComponent: 'prompt-editor',
                description: '留空则用配置默认模型；支持 ${} 模板',
              },
            },
            userPrompt: {
              type: 'string',
              extra: { label: '用户提示词', formComponent: 'prompt-editor' },
            },
            systemPrompt: {
              type: 'string',
              extra: { label: '系统提示词', formComponent: 'prompt-editor' },
            },
            enableSkillTool: {
              type: 'boolean',
              extra: { label: '启用 run_skill', description: '允许模型调用 Skill 执行器' },
            },
            enableMcpTool: {
              type: 'boolean',
              extra: { label: '启用 call_mcp_tool', description: '允许模型调用 MCP' },
            },
            enableUUIDTool: {
              type: 'boolean',
              extra: { label: '启用 generate_uuid' },
            },
            enableWorkspaceTools: {
              type: 'boolean',
              extra: {
                label: '启用 Workspace 工具',
                description: '读/写文件与 shell（与 Chat Agent 一致）',
              },
            },
            skillAllowlist: {
              type: 'array',
              items: { type: 'string' },
              extra: {
                label: 'Skill 白名单',
                description: '由侧边栏勾选维护（string[]）；空=不限制',
              },
            },
            mcpAllowlist: {
              type: 'array',
              items: { type: 'string' },
              extra: {
                label: 'MCP 白名单',
                description: '勾选 MCP 后写入 server:*；空=不限制',
              },
            },
            maxIterations: {
              type: 'number',
              extra: { label: '最大迭代轮次', description: '0 表示使用服务默认' },
            },
            maxToolCalls: {
              type: 'number',
              extra: { label: '最大工具调用次数', description: '0 表示使用服务默认' },
            },
            toolTimeoutSecs: {
              type: 'number',
              extra: { label: '单次工具超时(秒)', description: '0 表示使用服务默认' },
            },
          },
        },
        outputs: {
          type: 'object',
          properties: {},
        },
      },
    } as any;
  },
};
