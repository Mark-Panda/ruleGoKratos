/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { FormMeta, FormRenderProps, ValidateTrigger } from '@flowgram.ai/free-layout-editor';

import { FormHeader, FormContent } from '../../form-components';
import { JsFilterNodeJSON } from './types';
import { JsFilterCode } from './components/code';

export const FormRender = ({ form }: FormRenderProps<JsFilterNodeJSON>) => (
  <>
    <FormHeader />
    <FormContent>
      <JsFilterCode />
    </FormContent>
  </>
);

const SIGNATURE_REG =
  /^(?:\s*async\s+)?function\s+Filter\s*\(\s*msg\s*,\s*metadata\s*,\s*msgType\s*,\s*dataType\s*\)\s*\{/m;

export const formMeta: FormMeta = {
  render: (props) => <FormRender {...props} />,
  validateTrigger: ValidateTrigger.onChange,
  validate: {
    title: ({ value }) => (value ? undefined : 'Title is required'),
    'script.content': ({ value }) => {
      const text = String(value ?? '');
      if (!SIGNATURE_REG.test(text)) {
        return '函数名必须为 Filter，入参必须为 msg, metadata, msgType, dataType';
      }
      return undefined;
    },
  },
  effect: {},
};
