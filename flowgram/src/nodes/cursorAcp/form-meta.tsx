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

function CursorAcpCollapsedPreview() {
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
      <Field name="inputsValues.stdinLines">
        {({ field }) => {
          const raw = (field.value as { content?: unknown })?.content;
          const lines = Array.isArray(raw) ? raw : [];
          const first = lines.length ? String(lines[0]) : '';
          return (
            <div style={{ color: '#4e5969', wordBreak: 'break-all', marginBottom: 6 }}>
              <span style={{ color: '#86909c' }}>stdin 首行 </span>
              {truncOneLine(first, 160)}
              {lines.length > 1 ? (
                <span style={{ color: '#86909c', marginLeft: 4 }}>（共 {lines.length} 行）</span>
              ) : null}
            </div>
          );
        }}
      </Field>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px 14px' }}>
        <Field name="inputsValues.agentPath">
          {({ field }) => (
            <span style={{ color: '#86909c' }}>
              agent{' '}
              <strong style={{ color: '#1d2129' }}>
                {truncOneLine(String((field.value as { content?: unknown })?.content ?? ''), 36)}
              </strong>
            </span>
          )}
        </Field>
        <Field name="inputsValues.timeoutMs">
          {({ field }) => (
            <span style={{ color: '#86909c' }}>
              超时(ms){' '}
              <strong style={{ color: '#1d2129' }}>
                {String((field.value as { content?: unknown })?.content ?? '')}
              </strong>
            </span>
          )}
        </Field>
      </div>
    </div>
  );
}

const renderForm = (_props: FormRenderProps<FlowNodeJSON>) => (
  <>
    <FormHeader />
    <FormContent collapsedPreview={<CursorAcpCollapsedPreview />}>
      <FormInputs />
      <Divider />
      <DisplayOutputs displayFromScope />
    </FormContent>
  </>
);

export const cursorAcpFormMeta: FormMeta<FlowNodeJSON> = {
  ...defaultFormMeta,
  render: renderForm,
};
