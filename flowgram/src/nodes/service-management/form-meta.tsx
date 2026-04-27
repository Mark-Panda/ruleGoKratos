/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { Field, FormMeta, FormRenderProps } from '@flowgram.ai/free-layout-editor';
import { Divider } from '@douyinfe/semi-ui';

import { defaultFormMeta } from '../default-form-meta';
import { FlowNodeJSON } from '../../typings';
import { FormContent, FormHeader, FormInputs, OutputsPeek } from '../../form-components';

type ServiceAction = 'create' | 'get' | 'update' | 'delete';

function parseServiceAction(raw: unknown): ServiceAction {
  if (raw === 'create' || raw === 'get' || raw === 'update' || raw === 'delete') {
    return raw;
  }
  return 'create';
}

function isServicePropertyVisible(action: ServiceAction, key: string): boolean {
  if (key === 'action') return true;
  if (key === 'serviceId') return action !== 'create';
  if (['name', 'status', 'volcLogServiceId', 'gitRepoUrl', 'description'].includes(key)) {
    return action === 'create' || action === 'update';
  }
  return true;
}

const renderForm = (_props: FormRenderProps<FlowNodeJSON>) => (
  <Field name="inputsValues.action">
    {({ field }) => {
      const action = parseServiceAction((field.value as { content?: unknown } | undefined)?.content);
      return (
        <>
          <FormHeader />
          <FormContent>
            <FormInputs propertyFilter={(key) => isServicePropertyVisible(action, key)} />
            <Divider />
            <OutputsPeek />
          </FormContent>
        </>
      );
    }}
  </Field>
);

export const formMeta: FormMeta<FlowNodeJSON> = {
  ...defaultFormMeta,
  render: renderForm,
};
