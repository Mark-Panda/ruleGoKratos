/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconDB from '../../assets/icon_database-search.svg';
import { WorkflowNodeType, OutPutPortType } from '../constants';

let index = 0;
export const FileWriteNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.FileWrite,
  info: {
    icon: iconDB,
    description: '写入本地文件（x/fileWrite）；content 支持变量替换，默认 ${data}。',
  },
  meta: {
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: { width: 400, height: 280 },
  },
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.FileWrite,
      data: {
        title: `写入文件_${++index}`,
        positionType: 'middle',
        inputsValues: {
          path: { type: 'template', content: '/tmp/data.txt' },
          content: { type: 'template', content: '${data}' },
          append: { type: 'constant', content: false },
        },
        inputs: {
          type: 'object',
          required: ['path'],
          properties: {
            path: {
              type: 'string',
              extra: { label: '文件路径', formComponent: 'prompt-editor' },
            },
            content: {
              type: 'string',
              extra: {
                label: '写入内容',
                formComponent: 'prompt-editor',
                description: '可为 ${data} 或模板',
              },
            },
            append: {
              type: 'boolean',
              extra: { label: '追加写入' },
            },
          },
        },
      },
    };
  },
};
