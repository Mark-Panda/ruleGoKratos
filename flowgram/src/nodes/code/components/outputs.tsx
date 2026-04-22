/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { Field } from '@flowgram.ai/free-layout-editor';
import { IJsonSchema, JsonSchemaEditor } from '@flowgram.ai/form-materials';
import { Divider } from '@douyinfe/semi-ui';

import { useIsSidebar, useNodeRenderContext } from '../../../hooks';
import { FormItem } from '../../../form-components';
import {
  CANVAS_TWO_LINE_BOX_STYLE,
  canvasSchemaPreviewText,
} from '../../../utils/canvas-node-preview';

export function Outputs() {
  const { readonly } = useNodeRenderContext();
  const isSidebar = useIsSidebar();

  if (!isSidebar) {
    return (
      <>
        <Divider />
        <Field<IJsonSchema> name="outputs">
          {({ field }) => (
            <div style={{ ...CANVAS_TWO_LINE_BOX_STYLE, fontFamily: 'monospace', fontSize: '11px' }}>
              {canvasSchemaPreviewText(field.value, 220)}
            </div>
          )}
        </Field>
      </>
    );
  }

  return (
    <>
      <Divider />
      <FormItem name="outputs" type="object" vertical>
        <Field<IJsonSchema> name="outputs">
          {({ field }) => (
            <JsonSchemaEditor
              readonly={readonly}
              value={field.value}
              onChange={(value) => field.onChange(value)}
            />
          )}
        </Field>
      </FormItem>
    </>
  );
}
