/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { Divider } from '@douyinfe/semi-ui';
import { DisplayOutputs } from '@flowgram.ai/form-materials';
import { Field, FormMeta, FormRenderProps } from '@flowgram.ai/free-layout-editor';

import { FlowNodeJSON } from '../../typings';
import { FormContent, FormHeader, FormInputs } from '../../form-components';
import { defaultFormMeta } from '../default-form-meta';

function truncOneLine(s: string, max: number) {
  const one = s.replace(/\s+/g, ' ').trim();
  if (!one) return '（空）';
  if (one.length <= max) return one;
  return `${one.slice(0, max)}…`;
}

function CursorCliCollapsedPreview() {
  return (
    <div
      style={{
        margin: '0 10px 6px',
        padding: '8px',
        borderRadius: 6,
        background: '#f7f8fa',
        fontSize: 12,
        color: '#1d2129',
        lineHeight: 1.55,
      }}
    >
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px 14px', marginBottom: 6 }}>
        <Field name="inputsValues.model">
          {({ field }) => (
            <span style={{ color: '#86909c' }}>
              模型{' '}
              <strong style={{ color: '#1d2129' }}>
                {truncOneLine(String((field.value as { content?: unknown })?.content ?? 'auto'), 28)}
              </strong>
            </span>
          )}
        </Field>
        <Field name="inputsValues.outputFormat">
          {({ field }) => (
            <span style={{ color: '#86909c' }}>
              输出{' '}
              <strong style={{ color: '#1d2129' }}>
                {String((field.value as { content?: unknown })?.content ?? 'text')}
              </strong>
            </span>
          )}
        </Field>
        <Field name="inputsValues.printMode">
          {({ field }) => (
            <span style={{ color: '#86909c' }}>
              打印模式{' '}
              <strong style={{ color: '#1d2129' }}>
                {(field.value as { content?: unknown })?.content ? '开' : '关'}
              </strong>
            </span>
          )}
        </Field>
      </div>
      <Field name="inputsValues.prompt">
        {({ field }) => (
          <div style={{ color: '#4e5969', wordBreak: 'break-word' }}>
            <span style={{ color: '#86909c' }}>任务 </span>
            {truncOneLine(String((field.value as { content?: unknown })?.content ?? ''), 140)}
          </div>
        )}
      </Field>
    </div>
  );
}

const renderForm = (_props: FormRenderProps<FlowNodeJSON>) => (
  <>
    <FormHeader />
    <FormContent collapsedPreview={<CursorCliCollapsedPreview />}>
      <FormInputs />
      <Divider />
      <DisplayOutputs displayFromScope />
    </FormContent>
  </>
);

export const cursorCliFormMeta: FormMeta<FlowNodeJSON> = {
  ...defaultFormMeta,
  render: renderForm,
};
