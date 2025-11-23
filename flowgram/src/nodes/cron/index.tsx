/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconClock from '../../assets/icon_alarm-clock.svg';

let index = 0;

export const CronNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.Cron,
  info: {
    icon: iconClock,
    description: '定时任务：通过 Cron 表达式周期触发工作流',
  },
  meta: {
    nodePanelVisible: true,
    deleteDisable: false,
    copyDisable: true,
    defaultPorts: [{ type: 'output' }],
    size: {
      width: 360,
      height: 160,
    },
  },
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.Cron,
      data: {
        title: `定时任务_${++index}`,
        positionType: 'header',
        inputsValues: {
          cron: {
            type: 'constant',
            content: '*/10 * * * * *',
          },
        },
        inputs: {
          type: 'object',
          required: ['cron'],
          properties: {
            cron: {
              type: 'string',
              extra: {
                label: 'Cron 表达式',
                description: '支持秒级（六位）Quartz 表达式，例如：*/10 * * * * *',
                formComponent: 'cron-editor',
              },
            },
          },
        },
      },
    } as any;
  },
};
