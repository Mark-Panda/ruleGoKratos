/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { FlowNodeJSON } from '@flowgram.ai/free-layout-editor';
import { IJsonSchema } from '@flowgram.ai/form-materials';

export interface TransformNodeJSON extends FlowNodeJSON {
  data: {
    title: string;
    script: {
      language: 'javascript';
      content: string;
    };
  };
}
