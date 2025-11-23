/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { FormMeta, FormRenderProps, ValidateTrigger } from '@flowgram.ai/free-layout-editor';

import { FormHeader, FormContent } from '../../form-components';
import { YapiNodeJSON } from './types';
import { YapiConfig } from './components/yapi-config';

export const FormRender = ({ form }: FormRenderProps<YapiNodeJSON>) => (
  <>
    <FormHeader />
    <FormContent>
      <YapiConfig />
    </FormContent>
  </>
);

export const formMeta: FormMeta = {
  render: (props) => <FormRender {...props} />,
  validateTrigger: ValidateTrigger.onChange,
  validate: {
    title: ({ value }) => (value ? undefined : '标题不能为空'),
    'yapiConfig.baseUrl': ({ value }) => (value ? undefined : 'Base URL 不能为空'),
    'yapiConfig.userName': ({ value }) => (value ? undefined : '用户名不能为空'),
    'yapiConfig.password': ({ value }) => (value ? undefined : '密码不能为空'),
    'yapiConfig.interfacePath': ({ value }) => (value ? undefined : '接口路径不能为空'),
  },
};
