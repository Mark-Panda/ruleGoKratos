/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconCollectLaptop from '../../assets/icon_collect-laptop.svg';

let index = 0;

export const MultiNodeOutputRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.MultiNodeOutput,
  info: {
    icon: iconCollectLaptop,
    description: '获取多个已完成节点的输出信息（多选）',
  },
  meta: {
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: {
      width: 360,
      height: 260,
    },
  },
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.MultiNodeOutput,
      data: {
        title: `获取多节点输出_${++index}`,
        positionType: 'middle',
        inputsValues: {
          nodeIds: { type: 'constant', content: [] },
        },
        inputs: {
          type: 'object',
          required: ['nodeIds'],
          properties: {
            nodeIds: {
              type: 'array',
              items: { type: 'string' },
              extra: { formComponent: 'node-selector-multi', label: '节点列表' },
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
