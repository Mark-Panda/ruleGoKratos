/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconSwitch from '../../assets/icon_switch.svg';
import { formMeta } from './form-meta';
import { WorkflowNodeType, OutPutPortType } from '../constants';

export const CaseConditionNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.CaseCondition,
  info: {
    icon: iconSwitch,
    description: '按 CASE 分组（IF / ELSE IF / ELSE）的条件节点，支持 AND/OR 组合。',
  },
  meta: {
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    useDynamicPort: true,
    expandable: false,
    size: {
      width: 420,
      height: 260,
    },
    wrapperStyle: {
      borderRadius: 12,
      border: '1px solid rgba(6, 7, 9, 0.12)',
      boxShadow: '0 4px 12px rgba(0,0,0,0.06)',
      backgroundColor: '#fff',
    },
  },
  formMeta,
  onAdd() {
    const caseId = (i: number) => `case_${i}_${alphaNanoid(4)}`;
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.CaseCondition,
      data: {
        title: '条件列表',
        positionType: 'middle',
        cases: [
          {
            key: caseId(1),
            groups: [
              {
                operator: 'and',
                rows: [{ type: 'expression', content: '' }],
              },
            ],
          },
        ],
      },
    } as any;
  },
};
