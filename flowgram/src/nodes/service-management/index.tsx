/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconApi from '../../assets/icon_api.svg';
import { formMeta } from './form-meta';

let index = 0;

export const ServiceManagementNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.ServiceManagement,
  info: {
    icon: iconApi,
    description:
      '服务管理：创建、查询、更新、删除服务目录中的服务。',
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
      type: WorkflowNodeType.ServiceManagement,
      data: {
        title: `ServiceManagement_${++index}`,
        positionType: 'middle',
        inputsValues: {
          action: { type: 'constant', content: 'create' },
          name: { type: 'template', content: '' },
          status: { type: 'constant', content: 0 },
          volcLogServiceId: { type: 'template', content: '' },
          gitRepoUrl: { type: 'template', content: '' },
          description: { type: 'template', content: '' },
          serviceId: { type: 'constant', content: 0 },
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
                description: 'create: 创建服务 | get: 获取服务 | update: 更新服务 | delete: 删除服务',
              },
            },
            name: {
              type: 'string',
              extra: {
                label: '服务名称',
                formComponent: 'prompt-editor',
                description: '服务名称（create/update 时必填）',
              },
            },
            status: {
              type: 'number',
              extra: {
                label: '服务状态',
                description: '服务状态（0=正常, 1=维护中, 2=已下线）',
              },
            },
            volcLogServiceId: {
              type: 'string',
              extra: {
                label: '火山引擎日志服务ID',
                formComponent: 'prompt-editor',
                description: '火山引擎日志服务ID',
              },
            },
            gitRepoUrl: {
              type: 'string',
              extra: {
                label: 'Git仓库URL',
                formComponent: 'prompt-editor',
                description: 'Git仓库地址',
              },
            },
            description: {
              type: 'string',
              extra: {
                label: '服务描述',
                formComponent: 'prompt-editor',
                description: '服务描述',
              },
            },
            serviceId: {
              type: 'number',
              extra: {
                label: '服务ID',
                description: '服务ID（get/update/delete 时必填）',
              },
            },
          },
        },
        outputs: {
          type: 'object',
          properties: {
            success: { type: 'boolean' },
            service: { type: 'object' },
            message: { type: 'string' },
          },
        },
      },
    } as any;
  },
};
