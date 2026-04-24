/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { Field, FormMeta, FormRenderProps } from '@flowgram.ai/free-layout-editor';
import { Divider } from '@douyinfe/semi-ui';

import { defaultFormMeta } from '../default-form-meta';
import { FlowNodeJSON } from '../../typings';
import { FormContent, FormHeader, FormInputs, OutputsPeek } from '../../form-components';
import type { FormInputsProps } from '../../form-components';

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

function feishuInputVisible(
  propertyKey: string,
  msgType: string,
  interactivePreset: string
): boolean {
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
    <FormContent>
      <FeishuWebhookFormInputs />
      <Divider />
      <OutputsPeek />
    </FormContent>
  </>
);

export const feishuWebhookFormMeta: FormMeta<FlowNodeJSON> = {
  ...defaultFormMeta,
  render: renderForm,
};
