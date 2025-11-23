/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconCode from '../../assets/icon-script.png';
import { formMeta } from './form-meta';

const defaultLua = `-- 函数签名不可修改
function Transform(msg, metadata, msgType, dataType)
  -- 在此处编写 Lua 逻辑
  return msg, metadata, msgType
end`;

export const LuaTransformNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.LuaTransform,
  info: {
    icon: iconCode,
    description: '使用 Lua 脚本转换消息内容，函数签名与入参固定',
  },
  meta: {
    defaultPorts: [
      { type: 'input', location: 'left' },
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
      type: WorkflowNodeType.LuaTransform,
      data: {
        title: `LuaTransform`,
        positionType: 'middle',
        script: {
          language: 'lua',
          content: defaultLua,
        },
      },
    } as any;
  },
  formMeta,
};
