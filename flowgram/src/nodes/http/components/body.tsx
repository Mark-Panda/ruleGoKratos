/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { Field } from '@flowgram.ai/free-layout-editor';
import {
  IFlowTemplateValue,
  JsonEditorWithVariables,
  PromptEditorWithVariables,
} from '@flowgram.ai/form-materials';
import { Button, Select, TextArea, Toast } from '@douyinfe/semi-ui';

import { useEffectiveReadonly, useIsSidebar } from '../../../hooks';
import { VariablePicker } from '../../../form-components/variable-picker';
import { FormItem } from '../../../form-components';
import { tryFormatJsonPretty } from '../../../utils/format-json-pretty';

const BODY_TYPE_OPTIONS = [
  {
    label: 'None',
    value: 'none',
  },
  {
    label: 'JSON',
    value: 'JSON',
  },
  {
    label: 'Raw Text',
    value: 'raw-text',
  },
];

export function Body() {
  const readonly = useEffectiveReadonly();
  const isSidebar = useIsSidebar();

  const renderBodyEditor = (bodyType: string) => {
    switch (bodyType) {
      case 'JSON':
        return (
          <Field<IFlowTemplateValue> name="body.json">
            {({ field }) => {
              const rawJson = String(field.value?.content ?? '');
              const handleFormatJson = () => {
                const r = tryFormatJsonPretty(rawJson);
                if (!r.ok) {
                  Toast.warning({
                    content: `无法格式化：${r.error}。含 @ 变量时请保证整体仍是合法 JSON，或先改为合法 JSON 再格式化。`,
                  });
                  return;
                }
                field.onChange({ type: 'template', content: r.text });
                Toast.success({ content: 'JSON 已格式化' });
              };
              return (
              <div style={{ display: 'flex', alignItems: 'flex-start', gap: 6 }}>
                <div style={{ flex: 1, minWidth: 0 }}>
                  {!readonly && (
                    <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 4 }}>
                      <Button size="small" type="tertiary" onClick={handleFormatJson}>
                        格式化 JSON
                      </Button>
                    </div>
                  )}
                  {isSidebar ? (
                    <JsonEditorWithVariables
                      value={field.value?.content}
                      readonly={readonly}
                      activeLinePlaceholder="use var by '@'"
                      onChange={(value) => {
                        field.onChange({ type: 'template', content: value });
                      }}
                    />
                  ) : (
                    <TextArea
                      value={rawJson}
                      onChange={(value) => {
                        field.onChange({ type: 'template', content: String(value ?? '') });
                      }}
                      disabled={readonly}
                      autosize={{ minRows: 2, maxRows: 2 }}
                      placeholder="输入 JSON（支持变量）"
                      style={{ fontFamily: 'monospace' }}
                    />
                  )}
                </div>
                <VariablePicker
                  size="small"
                  disabled={readonly}
                  onInsert={(text) => {
                    const oldText =
                      typeof field.value?.content === 'string' ? String(field.value?.content) : '';
                    const nextText = oldText ? `${oldText}${text}` : text;
                    field.onChange({ type: 'template', content: nextText } as any);
                  }}
                />
              </div>
              );
            }}
          </Field>
        );
      case 'raw-text':
        return (
          <Field<IFlowTemplateValue> name="body.rawText">
            {({ field }) => (
              <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                {isSidebar ? (
                  <PromptEditorWithVariables
                    disableMarkdownHighlight
                    readonly={readonly}
                    style={{ flexGrow: 1 }}
                    placeholder="Input raw text, use var by '${'"
                    value={field.value}
                    onChange={(value) => {
                      field.onChange(value!);
                    }}
                  />
                ) : (
                  <TextArea
                    value={String((field.value as any)?.content ?? '')}
                    onChange={(value) => {
                      field.onChange({ type: 'template', content: String(value ?? '') } as any);
                    }}
                    disabled={readonly}
                    autosize={{ minRows: 2, maxRows: 2 }}
                    placeholder="输入原始文本（支持变量）"
                  />
                )}
                <VariablePicker
                  size="small"
                  disabled={readonly}
                  onInsert={(text) => {
                    const oldText =
                      typeof (field.value as any)?.content === 'string'
                        ? String((field.value as any)?.content)
                        : '';
                    const nextText = oldText ? `${oldText}${text}` : text;
                    field.onChange({ type: 'template', content: nextText } as any);
                  }}
                />
              </div>
            )}
          </Field>
        );
      default:
        return null;
    }
  };

  return (
    <Field<string> name="body.bodyType" defaultValue="JSON">
      {({ field }) => (
        <div style={{ marginTop: 5 }}>
          <FormItem name="Body" vertical type="object">
            <Select
              value={field.value}
              onChange={(value) => {
                field.onChange(value as string);
              }}
              style={{ width: '100%', marginBottom: 10 }}
              disabled={readonly}
              size="small"
              optionList={BODY_TYPE_OPTIONS}
            />
            {renderBodyEditor(field.value)}
          </FormItem>
        </div>
      )}
    </Field>
  );
}
