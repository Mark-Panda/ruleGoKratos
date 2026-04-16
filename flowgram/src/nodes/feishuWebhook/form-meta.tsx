/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { Divider } from '@douyinfe/semi-ui';
import { DisplayOutputs } from '@flowgram.ai/form-materials';
import { Field, FormMeta, FormRenderProps } from '@flowgram.ai/free-layout-editor';

import { FlowNodeJSON } from '../../typings';
import { FormContent, FormHeader, FormInputs } from '../../form-components';
import type { FormInputsProps } from '../../form-components';
import { defaultFormMeta } from '../default-form-meta';

/** 与节点 inputs.properties 键一致，用于侧边栏展示顺序 */
const FEISHU_INPUT_KEY_ORDER = [
  'msgType',
  'webhookUrl',
  'text',
  'postTitle',
  'postBody',
  'postLang',
  'postSplitByLine',
  'postAtAllBefore',
  'postAtAllAfter',
  'postMentionUserIds',
  'interactivePreset',
  'cardNoticeTitle',
  'cardNoticeMarkdown',
  'cardJson',
  'rawJson',
  'timeoutMs',
  'replaceData',
] as const;

function feishuInputVisible(propertyKey: string, msgType: string, interactivePreset: string): boolean {
  const common = new Set(['msgType', 'webhookUrl', 'timeoutMs', 'replaceData']);
  if (common.has(propertyKey)) return true;
  switch (msgType) {
    case 'text':
      return propertyKey === 'text';
    case 'post':
      return (
        propertyKey === 'postTitle' ||
        propertyKey === 'postBody' ||
        propertyKey === 'postLang' ||
        propertyKey === 'postSplitByLine' ||
        propertyKey === 'postAtAllBefore' ||
        propertyKey === 'postAtAllAfter' ||
        propertyKey === 'postMentionUserIds'
      );
    case 'interactive':
      if (propertyKey === 'interactivePreset') return true;
      if (interactivePreset === 'notice_card') {
        return propertyKey === 'cardNoticeTitle' || propertyKey === 'cardNoticeMarkdown';
      }
      return propertyKey === 'cardJson';
    case 'raw':
      return propertyKey === 'rawJson';
    default:
      return false;
  }
}

function truncOneLine(s: string, max: number) {
  const one = s.replace(/\s+/g, ' ').trim();
  if (!one) return '（空）';
  if (one.length <= max) return one;
  return `${one.slice(0, max)}…`;
}

function boolLabel(name: string, fieldName: string) {
  return (
    <Field name={fieldName}>
      {({ field }) => (
        <span style={{ color: '#86909c' }}>
          {name}{' '}
          <strong style={{ color: '#1d2129' }}>
            {(field.value as { content?: unknown })?.content ? '是' : '否'}
          </strong>
        </span>
      )}
    </Field>
  );
}

