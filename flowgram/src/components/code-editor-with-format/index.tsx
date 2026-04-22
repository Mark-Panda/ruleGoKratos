/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { useState } from 'react';

import { TypeScriptCodeEditor } from '@flowgram.ai/form-materials';
import { Button, Modal, Toast, Tooltip } from '@douyinfe/semi-ui';
import { IconCode, IconExpand } from '@douyinfe/semi-icons';

interface CodeEditorWithFormatProps {
  value: string;
  onChange: (value: string) => void;
  readonly?: boolean;
}

/**
 * 简单的 JavaScript 代码格式化函数
 * 使用基本的缩进规则,不依赖外部库
 */
function formatJavaScript(code: string): string {
  try {
    let formatted = '';
    let indentLevel = 0;
    const indentSize = 2;
    const lines = code.split('\n');

    for (let i = 0; i < lines.length; i++) {
      let line = lines[i].trim();

      if (!line) {
        formatted += '\n';
        continue;
      }

      // 减少缩进: }, ], )
      if (line.startsWith('}') || line.startsWith(']') || line.startsWith(')')) {
        indentLevel = Math.max(0, indentLevel - 1);
      }

      // 添加缩进
      formatted += ' '.repeat(indentLevel * indentSize) + line + '\n';

      // 增加缩进: {, [, (
      const openBraces = (line.match(/\{/g) || []).length;
      const closeBraces = (line.match(/\}/g) || []).length;
      const openBrackets = (line.match(/\[/g) || []).length;
      const closeBrackets = (line.match(/\]/g) || []).length;
      const openParens = (line.match(/\(/g) || []).length;
      const closeParens = (line.match(/\)/g) || []).length;

      indentLevel +=
        openBraces - closeBraces + (openBrackets - closeBrackets) + (openParens - closeParens);
      indentLevel = Math.max(0, indentLevel);
    }

    return formatted.trimEnd();
  } catch (error) {
    console.error('格式化失败:', error);
    return code;
  }
}

export function CodeEditorWithFormat({ value, onChange, readonly }: CodeEditorWithFormatProps) {
  const [formatting, setFormatting] = useState(false);
  const [expandOpen, setExpandOpen] = useState(false);

  const handleFormat = () => {
    if (readonly || !value) return;

    setFormatting(true);
    try {
      const formatted = formatJavaScript(value);
      onChange(formatted);
      Toast.success('代码格式化成功');
    } catch (error) {
      Toast.error('代码格式化失败');
      console.error('Format error:', error);
    } finally {
      setFormatting(false);
    }
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
        {!readonly ? (
          <Tooltip content="格式化">
            <Button
              icon={<IconCode />}
              size="small"
              theme="borderless"
              onClick={handleFormat}
              loading={formatting}
              disabled={!value}
              style={{
                background: 'rgba(255, 255, 255, 0.9)',
                boxShadow: '0 1px 4px rgba(0,0,0,0.1)',
              }}
            >
              格式化
            </Button>
          </Tooltip>
        ) : null}
      </div>
      <TypeScriptCodeEditor value={value} onChange={onChange} readonly={readonly} />
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
        <div style={{ position: 'relative', minHeight: 'min(72vh, 720px)' }}>
          {!readonly ? (
            <div
              style={{
                position: 'absolute',
                top: 0,
                right: 0,
                zIndex: 10,
                display: 'flex',
                gap: 6,
              }}
            >
              <Tooltip content="格式化">
                <Button
                  icon={<IconCode />}
                  size="small"
                  theme="borderless"
                  onClick={handleFormat}
                  loading={formatting}
                  disabled={!value}
                  style={{
                    background: 'rgba(255, 255, 255, 0.96)',
                    boxShadow: '0 1px 4px rgba(0,0,0,0.08)',
                  }}
                >
                  格式化
                </Button>
              </Tooltip>
            </div>
          ) : null}
          <div style={{ height: 'min(68vh, 680px)' }}>
            <TypeScriptCodeEditor value={value} onChange={onChange} readonly={readonly} />
          </div>
        </div>
      </Modal>
    </div>
  );
}
