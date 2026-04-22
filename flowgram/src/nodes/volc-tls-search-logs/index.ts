/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconDB from '../../assets/icon_database-search.svg';

let index = 0;

export const VolcTlsSearchLogsNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.VolcTlsSearchLogs,
  info: {
    icon: iconDB,
    description:
      '调用火山引擎 TLS SearchLogs / SearchLogsV2 检索日志，支持时间窗预设、topic、query、排序与高亮。',
  },
  meta: {
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'input', location: 'top', portID: 'input_top' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: {
      width: 380,
      height: 500,
    },
  },
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.VolcTlsSearchLogs,
      data: {
        title: `TLS日志检索_${++index}`,
        positionType: 'middle',
        inputsValues: {
          endpoint: { type: 'constant', content: '' },
          region: { type: 'constant', content: '' },
          accessKeyId: { type: 'template', content: '' },
          secretAccessKey: { type: 'template', content: '' },
          sessionToken: { type: 'template', content: '' },
          topicId: { type: 'template', content: '' },
          defaultQuery: { type: 'template', content: '*' },
          limit: { type: 'constant', content: 100 },
          useApiV3: { type: 'constant', content: false },
          timeoutSec: { type: 'constant', content: 60 },
          timeRangePreset: { type: 'constant', content: 'last_15m' },
          defaultStartTimeMs: { type: 'constant', content: 0 },
          defaultEndTimeMs: { type: 'constant', content: 0 },
          defaultSort: { type: 'constant', content: 'desc' },
          highLight: { type: 'constant', content: false },
        },
        inputs: {
          type: 'object',
          required: ['region', 'accessKeyId', 'secretAccessKey', 'topicId'],
          properties: {
            endpoint: {
              type: 'string',
              extra: {
                label: 'Endpoint（可选）',
                description: '留空时按 region 自动拼接，如 https://tls.cn-beijing.volces.com',
              },
            },
            region: { type: 'string', extra: { label: 'Region（必填）' } },
            accessKeyId: {
              type: 'string',
              extra: {
                label: 'AccessKeyId',
                formComponent: 'prompt-editor',
              },
            },
            secretAccessKey: {
              type: 'string',
              extra: {
                label: 'SecretAccessKey',
                formComponent: 'prompt-editor',
              },
            },
            sessionToken: {
              type: 'string',
              extra: { label: 'SessionToken（可选）', formComponent: 'prompt-editor' },
            },
            topicId: {
              type: 'string',
              extra: { label: 'TopicId（默认）', formComponent: 'prompt-editor' },
            },
            defaultQuery: {
              type: 'string',
              extra: { label: '默认查询语句', formComponent: 'prompt-editor' },
            },
            limit: { type: 'number', extra: { label: 'Limit（1-500）' } },
            useApiV3: { type: 'boolean', extra: { label: '使用 SearchLogsV2' } },
            timeoutSec: { type: 'number', extra: { label: '超时（秒）' } },
            timeRangePreset: {
              type: 'string',
              enum: [
                'last_15m',
                'last_30m',
                'last_1h',
                'last_6h',
                'last_24h',
                'last_7d',
                'today_local',
                'custom',
              ],
              extra: {
                label: '默认时间窗',
                formComponent: 'enum-select',
              },
            },
            defaultStartTimeMs: {
              type: 'number',
              extra: { label: '自定义开始时间（毫秒）', description: '仅 preset=custom 时生效' },
            },
            defaultEndTimeMs: {
              type: 'number',
              extra: { label: '自定义结束时间（毫秒）', description: '仅 preset=custom 时生效' },
            },
            defaultSort: {
              type: 'string',
              enum: ['desc', 'asc'],
              extra: { label: '默认排序', formComponent: 'enum-select' },
            },
            highLight: { type: 'boolean', extra: { label: '默认开启高亮' } },
          },
        },
      },
    } as any;
  },
};
