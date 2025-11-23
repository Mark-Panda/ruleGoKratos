/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import React, { useMemo } from 'react';

import { IFlowValue } from '@flowgram.ai/form-materials';
import { Button, Input } from '@douyinfe/semi-ui';
import { IconPlus, IconMinus } from '@douyinfe/semi-icons';

export function ArrayEditor({
  value,
  onChange,
  readonly,
  hasError,
}: {
  value: IFlowValue | undefined;
  onChange: (val: IFlowValue) => void;
  readonly?: boolean;
  hasError?: boolean;
}) {
  const items = useMemo(() => {
    const arr = Array.isArray(value?.content) ? (value?.content as unknown[]) : [];
    return arr.map((v) => (typeof v === 'string' ? v : String(v ?? '')));
  }, [value]);

  const setItems = (next: string[]) => {
    onChange({ type: 'constant', content: next });
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      {items.map((item, idx) => (
        <div key={idx} style={{ display: 'flex', gap: 8 }}>
          <Input
            value={item}
            onChange={(val) => {
              const next = [...items];
              next[idx] = String(val ?? '');
              setItems(next);
            }}
            disabled={readonly}
          />
          {!readonly && (
            <Button
              icon={<IconMinus />}
              type="tertiary"
              onClick={() => {
                const next = items.filter((_, i) => i !== idx);
                setItems(next);
              }}
            />
          )}
        </div>
      ))}
      {!readonly && (
        <Button icon={<IconPlus />} type="secondary" onClick={() => setItems([...items, ''])}>
          添加参数
        </Button>
      )}
    </div>
  );
}
