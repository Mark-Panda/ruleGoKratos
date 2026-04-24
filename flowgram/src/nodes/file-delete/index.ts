/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconDB from '../../assets/icon_database-search.svg';

let index = 0;
export const FileDeleteNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.FileDelete,
  info: {
    icon: iconDB,
    description: '删除文件（x/fileDelete），支持 Glob；高危操作请谨慎配置。',
  },
  meta: {
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'input', location: 'top', portID: 'input_top' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: { width: 380, height: 200 },
  },
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.FileDelete,
      data: {
        title: `删除文件_${++index}`,
        positionType: 'middle',
        inputsValues: {
          path: { type: 'template', content: '/tmp/data.txt' },
        },
        inputs: {
          type: 'object',
          required: ['path'],
          properties: {
            path: {
              type: 'string',
              extra: {
                label: '路径或 Glob',
                formComponent: 'prompt-editor',
              },
            },
          },
        },
      },
    };
  },
};
