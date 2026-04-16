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

function flowBool(v: unknown): boolean {
  if (v == null || typeof v !== 'object' || !('content' in (v as object))) return false;
  return (v as { content?: unknown }).content === true;
}

function AgentHarnessCollapsedPreview() {
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
      <Field name="inputsValues.model">
        {({ field }) => (
          <div style={{ marginBottom: 6 }}>
            <span style={{ color: '#86909c' }}>模型 </span>
            <strong style={{ color: '#1d2129' }}>{truncOneLine(flowStr(field.value), 40)}</strong>
          </div>
        )}
      </Field>
      <Field name="inputsValues.userPrompt">
        {({ field }) => (
          <div style={{ color: '#4e5969', wordBreak: 'break-word', marginBottom: 6 }}>
            <span style={{ color: '#86909c' }}>用户提示 </span>
            {truncOneLine(flowStr(field.value), 120)}
          </div>
        )}
      </Field>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px 10px', fontSize: 11, color: '#86909c' }}>
        <Field name="inputsValues.enableSkillTool">
          {({ field }) => <span>Skill {flowBool(field.value) ? '开' : '关'}</span>}
        </Field>
        <Field name="inputsValues.enableMcpTool">
          {({ field }) => <span>MCP {flowBool(field.value) ? '开' : '关'}</span>}
        </Field>
        <Field name="inputsValues.enableUUIDTool">
          {({ field }) => <span>UUID {flowBool(field.value) ? '开' : '关'}</span>}
        </Field>
        <Field name="inputsValues.enableWorkspaceTools">
          {({ field }) => <span>WS {flowBool(field.value) ? '开' : '关'}</span>}
        </Field>
        <Field name="inputsValues.maxIterations">
          {({ field }) => <span>迭代 {flowStr(field.value) || '0'}</span>}
        </Field>
      </div>
    </div>
  );
}

const renderForm = (_props: FormRenderProps<FlowNodeJSON>) => (
  <>
    <FormHeader />
    <FormContent collapsedPreview={<AgentHarnessCollapsedPreview />}>
      <FormInputs />
      <Divider />
      <DisplayOutputs displayFromScope />
    </FormContent>
  </>
);

export const agentHarnessFormMeta: FormMeta<FlowNodeJSON> = {
  ...defaultFormMeta,
  render: renderForm,
};
