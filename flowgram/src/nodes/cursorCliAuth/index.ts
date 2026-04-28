/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconLaptop from '../../assets/icon_collect-laptop.svg';
import { cursorCliAuthFormMeta } from './form-meta';

let index = 0;

export const CursorCliAuthNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.CursorCliAuth,
  info: {
    icon: iconLaptop,
    description:
      '执行 agent status 判断 Cursor CLI 是否已登录；已登录走 Success，未登录走 Failure。可配置 workspacePath / worktree / force。',
  },
  meta: {
    panelCategory: 'integration-x',
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'input', location: 'top', portID: 'input_top' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: {
      width: 320,
      height: 240,
    },
    defaultExpanded: false,
    expandable: true,
  },
  formMeta: cursorCliAuthFormMeta,
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.CursorCliAuth,
      data: {
        title: `Cursor_CLI_Auth_${++index}`,
        positionType: 'middle',
        inputsValues: {
          agentPath: { type: 'constant', content: 'agent' },
          workspacePath: { type: 'template', content: '$HOME' },
          worktree: { type: 'constant', content: false },
          force: { type: 'constant', content: true },
          workDir: { type: 'template', content: '' },
          timeoutMs: { type: 'constant', content: 15000 },
          replaceData: { type: 'constant', content: true },
        },
        inputs: {
          type: 'object',
          required: ['agentPath', 'workspacePath', 'timeoutMs'],
          properties: {
            agentPath: {
              type: 'string',
              extra: {
                label: 'agent 可执行文件',
                description: '后端仅允许 basename 为 agent 或 cursor',
              },
            },
            workspacePath: {
              type: 'string',
              extra: {
                label: '工作区路径（--workspace）',
                formComponent: 'prompt-editor',
                description: '用于 status 命令上下文；留空回退 $HOME',
              },
            },
            worktree: {
              type: 'boolean',
              extra: {
                label: 'Git Worktree（--worktree）',
              },
            },
            force: {
              type: 'boolean',
              extra: {
                label: '强制允许命令（--force）',
              },
            },
            workDir: {
              type: 'string',
              extra: {
                label: '进程工作目录（cwd）',
                formComponent: 'prompt-editor',
              },
            },
            timeoutMs: {
              type: 'number',
              extra: {
                label: '超时(毫秒)',
              },
            },
            replaceData: {
              type: 'boolean',
              extra: {
                label: '用状态 JSON 替换消息体',
              },
            },
          },
        },
      },
    } as any;
  },
};
