/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconCycle from '../../assets/icon_cycle.svg';
import { workspaceSyncFormMeta } from './form-meta';

let index = 0;

export const WorkspaceSyncNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.WorkspaceSync,
  info: {
    icon: iconCycle,
    description:
      '同步「工作区管理」中指定工作区的全部 Git 仓库（与后台「同步仓库」一致），成功走 Success，失败走 Failure。',
  },
  meta: {
    panelCategory: 'project-mgmt',
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'input', location: 'top', portID: 'input_top' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: {
      width: 320,
      height: 260,
    },
    defaultExpanded: false,
    expandable: true,
  },
  formMeta: workspaceSyncFormMeta,
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.WorkspaceSync,
      data: {
        title: `WorkspaceSync_${++index}`,
        positionType: 'middle',
        inputsValues: {
          workspaceId: { type: 'constant', content: '' },
          replaceData: { type: 'constant', content: true },
        },
        inputs: {
          type: 'object',
          required: ['workspaceId'],
          properties: {
            workspaceId: {
              type: 'string',
              extra: {
                label: '工作区 ID',
                description: '由上方下拉选择；也可在 DSL 中改为模板表达式',
              },
            },
            replaceData: {
              type: 'boolean',
              extra: {
                label: '用结果 JSON 替换消息体',
                description: '成功时写入 { ok, workspaceId, message }',
              },
            },
          },
        },
      },
    } as any;
  },
};
