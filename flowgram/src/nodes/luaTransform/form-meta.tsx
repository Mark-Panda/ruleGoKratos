/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { FormMeta, FormRenderProps, ValidateTrigger } from '@flowgram.ai/free-layout-editor';

import { FormHeader, FormContent } from '../../form-components';
import { LuaTransformCode } from './components/code';

export interface LuaTransformNodeJSON {
  data: {
    title: string;
    script: {
      language: 'lua';
      content: string;
    };
  };
}

export const FormRender = ({ form }: FormRenderProps<LuaTransformNodeJSON>) => (
  <>
    <FormHeader />
    <FormContent>
      <LuaTransformCode />
    </FormContent>
  </>
);

const SIGNATURE_START =
  /\bfunction\s+Transform\s*\(\s*msg\s*,\s*metadata\s*,\s*msgType\s*,\s*dataType\s*\)\s*/m;
const SIGNATURE_END = /\bend\s*$/m;

export const formMeta: FormMeta = {
  render: (props) => <FormRender {...props} />,
  validateTrigger: ValidateTrigger.onChange,
  validate: {
    title: ({ value }) => (value ? undefined : 'Title is required'),
    'script.content': ({ value }) => {
      const text = String(value ?? '');
      if (!SIGNATURE_START.test(text) || !SIGNATURE_END.test(text)) {
        return '函数签名必须为 function Transform(...) 开始并以 end 结束，且不可修改';
      }
      return undefined;
    },
  },
  effect: {},
};
