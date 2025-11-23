/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconDB from '../../assets/icon_database-search.svg';

let index = 0;
export const RedisClientNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.RedisClient,
  info: {
    icon: iconDB,
    description:
      '通过 Redis 客户端执行命令，支持设置服务器、密码、数据库、连接池等参数；命令与参数支持变量。',
  },
  meta: {
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: {
      width: 360,
      height: 360,
    },
  },
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.RedisClient,
      data: {
        title: `Redis操作_${alphaNanoid(5)}`,
        positionType: 'middle',
        inputsValues: {
          server: { type: 'constant', content: '' },
          password: { type: 'constant', content: '' },
          poolSize: { type: 'constant', content: 0 },
          db: { type: 'constant', content: 0 },
          cmd: { type: 'template', content: '' },
          params: { type: 'constant', content: [] },
        },
        inputs: {
          type: 'object',
          required: ['server', 'cmd'],
          properties: {
            server: {
              type: 'string',
              extra: { label: '服务器地址', description: '示例：redis://host:port 或 host:port' },
            },
            password: { type: 'string', extra: { label: '密码' } },
            poolSize: {
              type: 'number',
              extra: {
                label: '连接池大小',
                description: '并发量大时适当增大；留空或 0 使用默认值',
              },
            },
            db: { type: 'number', extra: { label: '数据库编号', description: '默认为 0' } },
            cmd: {
              type: 'string',
              extra: {
                label: '命令',
                formComponent: 'prompt-editor',
                description: '支持使用 ${metadata.key} 或 ${msg.key} 变量进行模板插值',
              },
            },
            params: {
              type: 'array',
              items: { type: 'string' },
              extra: {
                label: '参数列表',
                formComponent: 'array-editor',
                description:
                  '按顺序传入命令参数；支持使用 ${metadata.key} 或 ${msg.key} 变量进行替换',
              },
            },
          },
        },
      },
    } as any;
  },
};
