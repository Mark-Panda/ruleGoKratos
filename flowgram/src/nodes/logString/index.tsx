/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconLog from '../../assets/icon_log.svg';
import { formMeta } from './form-meta';

let index = 0;
const defaultCode = `// 函数签名不可修改
async function ToString(msg, metadata, msgType, dataType) {
  return 'Incoming message:' + JSON.stringify(msg) + 'Incoming metadata:' + JSON.stringify(metadata);
}`;

export const LogStringNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.LogString,
  info: {
    icon: iconLog,
    description: 'String 消息内容，函数签名与入参固定',
  },
  meta: {
    // 设置端口：一个输入，两个输出（success / failed）
    defaultPorts: [
      { type: 'input', location: 'top' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: {
      width: 360,
      height: 330,
    },
  },
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.LogString,
      data: {
        title: `Log_${++index}`,
        positionType: 'middle',
        script: {
          language: 'javascript',
          content: defaultCode,
        },
      },
    };
  },
  formMeta,
};
