/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconExec from '../../assets/icon_collect-laptop.svg';
import { WorkflowNodeType, OutPutPortType } from '../constants';

let index = 0;
export const ExecNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.Exec,
  info: {
    icon: iconExec,
    description:
      '执行本地命令（exec）。需在引擎配置命令白名单；可通过 metadata.workDir 指定工作目录。',
  },
  meta: {
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'input', location: 'top', portID: 'input_top' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: { width: 380, height: 340 },
  },
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.Exec,
      data: {
        title: `Exec_${++index}`,
        positionType: 'middle',
        inputsValues: {
          cmd: { type: 'template', content: '' },
          args: { type: 'constant', content: [] as string[] },
          log: { type: 'constant', content: false },
          replaceData: { type: 'constant', content: false },
        },
        inputs: {
          type: 'object',
          required: ['cmd'],
          properties: {
            cmd: {
              type: 'string',
              extra: {
                label: '命令',
                formComponent: 'prompt-editor',
                description: '可含变量；必须在 execNodeWhitelist 中',
              },
            },
            args: {
              type: 'array',
              items: { type: 'string' },
              extra: { label: '参数', formComponent: 'array-editor' },
            },
            log: {
              type: 'boolean',
              extra: { label: '记录标准输出到调试日志' },
            },
            replaceData: {
              type: 'boolean',
              extra: { label: '用 stdout 替换 msg.Data' },
            },
          },
        },
      },
    };
  },
};
