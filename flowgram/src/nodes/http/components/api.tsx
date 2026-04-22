/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { Field } from '@flowgram.ai/free-layout-editor';
import { IFlowTemplateValue, PromptEditorWithVariables } from '@flowgram.ai/form-materials';
import { Select, TextArea } from '@douyinfe/semi-ui';

import { useEffectiveReadonly, useIsSidebar, useNodeRenderContext } from '../../../hooks';
import { VariablePicker } from '../../../form-components/variable-picker';
import { FormItem } from '../../../form-components';

export function Api() {
  const readonly = useEffectiveReadonly();
  const isSidebar = useIsSidebar();
  const { readonly: playgroundReadonly } = useNodeRenderContext();

  return (
    <div>
      <FormItem name="API" required vertical type="string">
        <div style={{ display: 'flex', gap: 5 }}>
          <Field<string> name="api.method" defaultValue="GET">
            {({ field }) => (
              <Select
                value={field.value}
                onChange={(value) => {
                  field.onChange(value as string);
                }}
                style={{ width: 85, maxWidth: 85, minWidth: 85 }}
                size="small"
                disabled={readonly}
                optionList={[
                  { label: 'GET', value: 'GET' },
                  { label: 'POST', value: 'POST' },
                  { label: 'PUT', value: 'PUT' },
                  { label: 'DELETE', value: 'DELETE' },
                  { label: 'PATCH', value: 'PATCH' },
                  { label: 'HEAD', value: 'HEAD' },
                ]}
              />
            )}
          </Field>

          <Field<IFlowTemplateValue> name="api.url">
            {({ field }) => (
              <div style={{ display: 'flex', alignItems: 'center', gap: 6, flex: 1 }}>
                {isSidebar ? (
                  <PromptEditorWithVariables
                    disableMarkdownHighlight
                    readonly={readonly}
                    style={{ flexGrow: 1 }}
                    placeholder="Input URL, use var by '${'"
                    value={field.value}
                    onChange={(value) => {
                      field.onChange(value!);
                    }}
                  />
                ) : (
                  <TextArea
                    style={{ flexGrow: 1 }}
                    value={String((field.value as any)?.content ?? '')}
                    onChange={(value) =>
                      field.onChange({ type: 'template', content: String(value ?? '') } as any)
                    }
                    disabled={playgroundReadonly}
                    autosize={{ minRows: 2, maxRows: 2 }}
                    placeholder="输入 URL（支持变量）"
                  />
                )}
                <VariablePicker
                  size="small"
                  disabled={isSidebar ? readonly : playgroundReadonly}
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
        </div>
      </FormItem>
    </div>
  );
}
