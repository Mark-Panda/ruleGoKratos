/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import React from 'react';

import { IFlowTemplateValue, SQLEditorWithVariables } from '@flowgram.ai/form-materials';

export function SqlTemplateEditor({
  value,
  onChange,
  readonly,
  hasError,
}: {
  value?: IFlowTemplateValue;
  onChange: (val?: IFlowTemplateValue) => void;
  readonly?: boolean;
  hasError?: boolean;
}) {
  const text = typeof value?.content === 'string' ? (value?.content as string) : '';
  return (
    <SQLEditorWithVariables
      value={text}
      onChange={(v) => onChange({ type: 'template', content: v })}
      readonly={readonly}
    />
  );
}
