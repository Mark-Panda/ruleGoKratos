/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconDB from '../../assets/icon_database-search.svg';

let index = 0;
export const FileListNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.FileList,
  info: {
    icon: iconDB,
    description: '列出匹配模式的文件路径（x/fileList），结果为 JSON 数组字符串。',
  },
  meta: {
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'input', location: 'top', portID: 'input_top' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: { width: 400, height: 240 },
  },
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.FileList,
      data: {
        title: `文件列表_${++index}`,
        positionType: 'middle',
        inputsValues: {
          path: { type: 'template', content: '/tmp/*.txt' },
          recursive: { type: 'constant', content: false },
        },
        inputs: {
          type: 'object',
          required: ['path'],
          properties: {
            path: {
              type: 'string',
              extra: {
                label: '路径模式',
                formComponent: 'prompt-editor',
              },
            },
            recursive: {
              type: 'boolean',
              extra: { label: '递归子目录' },
            },
          },
        },
      },
    };
  },
};
