/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconCondition from '../../assets/icon-condition.svg';
import { formMeta } from './form-meta';
import { WorkflowNodeType } from '../constants';

export const ConditionNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.Condition,
  info: {
    icon: iconCondition,
    description:
      'Connect multiple downstream branches. Only the corresponding branch will be executed if the set conditions are met.',
  },
  meta: {
    defaultPorts: [{ type: 'input' }],
    // Condition Outputs use dynamic port
    useDynamicPort: true,
    expandable: false, // disable expanded
    size: {
      width: 360,
      height: 210,
    },
  },
  formMeta,
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: 'condition',
      data: {
        title: 'Condition',
        positionType: 'middle',
        conditions: [
          {
            key: `if_${alphaNanoid(5)}`,
            value: {},
          },
          {
            key: `if_${alphaNanoid(5)}`,
            value: {},
          },
        ],
      },
    };
  },
};
