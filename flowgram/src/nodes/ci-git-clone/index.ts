/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconLink from '../../assets/icon_link-one.svg';
import { WorkflowNodeType, OutPutPortType } from '../constants';

let index = 0;
export const CiGitCloneNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.GitClone,
  info: {
    icon: iconLink,
    description: 'ci/gitClone：仓库不存在则 clone，否则 pull。需 rulego-components-ci。',
  },
  meta: {
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: { width: 420, height: 460 },
  },
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.GitClone,
      data: {
        title: `Git拉取_${++index}`,
        positionType: 'middle',
        inputsValues: {
          repository: { type: 'template', content: '' },
          directory: { type: 'template', content: '' },
          reference: { type: 'constant', content: 'refs/heads/main' },
          authType: { type: 'constant', content: 'token' },
          authUser: { type: 'constant', content: '' },
          authPassword: { type: 'template', content: '' },
          authPemFile: { type: 'constant', content: '' },
          proxyUrl: { type: 'constant', content: '' },
          proxyUsername: { type: 'constant', content: '' },
          proxyPassword: { type: 'constant', content: '' },
        },
        inputs: {
          type: 'object',
          properties: {
            repository: {
              type: 'string',
              extra: {
                label: '仓库 URL',
                formComponent: 'prompt-editor',
              },
            },
            directory: {
              type: 'string',
              extra: { label: '本地目录', formComponent: 'prompt-editor' },
            },
            reference: { type: 'string', extra: { label: '分支/引用' } },
            authType: {
              type: 'string',
              enum: ['ssh', 'password', 'token'],
              extra: { label: '认证类型', formComponent: 'enum-select' },
            },
            authUser: { type: 'string', extra: { label: '用户名' } },
            authPassword: {
              type: 'string',
              extra: { label: '密码/token', formComponent: 'prompt-editor' },
            },
            authPemFile: { type: 'string', extra: { label: 'SSH 密钥路径' } },
            proxyUrl: { type: 'string', extra: { label: '代理地址' } },
            proxyUsername: { type: 'string', extra: { label: '代理用户' } },
            proxyPassword: { type: 'string', extra: { label: '代理密码' } },
          },
        },
      },
    };
  },
};
