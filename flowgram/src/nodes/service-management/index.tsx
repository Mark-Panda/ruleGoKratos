/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconApi from '../../assets/icon_api.svg';
import { formMeta } from './form-meta';
import { serviceStatusOptions, ServiceStatus } from '../../services/api-service';

let index = 0;
const SERVICE_STATUS_VALUES = serviceStatusOptions.map((o) => o.value);
const SERVICE_ACTION_OPTIONS = [
  { label: '保存服务（按名称）', value: 'save' },
  { label: '获取服务', value: 'get' },
  { label: '删除服务', value: 'delete' },
];

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
          action: { type: 'constant', content: 'save' },
          name: { type: 'template', content: '' },
          status: { type: 'constant', content: ServiceStatus.STOPPED },
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
              enum: ['save', 'get', 'delete'],
              default: { type: 'constant', content: 'save' } as any,
              extra: {
                label: '操作类型',
                formComponent: 'enum-select',
                options: SERVICE_ACTION_OPTIONS,
                description:
                  'save: 按名称保存（存在更新，不存在新建） | get: 获取服务 | delete: 删除服务',
              },
            },
            name: {
              type: 'string',
              extra: {
                label: '服务名称',
                formComponent: 'prompt-editor',
                description: '服务名称（save 时必填）',
              },
            },
            status: {
              type: 'number',
              enum: SERVICE_STATUS_VALUES,
              default: { type: 'constant', content: ServiceStatus.STOPPED } as any,
              extra: {
                label: '服务状态',
                formComponent: 'enum-select',
                options: serviceStatusOptions.map((o) => ({ label: o.label, value: o.value })),
                description: '服务状态（与服务管理一致）',
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
                description: '服务ID（get/delete 时必填）',
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
