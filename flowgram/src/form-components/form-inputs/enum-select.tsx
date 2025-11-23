/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import React, { useMemo } from 'react';

import { IFlowValue } from '@flowgram.ai/form-materials';
import { Select } from '@douyinfe/semi-ui';

import { JsonSchema } from '../../typings';

interface EnumSelectProps {
  value?: IFlowValue;
  onChange: (next: IFlowValue) => void;
  readonly?: boolean;
  hasError?: boolean;
  schema: JsonSchema;
}

export const EnumSelect: React.FC<EnumSelectProps> = ({
  value,
  onChange,
  readonly,
  hasError,
  schema,
}) => {
  const items = useMemo(() => {
    const raw = (schema as any).extra?.options || (schema as any).enum || [];
    if (!Array.isArray(raw)) return [];
    // 支持 [{label, value}] 或 原始枚举 ["text", ...]
    return raw.map((v: any) =>
      typeof v === 'object' && v !== null && 'value' in v
        ? { label: String(v.label ?? v.value), value: v.value }
        : { label: String(v), value: v }
    );
  }, [schema]);

  const selectedValue =
    (value?.type === 'constant' ? (value.content as any) : undefined) ?? undefined;

  return (
    <div style={{ width: '100%' }}>
      <Select
        value={selectedValue}
        onChange={(val) => onChange({ type: 'constant', content: val })}
        optionList={items}
        placeholder={readonly ? '只读' : '请选择'}
        disabled={readonly}
        insetLabel={hasError ? '!' : undefined}
        style={{ width: '100%' }}
      />
    </div>
  );
};
