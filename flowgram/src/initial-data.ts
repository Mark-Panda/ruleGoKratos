/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { FlowDocumentJSON } from './typings';

/** 与 `AgentHarnessNodeRegistry.onAdd` 对齐；示例画布在移除独立 LLM 组件后改用本结构。 */
function demoAgentHarnessNodeData(opts: {
  title: string;
  userPrompt: string;
  systemPrompt?: string;
}) {
  const defaultSys =
    'You are a helpful assistant. You may call run_skill and call_mcp_tool when they help answer the user.';
  return {
    title: opts.title,
    positionType: 'middle',
    inputsValues: {
      model: { type: 'constant' as const, content: '' },
      userPrompt: { type: 'template' as const, content: opts.userPrompt },
      systemPrompt: { type: 'template' as const, content: opts.systemPrompt ?? defaultSys },
      enableSkillTool: { type: 'constant' as const, content: true },
      enableMcpTool: { type: 'constant' as const, content: true },
      enableWorkspaceTools: { type: 'constant' as const, content: false },
      skillAllowlist: { type: 'constant' as const, content: [] as string[] },
      mcpAllowlist: { type: 'constant' as const, content: [] as string[] },
      maxIterations: { type: 'constant' as const, content: 0 },
      maxToolCalls: { type: 'constant' as const, content: 0 },
      toolTimeoutSecs: { type: 'constant' as const, content: 0 },
    },
    inputs: {
      type: 'object',
      required: [
        'model',
        'userPrompt',
        'systemPrompt',
        'enableSkillTool',
        'enableMcpTool',
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
      properties: {
        result: { type: 'string' },
      },
    },
  };
}

export const initialData: FlowDocumentJSON = {
  nodes: [
    {
      id: 'start_0',
      type: 'start',
      meta: {
        position: {
          x: 180,
          y: 601.2,
        },
      },
      data: {
        title: 'Start',
        positionType: 'header',
        outputs: {
          type: 'object',
          properties: {
            query: {
              type: 'string',
              default: 'Hello Flow.',
            },
            temperature: {
              type: 'number',
              default: 30,
            },
            enable: {
              type: 'boolean',
              default: true,
            },
            array_obj: {
              type: 'array',
              items: {
                type: 'object',
                properties: {
                  int: {
                    type: 'number',
                  },
                  str: {
                    type: 'string',
                  },
                },
              },
            },
          },
        },
      },
    },
    {
      id: 'transform_0',
      type: 'transform',
      meta: {
        position: {
          x: 2000,
          y: 546.2,
        },
      },
      data: {
        title: 'Transform',
        script: {
          language: 'javascript',
          content: `async function Transform(msg, metadata, msgType, dataType) {\n  return {\n    msg: msg,\n    metadata: metadata,\n    msgType: msgType,\n    dataType: dataType\n  };\n}`,
        },
      },
    },
    {
      id: 'condition_0',
      type: 'condition',
      meta: {
        position: {
          x: 1100,
          y: 546.2,
        },
      },
      data: {
        title: 'Condition',
        conditions: [
          {
            key: 'if_0',
            value: {
              // Case1: msg.temperature >= 20 AND msg.temperature <= 50
              type: 'group',
              operator: 'and',
              children: [
                {
                  left: {
                    type: 'ref',
                    content: ['start_0', 'temperature'],
                  },
                  operator: '>=',
                  right: {
                    type: 'constant',
                    content: 20,
                  },
                },
                {
                  left: {
                    type: 'ref',
                    content: ['start_0', 'temperature'],
                  },
                  operator: '<=',
                  right: {
                    type: 'constant',
                    content: 50,
                  },
                },
              ],
            },
          },
          {
            key: 'if_1',
            value: {
              // Case2: msg.temperature > 50 (ELSE IF)
              left: {
                type: 'ref',
                content: ['start_0', 'temperature'],
              },
              operator: '>',
              right: {
                type: 'constant',
                content: 50,
              },
            },
          },
        ],
      },
    },
    {
      id: 'case_condition_0',
      type: 'case-condition',
      meta: {
        position: {
          x: 1580,
          y: 546.2,
        },
      },
      data: {
        title: '条件列表',
        cases: [
          {
            key: 'case_a',
            groups: [
              {
                operator: 'and',
                rows: [
                  { type: 'expression', content: '' },
                  { type: 'expression', content: '' },
                ],
              },
              {
                operator: 'and',
                rows: [
                  { type: 'expression', content: '' },
                  { type: 'expression', content: '' },
                ],
              },
            ],
          },
          {
            key: 'case_b',
            groups: [
              {
                operator: 'and',
                rows: [{ type: 'expression', content: '' }],
              },
            ],
          },
        ],
      },
    },
    {
      id: 'end_0',
      type: 'end',
      meta: {
        position: {
          x: 2968,
          y: 601.2,
        },
      },
      data: {
        title: 'End',
        positionType: 'tail',
        inputsValues: {
          success: {
            type: 'constant',
            content: true,
            schema: {
              type: 'boolean',
            },
          },
          query: {
            type: 'ref',
            content: ['start_0', 'query'],
          },
        },
        inputs: {
          type: 'object',
          properties: {
            success: {
              type: 'boolean',
            },
            query: {
              type: 'string',
            },
          },
        },
      },
    },
    {
      id: '159623',
      type: 'comment',
      meta: {
        position: {
          x: 180,
          y: 775.2,
        },
      },
      data: {
        size: {
          width: 240,
          height: 150,
        },
        note: 'hi ~\n\nthis is a comment node\n\n- flowgram.ai',
      },
    },
    {
      id: 'http_rDGIH',
      type: 'http',
      meta: {
        position: {
          x: 640,
          y: 421.35,
        },
      },
      data: {
        title: 'HTTP_1',
        outputs: {
          type: 'object',
          properties: {
            body: {
              type: 'string',
            },
            headers: {
              type: 'object',
            },
            statusCode: {
              type: 'integer',
            },
          },
        },
        api: {
          method: 'GET',
          url: {
            type: 'template',
            content: '',
          },
        },
        body: {
          bodyType: 'JSON',
        },
        timeout: {
          timeout: 10000,
          retryTimes: 1,
        },
      },
    },
    {
      id: 'loop_Ycnsk',
      type: 'loop',
      meta: {
        position: {
          x: 1460,
          y: 0,
        },
      },
      data: {
        title: 'Loop_1',
        loopFor: {
          type: 'ref',
          content: ['start_0', 'array_obj'],
        },
        loopOutputs: {
          acm: {
            type: 'ref',
            content: ['llm_6aSyo', 'result'],
          },
        },
        outputs: {
          type: 'object',
          required: [],
          properties: {
            acm: {
              type: 'array',
              items: {
                type: 'string',
              },
            },
          },
        },
      },
      blocks: [
        {
          id: 'llm_6aSyo',
          type: 'ai/agentHarness',
          meta: {
            position: {
              x: 344,
              y: 0,
            },
          },
          data: demoAgentHarnessNodeData({
            title: 'AgentLLM_3',
            userPrompt: '',
            systemPrompt: '# Role\nYou are an AI assistant.\n',
          }),
        },
        {
          id: 'llm_ZqKlP',
          type: 'ai/agentHarness',
          meta: {
            position: {
              x: 804,
              y: 0,
            },
          },
          data: demoAgentHarnessNodeData({
            title: 'AgentLLM_4',
            userPrompt: '',
            systemPrompt: '# Role\nYou are an AI assistant.\n',
          }),
        },
        {
          id: 'block_start_PUDtS',
          type: 'block-start',
          meta: {
            position: {
              x: 32,
              y: 167.1,
            },
          },
          data: {},
        },
        {
          id: 'block_end_leBbs',
          type: 'block-end',
          meta: {
            position: {
              x: 1116,
              y: 167.1,
            },
          },
          data: {},
        },
      ],
      edges: [
        {
          sourceNodeID: 'block_start_PUDtS',
          targetNodeID: 'llm_6aSyo',
        },
        {
          sourceNodeID: 'llm_6aSyo',
          targetNodeID: 'llm_ZqKlP',
        },
        {
          sourceNodeID: 'llm_ZqKlP',
          targetNodeID: 'block_end_leBbs',
        },
      ],
    },
    {
      id: 'group_nYl6D',
      type: 'group',
      meta: {
        position: {
          x: 1624,
          y: 698.2,
        },
      },
      data: {
        parentID: 'root',
        blockIDs: ['llm_8--A3', 'llm_vTyMa'],
      },
    },
    {
      id: 'llm_8--A3',
      type: 'ai/agentHarness',
      meta: {
        position: {
          x: 180,
          y: 0,
        },
      },
      data: demoAgentHarnessNodeData({
        title: 'AgentLLM_1',
        userPrompt: '# User Input\nquery:{{start_0.query}}\nenable:{{start_0.enable}}',
        systemPrompt: '# Role\nYou are an AI assistant.\n',
      }),
    },
    {
      id: 'llm_vTyMa',
      type: 'ai/agentHarness',
      meta: {
        position: {
          x: 640,
          y: 10,
        },
      },
      data: demoAgentHarnessNodeData({
        title: 'AgentLLM_2',
        userPrompt: '# Agent Input\nresult:{{llm_8--A3.result}}',
        systemPrompt: '# Role\nYou are an AI assistant.\n',
      }),
    },
  ],
  edges: [
    {
      sourceNodeID: 'start_0',
      targetNodeID: 'http_rDGIH',
    },
    {
      sourceNodeID: 'http_rDGIH',
      targetNodeID: 'condition_0',
    },
    {
      sourceNodeID: 'condition_0',
      targetNodeID: 'loop_Ycnsk',
      sourcePortID: 'if_0',
    },
    {
      sourceNodeID: 'condition_0',
      targetNodeID: 'llm_8--A3',
      sourcePortID: 'else',
    },
    {
      sourceNodeID: 'llm_vTyMa',
      targetNodeID: 'end_0',
    },
    {
      sourceNodeID: 'loop_Ycnsk',
      targetNodeID: 'end_0',
    },
    {
      sourceNodeID: 'llm_8--A3',
      targetNodeID: 'llm_vTyMa',
    },
  ],
  globalVariable: {
    type: 'object',
    required: [],
    properties: {
      userId: {
        type: 'string',
      },
    },
  },
};
