/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconFork from '../../assets/icon_right-branch.svg';

let index = 0;

export const ForkNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.Fork,
  info: {
    icon: iconFork,
    description: '并行执行多节点',
  },
  meta: {
    // 设置端口：一个输入，多个输出（success / failed）
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
      type: WorkflowNodeType.Fork,
      data: {
        title: `Fork_${++index}`,
        positionType: 'middle',
        // inputs: {
        //   type: 'object',
        //   properties: {},
        // },
        // outputs: {
        //   type: 'object',
        //   properties: {},
        // },
      },
    };
  },
};
