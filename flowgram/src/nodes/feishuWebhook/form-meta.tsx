/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { Divider } from '@douyinfe/semi-ui';
import { Field, FormMeta, FormRenderProps } from '@flowgram.ai/free-layout-editor';

import { FlowNodeJSON } from '../../typings';
import { FormContent, FormHeader, FormInputs, OutputsPeek } from '../../form-components';
import type { FormInputsProps } from '../../form-components';
import { defaultFormMeta } from '../default-form-meta';
import {
  CANVAS_TWO_LINE_BOX_STYLE,
  truncateCanvasText,
} from '../../utils/canvas-node-preview';

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

function flowContent(v: unknown): string {
  return String((v as { content?: unknown })?.content ?? '');
}

/** 画布折叠态：整块最多两行（CSS clamp + 文本截断） */
function FeishuWebhookCollapsedBody(props: {
  msgType: string;
  interactivePreset: string;
  summaryPrefix: string;
}) {
  const { msgType, interactivePreset, summaryPrefix } = props;
  if (msgType === 'text') {
    return (
      <Field name="inputsValues.text">
        {({ field }) => (
          <div style={{ margin: '0 10px 6px' }}>
            <div style={CANVAS_TWO_LINE_BOX_STYLE}>
              {truncateCanvasText(
                `${summaryPrefix} · ${truncateCanvasText(flowContent(field.value), 80)}`,
                220
              )}
            </div>
          </div>
        )}
      </Field>
    );
  }
  if (msgType === 'post') {
    return (
      <Field name="inputsValues.postTitle">
        {({ field: t }) => (
          <Field name="inputsValues.postBody">
            {({ field: b }) => (
              <div style={{ margin: '0 10px 6px' }}>
                <div style={CANVAS_TWO_LINE_BOX_STYLE}>
                  {truncateCanvasText(
                    `${summaryPrefix} · ${truncateCanvasText(flowContent(t.value), 24)} · ${truncateCanvasText(flowContent(b.value), 48)}`,
                    220
                  )}
                </div>
              </div>
            )}
          </Field>
        )}
      </Field>
    );
  }
  if (msgType === 'interactive' && interactivePreset === 'notice_card') {
    return (
      <Field name="inputsValues.cardNoticeTitle">
        {({ field: t }) => (
          <Field name="inputsValues.cardNoticeMarkdown">
            {({ field: md }) => (
              <div style={{ margin: '0 10px 6px' }}>
                <div style={CANVAS_TWO_LINE_BOX_STYLE}>
                  {truncateCanvasText(
                    `${summaryPrefix} · ${truncateCanvasText(flowContent(t.value), 24)} · ${truncateCanvasText(flowContent(md.value), 48)}`,
                    220
                  )}
                </div>
              </div>
            )}
          </Field>
        )}
      </Field>
    );
  }
  if (msgType === 'interactive' && interactivePreset === 'card_json') {
    return (
      <Field name="inputsValues.cardJson">
        {({ field }) => (
          <div style={{ margin: '0 10px 6px' }}>
            <div style={CANVAS_TWO_LINE_BOX_STYLE}>
              {truncateCanvasText(
                `${summaryPrefix} · ${truncateCanvasText(flowContent(field.value), 72)}`,
                220
              )}
            </div>
          </div>
        )}
      </Field>
    );
  }
  if (msgType === 'raw') {
    return (
      <Field name="inputsValues.rawJson">
        {({ field }) => (
          <div style={{ margin: '0 10px 6px' }}>
            <div style={CANVAS_TWO_LINE_BOX_STYLE}>
              {truncateCanvasText(
                `${summaryPrefix} · ${truncateCanvasText(flowContent(field.value), 72)}`,
                220
              )}
            </div>
          </div>
        )}
      </Field>
    );
  }
  return (
    <div style={{ margin: '0 10px 6px' }}>
      <div style={CANVAS_TWO_LINE_BOX_STYLE}>{truncateCanvasText(summaryPrefix, 220)}</div>
    </div>
  );
}

function FeishuWebhookCollapsedPreview() {
  return (
    <Field name="inputsValues.msgType">
      {({ field: mtField }) => (
        <Field name="inputsValues.interactivePreset">
          {({ field: ipField }) => (
            <Field name="inputsValues.webhookUrl">
              {({ field: wu }) => {
                const msgType = flowContent(mtField.value) || 'text';
                const interactivePreset = flowContent(ipField.value) || 'card_json';
                const wh = truncateCanvasText(flowContent(wu.value), 52);
                const summaryPrefix = `${msgType}${
                  msgType === 'interactive' ? `/${interactivePreset}` : ''
                } · ${wh}`;
                return (
                  <FeishuWebhookCollapsedBody
                    msgType={msgType}
                    interactivePreset={interactivePreset}
                    summaryPrefix={summaryPrefix}
                  />
                );
              }}
            </Field>
          )}
        </Field>
      )}
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
      <OutputsPeek />
    </FormContent>
  </>
);

export const feishuWebhookFormMeta: FormMeta<FlowNodeJSON> = {
  ...defaultFormMeta,
  render: renderForm,
};
