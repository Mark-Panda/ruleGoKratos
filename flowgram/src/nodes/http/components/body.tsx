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
import { Select } from '@douyinfe/semi-ui';

import { useNodeRenderContext } from '../../../hooks';
import { VariablePicker } from '../../../form-components/variable-picker';
import { FormItem } from '../../../form-components';

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
  const { readonly } = useNodeRenderContext();

  const renderBodyEditor = (bodyType: string) => {
    switch (bodyType) {
      case 'JSON':
        return (
          <Field<IFlowTemplateValue> name="body.json">
            {({ field }) => (
              <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                <div style={{ flex: 1 }}>
                  <JsonEditorWithVariables
                    value={field.value?.content}
                    readonly={readonly}
                    activeLinePlaceholder="use var by '@'"
                    onChange={(value) => {
                      field.onChange({ type: 'template', content: value });
                    }}
                  />
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
            )}
          </Field>
        );
      case 'raw-text':
        return (
          <Field<IFlowTemplateValue> name="body.rawText">
            {({ field }) => (
              <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
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
