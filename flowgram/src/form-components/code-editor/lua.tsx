/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { useEffect, useState } from 'react';

import { BaseCodeEditor } from '@flowgram.ai/form-materials';
import { languages } from '@flowgram.ai/coze-editor/preset-code';
import { Button, Modal, Tooltip } from '@douyinfe/semi-ui';
import { IconExpand } from '@douyinfe/semi-icons';

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
  const [expandOpen, setExpandOpen] = useState(false);

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

  const editorProps = {
    value,
    onChange,
    languageId: 'lua' as any,
    readonly,
  };

  return (
    <div style={{ position: 'relative' }}>
      <div
        style={{
          position: 'absolute',
          top: 8,
          right: 8,
          zIndex: 10,
          display: 'flex',
          gap: 6,
          alignItems: 'center',
        }}
      >
        <Tooltip content={readonly ? '放大查看' : '放大编辑'}>
          <Button
            icon={<IconExpand />}
            size="small"
            theme="borderless"
            onClick={() => setExpandOpen(true)}
            style={{
              background: 'rgba(255, 255, 255, 0.9)',
              boxShadow: '0 1px 4px rgba(0,0,0,0.1)',
            }}
          />
        </Tooltip>
      </div>
      <BaseCodeEditor {...editorProps} />
      <Modal
        title={readonly ? '放大查看' : '放大编辑'}
        visible={expandOpen}
        onCancel={() => setExpandOpen(false)}
        footer={
          <Button type="primary" onClick={() => setExpandOpen(false)}>
            关闭
          </Button>
        }
        width="min(1200px, 92vw)"
        bodyStyle={{ padding: 12 }}
      >
        <div style={{ height: 'min(68vh, 680px)' }}>
          <BaseCodeEditor {...editorProps} />
        </div>
      </Modal>
    </div>
  );
}
