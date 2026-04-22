/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { Divider } from '@douyinfe/semi-ui';
import { Field, FormMeta, FormRenderProps } from '@flowgram.ai/free-layout-editor';

import { FlowNodeJSON } from '../../typings';
import { FormContent, FormHeader, FormInputs, OutputsPeek } from '../../form-components';
import { defaultFormMeta } from '../default-form-meta';
import {
  CANVAS_TWO_LINE_BOX_STYLE,
  truncateCanvasText,
} from '../../utils/canvas-node-preview';

function cliFlowContent(v: unknown): string {
  return String((v as { content?: unknown })?.content ?? '');
}

function CursorCliCollapsedPreview() {
  return (
    <Field name="inputsValues.model">
      {({ field: mo }) => (
        <Field name="inputsValues.outputFormat">
          {({ field: of }) => (
            <Field name="inputsValues.printMode">
              {({ field: pm }) => (
                <Field name="inputsValues.prompt">
                  {({ field: pr }) => {
                    const model = truncateCanvasText(cliFlowContent(mo.value) || 'auto', 28);
                    const fmt = cliFlowContent(of.value) || 'text';
                    const printOn = (pm.value as { content?: unknown })?.content ? '打印开' : '打印关';
                    const task = truncateCanvasText(cliFlowContent(pr.value), 96);
                    const line = `${model} · ${fmt} · ${printOn} · ${task}`;
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
    <FormContent collapsedPreview={<CursorCliCollapsedPreview />}>
      <FormInputs />
      <Divider />
      <OutputsPeek />
    </FormContent>
  </>
);

export const cursorCliFormMeta: FormMeta<FlowNodeJSON> = {
  ...defaultFormMeta,
  render: renderForm,
};
