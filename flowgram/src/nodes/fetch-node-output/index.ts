/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconCollectLaptop from '../../assets/icon_collect-laptop.svg';

let index = 0;

export const FetchNodeOutputRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.FetchNodeOutput,
  info: {
    icon: iconCollectLaptop,
    description: '获取已经执行完节点的输出信息',
  },
  meta: {
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: {
      width: 360,
      height: 240,
    },
  },
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.FetchNodeOutput,
      data: {
        title: `获取完成节点信息_${++index}`,
        positionType: 'middle',
        inputsValues: {
          nodeId: { type: 'constant', content: '' },
        },
        inputs: {
          type: 'object',
          required: ['nodeId'],
          properties: {
            nodeId: {
              type: 'string',
              extra: { formComponent: 'node-selector' },
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
