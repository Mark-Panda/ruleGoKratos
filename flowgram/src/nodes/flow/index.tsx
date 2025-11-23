/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { defaultFormMeta } from '../default-form-meta';
import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconVariable from '../../assets/icon-variable.png';

let index = 0;

export const FlowSubChainNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.Flow,
  info: {
    icon: iconVariable,
    description: '子规则链调用',
  },
  meta: {
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: {
      width: 360,
      height: 220,
    },
  },
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.Flow,
      data: {
        title: `子规则链_${++index}`,
        positionType: 'middle',
        inputsValues: {
          targetId: { type: 'constant', content: '' },
          extend: { type: 'constant', content: false },
        },
        inputs: {
          type: 'object',
          required: ['targetId'],
          properties: {
            targetId: {
              type: 'string',
              extra: { formComponent: 'rule-select', label: '子规则链ID' },
            },
            extend: {
              type: 'boolean',
              extra: { label: '继承模式' },
              default: { type: 'constant', content: false } as any,
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
  formMeta: defaultFormMeta,
};
