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

function flowStr(v: unknown): string {
  if (v == null || typeof v !== 'object' || !('content' in (v as object))) return '';
  const c = (v as { content?: unknown }).content;
  return c == null ? '' : String(c);
}

function LlmCollapsedPreview() {
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
              <strong style={{ color: '#1d2129' }}>{truncOneLine(flowStr(field.value), 32)}</strong>
            </span>
          )}
        </Field>
        <Field name="inputsValues.responseFormat">
          {({ field }) => (
            <span style={{ color: '#86909c' }}>
              输出{' '}
              <strong style={{ color: '#1d2129' }}>{flowStr(field.value) || 'text'}</strong>
            </span>
          )}
        </Field>
        <Field name="inputsValues.temperature">
          {({ field }) => (
            <span style={{ color: '#86909c' }}>
              temp{' '}
              <strong style={{ color: '#1d2129' }}>{flowStr(field.value)}</strong>
            </span>
          )}
        </Field>
      </div>
      <Field name="inputsValues.userPrompt">
        {({ field }) => (
          <div style={{ color: '#4e5969', wordBreak: 'break-word', marginBottom: 4 }}>
            <span style={{ color: '#86909c' }}>用户提示 </span>
            {truncOneLine(flowStr(field.value), 140)}
          </div>
        )}
      </Field>
      <Field name="inputsValues.url">
        {({ field }) => (
          <div style={{ color: '#86909c', fontSize: 11, wordBreak: 'break-all' }}>
            URL {truncOneLine(flowStr(field.value), 72)}
          </div>
        )}
      </Field>
    </div>
  );
}

const renderForm = (_props: FormRenderProps<FlowNodeJSON>) => (
  <>
    <FormHeader />
    <FormContent collapsedPreview={<LlmCollapsedPreview />}>
      <FormInputs />
      <Divider />
      <DisplayOutputs displayFromScope />
    </FormContent>
  </>
);

export const llmFormMeta: FormMeta<FlowNodeJSON> = {
  ...defaultFormMeta,
  render: renderForm,
};
