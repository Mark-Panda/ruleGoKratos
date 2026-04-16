/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconLLM from '../../assets/icon-llm.jpg';
import { llmFormMeta } from './form-meta';

let index = 0;
export const LLMNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.LLM,
  info: {
    icon: iconLLM,
    description:
      'Call the large language model and use variables and prompt words to generate responses.',
  },
  meta: {
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: {
      width: 320,
      height: 260,
    },
    defaultExpanded: false,
    expandable: true,
  },
  formMeta: llmFormMeta,
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
          userPrompt: {
            type: 'template',
            content: '',
          },
          systemPrompt: {
            type: 'template',
            content: '# Role\nYou are an AI assistant.\n',
          },
          responseFormat: {
            type: 'constant',
            content: 'text',
          },
          maxTokens: {
            type: 'constant',
            content: null,
          },
          temperature: {
            type: 'constant',
            content: 0.5,
          },
          topP: {
            type: 'constant',
            content: 0.5,
          },
          key: {
            type: 'constant',
            content: 'sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',
          },
          url: {
            type: 'constant',
            content: 'https://mock-ai-url/api/v3',
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
              },
            },
            userPrompt: {
              type: 'string',
              extra: {
                label: '用户提示词',
                formComponent: 'prompt-editor',
              },
            },
            systemPrompt: {
              type: 'string',
              extra: {
                label: '系统提示词',
                formComponent: 'prompt-editor',
              },
            },
            responseFormat: {
              type: 'string',
              enum: ['text', 'json_object', 'json_schema'],
              extra: { label: '输出格式', formComponent: 'enum-select' },
            },
            maxTokens: {
              type: 'number',
              extra: {
                label: '最大输出长度',
              },
            },
            temperature: {
              type: 'number',
            },
            topP: {
              type: 'number',
            },
            key: {
              type: 'string',
            },
            url: {
              type: 'string',
            },
          },
        },
        outputs: {
          type: 'object',
          properties: {},
        },
      },
    };
  },
};
