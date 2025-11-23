/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { useEffect, useState } from 'react';

import { BaseCodeEditor } from '@flowgram.ai/form-materials';
import { languages } from '@flowgram.ai/coze-editor/preset-code';

export function LuaCodeEditor({
  value,
  onChange,
  readonly,
}: {
  value?: string;
  onChange?: (val: string) => void;
  readonly?: boolean;
}) {
  const [loaded, setLoaded] = useState<boolean>(() => !!languages.get('lua'));

  useEffect(() => {
    if (!loaded) {
      import('@flowgram.ai/coze-editor/language-typescript').then((mod) => {
        // 复用 typescript 语法高亮以支持关键字与函数结构，注册为 lua
        languages.register('lua', (mod as any).typescript);
        setLoaded(true);
      });
    }
  }, [loaded]);

  if (!loaded) return null;

  return (
    <BaseCodeEditor
      value={value}
      onChange={onChange}
      languageId={'lua' as any}
      readonly={readonly}
    />
  );
}
