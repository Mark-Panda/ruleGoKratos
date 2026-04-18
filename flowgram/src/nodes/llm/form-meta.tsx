/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import React, { useEffect, useMemo, useState } from 'react';

import { Divider, Select, Spin, Typography } from '@douyinfe/semi-ui';
import { DisplayOutputs } from '@flowgram.ai/form-materials';
import { Field, FormMeta, FormRenderProps } from '@flowgram.ai/free-layout-editor';

import { FlowNodeJSON } from '../../typings';
import { FormContent, FormHeader, FormInputs } from '../../form-components';
import type { FormInputsProps } from '../../form-components';
import { defaultFormMeta } from '../default-form-meta';
import { listLlmConfigs, type LlmConfigItem } from '../../services/api-agent';

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

function flowNum(v: unknown): number {
  if (v == null || typeof v !== 'object' || !('content' in (v as object))) return 0;
  const n = Number((v as { content?: unknown }).content);
  return Number.isFinite(n) ? n : 0;
}

const llmFormInputsProps: FormInputsProps = {
  propertyFilter: (k) =>
    !['llmConfigId', 'llmModelEntryId', 'model', 'key', 'url'].includes(k),
};

function LlmManagedPanel() {
  const [configs, setConfigs] = useState<LlmConfigItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const rows = await listLlmConfigs();
        if (!cancelled) setConfigs(Array.isArray(rows) ? rows : []);
      } catch {
        if (!cancelled) setConfigs([]);
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const enabledConfigs = useMemo(
    () => configs.filter((c) => c.enabled && Array.isArray(c.models)),
    [configs]
  );

  return (
    <div style={{ margin: '0 10px 12px' }}>
      <Typography.Text strong style={{ display: 'block', marginBottom: 6 }}>
        模型（模型管理）
      </Typography.Text>
      <Typography.Paragraph type="tertiary" size="small" style={{ marginBottom: 10 }}>
        选择启用中的配置及模型；API Key / Base URL 由服务端根据配置注入，无需填写下方旧字段。
      </Typography.Paragraph>
      {loading ? (
        <Spin size="small" />
      ) : enabledConfigs.length === 0 ? (
        <Typography.Text type="warning" size="small">
          暂无可用配置，请先在「Agent 管理 → 模型管理」中添加。
        </Typography.Text>
      ) : (
        <Field name="inputsValues.llmConfigId">
          {({ field: cfgField }) => (
            <Field name="inputsValues.llmModelEntryId">
              {({ field: entField }) => (
                <Field name="inputsValues.model">
                  {({ field: modelField }) => {
                    const cid = flowNum(cfgField.value);
                    const selCfg = enabledConfigs.find((c) => c.id === cid);
                    const models = (selCfg?.models || []).filter((m) => m.enabled);
                    return (
                      <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                        <Select
                          placeholder="LLM 配置"
                          style={{ width: '100%' }}
                          value={cid ? String(cid) : ''}
                          onChange={(v) => {
                            cfgField.onChange({ type: 'constant', content: Number(v) } as any);
                            entField.onChange({ type: 'constant', content: 0 } as any);
                            modelField.onChange({ type: 'constant', content: '' } as any);
                          }}
                        >
                          {enabledConfigs.map((c) => (
                            <Select.Option key={c.id} value={String(c.id)}>
                              {c.name}
                            </Select.Option>
                          ))}
                        </Select>
                        <Select
                          placeholder="模型"
                          style={{ width: '100%' }}
                          value={flowNum(entField.value) ? String(flowNum(entField.value)) : ''}
                          disabled={!cid}
                          onChange={(v) => {
                            const eid = Number(v);
                            entField.onChange({ type: 'constant', content: eid } as any);
                            const row = models.find((m) => m.id === eid);
                            modelField.onChange({
                              type: 'constant',
                              content: row?.modelName ?? '',
                            } as any);
                          }}
                        >
                          {models.map((m) => (
                            <Select.Option key={m.id} value={String(m.id)}>
                              {m.modelName}
                            </Select.Option>
                          ))}
                        </Select>
                      </div>
                    );
                  }}
                </Field>
              )}
            </Field>
          )}
        </Field>
      )}
    </div>
  );
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
      <LlmManagedPanel />
      <Divider />
      <FormInputs {...llmFormInputsProps} />
      <Divider />
      <DisplayOutputs displayFromScope />
    </FormContent>
  </>
);

export const llmFormMeta: FormMeta<FlowNodeJSON> = {
  ...defaultFormMeta,
  render: renderForm,
};
