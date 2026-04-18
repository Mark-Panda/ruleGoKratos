/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconDB from '../../assets/icon_database-search.svg';
import { WorkflowNodeType, OutPutPortType } from '../constants';

let index = 0;
export const FileReadNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.FileRead,
  info: {
    icon: iconDB,
    description: '读取本地文件（x/fileRead），支持 Glob 批量；需配置路径白名单保障安全。',
  },
  meta: {
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: { width: 400, height: 260 },
  },
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.FileRead,
      data: {
        title: `读取文件_${++index}`,
        positionType: 'middle',
        inputsValues: {
          path: { type: 'template', content: '/tmp/data.txt' },
          dataType: { type: 'constant', content: 'text' },
          recursive: { type: 'constant', content: false },
        },
        inputs: {
          type: 'object',
          required: ['path', 'dataType'],
          properties: {
            path: {
              type: 'string',
              extra: {
                label: '路径或 Glob',
                formComponent: 'prompt-editor',
              },
            },
            dataType: {
              type: 'string',
              enum: ['text', 'base64'],
              extra: { label: '数据类型', formComponent: 'enum-select' },
            },
            recursive: {
              type: 'boolean',
              extra: { label: 'Glob 时递归子目录' },
            },
          },
        },
      },
    };
  },
};
