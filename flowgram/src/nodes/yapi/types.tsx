/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { FlowNodeJSON } from '@flowgram.ai/free-layout-editor';

export interface YapiNodeJSON extends FlowNodeJSON {
  data: {
    title: string;
    yapiConfig: {
      baseUrl: string;
      userName: string;
      password: string;
      interfacePath: string;
      loginType: 'ldap' | 'normal';
    };
  };
}
