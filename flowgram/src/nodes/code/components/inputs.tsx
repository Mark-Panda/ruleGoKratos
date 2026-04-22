/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { Field } from '@flowgram.ai/free-layout-editor';
import { IFlowValue, InputsValues } from '@flowgram.ai/form-materials';

import { useIsSidebar, useNodeRenderContext } from '../../../hooks';
import { FormItem } from '../../../form-components';
import {
  CANVAS_TWO_LINE_BOX_STYLE,
  summarizeFlowValuesRecordCompact,
} from '../../../utils/canvas-node-preview';

export function Inputs() {
  const isSidebar = useIsSidebar();

  const { readonly } = useNodeRenderContext();

  if (!isSidebar) {
    return (
      <Field<Record<string, IFlowValue | undefined> | undefined> name="inputsValues">
        {({ field }) => (
          <div style={CANVAS_TWO_LINE_BOX_STYLE}>{summarizeFlowValuesRecordCompact(field.value)}</div>
        )}
      </Field>
    );
  }

  return (
    <FormItem name="inputs" type="object" vertical>
      <Field<Record<string, IFlowValue | undefined> | undefined> name="inputsValues">
        {({ field }) => (
          <InputsValues
            value={field.value}
            onChange={(value) => field.onChange(value)}
            readonly={readonly}
          />
        )}
      </Field>
    </FormItem>
  );
}
