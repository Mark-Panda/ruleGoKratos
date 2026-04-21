/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconLoop from '../../assets/icon-loop.jpg';
import { WorkflowNodeType, OutPutPortType } from '../constants';

let index = 0;
export const WhileNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.While,
  info: {
    icon: iconLoop,
    description:
      'RuleGo while：按条件重复执行目标节点；循环体内需推进状态以免死循环。可与 break 配合终止。',
  },
  meta: {
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'input', location: 'top', portID: 'input_top' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: { width: 380, height: 280 },
  },
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.While,
      data: {
        title: `While_${++index}`,
        positionType: 'middle',
        inputsValues: {
          condition: {
            type: 'template',
            content: 'msg.count==nil || msg.count < 5',
          },
          do: {
            type: 'constant',
            content: '',
          },
        },
        inputs: {
          type: 'object',
          required: ['condition', 'do'],
          properties: {
            condition: {
              type: 'string',
              extra: {
                label: '循环条件',
                formComponent: 'prompt-editor',
                description: 'el 表达式，每次迭代前求值；为真则执行「执行目标」',
              },
            },
            do: {
              type: 'string',
              extra: {
                label: '执行目标',
                formComponent: 'while-do-target',
                description:
                  '可选画布节点（节点 ID）或子规则链（RuleGo：chain:规则链ID）。选「画布上的节点」时优先列出 Success 出线下游。',
                nodeSelectorExcludeSelf: true,
                nodeSelectorPreferSuccessDownstream: true,
                nodeSelectorExcludeTypes: [
                  WorkflowNodeType.Start,
                  WorkflowNodeType.End,
                  WorkflowNodeType.Comment,
                  WorkflowNodeType.BlockStart,
                  WorkflowNodeType.BlockEnd,
                  WorkflowNodeType.Cron,
                ],
              },
            },
          },
        },
      },
    };
  },
};
