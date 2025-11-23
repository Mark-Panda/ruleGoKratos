/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconHTTP from '../../assets/icon-http.svg';
import { formMeta } from './form-meta';

let index = 0;

export const HTTPNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.HTTP,
  info: {
    icon: iconHTTP,
    description: 'Call the HTTP API',
  },
  meta: {
    // 设置端口：一个输入，两个输出（success / failed）
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: {
      width: 360,
      height: 390,
    },
  },
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.HTTP,
      data: {
        title: `HTTP_${++index}`,
        positionType: 'middle',
        api: {
          method: 'GET',
        },
        body: {
          bodyType: 'JSON',
        },
        headers: {},
        params: {},
        outputs: {
          type: 'object',
          properties: {
            // body: { type: 'string' },
            // headers: { type: 'object' },
            // statusCode: { type: 'integer' },
          },
        },
      },
    };
  },
  formMeta: formMeta,
};