function FeishuWebhookCollapsedPreview() {
  return (
    <Field name="inputsValues.msgType">
      {({ field: mtField }) => {
        const msgType = String((mtField.value as { content?: unknown })?.content ?? 'text');
        return (
          <Field name="inputsValues.interactivePreset">
            {({ field: ipField }) => {
              const interactivePreset = String(
                (ipField.value as { content?: unknown })?.content ?? 'card_json'
              );
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
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px 12px', marginBottom: 6 }}>
                    <span style={{ color: '#86909c' }}>
                      类型{' '}
                      <strong style={{ color: '#1d2129' }}>{msgType}</strong>
                    </span>
                    {msgType === 'interactive' ? (
                      <span style={{ color: '#86909c' }}>
                        卡片{' '}
                        <strong style={{ color: '#1d2129' }}>{interactivePreset}</strong>
                      </span>
                    ) : null}
                    {msgType === 'post' ? (
                      <span style={{ color: '#86909c' }}>
                        post 语言{' '}
                        <Field name="inputsValues.postLang">
                          {({ field }) => (
                            <strong style={{ color: '#1d2129' }}>
                              {String((field.value as { content?: unknown })?.content ?? 'zh_cn')}
                            </strong>
                          )}
                        </Field>
                      </span>
                    ) : null}
                  </div>
                  {msgType === 'post' ? (
                    <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px 14px', marginBottom: 6 }}>
                      {boolLabel('换行分段', 'inputsValues.postSplitByLine')}
                      {boolLabel('前@全员', 'inputsValues.postAtAllBefore')}
                      {boolLabel('后@全员', 'inputsValues.postAtAllAfter')}
                    </div>
                  ) : null}
                  <Field name="inputsValues.webhookUrl">
                    {({ field }) => (
                      <div style={{ color: '#4e5969', wordBreak: 'break-all', marginBottom: 6 }}>
                        <span style={{ color: '#86909c' }}>Webhook </span>
                        {truncOneLine(String((field.value as { content?: unknown })?.content ?? ''), 120)}
                      </div>
                    )}
                  </Field>
                  {msgType === 'text' ? (
                    <Field name="inputsValues.text">
                      {({ field }) => (
                        <div style={{ color: '#4e5969', wordBreak: 'break-word' }}>
                          <span style={{ color: '#86909c' }}>正文 </span>
                          {truncOneLine(String((field.value as { content?: unknown })?.content ?? ''), 100)}
                        </div>
                      )}
                    </Field>
                  ) : null}
                  {msgType === 'post' ? (
                    <>
                      <Field name="inputsValues.postTitle">
                        {({ field }) => (
                          <div style={{ color: '#4e5969', wordBreak: 'break-word', marginBottom: 4 }}>
                            <span style={{ color: '#86909c' }}>标题 </span>
                            {truncOneLine(String((field.value as { content?: unknown })?.content ?? ''), 72)}
                          </div>
                        )}
                      </Field>
                      <Field name="inputsValues.postBody">
                        {({ field }) => (
                          <div style={{ color: '#4e5969', wordBreak: 'break-word' }}>
                            <span style={{ color: '#86909c' }}>正文 </span>
                            {truncOneLine(String((field.value as { content?: unknown })?.content ?? ''), 80)}
                          </div>
                        )}
                      </Field>
                    </>
                  ) : null}
                  {msgType === 'interactive' && interactivePreset === 'notice_card' ? (
                    <>
                      <Field name="inputsValues.cardNoticeTitle">
                        {({ field }) => (
                          <div style={{ color: '#4e5969', wordBreak: 'break-word', marginBottom: 4 }}>
                            <span style={{ color: '#86909c' }}>通知卡标题 </span>
                            {truncOneLine(String((field.value as { content?: unknown })?.content ?? ''), 64)}
                          </div>
                        )}
                      </Field>
                      <Field name="inputsValues.cardNoticeMarkdown">
                        {({ field }) => (
                          <div style={{ color: '#4e5969', wordBreak: 'break-word' }}>
                            <span style={{ color: '#86909c' }}>Markdown </span>
                            {truncOneLine(String((field.value as { content?: unknown })?.content ?? ''), 80)}
                          </div>
                        )}
                      </Field>
                    </>
                  ) : null}
                  {msgType === 'interactive' && interactivePreset === 'card_json' ? (
                    <Field name="inputsValues.cardJson">
                      {({ field }) => (
                        <div style={{ color: '#4e5969', wordBreak: 'break-all' }}>
                          <span style={{ color: '#86909c' }}>卡片 JSON </span>
                          {truncOneLine(String((field.value as { content?: unknown })?.content ?? ''), 96)}
                        </div>
                      )}
                    </Field>
                  ) : null}
                  {msgType === 'raw' ? (
                    <Field name="inputsValues.rawJson">
                      {({ field }) => (
                        <div style={{ color: '#4e5969', wordBreak: 'break-all' }}>
                          <span style={{ color: '#86909c' }}>raw JSON </span>
                          {truncOneLine(String((field.value as { content?: unknown })?.content ?? ''), 96)}
                        </div>
                      )}
                    </Field>
                  ) : null}
                </div>
              );
            }}
          </Field>
        );
      }}
    </Field>
  );
}

function FeishuWebhookFormInputs() {
  return (
    <Field name="inputsValues.msgType">
      {({ field: mtField }) => {
        const msgType = String((mtField.value as { content?: unknown })?.content ?? 'text');
        return (
          <Field name="inputsValues.interactivePreset">
            {({ field: ipField }) => {
              const interactivePreset = String(
                (ipField.value as { content?: unknown })?.content ?? 'card_json'
              );
              const filter = (k: string) => feishuInputVisible(k, msgType, interactivePreset);
              const formInputsProps: FormInputsProps = {
                propertyFilter: filter,
                propertyKeyOrder: FEISHU_INPUT_KEY_ORDER,
              };
              return <FormInputs {...formInputsProps} />;
            }}
          </Field>
        );
      }}
    </Field>
  );
}

const renderForm = (_props: FormRenderProps<FlowNodeJSON>) => (
  <>
    <FormHeader />
    <FormContent collapsedPreview={<FeishuWebhookCollapsedPreview />}>
      <FeishuWebhookFormInputs />
      <Divider />
      <DisplayOutputs displayFromScope />
    </FormContent>
  </>
);

export const feishuWebhookFormMeta: FormMeta<FlowNodeJSON> = {
  ...defaultFormMeta,
  render: renderForm,
};
