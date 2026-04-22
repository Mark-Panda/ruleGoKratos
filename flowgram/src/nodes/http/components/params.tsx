/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { Field } from '@flowgram.ai/free-layout-editor';
import { IFlowValue, InputsValues } from '@flowgram.ai/form-materials';

import { useEffectiveReadonly, useIsSidebar } from '../../../hooks';
import { FormItem } from '../../../form-components';
import {
  CANVAS_TWO_LINE_BOX_STYLE,
  summarizeFlowValuesRecordCompact,
} from '../../../utils/canvas-node-preview';

export function Params() {
  const readonly = useEffectiveReadonly();
  const isSidebar = useIsSidebar();

  if (!isSidebar) {
    return (
      <FormItem name="params" type="object" vertical>
        <Field<Record<string, IFlowValue | undefined> | undefined> name="paramsValues">
          {({ field }) => (
            <div style={CANVAS_TWO_LINE_BOX_STYLE}>{summarizeFlowValuesRecordCompact(field.value)}</div>
          )}
        </Field>
      </FormItem>
    );
  }

  return (
    <FormItem name="params" type="object" vertical>
      <Field<Record<string, IFlowValue | undefined> | undefined> name="paramsValues">
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
