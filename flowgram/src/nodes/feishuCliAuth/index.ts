/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconLink from '../../assets/icon_link-one.svg';
import { feishuCliAuthFormMeta } from './form-meta';

let index = 0;

export const FeishuCliAuthNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.FeishuCliAuth,
  info: {
    icon: iconLink,
    description:
      '执行 lark-cli auth status 判断飞书 CLI 是否已授权；已授权走 Success，未授权走 Failure。',
  },
  meta: {
    panelCategory: 'service-calls',
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'input', location: 'top', portID: 'input_top' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: {
      width: 320,
      height: 240,
    },
    defaultExpanded: false,
    expandable: true,
  },
  formMeta: feishuCliAuthFormMeta,
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.FeishuCliAuth,
      data: {
        title: `Feishu_CLI_Auth_${++index}`,
        positionType: 'middle',
        inputsValues: {
          cliPath: { type: 'constant', content: 'lark-cli' },
          args: { type: 'constant', content: ['auth', 'status'] },
          workDir: { type: 'template', content: '' },
          timeoutMs: { type: 'constant', content: 15000 },
          replaceData: { type: 'constant', content: true },
        },
        inputs: {
          type: 'object',
          required: ['cliPath', 'timeoutMs'],
          properties: {
            cliPath: {
              type: 'string',
              extra: {
                label: 'CLI 可执行文件',
                description: '后端仅允许 basename 为 lark-cli 或 lark',
              },
            },
            args: {
              type: 'array',
              items: { type: 'string' },
              extra: {
                label: '命令参数',
                formComponent: 'array-editor',
                description: '默认 ["auth","status"]',
              },
            },
            workDir: {
              type: 'string',
              extra: {
                label: '进程工作目录（cwd）',
                formComponent: 'prompt-editor',
              },
            },
            timeoutMs: {
              type: 'number',
              extra: {
                label: '超时(毫秒)',
              },
            },
            replaceData: {
              type: 'boolean',
              extra: {
                label: '用状态 JSON 替换消息体',
              },
            },
          },
        },
      },
    } as any;
  },
};
