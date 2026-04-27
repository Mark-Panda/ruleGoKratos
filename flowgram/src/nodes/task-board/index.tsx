/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconRouter from '../../assets/icon_router.svg';
import { formMeta } from './form-meta';

let index = 0;

export const TaskBoardNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.TaskBoard,
  info: {
    icon: iconRouter,
    description:
      '任务看板：创建、查询、更新、删除任务看板中的任务。',
  },
  meta: {
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'input', location: 'top', portID: 'input_top' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: {
      width: 320,
      height: 360,
    },
    defaultExpanded: false,
    expandable: true,
  },
  formMeta,
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.TaskBoard,
      data: {
        title: `TaskBoard_${++index}`,
        positionType: 'middle',
        inputsValues: {
          action: { type: 'constant', content: 'create' },
          name: { type: 'template', content: '' },
          priority: { type: 'constant', content: 0 },
          taskType: { type: 'constant', content: 0 },
          handlerUserId: { type: 'template', content: '' },
          description: { type: 'template', content: '' },
          taskId: { type: 'constant', content: 0 },
          status: { type: 'constant', content: 0 },
        },
        inputs: {
          type: 'object',
          required: ['action'],
          properties: {
            action: {
              type: 'string',
              enum: ['create', 'get', 'update', 'delete'],
              default: { type: 'constant', content: 'create' } as any,
              extra: {
                label: '操作类型',
                formComponent: 'enum-select',
                description: 'create: 创建任务 | get: 获取任务 | update: 更新任务 | delete: 删除任务',
              },
            },
            name: {
              type: 'string',
              extra: {
                label: '任务名称',
                formComponent: 'prompt-editor',
                description: '任务名称（create/update 时必填）',
              },
            },
            priority: {
              type: 'number',
              extra: {
                label: '优先级',
                description: '优先级（0=无, 1=低, 2=中, 3=高）',
              },
            },
            taskType: {
              type: 'number',
              extra: {
                label: '任务类型',
                description: '任务类型（0=普通, 1=紧急, 2=重要）',
              },
            },
            handlerUserId: {
              type: 'string',
              extra: {
                label: '处理人用户ID',
                formComponent: 'prompt-editor',
                description: '处理人的用户ID',
              },
            },
            description: {
              type: 'string',
              extra: {
                label: '任务描述',
                formComponent: 'prompt-editor',
                description: '任务描述',
              },
            },
            taskId: {
              type: 'number',
              extra: {
                label: '任务ID',
                description: '任务ID（get/update/delete 时必填）',
              },
            },
            status: {
              type: 'number',
              extra: {
                label: '状态',
                description: '状态（0=待处理, 1=进行中, 2=已完成）',
              },
            },
          },
        },
        outputs: {
          type: 'object',
          properties: {
            success: { type: 'boolean' },
            task: { type: 'object' },
            message: { type: 'string' },
          },
        },
      },
    } as any;
  },
};
