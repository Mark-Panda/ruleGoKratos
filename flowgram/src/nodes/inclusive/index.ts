/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconSwitch from '../../assets/icon_switch.svg';
import { formMeta } from './form-meta';

export const InclusiveNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.Inclusive,
  info: {
    icon: iconSwitch,
    description:
      '包容分支：评估全部 case 表达式，命中分支可同时路由（RuleGo inclusive）。与 switch 不同，不会短路。',
  },
  meta: {
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'input', location: 'top', portID: 'input_top' },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    useDynamicPort: true,
    expandable: false,
    size: {
      width: 450,
      height: 260,
    },
    wrapperStyle: {
      width: '360px',
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
      type: WorkflowNodeType.Inclusive,
      data: {
        title: '包容分支',
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
