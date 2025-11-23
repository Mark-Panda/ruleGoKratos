/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconCode from '../../assets/icon-script.png';
import { formMeta } from './form-meta';

let index = 0;
const defaultCode = `// 函数签名不可修改
async function Filter(msg, metadata, msgType, dataType) {
  return msg.temperature > 50;
}`;

export const JsFilterNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.JsFilter,
  info: {
    icon: iconCode,
    description: 'JS 表达式，函数签名与入参固定',
  },
  meta: {
    // 设置端口：一个输入，两个输出（success / failed）
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'output', location: 'right', portID: 'True' },
      { type: 'output', location: 'bottom', portID: 'False' },
    ],
    size: {
      width: 360,
      height: 330,
    },
  },
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.JsFilter,
      data: {
        title: `JsFilter_${++index}`,
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
