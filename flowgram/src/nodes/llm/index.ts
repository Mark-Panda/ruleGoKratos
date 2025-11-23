/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconLLM from '../../assets/icon-llm.jpg';

let index = 0;
export const LLMNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.LLM,
  info: {
    icon: iconLLM,
    description:
      'Call the large language model and use variables and prompt words to generate responses.',
  },
  meta: {
    // 设置端口：一个输入，两个输出（success / failed）
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: {
      width: 360,
      height: 390,
    },
  },
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: 'ai/llm',
      data: {
        title: `LLM_${++index}`,
        positionType: 'middle',
        inputsValues: {
          model: {
            type: 'constant',
            content: 'gpt-3.5-turbo',
          },
          key: {
            type: 'constant',
            content: 'sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',
          },
          url: {
            type: 'constant',
            content: 'https://mock-ai-url/api/v3',
          },
          systemPrompt: {
            type: 'template',
            content: '# Role\nYou are an AI assistant.\n',
          },
          userPrompt: {
            type: 'template',
            content: '',
          },
          temperature: {
            // 采样温度控制输出的随机性。范围 [0.0, 2.0]，值越高越随机
            type: 'constant',
            content: 0.5,
          },
          topP: {
            // 采样方法范围 [0.0,1.0]，从概率最高的前p%候选词中选取 tokens
            type: 'constant',
            content: 0.5,
          },
          maxTokens: {
            // 最大输出长度
            type: 'constant',
            content: null,
          },
          responseFormat: {
            // 输出格式枚举：text/json_object/json_schema（默认text）
            type: 'constant',
            content: 'text',
          },
        },
        inputs: {
          type: 'object',
          required: [
            'model',
            'key',
            'url',
            'temperature',
            'userPrompt',
            'topP',
            'maxTokens',
            'responseFormat',
          ],
          properties: {
            model: {
              type: 'string',
              extra: {
                label: '模型名称',
                // formComponent: 'enum-select',
                // options: [
                //   { label: 'GPT-3.5 通用', value: 'gpt-3.5-turbo' },
                //   { label: 'GPT-4o Mini', value: 'gpt-4o-mini' },
                //   { label: '通义·千问 Plus', value: 'qwen-plus' },
                //   { label: '智谱 GLM-4', value: 'glm-4' },
                //   { label: 'Moonshot 8K', value: 'moonshot-v1-8k' },
                // ],
              },
            },
            key: {
              type: 'string',
            },
            url: {
              type: 'string',
            },
            systemPrompt: {
              type: 'string',
              extra: {
                label: '系统提示词',
                formComponent: 'prompt-editor',
              },
            },
            userPrompt: {
              type: 'string',
              extra: {
                label: '用户提示词',
                formComponent: 'prompt-editor',
              },
            },
            maxTokens: {
              type: 'number',
              extra: {
                label: '最大输出长度',
              },
            },
            responseFormat: {
              type: 'string',
              enum: ['text', 'json_object', 'json_schema'],
              extra: { label: '输出格式', formComponent: 'enum-select' },
            },
            temperature: {
              type: 'number',
            },
            topP: {
              type: 'number',
            },
          },
        },
        outputs: {
          type: 'object',
          properties: {
            // result: { type: 'string' },
          },
        },
      },
    };
  },
};
