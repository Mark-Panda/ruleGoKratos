/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { Divider, Typography } from '@douyinfe/semi-ui';
import { Field, FormMeta, FormRenderProps } from '@flowgram.ai/free-layout-editor';

import { FlowNodeJSON } from '../../typings';
import { FormContent, FormHeader, FormInputs, OutputsPeek } from '../../form-components';
import { defaultFormMeta } from '../default-form-meta';
import {
  CANVAS_TWO_LINE_BOX_STYLE,
  truncateCanvasText,
} from '../../utils/canvas-node-preview';

const CURSOR_ACP_FIELD_ORDER: readonly string[] = [
  'acpSimpleMode',
  'acpTask',
  'agentPath',
  'apiKey',
  'workspacePath',
  'workDir',
  'replaceData',
  'timeoutMs',
  'log',
  'stdinLines',
  'args',
];

function acpFlowContent(v: unknown): string {
  return String((v as { content?: unknown })?.content ?? '');
}

function CursorAcpCollapsedPreview() {
  return (
    <Field name="inputsValues.acpSimpleMode">
      {({ field: sm }) => (
        <Field name="inputsValues.acpTask">
          {({ field: tk }) => (
            <Field name="inputsValues.agentPath">
              {({ field: ag }) => (
                <Field name="inputsValues.timeoutMs">
                  {({ field: to }) => {
                    const simple =
                      (sm.value as { content?: unknown })?.content !== false ? '简易开' : '简易关';
                    const task = truncateCanvasText(acpFlowContent(tk.value), 44);
                    const agent = truncateCanvasText(acpFlowContent(ag.value), 36);
                    const ms = acpFlowContent(to.value) || '—';
                    const line = `${simple} · ${task} · agent ${agent} · ${ms}ms`;
                    return (
                      <div style={{ margin: '0 10px 6px' }}>
                        <div style={CANVAS_TWO_LINE_BOX_STYLE}>{truncateCanvasText(line, 220)}</div>
                      </div>
                    );
                  }}
                </Field>
              )}
            </Field>
          )}
        </Field>
      )}
    </Field>
  );
}

const renderForm = (_props: FormRenderProps<FlowNodeJSON>) => (
  <>
    <FormHeader />
    <FormContent collapsedPreview={<CursorAcpCollapsedPreview />}>
      <Typography.Paragraph type="tertiary" size="small" style={{ margin: '0 10px 8px' }}>
        简易模式打开时：填写「API 密钥」与「任务说明」即可；引擎会用 API Key 启动 CLI（--api-key），并按官方 ACP 流程发送
        JSON-RPC。关闭简易模式后可自行编辑「stdin JSON-RPC 行」。文档：{' '}
        <a href="https://cursor.com/cn/docs/cli/acp" target="_blank" rel="noreferrer">
          cursor.com/docs/cli/acp
        </a>
      </Typography.Paragraph>
      <Field name="inputsValues.acpSimpleMode">
        {({ field }) => {
          const simple = (field.value as { content?: unknown })?.content !== false;
          return (
            <>
              <FormInputs
                propertyKeyOrder={CURSOR_ACP_FIELD_ORDER}
                propertyFilter={(k) => {
                  if (simple && (k === 'stdinLines' || k === 'args')) return false;
                  return true;
                }}
              />
              {simple ? (
                <Typography.Text
                  type="tertiary"
                  size="small"
                  style={{ display: 'block', margin: '6px 10px 0' }}
                >
                  高级选项（stdin 行与 argv）已隐藏；请先关闭「简易模式」再编辑。
                </Typography.Text>
              ) : (
                <Typography.Paragraph type="warning" size="small" style={{ margin: '8px 10px 0' }}>
                  已关闭简易模式：请填写「stdin JSON-RPC 行」，每行一条 JSON-RPC。
                </Typography.Paragraph>
              )}
            </>
          );
        }}
      </Field>
      <Divider />
      <OutputsPeek />
    </FormContent>
  </>
);

export const cursorAcpFormMeta: FormMeta<FlowNodeJSON> = {
  ...defaultFormMeta,
  render: renderForm,
};
