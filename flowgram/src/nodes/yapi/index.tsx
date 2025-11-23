/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconApi from '../../assets/icon_api.svg';
import { formMeta } from './form-meta';

export const YapiNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.Yapi,
  info: {
    icon: iconApi,
    description: 'Yapi 接口调用',
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
      type: WorkflowNodeType.Yapi,
      data: {
        title: 'Yapi 接口',
        yapiConfig: {
          baseUrl: '',
          userName: '',
          password: '',
          interfacePath: '',
          loginType: 'ldap',
        },
      },
    };
  },
  formMeta,
};
