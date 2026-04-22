/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { useEffect, useMemo, useRef, useState } from 'react';

import { JsonCodeEditor } from '@flowgram.ai/form-materials';
import { Button, Toast } from '@douyinfe/semi-ui';

import { tryFormatJsonPretty } from '../../../utils/format-json-pretty';

export function JsonValueEditor({
  value,
  onChange,
}: {
  value: Record<string, unknown>;
  onChange: (value: Record<string, unknown>) => void;
}) {
  const defaultJsonText = useMemo(() => JSON.stringify(value, null, 2), [value]);

  const [jsonText, setJsonText] = useState(defaultJsonText);

  const effectVersion = useRef(0);
  const changeVersion = useRef(0);

  const handleJsonTextChange = (text: string) => {
    setJsonText(text);
    try {
      const jsonValue = JSON.parse(text);
      onChange(jsonValue);
      changeVersion.current++;
    } catch (e) {
      // ignore
    }
  };

  useEffect(() => {
    // more effect compared with change
    effectVersion.current = effectVersion.current + 1;
    if (effectVersion.current === changeVersion.current) {
      return;
    }
    effectVersion.current = changeVersion.current;

    setJsonText(JSON.stringify(value, null, 2));
  }, [value]);

  const handleFormat = () => {
    const r = tryFormatJsonPretty(jsonText);
    if (!r.ok) {
      Toast.warning({ content: `无法格式化：${r.error}` });
      return;
    }
    setJsonText(r.text);
    try {
      const jsonValue = JSON.parse(r.text);
      onChange(jsonValue as Record<string, unknown>);
      changeVersion.current++;
      Toast.success({ content: 'JSON 已格式化' });
    } catch {
      Toast.warning({ content: '格式化后解析失败' });
    }
  };

  return (
    <div style={{ position: 'relative' }}>
      <div style={{ position: 'absolute', top: 8, right: 8, zIndex: 10 }}>
        <Button size="small" type="tertiary" onClick={handleFormat}>
          格式化 JSON
        </Button>
      </div>
      <JsonCodeEditor value={jsonText} onChange={handleJsonTextChange} />
    </div>
  );
}
