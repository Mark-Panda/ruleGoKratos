/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconDB from '../../assets/icon_database-search.svg';

let index = 0;

export const OpenSearchSearchNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.OpenSearchSearch,
  info: {
    icon: iconDB,
    description:
      '调用 OpenSearch / Elasticsearch 的 _search 接口检索日志，支持 endpoint/index 模板、默认查询体、searchType、认证与超时配置。',
  },
  meta: {
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'input', location: 'top', portID: 'input_top' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: {
      width: 360,
      height: 420,
    },
  },
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.OpenSearchSearch,
      data: {
        title: `OpenSearch检索_${++index}`,
        positionType: 'middle',
        inputsValues: {
          endpoint: { type: 'template', content: '' },
          index: { type: 'template', content: '' },
          username: { type: 'constant', content: '' },
          password: { type: 'constant', content: '' },
          insecureSkipVerify: { type: 'constant', content: false },
          timeoutSec: { type: 'constant', content: 60 },
          searchType: { type: 'constant', content: 'query_then_fetch' },
          ignoreUnavailable: { type: 'constant', content: false },
          defaultSearchBody: {
            type: 'template',
            content:
              '{"size":100,"sort":[{"@timestamp":{"order":"desc"}}],"query":{"match_all":{}}}',
          },
        },
        inputs: {
          type: 'object',
          required: ['endpoint', 'index'],
          properties: {
            endpoint: {
              type: 'string',
              extra: {
                label: 'Endpoint',
                formComponent: 'prompt-editor',
                description: 'OpenSearch 地址，例如 https://opensearch:9200，支持 ${metadata.xxx}',
              },
            },
            index: {
              type: 'string',
              extra: {
                label: '索引',
                formComponent: 'prompt-editor',
                description: '支持单索引、多索引（逗号分隔）和通配符，例如 logs-*',
              },
            },
            username: { type: 'string', extra: { label: '用户名（可选）' } },
            password: { type: 'string', extra: { label: '密码（可选）' } },
            insecureSkipVerify: {
              type: 'boolean',
              extra: { label: '跳过 TLS 证书校验', description: '仅测试环境建议开启' },
            },
            timeoutSec: { type: 'number', extra: { label: '超时（秒）' } },
            searchType: {
              type: 'string',
              enum: ['query_then_fetch', 'dfs_query_then_fetch'],
              extra: { label: 'search_type', formComponent: 'enum-select' },
            },
            ignoreUnavailable: {
              type: 'boolean',
              extra: { label: '忽略不可用索引（ignore_unavailable）' },
            },
            defaultSearchBody: {
              type: 'string',
              extra: {
                label: '默认查询体（JSON）',
                formComponent: 'prompt-editor',
                jsonFormat: true,
                description:
                  '当消息体为空时使用；若消息体是普通对象，会与该默认查询体合并后请求 OpenSearch。',
              },
            },
          },
        },
      },
    } as any;
  },
};
